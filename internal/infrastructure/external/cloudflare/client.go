package cloudflare

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"time"

	dnsdto "github.com/leoninew/pomelo-orbit/internal/application/dns/dto"
	dnsport "github.com/leoninew/pomelo-orbit/internal/application/dns/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var _ dnsport.Provider = (*Client)(nil)

type Client struct {
	baseUrl string
	http    *http.Client
}

func New() *Client {
	return &Client{baseUrl: "https://api.cloudflare.com/client/v4", http: &http.Client{
		Timeout:       15 * time.Second,
		CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse },
	}}
}

type envelope[T any] struct {
	Success    bool `json:"success"`
	Result     T    `json:"result"`
	ResultInfo struct {
		TotalCount int `json:"total_count"`
		TotalPages int `json:"total_pages"`
	} `json:"result_info"`
}

type zone struct {
	Id   string `json:"id"`
	Name string `json:"name"`
}

type record struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	Content   string `json:"content"`
	TTL       int    `json:"ttl"`
	Proxied   bool   `json:"proxied"`
	Proxiable bool   `json:"proxiable"`
}

func (r record) view() dnsdto.Record {
	return dnsdto.Record{Id: r.Id, Name: r.Name, Type: r.Type, Content: r.Content, TTL: r.TTL, Proxied: r.Proxied, Proxiable: r.Proxiable}
}

func (c *Client) request(ctx context.Context, token, method, path string, body any, output any) error {
	var payload []byte
	if body != nil {
		var err error
		payload, err = json.Marshal(body)
		if err != nil {
			return apperror.New(apperror.KindInternal, "Failed to encode Cloudflare request")
		}
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseUrl+path, bytes.NewReader(payload))
	if err != nil {
		return apperror.New(apperror.KindInternal, "Failed to create Cloudflare request")
	}
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := c.http.Do(req)
	if err != nil {
		return apperror.New(apperror.KindUnavailable, "Cloudflare request failed or timed out; refresh records to check the current state before retrying")
	}
	defer func() { _ = resp.Body.Close() }()
	switch resp.StatusCode {
	case http.StatusUnauthorized, http.StatusForbidden:
		return apperror.NewWithCode(apperror.KindValidation, "dns_permission_denied", "Cloudflare token is invalid or lacks Zone Read / DNS Edit permissions for the selected Zone")
	case http.StatusNotFound:
		return apperror.New(apperror.KindNotFound, "Cloudflare Zone or DNS record was not found")
	case http.StatusTooManyRequests:
		return apperror.New(apperror.KindRateLimited, "Cloudflare rate limit reached; try again later")
	}
	if resp.StatusCode >= 500 {
		return apperror.New(apperror.KindUnavailable, "Cloudflare is temporarily unavailable")
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return apperror.New(apperror.KindValidation, "Cloudflare rejected the request; check the record values and Zone permissions (30-second TTL requires Enterprise)")
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 4<<20)).Decode(output); err != nil {
		return apperror.New(apperror.KindUnavailable, "Cloudflare returned an invalid response; refresh records before retrying")
	}
	return nil
}

func (c *Client) ListZones(ctx context.Context, token string) ([]dnsdto.Zone, error) {
	items := make([]dnsdto.Zone, 0)
	for page := 1; ; page++ {
		var response envelope[[]zone]
		if err := c.request(ctx, token, http.MethodGet, "/zones?per_page=50&page="+strconv.Itoa(page), nil, &response); err != nil {
			return nil, err
		}
		if !response.Success {
			return nil, unsuccessfulResponse()
		}
		for _, item := range response.Result {
			items = append(items, dnsdto.Zone{Id: item.Id, Name: item.Name})
		}
		if page >= response.ResultInfo.TotalPages {
			return items, nil
		}
	}
}

func (c *Client) ListARecords(ctx context.Context, token, zoneId, search string, page, perPage int) (repository.Page[dnsdto.Record], error) {
	query := url.Values{"type": {"A"}, "page": {strconv.Itoa(page)}, "per_page": {strconv.Itoa(perPage)}, "order": {"name"}, "direction": {"asc"}}
	if search != "" {
		query.Set("name.contains", search)
	}
	var response envelope[[]record]
	if err := c.request(ctx, token, http.MethodGet, "/zones/"+zoneId+"/dns_records?"+query.Encode(), nil, &response); err != nil {
		return repository.Page[dnsdto.Record]{}, err
	}
	if !response.Success {
		return repository.Page[dnsdto.Record]{}, unsuccessfulResponse()
	}
	items := make([]dnsdto.Record, 0, len(response.Result))
	for _, item := range response.Result {
		if item.Type != "A" {
			return repository.Page[dnsdto.Record]{}, apperror.New(apperror.KindUnavailable, "Cloudflare returned an unexpected record type")
		}
		items = append(items, item.view())
	}
	return repository.Page[dnsdto.Record]{Items: items, Total: response.ResultInfo.TotalCount, Page: page, PerPage: perPage}, nil
}

func (c *Client) CreateARecord(ctx context.Context, token, zoneId string, input dnsdto.CreateRecordInput) (dnsdto.Record, error) {
	body := struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Content string `json:"content"`
		TTL     int    `json:"ttl"`
		Proxied bool   `json:"proxied"`
	}{input.Name, "A", input.Content, input.TTL, input.Proxied}
	var response envelope[record]
	if err := c.request(ctx, token, http.MethodPost, "/zones/"+zoneId+"/dns_records", body, &response); err != nil {
		return dnsdto.Record{}, err
	}
	if !response.Success {
		return dnsdto.Record{}, unsuccessfulResponse()
	}
	return response.Result.view(), nil
}

func (c *Client) DeleteARecord(ctx context.Context, token, zoneId, recordId string) error {
	var response envelope[struct {
		Id string `json:"id"`
	}]
	if err := c.request(ctx, token, http.MethodDelete, "/zones/"+zoneId+"/dns_records/"+recordId, nil, &response); err != nil {
		return err
	}
	if !response.Success {
		return unsuccessfulResponse()
	}
	return nil
}

func (c *Client) Record(ctx context.Context, token, zoneId, recordId string) (dnsdto.Record, error) {
	var response envelope[record]
	if err := c.request(ctx, token, http.MethodGet, "/zones/"+zoneId+"/dns_records/"+recordId, nil, &response); err != nil {
		return dnsdto.Record{}, err
	}
	if !response.Success {
		return dnsdto.Record{}, unsuccessfulResponse()
	}
	return response.Result.view(), nil
}

func (c *Client) UpdateARecord(ctx context.Context, token, zoneId, recordId, name string, input dnsdto.UpdateRecordInput) (dnsdto.Record, error) {
	body := struct {
		Name    string `json:"name"`
		Type    string `json:"type"`
		Content string `json:"content"`
		TTL     int    `json:"ttl"`
		Proxied bool   `json:"proxied"`
	}{Name: name, Type: "A", Content: input.Content, TTL: input.TTL, Proxied: input.Proxied}
	var response envelope[record]
	if err := c.request(ctx, token, http.MethodPatch, "/zones/"+zoneId+"/dns_records/"+recordId, body, &response); err != nil {
		return dnsdto.Record{}, err
	}
	if !response.Success {
		return dnsdto.Record{}, unsuccessfulResponse()
	}
	return response.Result.view(), nil
}

func unsuccessfulResponse() error {
	return apperror.New(apperror.KindUnavailable, "Cloudflare did not complete the request; refresh records before retrying")
}

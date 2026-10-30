package cloudflare

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	dnsdto "github.com/leoninew/pomelo-orbit/internal/application/dns/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

func testClient(t *testing.T, handler http.HandlerFunc) *Client {
	t.Helper()
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)
	client := New()
	client.baseUrl = server.URL
	return client
}

func TestListZonesReadsAllPages(t *testing.T) {
	calls := 0
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		calls++
		if r.Header.Get("Authorization") != "Bearer test-token" || r.URL.Path != "/zones" || r.URL.Query().Get("per_page") != "50" {
			t.Errorf("unexpected request %s", r.URL)
		}
		name := "first.test"
		if r.URL.Query().Get("page") == "2" {
			name = "second.test"
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []zone{{Id: name, Name: name}}, "result_info": map[string]int{"total_pages": 2}})
	})
	zones, err := client.ListZones(context.Background(), "test-token")
	if err != nil || len(zones) != 2 || zones[1].Name != "second.test" || calls != 2 {
		t.Fatalf("zones=%#v calls=%d err=%v", zones, calls, err)
	}
}

func TestListARecordsFiltersAndPaginates(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		if r.URL.Path != "/zones/zone/dns_records" || query.Get("name.contains") != "api" || query.Get("type") != "A" || query.Get("page") != "2" || query.Get("per_page") != "10" || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected request %s", r.URL)
		}
		_ = json.NewEncoder(w).Encode(map[string]any{"success": true, "result": []record{{Id: "record", Type: "A", Content: "192.0.2.1", TTL: 300, Proxiable: true}}, "result_info": map[string]int{"total_count": 11}})
	})
	page, err := client.ListARecords(context.Background(), "test-token", "zone", "api", 2, 10)
	if err != nil || page.Total != 11 || page.Page != 2 || page.PerPage != 10 || len(page.Items) != 1 || page.Items[0].Content != "192.0.2.1" || !page.Items[0].Proxiable {
		t.Fatalf("page=%#v err=%v", page, err)
	}
}

func TestUpdateARecordPatchesOnlyEditableFields(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/zones/zone/dns_records/record" || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Error(err)
		}
		if len(body) != 5 || body["name"] != "api.example.test" || body["type"] != "A" || body["content"] != "192.0.2.2" || body["ttl"] != float64(300) || body["proxied"] != false {
			t.Errorf("PATCH body=%#v", body)
		}
		_, _ = w.Write([]byte(`{"success":true,"result":{"id":"record","name":"api.example.test","type":"A","content":"192.0.2.2","ttl":300,"proxied":false,"proxiable":true,"comment":"keep","tags":["keep"]}}`))
	})
	record, err := client.UpdateARecord(context.Background(), "test-token", "zone", "record", "api.example.test", dnsdto.UpdateRecordInput{Content: "192.0.2.2", TTL: 300})
	if err != nil || record.Name != "api.example.test" || record.Content != "192.0.2.2" || record.TTL != 300 {
		t.Fatalf("record=%#v err=%v", record, err)
	}
}

func TestCloudflareErrorsAreSanitizedAndWritesAreNotRetried(t *testing.T) {
	for _, tc := range []struct {
		name   string
		status int
		body   string
		kind   apperror.Kind
	}{
		{"permission", 403, "secret-token upstream details", apperror.KindValidation},
		{"rate limit", 429, "secret-token upstream details", apperror.KindRateLimited},
		{"unavailable", 503, "secret-token upstream details", apperror.KindUnavailable},
		{"invalid response", 200, "secret-token upstream details", apperror.KindUnavailable},
		{"failed envelope", 200, `{"success":false,"errors":[{"message":"secret-token"}]}`, apperror.KindUnavailable},
	} {
		t.Run(tc.name, func(t *testing.T) {
			calls := 0
			client := testClient(t, func(w http.ResponseWriter, _ *http.Request) {
				calls++
				w.WriteHeader(tc.status)
				_, _ = w.Write([]byte(tc.body))
			})
			_, err := client.UpdateARecord(context.Background(), "secret-token", "zone", "record", "api.example.test", dnsdto.UpdateRecordInput{Content: "192.0.2.2", TTL: 1})
			if !apperror.IsKind(err, tc.kind) || strings.Contains(err.Error(), "secret-token") || calls != 1 {
				t.Fatalf("err=%v calls=%d", err, calls)
			}
		})
	}
}

func TestCloudflareDoesNotFollowRedirects(t *testing.T) {
	redirected := false
	target := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { redirected = true }))
	defer target.Close()
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusTemporaryRedirect)
	})
	_, err := client.ListZones(context.Background(), "secret-token")
	if err == nil || redirected {
		t.Fatalf("err=%v redirected=%v", err, redirected)
	}
}

func TestCreateARecordPostsFields(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/zones/zone/dns_records" || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("request %s %s", r.Method, r.URL)
		}
		var body map[string]any
		if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
			t.Fatal(err)
		}
		if len(body) != 5 || body["name"] != "api.example.test" || body["type"] != "A" || body["content"] != "192.0.2.1" || body["ttl"] != float64(300) || body["proxied"] != false {
			t.Errorf("body=%#v", body)
		}
		_, _ = w.Write([]byte(`{"success":true,"result":{"id":"new","type":"A","name":"api.example.test","content":"192.0.2.1","ttl":300}}`))
	})
	result, err := client.CreateARecord(context.Background(), "test-token", "zone", dnsdto.CreateRecordInput{Name: "api.example.test", UpdateRecordInput: dnsdto.UpdateRecordInput{Content: "192.0.2.1", TTL: 300}})
	if err != nil || result.Id != "new" {
		t.Fatalf("result=%#v err=%v", result, err)
	}
}

func TestDeleteARecord(t *testing.T) {
	client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodDelete || r.URL.Path != "/zones/zone/dns_records/record" || r.Header.Get("Authorization") != "Bearer test-token" {
			t.Errorf("request %s %s", r.Method, r.URL)
		}
		_, _ = w.Write([]byte(`{"success":true,"result":{"id":"record"}}`))
	})
	if err := client.DeleteARecord(context.Background(), "test-token", "zone", "record"); err != nil {
		t.Fatal(err)
	}
}

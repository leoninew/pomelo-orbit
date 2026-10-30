package dnshandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"
	dnsdto "github.com/leoninew/pomelo-orbit/internal/application/dns/dto"
	dnssvc "github.com/leoninew/pomelo-orbit/internal/application/dns/usecase"
	dnsv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/dns"
)

type Handler struct {
	service       dnssvc.Service
	authenticator security.Authenticator
}

func New(service dnssvc.Service, authenticator security.Authenticator) Handler {
	return Handler{service: service, authenticator: authenticator}
}

func (h Handler) ListZones(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListZones(c.Request.Context(), current.Id, c.Query("project_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := &dnsv1.ZoneListResp{Items: make([]*dnsv1.ZoneResp, 0, len(items))}
	for _, item := range items {
		resp.Items = append(resp.Items, &dnsv1.ZoneResp{Id: item.Id, Name: item.Name})
	}
	transport.WriteProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) ListARecords(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transport.QueryInt(c.Query("page"), 1)
	perPage := transport.QueryInt(c.Query("per_page"), 20)
	items, err := h.service.ListARecords(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("zone_id"), c.Query("search"), page, perPage)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := &dnsv1.RecordPaginatedResp{Items: make([]*dnsv1.RecordResp, 0, len(items.Items)), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transport.PageCount(items.Total, items.PerPage))}
	for _, item := range items.Items {
		resp.Items = append(resp.Items, recordResponse(item))
	}
	transport.WriteProtoJSON(c, http.StatusOK, resp)
}

func (h Handler) CreateARecord(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req dnsv1.RecordCreateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Proxied == nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "proxied is required")
		return
	}
	item, err := h.service.CreateARecord(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("zone_id"), dnsdto.CreateRecordInput{Name: req.Name, UpdateRecordInput: dnsdto.UpdateRecordInput{Content: req.Content, TTL: int(req.Ttl), Proxied: *req.Proxied}})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusCreated, recordResponse(item))
}

func (h Handler) UpdateARecord(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req dnsv1.RecordUpdateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Proxied == nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "proxied is required")
		return
	}
	item, err := h.service.UpdateARecord(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("zone_id"), c.Param("record_id"), dnsdto.UpdateRecordInput{Content: req.Content, TTL: int(req.Ttl), Proxied: *req.Proxied})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, recordResponse(item))
}

func (h Handler) DeleteARecord(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteARecord(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("zone_id"), c.Param("record_id")); err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func recordResponse(item dnsdto.Record) *dnsv1.RecordResp {
	return &dnsv1.RecordResp{Id: item.Id, Name: item.Name, Type: item.Type, Content: item.Content, Ttl: int32(item.TTL), Proxied: item.Proxied, Proxiable: item.Proxiable}
}

package gatewayhandler

import (
	"net/http"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	gatewayv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/gateway"
)

func (h Handler) ListGateways(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transport.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transport.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListGateways(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := gatewayResponses(items.Items)
	transport.WriteProtoJSON(c, http.StatusOK, &gatewayv1.GatewayPaginatedResp{
		Items:   transport.Ptrs(resp),
		Total:   int32(items.Total),
		Page:    int32(items.Page),
		PerPage: int32(items.PerPage),
		Pages:   int32(transport.PageCount(items.Total, items.PerPage)),
	})
}

func (h Handler) GetGateway(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.GatewayForUser(c.Request.Context(), current.Id, c.Param("gateway_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := gatewayResponse(view)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateGateway(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req gatewayv1.GatewayUpdateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.UpdateGateway(c.Request.Context(), current.Id, c.Param("gateway_id"), gatewayUpdateInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := gatewayResponse(view)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteGateway(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteGateway(c.Request.Context(), current.Id, c.Param("gateway_id")); err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

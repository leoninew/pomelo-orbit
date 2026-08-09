package gatewayhandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	gatewayv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/gateway"
	"github.com/gin-gonic/gin"
)

func (h Handler) ListGateways(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListGateways(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := gatewayResponses(items.Items)
	transportresponse.ProtoJSON(c, http.StatusOK, &gatewayv1.GatewayPaginatedResp{
		Items:   transportresponse.Ptrs(resp),
		Total:   int32(items.Total),
		Page:    int32(items.Page),
		PerPage: int32(items.PerPage),
		Pages:   int32(transportresponse.PageCount(items.Total, items.PerPage)),
	})
}

func (h Handler) CreateGateway(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req gatewayv1.GatewayCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if projectId := c.Request.URL.Query().Get("project_id"); projectId != "" && req.ProjectId == "" {
		req.ProjectId = projectId
	}
	view, err := h.service.CreateGateway(c.Request.Context(), current.Id, gatewayCreateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := gatewayResponse(view)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetGateway(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.GatewayForUser(c.Request.Context(), current.Id, c.Param("gateway_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := gatewayResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateGateway(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req gatewayv1.GatewayUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.UpdateGateway(c.Request.Context(), current.Id, c.Param("gateway_id"), gatewayUpdateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := gatewayResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteGateway(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteGateway(c.Request.Context(), current.Id, c.Param("gateway_id")); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

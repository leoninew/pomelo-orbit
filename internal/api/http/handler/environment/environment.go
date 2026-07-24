package environmenthandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	environmentv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/environment"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
)

func (h Handler) ListEnvironments(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListEnvironments(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := environmentListResponses(items.Items)
	transportresponse.ProtoJSON(c, http.StatusOK, &environmentv1.EnvironmentPaginatedResp{
		Items:   transportresponse.Ptrs(resp),
		Total:   int32(items.Total),
		Page:    int32(items.Page),
		PerPage: int32(items.PerPage),
		Pages:   int32(transportresponse.PageCount(items.Total, items.PerPage)),
	})
}

func (h Handler) CreateEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req environmentv1.EnvironmentCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.ProjectId == "" {
		req.ProjectId = c.Request.URL.Query().Get("project_id")
	}
	view, err := h.service.CreateEnvironment(c.Request.Context(), current.Id, environmentCreateInput(&req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := environmentResponse(view)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.EnvironmentForUser(c.Request.Context(), current.Id, c.Param("env_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := environmentResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req environmentv1.EnvironmentUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.UpdateEnvironment(c.Request.Context(), current.Id, c.Param("env_id"), environmentUpdateInput(&req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := environmentResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteEnvironment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteEnvironment(c.Request.Context(), current.Id, c.Param("env_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

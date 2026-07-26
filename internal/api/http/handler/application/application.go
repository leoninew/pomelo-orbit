package applicationhandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	applicationv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/application"
	"github.com/gin-gonic/gin"
)

func (h Handler) ListApplications(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListApplications(c.Request.Context(), current.Id, binding.QueryProjectId(c.Request.URL.Query().Get("project_id")), page, perPage, c.Request.URL.Query().Get("search"), c.Request.URL.Query().Get("kind"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponses(items.Items)
	transportresponse.ProtoJSON(c, http.StatusOK, &applicationv1.ApplicationPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreateApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.ApplicationCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	app, err := h.service.CreateApplication(c.Request.Context(), current.Id, applicationCreateInput(c.Request.URL.Query().Get("project_id"), &req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponse(app)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	app, err := h.service.ApplicationForUser(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponse(app)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.ApplicationUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	app, err := h.service.UpdateApplication(c.Request.Context(), current.Id, c.Param("app_id"), applicationUpdateInput(&req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponse(app)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

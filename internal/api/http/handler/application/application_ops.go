package applicationhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	applicationv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/application"
)

func (h Handler) ListVersions(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	views, err := h.service.ListVersionsPage(c.Request.Context(), current.Id, c.Param("app_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionResponses(views.Items)
	transportresponse.ProtoJSON(c, http.StatusOK, &applicationv1.VersionPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(views.Total), Page: int32(views.Page), PerPage: int32(views.PerPage), Pages: int32(transportresponse.PageCount(views.Total, views.PerPage))})
}

func (h Handler) CreateVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.ApplicationId == "" {
		req.ApplicationId = c.Param("app_id")
	}
	view, err := h.service.CreateVersion(c.Request.Context(), current.Id, versionCreateInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.VersionForUser(c.Request.Context(), current.Id, c.Param("version_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	input := versionUpdateInput(&req)
	view, err := h.service.UpdateVersion(c.Request.Context(), current.Id, c.Param("version_id"), input)
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) PublishVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.PublishVersion(c.Request.Context(), current.Id, c.Param("version_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UnpublishVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.UnpublishVersion(c.Request.Context(), current.Id, c.Param("version_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteVersion(c.Request.Context(), current.Id, c.Param("version_id")); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) ForkVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionForkReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.ForkVersion(c.Request.Context(), current.Id, c.Param("version_id"), req.Label)
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

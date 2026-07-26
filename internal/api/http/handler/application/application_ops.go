package applicationhandler

import (
	"io"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	applicationv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/application"
	"github.com/gin-gonic/gin"
)

func (h Handler) ImportApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.ApplicationImportReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	app, err := h.service.ImportApplication(c.Request.Context(), current.Id, applicationImportInput(c.Request.URL.Query().Get("project_id"), &req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	services, _ := h.runtimeService.ListServicesByApplication(c.Request.Context(), current.Id, app.Id)
	resp := applicationResponse(app, services)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) ExportApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	exported, err := h.service.ExportApplication(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	services, err := h.runtimeService.ListServicesByApplication(c.Request.Context(), current.Id, exported.Application.Id)
	if err != nil {
		h.writeError(c, err)
		return
	}
	exported.Services = services
	transportresponse.ProtoJSON(c, http.StatusOK, applicationExportResponse(exported))
}

func (h Handler) ListVersions(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	views, err := h.service.ListVersionsPage(c.Request.Context(), current.Id, c.Param("app_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
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
		h.writeError(c, err)
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
		h.writeError(c, err)
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
	data, err := io.ReadAll(c.Request.Body)
	if err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	input, err := versionUpdateInputFromJSON(data)
	if err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.UpdateVersion(c.Request.Context(), current.Id, c.Param("version_id"), input)
	if err != nil {
		h.writeError(c, err)
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
		h.writeError(c, err)
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
		h.writeError(c, err)
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
		h.writeError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

package cihandler

import (
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/codec"
	"github.com/gin-gonic/gin"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/service/ci"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/response"
)

func (h Handler) RegisterCredentialRoutes(r router) {
	r.GET("/api/ci/credential", h.listCredentials)
	r.POST("/api/ci/credential", h.createCredential)
	r.POST("/api/ci/credential/import", h.importCredential)
	r.GET("/api/ci/credential/:credential_id", h.getCredential)
	r.PUT("/api/ci/credential/:credential_id", h.updateCredential)
	r.DELETE("/api/ci/credential/:credential_id", h.deleteCredential)
	r.GET("/api/ci/credential/:credential_id/export", h.exportCredential)
}

func (h Handler) listCredentials(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 20)
	items, err := h.service.ListCredentials(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := mapPage(items, credentialResponse)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.CredentialPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) createCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.CredentialCreateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	created, err := h.service.CreateCredential(c.Request.Context(), current.Id, cisvc.CredentialCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Type: req.Type, Data: req.Data})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := credentialResponse(created)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) importCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.CredentialImportReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	if req.Version == "" {
		req.Version = cisvc.CredentialExportVersion
	}
	created, err := h.service.ImportCredential(c.Request.Context(), current.Id, cisvc.CredentialCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Type: req.Type, Data: req.Data})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := credentialResponse(created)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	credential, err := h.service.CredentialDetailForUser(c.Request.Context(), current.Id, c.Param("credential_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := credentialDetailResponse(credential)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) updateCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.CredentialUpdateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	updated, err := h.service.UpdateCredential(c.Request.Context(), current.Id, c.Param("credential_id"), cisvc.CredentialUpdateInput{Name: req.Name, Data: req.Data})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := credentialResponse(updated)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deleteCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteCredential(c.Request.Context(), current.Id, c.Param("credential_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) exportCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	exported, err := h.service.ExportCredential(c.Request.Context(), current.Id, c.Param("credential_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.CredentialExportResp{Version: exported.Version, Name: exported.Name, Type: exported.Type, Data: exported.Data}})
}

func credentialResponse(item model.Credential) pomeloorbit.CredentialResp {
	return pomeloorbit.CredentialResp{Id: item.Id, Name: item.Name, Type: item.Type, CreatedAt: transportresponse.FormatTime(item.CreatedAt)}
}

func credentialDetailResponse(item cisvc.CredentialDetail) pomeloorbit.CredentialDetailResp {
	credential := credentialResponse(item.Credential)
	return pomeloorbit.CredentialDetailResp{Id: credential.Id, Name: credential.Name, Type: credential.Type, Data: item.Data, CreatedAt: credential.CreatedAt}
}

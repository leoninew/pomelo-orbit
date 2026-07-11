package cihandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
)

func (h Handler) ListCredentials(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 20)
	items, err := h.service.ListCredentials(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.CredentialResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, credentialResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.CredentialPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreateCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.CredentialCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	created, err := h.service.CreateCredential(c.Request.Context(), current.Id, cidto.CredentialCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Type: req.Type, Data: req.Data})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := credentialResponse(created)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) ImportCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.CredentialImportReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Version == "" {
		req.Version = cidto.CredentialExportVersion
	}
	created, err := h.service.ImportCredential(c.Request.Context(), current.Id, cidto.CredentialCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Type: req.Type, Data: req.Data})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := credentialResponse(created)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetCredential(c *gin.Context) {
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
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.CredentialUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	updated, err := h.service.UpdateCredential(c.Request.Context(), current.Id, c.Param("credential_id"), cidto.CredentialUpdateInput{Name: req.Name, Data: req.Data})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := credentialResponse(updated)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteCredential(c *gin.Context) {
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

func (h Handler) ExportCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	exported, err := h.service.ExportCredential(c.Request.Context(), current.Id, c.Param("credential_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.CredentialExportResp{Version: exported.Version, Name: exported.Name, Type: exported.Type, Data: exported.Data})
}

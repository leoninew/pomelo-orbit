package credentialhandler

import (
	"net/http"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	credentialv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/credential"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
)

func (h Handler) ListCredentials(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transport.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transport.QueryInt(c.Request.URL.Query().Get("per_page"), 20)
	items, err := h.service.ListCredentials(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := make([]credentialv1.CredentialResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, credentialResponse(item))
	}
	transport.WriteProtoJSON(c, http.StatusOK, &credentialv1.CredentialPaginatedResp{Items: transport.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transport.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreateCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req credentialv1.CredentialCreateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	created, err := h.service.CreateCredential(c.Request.Context(), current.Id, credentialdto.CredentialCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Type: req.Type, Data: req.Data})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := credentialResponse(created)
	transport.WriteProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) ImportCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req credentialv1.CredentialImportReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Version == "" {
		req.Version = credentialdto.CredentialExportVersion
	}
	created, err := h.service.ImportCredential(c.Request.Context(), current.Id, credentialdto.CredentialCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Type: req.Type, Data: req.Data})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := credentialResponse(created)
	transport.WriteProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	credential, err := h.service.CredentialDetailForUser(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("credential_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := credentialDetailResponse(credential)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req credentialv1.CredentialUpdateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	updated, err := h.service.UpdateCredential(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("credential_id"), credentialdto.CredentialUpdateInput{Name: req.Name, Data: req.Data})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := credentialResponse(updated)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteCredential(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("credential_id")); err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) ExportCredential(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	exported, err := h.service.ExportCredential(c.Request.Context(), current.Id, c.Request.URL.Query().Get("project_id"), c.Param("credential_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &credentialv1.CredentialExportResp{Version: exported.Version, Name: exported.Name, Type: exported.Type, Data: exported.Data})
}

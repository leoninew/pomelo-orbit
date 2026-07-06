package cihandler

import (
	pomeloorbit "gitee.com/leoninew/pomelo-orbit/internal/gen/proto/orbit"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitee.com/leoninew/pomelo-orbit/internal/repository/model"
	cisvc "gitee.com/leoninew/pomelo-orbit/internal/service/ci"
	transportresponse "gitee.com/leoninew/pomelo-orbit/internal/transport/http/response"
)

func (h Handler) RegisterCredentialRoutes(r router) {
	r.Get("/api/ci/credential", h.listCredentials)
	r.Post("/api/ci/credential", h.createCredential)
	r.Post("/api/ci/credential/import", h.importCredential)
	r.Get("/api/ci/credential/{credential_id}", h.getCredential)
	r.Put("/api/ci/credential/{credential_id}", h.updateCredential)
	r.Delete("/api/ci/credential/{credential_id}", h.deleteCredential)
	r.Get("/api/ci/credential/{credential_id}/export", h.exportCredential)
}

func (h Handler) listCredentials(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 20)
	items, err := h.service.ListCredentials(r.Context(), current.Id, r.URL.Query().Get("project_id"), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := mapPage(items, credentialResponse)
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.CredentialPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))})
}

func (h Handler) createCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.CredentialCreateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	created, err := h.service.CreateCredential(r.Context(), current.Id, cisvc.CredentialCreateInput{ProjectId: r.URL.Query().Get("project_id"), Name: req.Name, Type: req.Type, Data: req.Data})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := credentialResponse(created)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) importCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.CredentialImportReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.Version == "" {
		req.Version = cisvc.CredentialExportVersion
	}
	created, err := h.service.ImportCredential(r.Context(), current.Id, cisvc.CredentialCreateInput{ProjectId: r.URL.Query().Get("project_id"), Name: req.Name, Type: req.Type, Data: req.Data})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := credentialResponse(created)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) getCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	credential, err := h.service.CredentialDetailForUser(r.Context(), current.Id, chi.URLParam(r, "credential_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := credentialDetailResponse(credential)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) updateCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.CredentialUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	updated, err := h.service.UpdateCredential(r.Context(), current.Id, chi.URLParam(r, "credential_id"), cisvc.CredentialUpdateInput{Name: req.Name, Data: req.Data})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := credentialResponse(updated)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) deleteCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteCredential(r.Context(), current.Id, chi.URLParam(r, "credential_id")); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) exportCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	exported, err := h.service.ExportCredential(r.Context(), current.Id, chi.URLParam(r, "credential_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.CredentialExportResp{Version: exported.Version, Name: exported.Name, Type: exported.Type, Data: exported.Data})
}

func credentialResponse(item model.Credential) pomeloorbit.CredentialResp {
	return pomeloorbit.CredentialResp{Id: item.Id, Name: item.Name, Type: item.Type, CreatedAt: transportresponse.FormatTime(item.CreatedAt)}
}

func credentialDetailResponse(item cisvc.CredentialDetail) pomeloorbit.CredentialDetailResp {
	credential := credentialResponse(item.Credential)
	return pomeloorbit.CredentialDetailResp{Id: credential.Id, Name: credential.Name, Type: credential.Type, Data: item.Data, CreatedAt: credential.CreatedAt}
}

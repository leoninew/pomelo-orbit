package cihandler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/repository"
	cisvc "backend/internal/service/ci"
	transportresponse "backend/internal/transport/http/response"
)

type CredentialResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

type CredentialCreateReq struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data string `json:"data"`
}

type CredentialUpdateReq struct {
	Name *string `json:"name"`
	Data *string `json:"data"`
}

type CredentialExportResp struct {
	Version string `json:"version"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Data    string `json:"data"`
}

type CredentialImportReq struct {
	Version string `json:"version"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Data    string `json:"data"`
}

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
	transportresponse.JSON(w, http.StatusOK, transportresponse.NewPaginatedResp(mapPage(items, credentialResponse)))
}

func (h Handler) createCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req CredentialCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	created, err := h.service.CreateCredential(r.Context(), current.Id, cisvc.CredentialCreateInput{ProjectId: r.URL.Query().Get("project_id"), Name: req.Name, Type: req.Type, Data: req.Data})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusCreated, credentialResponse(created))
}

func (h Handler) importCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req CredentialImportReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
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
	transportresponse.JSON(w, http.StatusCreated, credentialResponse(created))
}

func (h Handler) getCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	credential, err := h.service.CredentialForUser(r.Context(), current.Id, chi.URLParam(r, "credential_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, credentialResponse(credential))
}

func (h Handler) updateCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req CredentialUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	updated, err := h.service.UpdateCredential(r.Context(), current.Id, chi.URLParam(r, "credential_id"), cisvc.CredentialUpdateInput{Name: req.Name, Data: req.Data})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, credentialResponse(updated))
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
	transportresponse.JSON(w, http.StatusOK, CredentialExportResp{Version: exported.Version, Name: exported.Name, Type: exported.Type, Data: exported.Data})
}

func credentialResponse(item repository.Credential) CredentialResp {
	return CredentialResp{Id: item.Id, Name: item.Name, Type: item.Type, CreatedAt: transportresponse.FormatTime(item.CreatedAt)}
}

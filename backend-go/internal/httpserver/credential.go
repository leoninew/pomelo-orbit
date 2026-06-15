package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"backend/internal/orbit"
	"backend/internal/security"
)

const credentialExportVersion = "1.0"

type credentialResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Type      string `json:"type"`
	CreatedAt string `json:"created_at"`
}

type credentialCreateReq struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Data string `json:"data"`
}

type credentialUpdateReq struct {
	Name *string `json:"name"`
	Data *string `json:"data"`
}

type credentialExportResp struct {
	Version string `json:"version"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Data    string `json:"data"`
}

type credentialImportReq struct {
	Version string `json:"version"`
	Name    string `json:"name"`
	Type    string `json:"type"`
	Data    string `json:"data"`
}

func (s Server) listCredentials(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}
	page, perPage := credentialPageParams(r)
	items, err := s.store.ListCredentials(r.Context(), projectId, page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list credentials failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list credentials"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, credentialResponse)))
}

func (s Server) createCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}
	var req credentialCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeCredentialCreateReq(w, &req) || !s.ensureCredentialNameAvailable(w, r, projectId, req.Name, "") {
		return
	}
	created, ok := s.createCredentialRecord(w, r, projectId, req.Name, req.Type, req.Data)
	if !ok {
		return
	}
	writeJSON(w, http.StatusCreated, credentialResponse(created))
}

func (s Server) importCredential(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}
	var req credentialImportReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeCredentialImportReq(w, &req) || !s.ensureCredentialNameAvailable(w, r, projectId, req.Name, "") {
		return
	}
	created, ok := s.createCredentialRecord(w, r, projectId, req.Name, req.Type, req.Data)
	if !ok {
		return
	}
	writeJSON(w, http.StatusCreated, credentialResponse(created))
}

func (s Server) getCredential(w http.ResponseWriter, r *http.Request) {
	credential, ok := s.loadCredentialForCurrentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, credentialResponse(credential))
}

func (s Server) updateCredential(w http.ResponseWriter, r *http.Request) {
	credential, ok := s.loadCredentialForCurrentUser(w, r)
	if !ok {
		return
	}
	var req credentialUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid credential fields"})
			return
		}
		projectId := credentialProjectId(credential)
		if !s.ensureCredentialNameAvailable(w, r, projectId, name, credential.Id) {
			return
		}
		credential.Name = name
	}
	if req.Data != nil {
		if *req.Data == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid credential fields"})
			return
		}
		encrypted, ok := s.encryptCredentialData(w, *req.Data)
		if !ok {
			return
		}
		credential.EncryptedData = encrypted
	}
	if err := s.store.UpdateCredential(r.Context(), credential); err != nil {
		s.logger.Error("update credential failed", "credential_id", credential.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update credential"})
		return
	}
	updated, err := s.store.Credential(r.Context(), credential.Id)
	if err != nil {
		s.logger.Error("load updated credential failed", "credential_id", credential.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load credential"})
		return
	}
	writeJSON(w, http.StatusOK, credentialResponse(updated))
}

func (s Server) deleteCredential(w http.ResponseWriter, r *http.Request) {
	credential, ok := s.loadCredentialForCurrentUser(w, r)
	if !ok {
		return
	}
	referenced, err := s.store.CredentialReferencedByRepositories(r.Context(), credentialProjectId(credential), credential.Id)
	if err != nil {
		s.logger.Error("check credential references failed", "credential_id", credential.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check credential references"})
		return
	}
	if referenced {
		writeJSON(w, http.StatusConflict, map[string]string{"detail": "Credential is referenced by projects, cannot delete"})
		return
	}
	if err := s.store.DeleteCredential(r.Context(), credential.Id); err != nil {
		s.logger.Error("delete credential failed", "credential_id", credential.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete credential"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) exportCredential(w http.ResponseWriter, r *http.Request) {
	credential, ok := s.loadCredentialForCurrentUser(w, r)
	if !ok {
		return
	}
	decrypted, err := security.DecryptString(s.appCfg.JWT.SecretKey, credential.EncryptedData)
	if err != nil {
		s.logger.Error("decrypt credential failed", "credential_id", credential.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to decrypt credential"})
		return
	}
	writeJSON(w, http.StatusOK, credentialExportResp{Version: credentialExportVersion, Name: credential.Name, Type: credential.Type, Data: decrypted})
}

func (s Server) createCredentialRecord(w http.ResponseWriter, r *http.Request, projectId string, name string, credentialType string, data string) (orbit.Credential, bool) {
	encrypted, ok := s.encryptCredentialData(w, data)
	if !ok {
		return orbit.Credential{}, false
	}
	credential := orbit.Credential{Id: orbit.NewId(), ProjectId: &projectId, Name: name, Type: credentialType, EncryptedData: encrypted}
	if err := s.store.CreateCredential(r.Context(), credential); err != nil {
		s.logger.Error("create credential failed", "credential_name", credential.Name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create credential"})
		return orbit.Credential{}, false
	}
	created, err := s.store.Credential(r.Context(), credential.Id)
	if err != nil {
		s.logger.Error("load created credential failed", "credential_id", credential.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load credential"})
		return orbit.Credential{}, false
	}
	return created, true
}

func (s Server) loadCredentialForCurrentUser(w http.ResponseWriter, r *http.Request) (orbit.Credential, bool) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return orbit.Credential{}, false
	}
	credentialId := urlParam(r, "credential_id")
	credential, err := s.store.Credential(r.Context(), credentialId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Credential " + credentialId + " not found"})
			return orbit.Credential{}, false
		}
		s.logger.Error("load credential failed", "credential_id", credentialId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load credential"})
		return orbit.Credential{}, false
	}
	if credential.ProjectId != nil && !s.ensureProjectMembership(w, r, *credential.ProjectId, current.Id) {
		return orbit.Credential{}, false
	}
	return credential, true
}

func (s Server) ensureCredentialNameAvailable(w http.ResponseWriter, r *http.Request, projectId string, name string, excludeId string) bool {
	existing, err := s.store.CredentialByName(r.Context(), projectId, name)
	if err == nil {
		if existing.Id != excludeId {
			writeJSON(w, http.StatusConflict, map[string]string{"detail": "凭据名称 '" + name + "' 已存在"})
			return false
		}
		return true
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("check credential name failed", "credential_name", name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check credential name"})
		return false
	}
	return true
}

func (s Server) encryptCredentialData(w http.ResponseWriter, data string) (string, bool) {
	encrypted, err := security.EncryptString(s.appCfg.JWT.SecretKey, data)
	if err != nil {
		s.logger.Error("encrypt credential failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to encrypt credential"})
		return "", false
	}
	return encrypted, true
}

func credentialResponse(item orbit.Credential) credentialResp {
	return credentialResp{Id: item.Id, Name: item.Name, Type: item.Type, CreatedAt: formatTime(item.CreatedAt)}
}

func credentialPageParams(r *http.Request) (int, int) {
	return queryInt(r.URL.Query().Get("page"), 1), queryInt(r.URL.Query().Get("per_page"), 20)
}

func credentialProjectId(item orbit.Credential) string {
	if item.ProjectId == nil {
		return ""
	}
	return *item.ProjectId
}

func normalizeCredentialCreateReq(w http.ResponseWriter, req *credentialCreateReq) bool {
	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.TrimSpace(req.Type)
	if req.Name == "" || req.Data == "" || !validCredentialType(req.Type) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid credential fields"})
		return false
	}
	return true
}

func normalizeCredentialImportReq(w http.ResponseWriter, req *credentialImportReq) bool {
	if req.Version == "" {
		req.Version = credentialExportVersion
	}
	req.Name = strings.TrimSpace(req.Name)
	req.Type = strings.TrimSpace(req.Type)
	if req.Name == "" || req.Data == "" || !validCredentialType(req.Type) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid credential fields"})
		return false
	}
	return true
}

func validCredentialType(value string) bool {
	switch value {
	case "git_ssh", "github_token", "gitee_token", "registry_token":
		return true
	default:
		return false
	}
}

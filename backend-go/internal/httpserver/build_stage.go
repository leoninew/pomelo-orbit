package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"

	"backend/internal/orbit"
)

var buildStageCopyPattern = regexp.MustCompile(` copy( [0-9]+)?$`)

type artifactConfigResp struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
}

type buildStageResp struct {
	Id          string               `json:"id"`
	Name        string               `json:"name"`
	Image       string               `json:"image"`
	Script      string               `json:"script"`
	Artifacts   []artifactConfigResp `json:"artifacts"`
	Description string               `json:"description"`
	Version     int                  `json:"version"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

type buildStageCreateReq struct {
	Name        string               `json:"name"`
	Image       string               `json:"image"`
	Script      string               `json:"script"`
	Artifacts   []artifactConfigResp `json:"artifacts"`
	Description string               `json:"description"`
}

func (s Server) listBuildStages(w http.ResponseWriter, r *http.Request) {
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
	page, perPage := pageParams(r)
	items, err := s.store.ListBuildStages(r.Context(), projectId, page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list build stages failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list build stages"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, buildStageResponse)))
}

func (s Server) createBuildStage(w http.ResponseWriter, r *http.Request) {
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
	var req buildStageCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeBuildStageCreateReq(w, &req) || !s.ensureBuildStageNameAvailable(w, r, projectId, req.Name, "") {
		return
	}
	artifacts, ok := marshalBuildStageArtifacts(w, req.Artifacts)
	if !ok {
		return
	}
	stage := orbit.BuildStage{Id: orbit.NewId(), ProjectId: &projectId, Name: req.Name, Image: req.Image, Script: req.Script, Artifacts: artifacts, Description: req.Description, Version: 1}
	if err := s.store.CreateBuildStage(r.Context(), stage); err != nil {
		s.logger.Error("create build stage failed", "stage_name", stage.Name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create build stage"})
		return
	}
	created, err := s.store.BuildStage(r.Context(), stage.Id)
	if err != nil {
		s.logger.Error("load created build stage failed", "stage_id", stage.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load build stage"})
		return
	}
	writeJSON(w, http.StatusCreated, buildStageResponse(created))
}

func (s Server) getBuildStage(w http.ResponseWriter, r *http.Request) {
	stage, ok := s.loadBuildStageForCurrentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, buildStageResponse(stage))
}

func (s Server) updateBuildStage(w http.ResponseWriter, r *http.Request) {
	stage, ok := s.loadBuildStageForCurrentUser(w, r)
	if !ok {
		return
	}
	var req map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !s.applyBuildStageUpdateReq(w, r, &stage, req) {
		return
	}
	if err := s.store.UpdateBuildStage(r.Context(), stage); err != nil {
		s.logger.Error("update build stage failed", "stage_id", stage.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update build stage"})
		return
	}
	updated, err := s.store.BuildStage(r.Context(), stage.Id)
	if err != nil {
		s.logger.Error("load updated build stage failed", "stage_id", stage.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load build stage"})
		return
	}
	writeJSON(w, http.StatusOK, buildStageResponse(updated))
}

func (s Server) deleteBuildStage(w http.ResponseWriter, r *http.Request) {
	stage, ok := s.loadBuildStageForCurrentUser(w, r)
	if !ok {
		return
	}
	projectId := buildStageProjectId(stage)
	referenced, err := s.store.BuildStageReferencedByTemplates(r.Context(), projectId, stage.Id)
	if err != nil {
		s.logger.Error("check build stage references failed", "stage_id", stage.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check build stage references"})
		return
	}
	if referenced {
		writeJSON(w, http.StatusConflict, map[string]string{"detail": "Stage is referenced by templates, cannot delete"})
		return
	}
	if err := s.store.DeleteBuildStage(r.Context(), stage.Id); err != nil {
		s.logger.Error("delete build stage failed", "stage_id", stage.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete build stage"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) duplicateBuildStage(w http.ResponseWriter, r *http.Request) {
	stage, ok := s.loadBuildStageForCurrentUser(w, r)
	if !ok {
		return
	}
	projectId := buildStageProjectId(stage)
	name, ok := s.nextBuildStageCopyName(w, r, projectId, stage.Name)
	if !ok {
		return
	}
	duplicated := orbit.BuildStage{Id: orbit.NewId(), ProjectId: stage.ProjectId, Name: name, Image: stage.Image, Script: stage.Script, Artifacts: stage.Artifacts, Description: stage.Description, Version: 1}
	if err := s.store.CreateBuildStage(r.Context(), duplicated); err != nil {
		s.logger.Error("duplicate build stage failed", "stage_id", stage.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to duplicate build stage"})
		return
	}
	created, err := s.store.BuildStage(r.Context(), duplicated.Id)
	if err != nil {
		s.logger.Error("load duplicated build stage failed", "stage_id", duplicated.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load build stage"})
		return
	}
	writeJSON(w, http.StatusCreated, buildStageResponse(created))
}

func (s Server) loadBuildStageForCurrentUser(w http.ResponseWriter, r *http.Request) (orbit.BuildStage, bool) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return orbit.BuildStage{}, false
	}
	stageId := urlParam(r, "stage_id")
	stage, err := s.store.BuildStage(r.Context(), stageId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Stage " + stageId + " not found"})
			return orbit.BuildStage{}, false
		}
		s.logger.Error("load build stage failed", "stage_id", stageId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load build stage"})
		return orbit.BuildStage{}, false
	}
	projectId := buildStageProjectId(stage)
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return orbit.BuildStage{}, false
	}
	return stage, true
}

func (s Server) ensureBuildStageNameAvailable(w http.ResponseWriter, r *http.Request, projectId string, name string, currentStageId string) bool {
	existing, err := s.store.BuildStageByName(r.Context(), projectId, name)
	if err == nil {
		if existing.Id != currentStageId {
			writeJSON(w, http.StatusConflict, map[string]string{"detail": "Stage '" + name + "' already exists"})
			return false
		}
		return true
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("check build stage name failed", "stage_name", name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check build stage name"})
		return false
	}
	return true
}

func (s Server) nextBuildStageCopyName(w http.ResponseWriter, r *http.Request, projectId string, name string) (string, bool) {
	baseName := buildStageCopyPattern.ReplaceAllString(name, "")
	for i := 1; ; i++ {
		candidate := baseName + " copy"
		if i > 1 {
			candidate = fmt.Sprintf("%s copy %d", baseName, i)
		}
		_, err := s.store.BuildStageByName(r.Context(), projectId, candidate)
		if errors.Is(err, sql.ErrNoRows) {
			return candidate, true
		}
		if err != nil {
			s.logger.Error("check duplicate build stage name failed", "stage_name", candidate, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check build stage name"})
			return "", false
		}
	}
}

func (s Server) applyBuildStageUpdateReq(w http.ResponseWriter, r *http.Request, stage *orbit.BuildStage, req map[string]json.RawMessage) bool {
	projectId := buildStageProjectId(*stage)
	versionChanged := false
	if raw, exists := req["name"]; exists {
		value, ok := decodeRequiredString(w, raw, "Invalid build stage fields")
		if !ok || !s.ensureBuildStageNameAvailable(w, r, projectId, value, stage.Id) {
			return false
		}
		stage.Name = value
	}
	if raw, exists := req["image"]; exists {
		value, ok := decodeRequiredString(w, raw, "Invalid build stage fields")
		if !ok {
			return false
		}
		if value != stage.Image {
			stage.Image = value
			versionChanged = true
		}
	}
	if raw, exists := req["script"]; exists {
		value, ok := decodeRequiredString(w, raw, "Invalid build stage fields")
		if !ok {
			return false
		}
		if value != stage.Script {
			stage.Script = value
			versionChanged = true
		}
	}
	if raw, exists := req["artifacts"]; exists && string(raw) != "null" {
		var artifacts []artifactConfigResp
		if err := json.Unmarshal(raw, &artifacts); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid build stage fields"})
			return false
		}
		artifactJSON, ok := marshalBuildStageArtifacts(w, artifacts)
		if !ok {
			return false
		}
		if optionalStringValue(stage.Artifacts) != optionalStringValue(artifactJSON) {
			stage.Artifacts = artifactJSON
			versionChanged = true
		}
	}
	if raw, exists := req["description"]; exists {
		var value *string
		if string(raw) != "null" {
			decoded := ""
			if err := json.Unmarshal(raw, &decoded); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid build stage fields"})
				return false
			}
			value = &decoded
		}
		if value != nil {
			stage.Description = *value
		}
	}
	if versionChanged {
		stage.Version++
	}
	return true
}

func normalizeBuildStageCreateReq(w http.ResponseWriter, req *buildStageCreateReq) bool {
	req.Name = strings.TrimSpace(req.Name)
	req.Image = strings.TrimSpace(req.Image)
	req.Script = strings.TrimSpace(req.Script)
	if req.Name == "" || req.Image == "" || req.Script == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid build stage fields"})
		return false
	}
	if !validateBuildStageArtifacts(w, req.Artifacts) {
		return false
	}
	return true
}

func decodeRequiredString(w http.ResponseWriter, raw json.RawMessage, message string) (string, bool) {
	var value string
	if err := json.Unmarshal(raw, &value); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": message})
		return "", false
	}
	value = strings.TrimSpace(value)
	if value == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": message})
		return "", false
	}
	return value, true
}

func marshalBuildStageArtifacts(w http.ResponseWriter, artifacts []artifactConfigResp) (*string, bool) {
	if !validateBuildStageArtifacts(w, artifacts) {
		return nil, false
	}
	if len(artifacts) == 0 {
		return nil, true
	}
	data, err := json.Marshal(artifacts)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid build stage artifacts"})
		return nil, false
	}
	value := string(data)
	return &value, true
}

func validateBuildStageArtifacts(w http.ResponseWriter, artifacts []artifactConfigResp) bool {
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Type) == "" || strings.TrimSpace(artifact.Path) == "" || strings.TrimSpace(artifact.Name) == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid build stage artifacts"})
			return false
		}
	}
	return true
}

func buildStageResponse(stage orbit.BuildStage) buildStageResp {
	artifacts := []artifactConfigResp(nil)
	if stage.Artifacts != nil && strings.TrimSpace(*stage.Artifacts) != "" {
		_ = json.Unmarshal([]byte(*stage.Artifacts), &artifacts)
	}
	return buildStageResp{Id: stage.Id, Name: stage.Name, Image: stage.Image, Script: stage.Script, Artifacts: artifacts, Description: stage.Description, Version: stage.Version, CreatedAt: formatTime(stage.CreatedAt), UpdatedAt: formatTime(stage.UpdatedAt)}
}

func buildStageProjectId(stage orbit.BuildStage) string {
	if stage.ProjectId == nil {
		return ""
	}
	return *stage.ProjectId
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

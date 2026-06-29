package cihandler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	cisvc "backend/internal/service/ci"
	transportresponse "backend/internal/transport/http/response"
)

type BuildStageCreateReq struct {
	Name        string               `json:"name"`
	Image       string               `json:"image"`
	Script      string               `json:"script"`
	Artifacts   []ArtifactConfigResp `json:"artifacts"`
	Description string               `json:"description"`
}

func (h Handler) RegisterBuildStageRoutes(r router) {
	r.Get("/api/ci/build-stage", h.listBuildStages)
	r.Post("/api/ci/build-stage", h.createBuildStage)
	r.Get("/api/ci/build-stage/{stage_id}", h.getBuildStage)
	r.Put("/api/ci/build-stage/{stage_id}", h.updateBuildStage)
	r.Delete("/api/ci/build-stage/{stage_id}", h.deleteBuildStage)
	r.Post("/api/ci/build-stage/{stage_id}/duplicate", h.duplicateBuildStage)
}

func (h Handler) listBuildStages(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListBuildStages(r.Context(), current.Id, r.URL.Query().Get("project_id"), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, transportresponse.NewPaginatedResp(mapPage(items, buildStageDetailResponse)))
}

func (h Handler) createBuildStage(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req BuildStageCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(h.logger, w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	stage, err := h.service.CreateBuildStage(r.Context(), current.Id, cisvc.BuildStageCreateInput{ProjectId: r.URL.Query().Get("project_id"), Name: req.Name, Image: req.Image, Script: req.Script, Artifacts: serviceArtifacts(req.Artifacts), Description: req.Description})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusCreated, buildStageDetailResponse(stage))
}

func (h Handler) getBuildStage(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	stage, err := h.service.BuildStageForUser(r.Context(), current.Id, chi.URLParam(r, "stage_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, buildStageDetailResponse(stage))
}

func (h Handler) updateBuildStage(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(h.logger, w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	stage, err := h.service.UpdateBuildStage(r.Context(), current.Id, chi.URLParam(r, "stage_id"), cisvc.BuildStageUpdateInput{Fields: req})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, buildStageDetailResponse(stage))
}

func (h Handler) deleteBuildStage(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteBuildStage(r.Context(), current.Id, chi.URLParam(r, "stage_id")); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) duplicateBuildStage(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	stage, err := h.service.DuplicateBuildStage(r.Context(), current.Id, chi.URLParam(r, "stage_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusCreated, buildStageDetailResponse(stage))
}

func buildStageDetailResponse(item cisvc.BuildStageDetail) BuildStageResp {
	return BuildStageResp{Id: item.Id, Name: item.Name, Image: item.Image, Script: item.Script, Artifacts: artifactConfigsResponse(item.Artifacts), Description: item.Description, Version: item.Version, CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func serviceArtifacts(items []ArtifactConfigResp) []cisvc.ArtifactConfig {
	resp := make([]cisvc.ArtifactConfig, 0, len(items))
	for _, item := range items {
		resp = append(resp, cisvc.ArtifactConfig{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}

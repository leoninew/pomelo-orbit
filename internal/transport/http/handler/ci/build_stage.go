package cihandler

import (
	pomeloorbit "gitee.com/leoninew/pomelo-orbit/internal/gen/proto/orbit"
	"net/http"

	"github.com/go-chi/chi/v5"

	cisvc "gitee.com/leoninew/pomelo-orbit/internal/service/ci"
	transportresponse "gitee.com/leoninew/pomelo-orbit/internal/transport/http/response"
)

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
	resp := mapPage(items, buildStageDetailResponse)
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.BuildStagePaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))})
}

func (h Handler) createBuildStage(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.BuildStageCreateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	stage, err := h.service.CreateBuildStage(r.Context(), current.Id, cisvc.BuildStageCreateInput{ProjectId: r.URL.Query().Get("project_id"), Name: req.Name, Image: req.Image, Script: req.Script, Artifacts: serviceArtifacts(req.Artifacts), Description: req.Description})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := buildStageDetailResponse(stage)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
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
	resp := buildStageDetailResponse(stage)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) updateBuildStage(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.BuildStageUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var artifacts *[]cisvc.ArtifactConfig
	if req.Artifacts != nil {
		items := serviceArtifacts(req.Artifacts.Items)
		artifacts = &items
	}
	stage, err := h.service.UpdateBuildStage(r.Context(), current.Id, chi.URLParam(r, "stage_id"), cisvc.BuildStageUpdateInput{Name: req.Name, Image: req.Image, Script: req.Script, Artifacts: artifacts, Description: req.Description})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := buildStageDetailResponse(stage)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
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
	var req pomeloorbit.BuildStageDuplicateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	stage, err := h.service.DuplicateBuildStage(r.Context(), current.Id, chi.URLParam(r, "stage_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := buildStageDetailResponse(stage)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func buildStageDetailResponse(item cisvc.BuildStageDetail) pomeloorbit.BuildStageResp {
	return pomeloorbit.BuildStageResp{Id: item.Id, Name: item.Name, Image: item.Image, Script: item.Script, Artifacts: transportresponse.Ptrs(artifactConfigsResponse(item.Artifacts)), Description: item.Description, Version: int32(item.Version), CreatedAt: item.CreatedAt, UpdatedAt: item.UpdatedAt}
}

func serviceArtifacts(items []*pomeloorbit.ArtifactConfigReq) []cisvc.ArtifactConfig {
	resp := make([]cisvc.ArtifactConfig, 0, len(items))
	for _, item := range items {
		if item == nil {
			continue
		}
		resp = append(resp, cisvc.ArtifactConfig{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}

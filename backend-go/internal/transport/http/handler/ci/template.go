package cihandler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	cisvc "backend/internal/service/ci"
	transportresponse "backend/internal/transport/http/response"
)

type PipelineTemplateResp struct {
	Id                   string                   `json:"id"`
	Name                 string                   `json:"name"`
	Description          string                   `json:"description"`
	Orchestration        []StageOrchestrationResp `json:"orchestration"`
	Stages               []BuildStageResp         `json:"stages"`
	VariableDeclarations []map[string]any         `json:"variable_declarations"`
	Version              int                      `json:"version"`
	CreatedAt            string                   `json:"created_at"`
	UpdatedAt            string                   `json:"updated_at"`
}

type StageOrchestrationResp struct {
	StageId      string   `json:"stage_id"`
	StageName    string   `json:"stage_name"`
	StageVersion int      `json:"stage_version"`
	DependsOn    []string `json:"depends_on"`
	SortOrder    int      `json:"sort_order"`
}

type ArtifactConfigResp struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
}

type BuildStageResp struct {
	Id          string               `json:"id"`
	Name        string               `json:"name"`
	Image       string               `json:"image"`
	Script      string               `json:"script"`
	Artifacts   []ArtifactConfigResp `json:"artifacts"`
	Description string               `json:"description"`
	Version     int                  `json:"version"`
	CreatedAt   string               `json:"created_at"`
	UpdatedAt   string               `json:"updated_at"`
}

type PipelineTemplateCreateReq struct {
	Name                 string           `json:"name"`
	Description          string           `json:"description"`
	VariableDeclarations []map[string]any `json:"variable_declarations"`
}

type TemplateVariableResolveReq struct {
	Orchestration        []StageOrchestrationResp `json:"orchestration"`
	VariableDeclarations []map[string]any         `json:"variable_declarations"`
}

func (h Handler) RegisterTemplateRoutes(r router) {
	r.Get("/api/ci/template", h.listPipelineTemplates)
	r.Post("/api/ci/template", h.createPipelineTemplate)
	r.Post("/api/ci/template/resolve-variables", h.resolvePipelineTemplateVariables)
	r.Get("/api/ci/template/{template_id}", h.getPipelineTemplate)
	r.Put("/api/ci/template/{template_id}", h.updatePipelineTemplate)
	r.Delete("/api/ci/template/{template_id}", h.deletePipelineTemplate)
	r.Post("/api/ci/template/{template_id}/duplicate", h.duplicatePipelineTemplate)
}

func (h Handler) listPipelineTemplates(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListPipelineTemplates(r.Context(), current.Id, r.URL.Query().Get("project_id"), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, transportresponse.NewPaginatedResp(mapPage(items, pipelineTemplateResponse)))
}

func (h Handler) createPipelineTemplate(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req PipelineTemplateCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	detail, err := h.service.CreatePipelineTemplate(r.Context(), current.Id, cisvc.PipelineTemplateCreateInput{ProjectId: r.URL.Query().Get("project_id"), Name: req.Name, Description: req.Description, VariableDeclarations: req.VariableDeclarations})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusCreated, pipelineTemplateResponse(detail))
}

func (h Handler) getPipelineTemplate(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	detail, err := h.service.PipelineTemplateForUser(r.Context(), current.Id, chi.URLParam(r, "template_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, pipelineTemplateResponse(detail))
}

func (h Handler) updatePipelineTemplate(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	detail, err := h.service.UpdatePipelineTemplate(r.Context(), current.Id, chi.URLParam(r, "template_id"), cisvc.PipelineTemplateUpdateInput{Fields: req})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, pipelineTemplateResponse(detail))
}

func (h Handler) deletePipelineTemplate(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeletePipelineTemplate(r.Context(), current.Id, chi.URLParam(r, "template_id")); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) duplicatePipelineTemplate(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	detail, err := h.service.DuplicatePipelineTemplate(r.Context(), current.Id, chi.URLParam(r, "template_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusCreated, pipelineTemplateResponse(detail))
}

func (h Handler) resolvePipelineTemplateVariables(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req TemplateVariableResolveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	variables, err := h.service.ResolvePipelineTemplateVariables(r.Context(), current.Id, cisvc.PipelineTemplateResolveInput{ProjectId: r.URL.Query().Get("project_id"), Orchestration: serviceOrchestration(req.Orchestration), VariableDeclarations: req.VariableDeclarations})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, variables)
}

func pipelineTemplateResponse(detail cisvc.PipelineTemplateDetail) PipelineTemplateResp {
	item := detail.Template
	return PipelineTemplateResp{Id: item.Id, Name: item.Name, Description: item.Description, Orchestration: orchestrationResponse(detail.Orchestration), Stages: buildStageDetailsResponse(detail.Stages), VariableDeclarations: detail.VariableDeclarations, Version: item.Version, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func orchestrationResponse(items []cisvc.StageOrchestration) []StageOrchestrationResp {
	resp := make([]StageOrchestrationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, StageOrchestrationResp{StageId: item.StageId, StageName: item.StageName, StageVersion: item.StageVersion, DependsOn: item.DependsOn, SortOrder: item.SortOrder})
	}
	return resp
}

func serviceOrchestration(items []StageOrchestrationResp) []cisvc.StageOrchestration {
	resp := make([]cisvc.StageOrchestration, 0, len(items))
	for _, item := range items {
		resp = append(resp, cisvc.StageOrchestration{StageId: item.StageId, StageName: item.StageName, StageVersion: item.StageVersion, DependsOn: item.DependsOn, SortOrder: item.SortOrder})
	}
	return resp
}

func buildStageDetailsResponse(items []cisvc.BuildStageDetail) []BuildStageResp {
	resp := make([]BuildStageResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, buildStageDetailResponse(item))
	}
	return resp
}

func artifactConfigsResponse(items []cisvc.ArtifactConfig) []ArtifactConfigResp {
	if items == nil {
		return nil
	}
	resp := make([]ArtifactConfigResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, ArtifactConfigResp{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}

package cihandler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/repository/model"
	cisvc "backend/internal/service/ci"
	transportresponse "backend/internal/transport/http/response"
)

func (h Handler) RegisterPipelineRunRoutes(r router) {
	r.Get("/api/ci/repository/{repository_id}/run", h.listRepositoryRuns)
	r.Post("/api/ci/repository/{repository_id}/trigger", h.triggerRepository)
	r.Get("/api/ci/run", h.listPipelineRuns)
	r.Get("/api/ci/run/{run_id}", h.getPipelineRun)
	r.Get("/api/ci/run/{run_id}/artifacts", h.listPipelineRunArtifacts)
	r.Get("/api/ci/run/{run_id}/stages/{stage_run_id}/log", h.getPipelineStageLog)
	r.Post("/api/ci/run/{run_id}/cancel", h.cancelPipelineRun)
	r.Post("/api/ci/run/{run_id}/retry", h.retryPipelineRun)
}

func (h Handler) listRepositoryRuns(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListRepositoryRuns(r.Context(), current.Id, chi.URLParam(r, "repository_id"), page, perPage)
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, transportresponse.NewPaginatedResp(mapPage(items, pipelineRunResponse)))
}

func (h Handler) triggerRepository(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req PipelineRunTriggerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.TriggerRepository(r.Context(), current.Id, cisvc.PipelineRunTriggerInput{RepositoryId: chi.URLParam(r, "repository_id"), TemplateId: req.TemplateId, TriggerRef: req.TriggerRef, Variables: req.Variables})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusCreated, pipelineRunResponse(detail))
}

func (h Handler) listPipelineRuns(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 20)
	items, err := h.service.ListPipelineRuns(r.Context(), current.Id, cisvc.PipelineRunListInput{ProjectId: r.URL.Query().Get("project_id"), RepositoryId: r.URL.Query().Get("repository_id"), TemplateId: r.URL.Query().Get("template_id"), DateFrom: r.URL.Query().Get("date_from"), DateTo: r.URL.Query().Get("date_to"), Page: page, PerPage: perPage})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, transportresponse.NewPaginatedResp(mapPage(items, pipelineRunResponse)))
}

func (h Handler) getPipelineRun(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	detail, err := h.service.PipelineRunForUser(r.Context(), current.Id, chi.URLParam(r, "run_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, pipelineRunResponse(detail))
}

func (h Handler) listPipelineRunArtifacts(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListPipelineRunArtifacts(r.Context(), current.Id, chi.URLParam(r, "run_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	responses := make([]ArtifactResp, 0, len(items))
	for _, item := range items {
		responses = append(responses, artifactResponse(item))
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, PipelineRunArtifactListResp{Items: responses})
}

func (h Handler) getPipelineStageLog(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	result, err := h.service.PipelineStageLog(r.Context(), current.Id, chi.URLParam(r, "run_id"), chi.URLParam(r, "stage_run_id"), transportresponse.QueryInt(r.URL.Query().Get("offset"), 0))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, PipelineStageLogResp{Logs: result.Logs, Offset: result.Offset, IsComplete: result.IsComplete})
}

func (h Handler) cancelPipelineRun(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	detail, err := h.service.CancelPipelineRun(r.Context(), current.Id, chi.URLParam(r, "run_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, pipelineRunResponse(detail))
}

func (h Handler) retryPipelineRun(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	detail, err := h.service.RetryPipelineRun(r.Context(), current.Id, chi.URLParam(r, "run_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusCreated, pipelineRunResponse(detail))
}

func pipelineRunResponse(detail cisvc.PipelineRunDetail) PipelineRunResp {
	item := detail.Run
	return PipelineRunResp{Id: item.Id, ProjectId: item.ProjectId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, SnapshotId: item.SnapshotId, TemplateId: item.TemplateId, TemplateName: item.TemplateName, TemplateVersion: item.TemplateVersion, Trigger: item.Trigger, TriggerRef: item.TriggerRef, VariablesSnapshot: pipelineRunVariableDeclarationResponses(detail.VariablesSnapshot), Status: item.Status, RetryOf: item.RetryOf, StartedAt: transportresponse.FormatOptionalTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), ErrorMessage: item.ErrorMessage, CreatedAt: transportresponse.FormatTime(item.CreatedAt), StageRuns: stageRunsResponse(detail.StageRuns)}
}

func pipelineRunVariableDeclarationResponses(items []model.VariableDeclaration) []VariableDeclarationResp {
	resp := make([]VariableDeclarationResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, VariableDeclarationResp{Name: item.Name, Description: item.Description, Default: item.Default, Value: item.Value, Secret: item.Secret, Source: item.Source, Editable: item.Editable})
	}
	return resp
}

func stageRunsResponse(items []model.StageRun) []StageRunResp {
	resp := make([]StageRunResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, stageRunResponse(item))
	}
	return resp
}

func stageRunResponse(item model.StageRun) StageRunResp {
	return StageRunResp{Id: item.Id, PipelineRunId: item.PipelineRunId, StageId: item.StageId, StageName: item.StageName, Status: item.Status, StartedAt: transportresponse.FormatOptionalTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), ExitCode: item.ExitCode, ErrorMessage: item.ErrorMessage}
}

func artifactResponse(item model.Artifact) ArtifactResp {
	return ArtifactResp{Id: item.Id, PipelineRunId: item.PipelineRunId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, TemplateId: item.TemplateId, TemplateName: item.TemplateName, StageName: item.StageName, Type: item.Type, Name: item.Name, Path: item.Path, CreatedAt: transportresponse.FormatTime(item.CreatedAt)}
}

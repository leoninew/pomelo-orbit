package cihandler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/repository/model"
	cisvc "backend/internal/service/ci"
	transportresponse "backend/internal/transport/http/response"
)

type PipelineSnapshotResp struct {
	Id                string                      `json:"id"`
	TemplateId        string                      `json:"template_id"`
	Version           int                         `json:"version"`
	StagesSnapshot    []model.StageDefinition     `json:"stages_snapshot"`
	VariablesSnapshot []model.VariableDeclaration `json:"variables_snapshot"`
	CreatedAt         string                      `json:"created_at"`
}

func (h Handler) RegisterSnapshotRoutes(r router) {
	r.Get("/api/ci/snapshot/{snapshot_id}", h.getPipelineSnapshot)
}

func (h Handler) getPipelineSnapshot(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	snapshot, err := h.service.PipelineSnapshotForUser(r.Context(), current.Id, chi.URLParam(r, "snapshot_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, pipelineSnapshotResponse(snapshot))
}

func pipelineSnapshotResponse(detail cisvc.PipelineSnapshotDetail) PipelineSnapshotResp {
	item := detail.Snapshot
	return PipelineSnapshotResp{Id: item.Id, TemplateId: item.TemplateId, Version: item.Version, StagesSnapshot: detail.StagesSnapshot, VariablesSnapshot: detail.VariablesSnapshot, CreatedAt: transportresponse.FormatTime(item.CreatedAt)}
}

package cihandler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/repository/model"
	cisvc "backend/internal/service/ci"
	transportresponse "backend/internal/transport/http/response"
)

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
	resp := pipelineSnapshotResponse(snapshot)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func pipelineSnapshotResponse(detail cisvc.PipelineSnapshotDetail) PipelineSnapshotResp {
	item := detail.Snapshot
	return PipelineSnapshotResp{Id: item.Id, TemplateId: item.TemplateId, Version: int32(item.Version), StagesSnapshot: transportresponse.Ptrs(snapshotStageResponses(detail.StagesSnapshot)), VariablesSnapshot: transportresponse.Ptrs(pipelineRunVariableDeclarationResponses(detail.VariablesSnapshot)), CreatedAt: transportresponse.FormatTime(item.CreatedAt)}
}

func snapshotStageResponses(items []model.StageDefinition) []SnapshotStageResp {
	resp := make([]SnapshotStageResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, SnapshotStageResp{Name: item.Name, Id: item.Id, Image: item.Image, Version: int32(item.Version), DependsOn: item.DependsOn, Script: item.Script, Artifacts: transportresponse.Ptrs(snapshotArtifactConfigResponses(item.Artifacts))})
	}
	return resp
}

func snapshotArtifactConfigResponses(items []model.ArtifactConfig) []ArtifactConfigResp {
	if items == nil {
		return nil
	}
	resp := make([]ArtifactConfigResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, ArtifactConfigResp{Type: item.Type, Path: item.Path, Name: item.Name})
	}
	return resp
}

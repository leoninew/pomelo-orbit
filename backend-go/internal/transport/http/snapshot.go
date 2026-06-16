package transporthttp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strings"

	"backend/internal/repository"
)

type pipelineSnapshotResp struct {
	Id                string                           `json:"id"`
	TemplateId        string                           `json:"template_id"`
	Version           int                              `json:"version"`
	StagesSnapshot    []repository.StageDefinition     `json:"stages_snapshot"`
	VariablesSnapshot []repository.VariableDeclaration `json:"variables_snapshot"`
	CreatedAt         string                           `json:"created_at"`
}

func (s Server) getPipelineSnapshot(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	snapshotId := urlParam(r, "snapshot_id")
	snapshot, err := s.store.PipelineSnapshot(r.Context(), snapshotId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "PipelineSnapshot " + snapshotId + " not found"})
			return
		}
		s.logger.Error("load pipeline snapshot failed", "snapshot_id", snapshotId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline snapshot"})
		return
	}
	if snapshot.ProjectId != nil && !s.ensureProjectMembership(w, r, *snapshot.ProjectId, current.Id) {
		return
	}
	response, ok := pipelineSnapshotResponse(w, snapshot)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func pipelineSnapshotResponse(w http.ResponseWriter, snapshot repository.PipelineSnapshot) (pipelineSnapshotResp, bool) {
	stages, ok := pipelineSnapshotStages(w, snapshot.StagesSnapshot)
	if !ok {
		return pipelineSnapshotResp{}, false
	}
	variables, ok := pipelineSnapshotVariables(w, snapshot.VariablesSnapshot)
	if !ok {
		return pipelineSnapshotResp{}, false
	}
	return pipelineSnapshotResp{Id: snapshot.Id, TemplateId: snapshot.TemplateId, Version: snapshot.Version, StagesSnapshot: stages, VariablesSnapshot: variables, CreatedAt: formatTime(snapshot.CreatedAt)}, true
}

func pipelineSnapshotStages(w http.ResponseWriter, value string) ([]repository.StageDefinition, bool) {
	if strings.TrimSpace(value) == "" {
		return []repository.StageDefinition{}, true
	}
	var stages []repository.StageDefinition
	if err := json.Unmarshal([]byte(value), &stages); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Invalid pipeline snapshot stages"})
		return nil, false
	}
	return stages, true
}

func pipelineSnapshotVariables(w http.ResponseWriter, value string) ([]repository.VariableDeclaration, bool) {
	if strings.TrimSpace(value) == "" {
		return []repository.VariableDeclaration{}, true
	}
	var variables []repository.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Invalid pipeline snapshot variables"})
		return nil, false
	}
	return variables, true
}

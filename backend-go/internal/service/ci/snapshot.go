package cisvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"backend/internal/apperror"
	"backend/internal/repository"
)

type PipelineSnapshotDetail struct {
	Snapshot          repository.PipelineSnapshot
	StagesSnapshot    []repository.StageDefinition
	VariablesSnapshot []repository.VariableDeclaration
}

func (s Service) PipelineSnapshotForUser(ctx context.Context, userId string, snapshotId string) (PipelineSnapshotDetail, error) {
	snapshotId = strings.TrimSpace(snapshotId)
	snapshot, err := s.store.PipelineSnapshot(ctx, snapshotId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return PipelineSnapshotDetail{}, apperror.New(apperror.KindNotFound, "PipelineSnapshot "+snapshotId+" not found")
		}
		return PipelineSnapshotDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	if snapshot.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *snapshot.ProjectId, userId); err != nil {
			return PipelineSnapshotDetail{}, err
		}
	}
	stages, err := pipelineSnapshotStages(snapshot.StagesSnapshot)
	if err != nil {
		return PipelineSnapshotDetail{}, err
	}
	variables, err := pipelineSnapshotVariables(snapshot.VariablesSnapshot)
	if err != nil {
		return PipelineSnapshotDetail{}, err
	}
	return PipelineSnapshotDetail{Snapshot: snapshot, StagesSnapshot: stages, VariablesSnapshot: variables}, nil
}

func pipelineSnapshotStages(value string) ([]repository.StageDefinition, error) {
	if strings.TrimSpace(value) == "" {
		return []repository.StageDefinition{}, nil
	}
	var stages []repository.StageDefinition
	if err := json.Unmarshal([]byte(value), &stages); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	return stages, nil
}

func pipelineSnapshotVariables(value string) ([]repository.VariableDeclaration, error) {
	if strings.TrimSpace(value) == "" {
		return []repository.VariableDeclaration{}, nil
	}
	var variables []repository.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot variables")
	}
	return variables, nil
}

package pipelinesvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pipelinedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/dto"
	pipelinevariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/rule/pipelinevariable"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) PipelineSnapshotForUser(ctx context.Context, userId string, snapshotId string) (pipelinedto.PipelineSnapshotDetail, error) {
	snapshotId = strings.TrimSpace(snapshotId)
	snapshot, err := s.store.PipelineSnapshot(ctx, snapshotId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return pipelinedto.PipelineSnapshotDetail{}, apperror.New(apperror.KindNotFound, "PipelineSnapshot "+snapshotId+" not found")
		}
		return pipelinedto.PipelineSnapshotDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	if snapshot.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *snapshot.ProjectId, userId); err != nil {
			return pipelinedto.PipelineSnapshotDetail{}, err
		}
	}
	stages, err := pipelinevariable.PipelineSnapshotStages(snapshot.StagesSnapshot)
	if err != nil {
		return pipelinedto.PipelineSnapshotDetail{}, err
	}
	variables, err := pipelineSnapshotVariables(snapshot.VariablesSnapshot)
	if err != nil {
		return pipelinedto.PipelineSnapshotDetail{}, err
	}
	return pipelinedto.PipelineSnapshotDetail{Snapshot: snapshot, StagesSnapshot: stages, VariablesSnapshot: variables}, nil
}

func pipelineStageArtifactConfigs(stage model.PipelineStage) ([]model.ArtifactConfig, error) {
	artifacts := []model.ArtifactConfig(nil)
	if stage.Artifacts == nil || strings.TrimSpace(*stage.Artifacts) == "" {
		return artifacts, nil
	}
	if err := json.Unmarshal([]byte(*stage.Artifacts), &artifacts); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline stage artifacts")
	}
	return artifacts, nil
}

func pipelineSnapshotVariables(value string) ([]model.VariableDeclaration, error) {
	if strings.TrimSpace(value) == "" {
		return []model.VariableDeclaration{}, nil
	}
	var variables []model.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot variables")
	}
	return variables, nil
}

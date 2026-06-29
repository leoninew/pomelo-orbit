package cisvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"strings"

	"backend/internal/apperror"
	"backend/internal/repository"
	"backend/internal/repository/model"
)

type PipelineSnapshotDetail struct {
	Snapshot          model.PipelineSnapshot
	StagesSnapshot    []model.StageDefinition
	VariablesSnapshot []model.VariableDeclaration
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

func (s Service) getOrCreatePipelineSnapshot(ctx context.Context, template model.PipelineTemplate) (model.PipelineSnapshot, error) {
	latest, err := s.store.LatestPipelineSnapshot(ctx, template.Id)
	if err == nil && latest.Version == template.Version {
		return latest, nil
	}
	if err != nil && !errors.Is(err, sql.ErrNoRows) {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	if err == nil && latest.Version != template.Version {
		// Keep the old snapshot immutable and create one for the current template version.
	}
	snapshot, err := s.createPipelineSnapshot(ctx, template)
	if err == nil {
		return snapshot, nil
	}
	latest, latestErr := s.store.LatestPipelineSnapshot(ctx, template.Id)
	if latestErr == nil && latest.Version == template.Version {
		return latest, nil
	}
	return model.PipelineSnapshot{}, err
}

func (s Service) createPipelineSnapshot(ctx context.Context, template model.PipelineTemplate) (model.PipelineSnapshot, error) {
	stages, err := s.pipelineTemplateSnapshotStages(ctx, template)
	if err != nil {
		return model.PipelineSnapshot{}, err
	}
	stageData, err := json.Marshal(stages)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline snapshot stages", err)
	}
	custom, err := pipelineTemplateVariables(template.VariableDeclarations)
	if err != nil {
		return model.PipelineSnapshot{}, err
	}
	variables, err := variableDeclarationsFromMaps(resolveTemplateVariablesFromStageDefinitions(stages, custom))
	if err != nil {
		return model.PipelineSnapshot{}, err
	}
	variableData, err := json.Marshal(variables)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline snapshot variables", err)
	}
	snapshot := model.PipelineSnapshot{Id: repository.NewId(), ProjectId: template.ProjectId, TemplateId: template.Id, Version: template.Version, StagesSnapshot: string(stageData), VariablesSnapshot: string(variableData)}
	if err := s.store.CreatePipelineSnapshot(ctx, snapshot); err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline snapshot", err)
	}
	created, err := s.store.PipelineSnapshot(ctx, snapshot.Id)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	return created, nil
}

func (s Service) pipelineTemplateSnapshotStages(ctx context.Context, template model.PipelineTemplate) ([]model.StageDefinition, error) {
	projectId := pipelineTemplateProjectId(template)
	orchestration, err := s.pipelineTemplateOrchestration(ctx, template.Id)
	if err != nil {
		return nil, err
	}
	stageIds := make([]string, 0, len(orchestration))
	seen := map[string]struct{}{}
	for _, item := range orchestration {
		if _, exists := seen[item.StageId]; !exists {
			stageIds = append(stageIds, item.StageId)
			seen[item.StageId] = struct{}{}
		}
	}
	stages, err := s.store.BuildStagesByIds(ctx, projectId, stageIds)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load build stages", err)
	}
	stageMap := make(map[string]model.BuildStage, len(stages))
	for _, stage := range stages {
		stageMap[stage.Id] = stage
	}
	definitions := make([]model.StageDefinition, 0, len(orchestration))
	missing := make([]string, 0)
	for _, item := range orchestration {
		stage, exists := stageMap[item.StageId]
		if !exists {
			missing = append(missing, item.StageId)
			continue
		}
		artifacts, err := buildStageArtifactConfigs(stage)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, model.StageDefinition{Name: item.StageName, Id: stage.Id, Image: stage.Image, Version: stage.Version, DependsOn: item.DependsOn, Script: stage.Script, Artifacts: artifacts})
	}
	if len(missing) > 0 {
		return nil, apperror.New(apperror.KindNotFound, "Stage(s) not found: "+strings.Join(missing, ", "))
	}
	return definitions, nil
}

func buildStageArtifactConfigs(stage model.BuildStage) ([]model.ArtifactConfig, error) {
	artifacts := []model.ArtifactConfig(nil)
	if stage.Artifacts == nil || strings.TrimSpace(*stage.Artifacts) == "" {
		return artifacts, nil
	}
	if err := json.Unmarshal([]byte(*stage.Artifacts), &artifacts); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid build stage artifacts")
	}
	return artifacts, nil
}

func pipelineSnapshotStages(value string) ([]model.StageDefinition, error) {
	if strings.TrimSpace(value) == "" {
		return []model.StageDefinition{}, nil
	}
	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(value), &stages); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	return stages, nil
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

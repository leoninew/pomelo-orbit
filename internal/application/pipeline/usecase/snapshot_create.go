package pipelinesvc

import (
	"context"
	"encoding/json"
	"errors"

	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type SnapshotStore interface {
	LatestPipelineSnapshot(ctx context.Context, pipelineId string) (model.PipelineSnapshot, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error
	ApplicationPipelineStages(ctx context.Context, pipelineId string) ([]model.PipelineStage, error)
}

// GetOrCreatePipelineSnapshot only accepts an application Pipeline. Templates
// have a version for provenance, but are not executable inputs.
func GetOrCreatePipelineSnapshot(ctx context.Context, store SnapshotStore, pipeline model.Pipeline, repo model.Repository) (model.PipelineSnapshot, error) {
	if pipeline.Kind != model.PipelineKindApplication {
		return model.PipelineSnapshot{}, apperror.New(apperror.KindValidation, "template pipelines cannot create snapshots")
	}
	latest, err := store.LatestPipelineSnapshot(ctx, pipeline.Id)
	if err == nil && latest.PipelineVersion == pipeline.Version {
		return latest, nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	stages, err := store.ApplicationPipelineStages(ctx, pipeline.Id)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stages", err)
	}
	definitions, err := pipelineStageDefinitions(stages)
	if err != nil {
		return model.PipelineSnapshot{}, err
	}
	if err := validatePipelineDAG(definitions); err != nil {
		return model.PipelineSnapshot{}, apperror.New(apperror.KindValidation, err.Error())
	}
	stageData, err := json.Marshal(definitions)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline snapshot stages", err)
	}
	variables, err := pipelinevariable.RuntimeVariableDeclarations(repo, pipeline, definitions)
	if err != nil {
		return model.PipelineSnapshot{}, err
	}
	variableData, err := json.Marshal(variables)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline snapshot variables", err)
	}
	if pipeline.SourcePipelineId == nil || pipeline.SourceTemplateName == nil || pipeline.SourceTemplateVersion == nil || pipeline.RepositoryId == nil || pipeline.RepositoryName == nil || (pipeline.ApplicationId == nil) != (pipeline.ApplicationName == nil) {
		return model.PipelineSnapshot{}, apperror.New(apperror.KindValidation, "application pipeline identity is incomplete")
	}
	snapshot := model.PipelineSnapshot{
		Id: idutil.NewId(), ProjectId: pipeline.ProjectId, PipelineId: pipeline.Id, PipelineName: pipeline.Name, PipelineVersion: pipeline.Version,
		SourcePipelineId: *pipeline.SourcePipelineId, SourceTemplateName: *pipeline.SourceTemplateName, SourceTemplateVersion: *pipeline.SourceTemplateVersion,
		ApplicationId: pipeline.ApplicationId, ApplicationName: pipeline.ApplicationName,
		RepositoryId: *pipeline.RepositoryId, RepositoryName: *pipeline.RepositoryName,
		VersionForkStrategy: pipeline.VersionForkStrategy, FixedVersionId: pipeline.FixedVersionId, FixedVersionLabel: pipeline.FixedVersionLabel,
		StagesSnapshot: string(stageData), VariablesSnapshot: string(variableData),
	}
	if err := store.CreatePipelineSnapshot(ctx, snapshot); err != nil {
		latest, latestErr := store.LatestPipelineSnapshot(ctx, pipeline.Id)
		if latestErr == nil && latest.PipelineVersion == pipeline.Version {
			return latest, nil
		}
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline snapshot", err)
	}
	created, err := store.PipelineSnapshot(ctx, snapshot.Id)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	return created, nil
}

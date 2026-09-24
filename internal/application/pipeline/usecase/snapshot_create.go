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
	LatestPipelineSnapshot(ctx context.Context, projectId string, pipelineId string) (model.PipelineSnapshot, error)
	PipelineSnapshot(ctx context.Context, projectId string, id string) (model.PipelineSnapshot, error)
	CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error
	ApplicationPipelineStages(ctx context.Context, projectId string, pipelineId string) ([]model.PipelineStage, error)
	TemplatePipelineStageReferences(ctx context.Context, projectId string, pipelineId string) ([]model.PipelineStageReference, error)
}

// GetOrCreatePipelineSnapshot materializes the immutable definition of either
// pipeline kind. Application snapshots additionally contain the resolved
// runtime variable declarations used by a Run.
func GetOrCreatePipelineSnapshot(ctx context.Context, store SnapshotStore, projectId string, pipeline model.Pipeline, repo model.Repository) (model.PipelineSnapshot, error) {
	latest, err := store.LatestPipelineSnapshot(ctx, projectId, pipeline.Id)
	if err == nil && latest.PipelineVersion == pipeline.Version {
		return latest, nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	var definitions []model.StageDefinition
	switch pipeline.Kind {
	case model.PipelineKindApplication:
		stages, loadErr := store.ApplicationPipelineStages(ctx, projectId, pipeline.Id)
		if loadErr != nil {
			return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stages", loadErr)
		}
		definitions, err = pipelineStageDefinitions(stages)
	case model.PipelineKindTemplate:
		references, loadErr := store.TemplatePipelineStageReferences(ctx, projectId, pipeline.Id)
		if loadErr != nil {
			return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage references", loadErr)
		}
		if err := validateTemplatePipelineConfiguration(pipeline, references); err != nil {
			return model.PipelineSnapshot{}, err
		}
		definitions, err = templatePipelineSnapshotDefinitions(references)
	default:
		return model.PipelineSnapshot{}, apperror.New(apperror.KindValidation, "unknown pipeline kind")
	}
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
	var variables []model.VariableDeclaration
	if pipeline.Kind == model.PipelineKindApplication {
		variables, err = pipelinevariable.RuntimeVariableDeclarations(repo, pipeline, definitions)
		if err != nil {
			return model.PipelineSnapshot{}, err
		}
	} else {
		variables, err = pipelinevariable.VariableDeclarations(pipeline.VariableDeclarations)
		if err != nil {
			return model.PipelineSnapshot{}, err
		}
	}
	variableData, err := json.Marshal(variables)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline snapshot variables", err)
	}
	snapshot := model.PipelineSnapshot{
		Id: idutil.NewId(), ProjectId: pipeline.ProjectId, PipelineId: pipeline.Id, PipelineName: pipeline.Name, PipelineVersion: pipeline.Version,
		VersionForkStrategy: pipeline.VersionForkStrategy, FixedVersionId: pipeline.FixedVersionId, FixedVersionLabel: pipeline.FixedVersionLabel,
		StagesSnapshot: string(stageData), VariablesSnapshot: string(variableData),
	}
	if pipeline.Kind == model.PipelineKindApplication {
		if pipeline.SourcePipelineId == nil || pipeline.SourceTemplateName == nil || pipeline.SourceTemplateVersion == nil || pipeline.RepositoryId == nil || pipeline.RepositoryName == nil || (pipeline.ApplicationId == nil) != (pipeline.ApplicationName == nil) {
			return model.PipelineSnapshot{}, apperror.New(apperror.KindValidation, "application pipeline identity is incomplete")
		}
		snapshot.SourcePipelineId = *pipeline.SourcePipelineId
		snapshot.SourceTemplateName = *pipeline.SourceTemplateName
		snapshot.SourceTemplateVersion = *pipeline.SourceTemplateVersion
		snapshot.ApplicationId = pipeline.ApplicationId
		snapshot.ApplicationName = pipeline.ApplicationName
		snapshot.RepositoryId = *pipeline.RepositoryId
		snapshot.RepositoryName = *pipeline.RepositoryName
	}
	if err := store.CreatePipelineSnapshot(ctx, snapshot); err != nil {
		latest, latestErr := store.LatestPipelineSnapshot(ctx, projectId, pipeline.Id)
		if latestErr == nil && latest.PipelineVersion == pipeline.Version {
			return latest, nil
		}
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline snapshot", err)
	}
	created, err := store.PipelineSnapshot(ctx, projectId, snapshot.Id)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	return created, nil
}

func templatePipelineSnapshotDefinitions(references []model.PipelineStageReference) ([]model.StageDefinition, error) {
	definitions := make([]model.StageDefinition, 0, len(references))
	for _, reference := range references {
		artifacts := reference.Artifacts
		artifactDefinitions, err := templateStageArtifacts(model.PipelineStage{Artifacts: &artifacts})
		if err != nil {
			return nil, err
		}
		dependsOn, err := dependsOnFromJSON(reference.DependsOn)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, model.StageDefinition{
			Id: reference.Id, Name: reference.Name, Image: reference.Image, Script: reference.Script,
			Artifacts: artifactDefinitions, DependsOn: dependsOn, SortOrder: reference.SortOrder,
			Description: reference.Description, SourceTemplateStageId: reference.SourceTemplateStageId,
			SourceTemplateStageName: reference.SourceTemplateStageName, SourceTemplateStageVersion: reference.SourceTemplateStageVersion, SourceTemplateStageDescription: reference.SourceTemplateStageDescription,
		})
	}
	return definitions, nil
}

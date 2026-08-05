package pipelinesvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pipelinevariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/rule/pipelinevariable"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

// SnapshotStore is the pipeline-store surface needed to materialize a template snapshot.
type SnapshotStore interface {
	LatestPipelineSnapshot(ctx context.Context, templateId string) (model.PipelineSnapshot, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error
	PipelineTemplateStages(ctx context.Context, templateId string) ([]model.PipelineTemplateStage, error)
	PipelineStagesByIds(ctx context.Context, projectId string, ids []string) ([]model.PipelineStage, error)
}

// GetOrCreatePipelineSnapshot returns the current snapshot for a template version, creating it if needed.
func GetOrCreatePipelineSnapshot(ctx context.Context, store SnapshotStore, template model.PipelineTemplate) (model.PipelineSnapshot, error) {
	latest, err := store.LatestPipelineSnapshot(ctx, template.Id)
	if err == nil && latest.Version == template.Version {
		return latest, nil
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	snapshot, err := createPipelineSnapshot(ctx, store, template)
	if err == nil {
		return snapshot, nil
	}
	latest, latestErr := store.LatestPipelineSnapshot(ctx, template.Id)
	if latestErr == nil && latest.Version == template.Version {
		return latest, nil
	}
	return model.PipelineSnapshot{}, err
}

func createPipelineSnapshot(ctx context.Context, store SnapshotStore, template model.PipelineTemplate) (model.PipelineSnapshot, error) {
	stages, err := pipelineTemplateSnapshotStages(ctx, store, template)
	if err != nil {
		return model.PipelineSnapshot{}, err
	}
	stageData, err := json.Marshal(stages)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline snapshot stages", err)
	}
	custom, err := pipelinevariable.PipelineTemplateVariables(template.VariableDeclarations)
	if err != nil {
		return model.PipelineSnapshot{}, err
	}
	variables, err := pipelinevariable.VariableDeclarationsFromMaps(pipelinevariable.ResolveTemplateVariablesFromStageDefinitions(stages, custom))
	if err != nil {
		return model.PipelineSnapshot{}, err
	}
	variableData, err := json.Marshal(variables)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Invalid pipeline snapshot variables", err)
	}
	snapshot := model.PipelineSnapshot{
		Id:                idutil.NewId(),
		ProjectId:         template.ProjectId,
		TemplateId:        template.Id,
		Version:           template.Version,
		StagesSnapshot:    string(stageData),
		VariablesSnapshot: string(variableData),
	}
	if err := store.CreatePipelineSnapshot(ctx, snapshot); err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline snapshot", err)
	}
	created, err := store.PipelineSnapshot(ctx, snapshot.Id)
	if err != nil {
		return model.PipelineSnapshot{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	return created, nil
}

func pipelineTemplateSnapshotStages(ctx context.Context, store SnapshotStore, template model.PipelineTemplate) ([]model.StageDefinition, error) {
	projectId := pipelineTemplateProjectId(template)
	rows, err := store.PipelineTemplateStages(ctx, template.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template stages", err)
	}
	orchestration := make([]struct {
		StageId   string
		StageName string
		DependsOn []string
	}, 0, len(rows))
	for _, row := range rows {
		dependsOn, err := pipelineTemplateDependsOn(row.DependsOn)
		if err != nil {
			return nil, err
		}
		orchestration = append(orchestration, struct {
			StageId   string
			StageName string
			DependsOn []string
		}{StageId: row.StageId, StageName: row.StageName, DependsOn: dependsOn})
	}
	stageIds := make([]string, 0, len(orchestration))
	seen := map[string]struct{}{}
	for _, item := range orchestration {
		if _, exists := seen[item.StageId]; !exists {
			stageIds = append(stageIds, item.StageId)
			seen[item.StageId] = struct{}{}
		}
	}
	stages, err := store.PipelineStagesByIds(ctx, projectId, stageIds)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stages", err)
	}
	stageMap := make(map[string]model.PipelineStage, len(stages))
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
		artifacts, err := pipelineStageArtifactConfigs(stage)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, model.StageDefinition{
			Name: item.StageName, Id: stage.Id, Image: stage.Image, Version: stage.Version,
			DependsOn: item.DependsOn, Script: stage.Script, Artifacts: artifacts,
			BuildVersionBinding: cloneBuildVersionBinding(stage.BuildVersionBinding),
		})
	}
	if len(missing) > 0 {
		return nil, apperror.New(apperror.KindNotFound, "Stage(s) not found: "+strings.Join(missing, ", "))
	}
	if err := validateSourceCommitDependencies(definitions); err != nil {
		return nil, apperror.New(apperror.KindValidation, err.Error())
	}
	return definitions, nil
}

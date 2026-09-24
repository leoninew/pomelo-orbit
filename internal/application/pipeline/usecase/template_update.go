package pipelinesvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"

	pipelinedto "github.com/leoninew/pomelo-orbit/internal/application/pipeline/dto"
	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

const (
	pipelineTemplateUpdateStatusUpdate   = "update"
	pipelineTemplateUpdateStatusAdded    = "added"
	pipelineTemplateUpdateStatusKeep     = "keep"
	pipelineTemplateUpdateStatusConflict = "conflict"
)

func (s Service) PipelineTemplateUpdatePreviewForUser(ctx context.Context, userId, projectId, pipelineId string) (pipelinedto.PipelineTemplateUpdatePreview, error) {
	pipeline, err := s.loadPipelineForUser(ctx, userId, projectId, pipelineId)
	if err != nil {
		return pipelinedto.PipelineTemplateUpdatePreview{}, err
	}
	return s.pipelineTemplateUpdatePreview(ctx, projectId, pipeline)
}

func (s Service) ApplyPipelineTemplateUpdate(ctx context.Context, userId, projectId, pipelineId string, input pipelinedto.PipelineTemplateApplyUpdateInput) (pipelinedto.PipelineDetail, error) {
	pipeline, err := s.loadPipelineForUser(ctx, userId, projectId, pipelineId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if pipeline.Kind != model.PipelineKindApplication || pipeline.SourcePipelineId == nil || pipeline.SourceTemplateVersion == nil {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "only application pipelines created from a template can be updated")
	}
	if pipeline.Version != input.ExpectedPipelineVersion || *pipeline.SourceTemplateVersion != input.ExpectedSourceTemplateVersion {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindConflict, "Pipeline changed; reload the template update preview")
	}
	template, err := s.store.Pipeline(ctx, projectId, *pipeline.SourcePipelineId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindConflict, "source template pipeline no longer exists")
		}
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load source template pipeline", err)
	}
	preview, stages, variables, err := s.buildPipelineTemplateUpdate(ctx, projectId, pipeline, template)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if !preview.Available || len(preview.Conflicts) != 0 || preview.TargetSourceTemplateVersion != input.TargetSourceTemplateVersion {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindConflict, "template update preview is no longer applicable")
	}
	pipeline.SourceTemplateName = stringPtr(template.Name)
	pipeline.SourceTemplateVersion = intPtr(template.Version)
	pipeline.VariableDeclarations = variables
	pipeline.Version++
	for index := range stages {
		stages[index].ProjectId = projectId
		pipelineId := pipeline.Id
		stages[index].PipelineId = &pipelineId
	}
	if err := s.validatePipelineConfiguration(ctx, projectId, pipeline, stages); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if err := s.store.UpdateApplicationPipelineWithStagesIfVersion(ctx, projectId, pipeline, stages, input.ExpectedPipelineVersion); err != nil {
		if errors.Is(err, repository.ErrStateConflict) {
			return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindConflict, "Pipeline changed; reload the template update preview")
		}
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update application pipeline", err)
	}
	if _, err := GetOrCreatePipelineSnapshot(ctx, s.store, projectId, template, model.Repository{}); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	return s.pipelineDetail(ctx, projectId, pipeline)
}

func (s Service) pipelineTemplateUpdatePreview(ctx context.Context, projectId string, pipeline model.Pipeline) (pipelinedto.PipelineTemplateUpdatePreview, error) {
	if pipeline.Kind != model.PipelineKindApplication || pipeline.SourcePipelineId == nil || pipeline.SourceTemplateVersion == nil {
		return pipelinedto.PipelineTemplateUpdatePreview{}, apperror.New(apperror.KindValidation, "only application pipelines created from a template can be updated")
	}
	template, err := s.store.Pipeline(ctx, projectId, *pipeline.SourcePipelineId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return pipelinedto.PipelineTemplateUpdatePreview{Available: false, Conflicts: []string{"source template pipeline no longer exists"}}, nil
		}
		return pipelinedto.PipelineTemplateUpdatePreview{}, apperror.Wrap(apperror.KindInternal, "Failed to load source template pipeline", err)
	}
	preview, _, _, err := s.buildPipelineTemplateUpdate(ctx, projectId, pipeline, template)
	return preview, err
}

func (s Service) buildPipelineTemplateUpdate(ctx context.Context, projectId string, pipeline, template model.Pipeline) (pipelinedto.PipelineTemplateUpdatePreview, []model.PipelineStage, string, error) {
	preview := pipelinedto.PipelineTemplateUpdatePreview{
		Available: true, ExpectedPipelineVersion: pipeline.Version, ExpectedSourceTemplateVersion: valueOrZero(pipeline.SourceTemplateVersion),
		TargetSourceTemplateVersion: template.Version, SourceTemplateName: template.Name,
	}
	if template.Kind != model.PipelineKindTemplate {
		preview.Available = false
		preview.Conflicts = []string{"source pipeline is not a template pipeline"}
		return preview, nil, "", nil
	}
	if template.Version == *pipeline.SourceTemplateVersion {
		preview.Available = false
		return preview, nil, "", nil
	}
	if template.Version < *pipeline.SourceTemplateVersion {
		preview.Available = false
		preview.Conflicts = []string{"source template version cannot move backwards"}
		return preview, nil, "", nil
	}
	oldSnapshot, err := s.store.PipelineSnapshotAtVersion(ctx, projectId, template.Id, *pipeline.SourceTemplateVersion)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			preview.Available = false
			preview.Conflicts = []string{"adopted template version snapshot is missing"}
			return preview, nil, "", nil
		}
		return preview, nil, "", apperror.Wrap(apperror.KindInternal, "Failed to load adopted template snapshot", err)
	}
	oldStages, err := snapshotStages(oldSnapshot.StagesSnapshot)
	if err != nil {
		return preview, nil, "", err
	}
	oldVariables, err := snapshotVariables(oldSnapshot.VariablesSnapshot)
	if err != nil {
		return preview, nil, "", err
	}
	references, err := s.store.TemplatePipelineStageReferences(ctx, projectId, template.Id)
	if err != nil {
		return preview, nil, "", apperror.Wrap(apperror.KindInternal, "Failed to load source template stage references", err)
	}
	if err := validateTemplatePipelineConfiguration(template, references); err != nil {
		return preview, nil, "", err
	}
	currentStages, err := s.store.ApplicationPipelineStages(ctx, projectId, pipeline.Id)
	if err != nil {
		return preview, nil, "", apperror.Wrap(apperror.KindInternal, "Failed to load application pipeline stages", err)
	}
	mergedStages, stageUpdates, conflicts, err := mergeApplicationStages(currentStages, references, oldStages)
	if err != nil {
		return preview, nil, "", err
	}
	preview.Stages = stageUpdates
	preview.Conflicts = append(preview.Conflicts, conflicts...)
	variables, variableUpdates, variableConflicts, err := mergeApplicationVariables(pipeline.VariableDeclarations, template.VariableDeclarations, oldVariables, oldStages, references, mergedStages)
	if err != nil {
		return preview, nil, "", err
	}
	preview.Variables = variableUpdates
	preview.Conflicts = append(preview.Conflicts, variableConflicts...)
	preview.Available = len(preview.Conflicts) == 0
	return preview, mergedStages, variables, nil
}

func mergeApplicationStages(current []model.PipelineStage, references []model.PipelineStageReference, old []model.StageDefinition) ([]model.PipelineStage, []pipelinedto.PipelineTemplateStageUpdate, []string, error) {
	currentBySource := make(map[string]model.PipelineStage, len(current))
	for _, stage := range current {
		if stage.SourceTemplateStageId != nil && *stage.SourceTemplateStageId != "" {
			if _, exists := currentBySource[*stage.SourceTemplateStageId]; exists {
				continue
			}
			currentBySource[*stage.SourceTemplateStageId] = stage
		}
	}
	referenceBySource := make(map[string]model.PipelineStageReference, len(references))
	conflicts := []string{}
	for _, reference := range references {
		if previous, exists := referenceBySource[reference.SourceTemplateStageId]; exists && previous.SourceTemplateStageVersion == reference.SourceTemplateStageVersion && !sameReferenceContent(previous, reference) {
			conflicts = append(conflicts, "template contains conflicting references for stage "+reference.SourceTemplateStageId)
			continue
		}
		if previous, exists := referenceBySource[reference.SourceTemplateStageId]; !exists || previous.SourceTemplateStageVersion < reference.SourceTemplateStageVersion {
			referenceBySource[reference.SourceTemplateStageId] = reference
		}
	}
	oldBySource := make(map[string]model.StageDefinition, len(old))
	for _, stage := range old {
		if stage.SourceTemplateStageId != "" {
			oldBySource[stage.SourceTemplateStageId] = stage
		}
	}
	updates := make([]pipelinedto.PipelineTemplateStageUpdate, 0, len(current)+len(references))
	merged := make([]model.PipelineStage, 0, len(current)+len(references))
	seen := make(map[string]struct{}, len(current))
	for _, stage := range current {
		if stage.SourceTemplateStageId == nil || *stage.SourceTemplateStageId == "" {
			conflicts = append(conflicts, "application stage "+stage.Name+" has no template source")
			merged = append(merged, stage)
			continue
		}
		sourceId := *stage.SourceTemplateStageId
		target, exists := referenceBySource[sourceId]
		if !exists {
			conflicts = append(conflicts, "template stage removed: "+sourceId)
			merged = append(merged, stage)
			continue
		}
		seen[sourceId] = struct{}{}
		currentVersion := valueOrZero(stage.SourceTemplateStageVersion)
		status := pipelineTemplateUpdateStatusKeep
		contentChangedAtSameVersion := false
		if oldStage, exists := oldBySource[sourceId]; exists && target.SourceTemplateStageVersion == currentVersion {
			contentChangedAtSameVersion = referenceDiffersFromSnapshot(target, oldStage)
		}
		if target.SourceTemplateStageVersion > currentVersion || (target.SourceTemplateStageVersion == currentVersion && contentChangedAtSameVersion) {
			stage.Image, stage.Script, stage.Description = target.Image, target.Script, target.Description
			stage.SourceTemplateStageName, stage.SourceTemplateStageDescription = stringPtr(target.SourceTemplateStageName), stringPtr(target.SourceTemplateStageDescription)
			stage.SourceTemplateStageVersion = intPtr(target.SourceTemplateStageVersion)
			artifacts := target.Artifacts
			mergedArtifacts, err := mergeTemplateStageArtifacts(model.PipelineStage{Artifacts: &artifacts}, stage.Artifacts)
			if err != nil {
				return nil, nil, nil, err
			}
			dependsOn := stage.DependsOn
			stage.Artifacts = mergedArtifacts
			stage.DependsOn = dependsOn
			status = pipelineTemplateUpdateStatusUpdate
		}
		updates = append(updates, pipelinedto.PipelineTemplateStageUpdate{StageId: stage.Id, SourceTemplateStageId: sourceId, Status: status, CurrentVersion: currentVersion, TargetVersion: target.SourceTemplateStageVersion})
		merged = append(merged, stage)
	}
	stageIdBySource := make(map[string]string, len(currentBySource)+len(referenceBySource))
	for sourceId, stage := range currentBySource {
		stageIdBySource[sourceId] = stage.Id
	}
	for sourceId := range referenceBySource {
		if _, exists := stageIdBySource[sourceId]; !exists {
			stageIdBySource[sourceId] = idutil.NewId()
		}
	}
	stageIdByReference := make(map[string]string, len(references))
	for _, reference := range references {
		stageIdByReference[reference.Id] = stageIdBySource[reference.SourceTemplateStageId]
	}
	for _, reference := range references {
		sourceId := reference.SourceTemplateStageId
		if _, exists := seen[sourceId]; exists {
			continue
		}
		if referenceBySource[sourceId].Id != reference.Id {
			continue
		}
		id := stageIdBySource[sourceId]
		pipelineId := ""
		stage := model.PipelineStage{Id: id, ProjectId: "", Kind: model.PipelineStageKindApplication, PipelineId: &pipelineId, Name: reference.Name, Image: reference.Image, Script: reference.Script, Description: reference.Description}
		stage.SourceTemplateStageId = stringPtr(sourceId)
		stage.SourceTemplateStageName = stringPtr(reference.SourceTemplateStageName)
		stage.SourceTemplateStageVersion = intPtr(reference.SourceTemplateStageVersion)
		stage.SourceTemplateStageDescription = stringPtr(reference.SourceTemplateStageDescription)
		artifacts, dependsOn, sortOrder := reference.Artifacts, reference.DependsOn, reference.SortOrder
		dependencies, err := dependsOnFromJSON(dependsOn)
		if err != nil {
			conflicts = append(conflicts, "invalid dependencies for template stage "+sourceId)
			continue
		}
		for index, dependency := range dependencies {
			if mapped, exists := stageIdByReference[dependency]; exists {
				dependencies[index] = mapped
			}
		}
		dependsOnData, err := marshalDependsOn(dependencies)
		if err != nil {
			conflicts = append(conflicts, "invalid dependencies for template stage "+sourceId)
			continue
		}
		dependsOn = dependsOnData
		stage.Artifacts, stage.DependsOn, stage.SortOrder = &artifacts, &dependsOn, &sortOrder
		updates = append(updates, pipelinedto.PipelineTemplateStageUpdate{StageId: id, SourceTemplateStageId: sourceId, Status: pipelineTemplateUpdateStatusAdded, TargetVersion: reference.SourceTemplateStageVersion})
		merged = append(merged, stage)
	}
	return merged, updates, conflicts, nil
}

func mergeApplicationVariables(currentJSON, targetJSON string, old []model.VariableDeclaration, oldStages []model.StageDefinition, references []model.PipelineStageReference, stages []model.PipelineStage) (string, []pipelinedto.PipelineTemplateVariableUpdate, []string, error) {
	current, err := pipelinevariable.VariableDeclarations(currentJSON)
	if err != nil {
		return "", nil, nil, err
	}
	target, err := pipelinevariable.VariableDeclarations(targetJSON)
	if err != nil {
		return "", nil, nil, err
	}
	oldByKey := variableBySourceKey(old, oldStages, references)
	currentByKey := make(map[string]model.VariableDeclaration, len(current))
	for _, item := range current {
		currentByKey[variableKey(item.Name, item.StageId)] = item
	}
	stageIdBySource := make(map[string]string, len(stages))
	sourceByReference := make(map[string]string, len(references))
	for _, stage := range stages {
		if stage.SourceTemplateStageId != nil {
			stageIdBySource[*stage.SourceTemplateStageId] = stage.Id
		}
	}
	for _, reference := range references {
		sourceByReference[reference.Id] = reference.SourceTemplateStageId
	}
	oldSourceByReference := make(map[string]string, len(oldStages))
	for _, stage := range oldStages {
		if stage.SourceTemplateStageId != "" {
			oldSourceByReference[stage.Id] = stage.SourceTemplateStageId
		}
	}
	result := append([]model.VariableDeclaration(nil), current...)
	updates := []pipelinedto.PipelineTemplateVariableUpdate{}
	conflicts := []string{}
	seen := make(map[string]struct{}, len(target))
	for _, item := range target {
		sourceStageId := item.StageId
		appStageId := item.StageId
		if item.StageId != "" {
			sourceStageId = sourceByReference[item.StageId]
			appStageId = stageIdBySource[sourceStageId]
			if appStageId == "" {
				conflicts = append(conflicts, "template variable stage is not present: "+item.StageId)
				continue
			}
		}
		item.StageId = appStageId
		key := variableKey(item.Name, appStageId)
		seen[key] = struct{}{}
		oldItem, hadOld := oldByKey[variableKey(item.Name, sourceStageId)]
		currentItem, hadCurrent := currentByKey[key]
		status := pipelineTemplateUpdateStatusAdded
		if hadCurrent {
			status = pipelineTemplateUpdateStatusKeep
			if hadOld && reflect.DeepEqual(currentItem.Value, oldItem.Value) {
				replaceVariable(result, key, item)
				status = pipelineTemplateUpdateStatusUpdate
			}
		} else {
			result = append(result, item)
		}
		updates = append(updates, pipelinedto.PipelineTemplateVariableUpdate{Name: item.Name, StageId: item.StageId, Status: status, Current: fmt.Sprint(currentItem.Value), Target: fmt.Sprint(item.Value)})
	}
	for _, item := range old {
		stageId := item.StageId
		if sourceId := oldSourceByReference[item.StageId]; sourceId != "" {
			stageId = stageIdBySource[sourceId]
		}
		key := variableKey(item.Name, stageId)
		if _, exists := seen[key]; exists {
			continue
		}
		if _, exists := currentByKey[key]; exists {
			conflicts = append(conflicts, "template variable removed: "+item.Name)
		}
	}
	data, err := json.Marshal(result)
	if err != nil {
		return "", nil, nil, apperror.Wrap(apperror.KindInternal, "Invalid merged pipeline variables", err)
	}
	return string(data), updates, conflicts, nil
}

func variableBySourceKey(items []model.VariableDeclaration, oldStages []model.StageDefinition, references []model.PipelineStageReference) map[string]model.VariableDeclaration {
	refSourceById := make(map[string]string, len(references))
	for _, reference := range references {
		refSourceById[reference.Id] = reference.SourceTemplateStageId
	}
	for _, stage := range oldStages {
		if stage.SourceTemplateStageId != "" {
			refSourceById[stage.Id] = stage.SourceTemplateStageId
		}
	}
	result := make(map[string]model.VariableDeclaration, len(items))
	for _, item := range items {
		stageId := item.StageId
		if sourceId, exists := refSourceById[stageId]; exists {
			stageId = sourceId
		}
		result[variableKey(item.Name, stageId)] = item
	}
	return result
}

func variableKey(name, stageId string) string { return strings.TrimSpace(name) + "\x00" + stageId }

func replaceVariable(items []model.VariableDeclaration, key string, replacement model.VariableDeclaration) {
	for index := range items {
		if variableKey(items[index].Name, items[index].StageId) == key {
			items[index] = replacement
			return
		}
	}
}

func sameReferenceContent(left, right model.PipelineStageReference) bool {
	return left.Name == right.Name && left.Image == right.Image && left.Script == right.Script && left.Description == right.Description && left.Artifacts == right.Artifacts && left.DependsOn == right.DependsOn && left.SortOrder == right.SortOrder
}

func referenceDiffersFromSnapshot(reference model.PipelineStageReference, snapshot model.StageDefinition) bool {
	artifacts := reference.Artifacts
	targetArtifacts, err := templateStageArtifacts(model.PipelineStage{Artifacts: &artifacts})
	if err != nil {
		return true
	}
	targetDependsOn, err := dependsOnFromJSON(reference.DependsOn)
	if err != nil {
		return true
	}
	return reference.Name != snapshot.Name || reference.Image != snapshot.Image || reference.Script != snapshot.Script || reference.SourceTemplateStageName != snapshot.SourceTemplateStageName || reference.SourceTemplateStageVersion != snapshot.SourceTemplateStageVersion || reference.SourceTemplateStageDescription != snapshot.SourceTemplateStageDescription || reference.Description != snapshot.Description || reference.SortOrder != snapshot.SortOrder || !reflect.DeepEqual(targetArtifacts, snapshot.Artifacts) || !reflect.DeepEqual(targetDependsOn, snapshot.DependsOn)
}

func valueOrZero(value *int) int {
	if value == nil {
		return 0
	}
	return *value
}

func stringPtr(value string) *string { return &value }

func intPtr(value int) *int { return &value }

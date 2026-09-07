package pipelinesvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	pipelinedto "github.com/leoninew/pomelo-orbit/internal/application/pipeline/dto"
	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func (s Service) ListPipelineStageTemplates(ctx context.Context, userId, projectId string, page, perPage int, search string) (repository.Page[pipelinedto.PipelineStageTemplateDetail], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[pipelinedto.PipelineStageTemplateDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[pipelinedto.PipelineStageTemplateDetail]{}, err
	}
	pageOfStages, err := s.store.ListPipelineStageTemplates(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[pipelinedto.PipelineStageTemplateDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list pipeline stages", err)
	}
	items := make([]pipelinedto.PipelineStageTemplateDetail, 0, len(pageOfStages.Items))
	for _, stage := range pageOfStages.Items {
		detail, err := pipelineStageTemplateDetail(stage)
		if err != nil {
			return repository.Page[pipelinedto.PipelineStageTemplateDetail]{}, err
		}
		items = append(items, detail)
	}
	return repository.Page[pipelinedto.PipelineStageTemplateDetail]{Items: items, Total: pageOfStages.Total, Page: pageOfStages.Page, PerPage: pageOfStages.PerPage}, nil
}

func (s Service) PipelineStageTemplateForUser(ctx context.Context, userId, stageId string) (pipelinedto.PipelineStageTemplateDetail, error) {
	stage, err := s.pipelineStageTemplateForUser(ctx, userId, stageId)
	if err != nil {
		return pipelinedto.PipelineStageTemplateDetail{}, err
	}
	return pipelineStageTemplateDetail(stage)
}

func (s Service) CreatePipelineStageTemplate(ctx context.Context, userId string, input pipelinedto.PipelineStageTemplateCreateInput) (pipelinedto.PipelineStageTemplateDetail, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return pipelinedto.PipelineStageTemplateDetail{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return pipelinedto.PipelineStageTemplateDetail{}, err
	}
	stage, err := pipelineStageTemplateFromInput(projectId, input.Name, input.Image, input.Script, input.Description, input.Artifacts)
	if err != nil {
		return pipelinedto.PipelineStageTemplateDetail{}, err
	}
	if err := s.ensurePipelineStageTemplateNameAvailable(ctx, projectId, stage.Name, ""); err != nil {
		return pipelinedto.PipelineStageTemplateDetail{}, err
	}
	if err := validatePipelineStageTemplate(stage); err != nil {
		return pipelinedto.PipelineStageTemplateDetail{}, err
	}
	if err := s.store.CreatePipelineStageTemplate(ctx, stage); err != nil {
		return pipelinedto.PipelineStageTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline stage", err)
	}
	return pipelineStageTemplateDetail(stage)
}

func (s Service) UpdatePipelineStageTemplate(ctx context.Context, userId, stageId string, input pipelinedto.PipelineStageTemplateUpdateInput) (pipelinedto.PipelineStageTemplateDetail, error) {
	if input.DependsOn != nil || input.SortOrder != nil || input.PipelineId != nil || input.VersionForkStrategy != nil || input.FixedVersionId != nil || input.ClearVersionForkStrategy {
		return pipelinedto.PipelineStageTemplateDetail{}, apperror.New(apperror.KindValidation, "pipeline stage templates cannot contain pipeline configuration")
	}
	stage, err := s.pipelineStageTemplateForUser(ctx, userId, stageId)
	if err != nil {
		return pipelinedto.PipelineStageTemplateDetail{}, err
	}
	changed := false
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return pipelinedto.PipelineStageTemplateDetail{}, apperror.New(apperror.KindValidation, "name is required")
		}
		if err := s.ensurePipelineStageTemplateNameAvailable(ctx, stage.ProjectId, name, stage.Id); err != nil {
			return pipelinedto.PipelineStageTemplateDetail{}, err
		}
		if stage.Name != name {
			stage.Name, changed = name, true
		}
	}
	if input.Image != nil {
		image := strings.TrimSpace(*input.Image)
		if image == "" {
			return pipelinedto.PipelineStageTemplateDetail{}, apperror.New(apperror.KindValidation, "image is required")
		}
		if stage.Image != image {
			stage.Image, changed = image, true
		}
	}
	if input.Script != nil {
		script := strings.TrimSpace(*input.Script)
		if stage.Script != script {
			stage.Script, changed = script, true
		}
	}
	if input.Description != nil && stage.Description != *input.Description {
		stage.Description, changed = *input.Description, true
	}
	if input.Artifacts != nil {
		artifacts, err := marshalTemplateStageArtifacts(*input.Artifacts)
		if err != nil {
			return pipelinedto.PipelineStageTemplateDetail{}, err
		}
		if !stringPointerEqual(stage.Artifacts, artifacts) {
			stage.Artifacts, changed = artifacts, true
		}
	}
	if changed {
		if stage.Version == nil {
			return pipelinedto.PipelineStageTemplateDetail{}, apperror.New(apperror.KindValidation, "pipeline stage template version is required")
		}
		version := *stage.Version + 1
		stage.Version = &version
		if err := validatePipelineStageTemplate(stage); err != nil {
			return pipelinedto.PipelineStageTemplateDetail{}, err
		}
		if err := s.store.UpdatePipelineStageTemplate(ctx, stage); err != nil {
			return pipelinedto.PipelineStageTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline stage", err)
		}
	}
	return pipelineStageTemplateDetail(stage)
}

func (s Service) DeletePipelineStageTemplate(ctx context.Context, userId, stageId string) error {
	stage, err := s.pipelineStageTemplateForUser(ctx, userId, stageId)
	if err != nil {
		return err
	}
	if err := s.store.DeletePipelineStageTemplate(ctx, stage.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete pipeline stage", err)
	}
	return nil
}

func (s Service) ImportPipelineStage(ctx context.Context, userId, pipelineId string, input pipelinedto.PipelineStageImportInput) (pipelinedto.PipelineDetail, error) {
	pipeline, err := s.loadPipelineForUser(ctx, userId, pipelineId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	template, err := s.pipelineStageTemplateForUser(ctx, userId, input.SourceTemplateStageId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if template.ProjectId != pipelineProjectId(pipeline) {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindNotFound, "Pipeline stage template not found")
	}
	name, description, dependsOn, err := importNodeFields(template, input)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if pipeline.Kind == model.PipelineKindTemplate {
		references, err := s.store.TemplatePipelineStageReferences(ctx, pipeline.Id)
		if err != nil {
			return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage references", err)
		}
		artifacts, err := templateStageArtifactsJSON(template)
		if err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		references = append(references, model.PipelineStageReference{Id: idutil.NewId(), PipelineId: pipeline.Id, SourceTemplateStageId: template.Id, SourceTemplateStageName: template.Name, SourceTemplateStageVersion: requiredStageVersion(template), SourceTemplateStageDescription: template.Description, Name: name, Image: template.Image, Script: template.Script, Description: description, Artifacts: artifacts, DependsOn: dependsOn, SortOrder: input.SortOrder})
		if err := validateTemplatePipelineConfiguration(pipeline, references); err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		pipeline.Version++
		if err := s.store.UpdateTemplatePipelineWithReferences(ctx, pipeline, references); err != nil {
			return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to import pipeline stage", err)
		}
		return s.pipelineDetail(ctx, pipeline)
	}
	if pipeline.Kind != model.PipelineKindApplication {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "unknown pipeline kind")
	}
	stages, err := s.store.ApplicationPipelineStages(ctx, pipeline.Id)
	if err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load application pipeline stages", err)
	}
	pipelineIdCopy, sourceID, sourceName, sourceDescription := pipeline.Id, template.Id, template.Name, template.Description
	version := requiredStageVersion(template)
	artifacts, err := templateStageArtifactsJSON(template)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	sortOrder := input.SortOrder
	stages = append(stages, model.PipelineStage{Id: idutil.NewId(), ProjectId: pipelineProjectId(pipeline), Kind: model.PipelineStageKindApplication, PipelineId: &pipelineIdCopy, Name: name, Image: template.Image, Script: template.Script, Description: description, SourceTemplateStageId: &sourceID, SourceTemplateStageName: &sourceName, SourceTemplateStageVersion: &version, SourceTemplateStageDescription: &sourceDescription, Artifacts: &artifacts, DependsOn: &dependsOn, SortOrder: &sortOrder})
	if err := s.validatePipelineConfiguration(ctx, pipeline, stages); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	pipeline.Version++
	if err := s.store.UpdateApplicationPipelineWithStages(ctx, pipeline, stages); err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to import pipeline stage", err)
	}
	return s.pipelineDetail(ctx, pipeline)
}

func (s Service) UpdatePipelineStageNode(ctx context.Context, userId, pipelineId, stageId string, input pipelinedto.PipelineStageNodeUpdateInput) (pipelinedto.PipelineDetail, error) {
	pipeline, err := s.loadPipelineForUser(ctx, userId, pipelineId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if pipeline.Kind == model.PipelineKindTemplate {
		if input.Image != nil || input.Script != nil {
			return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "template pipeline stage references only accept name, description, dependencies, and sort order")
		}
		references, err := s.store.TemplatePipelineStageReferences(ctx, pipeline.Id)
		if err != nil {
			return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage references", err)
		}
		changed, found, err := applyTemplateReferenceUpdate(references, stageId, input)
		if err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		if !found {
			return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindNotFound, "Pipeline stage "+stageId+" not found")
		}
		if err := validateTemplatePipelineConfiguration(pipeline, references); err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		if changed {
			pipeline.Version++
			if err := s.store.UpdateTemplatePipelineWithReferences(ctx, pipeline, references); err != nil {
				return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline stage", err)
			}
		}
		return s.pipelineDetail(ctx, pipeline)
	}
	stages, err := s.store.ApplicationPipelineStages(ctx, pipeline.Id)
	if err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load application pipeline stages", err)
	}
	changed, found, err := applyApplicationStageUpdate(stages, stageId, input)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if !found {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindNotFound, "Pipeline stage "+stageId+" not found")
	}
	if err := s.validatePipelineConfiguration(ctx, pipeline, stages); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if changed {
		pipeline.Version++
		if err := s.store.UpdateApplicationPipelineWithStages(ctx, pipeline, stages); err != nil {
			return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline stage", err)
		}
	}
	return s.pipelineDetail(ctx, pipeline)
}

func (s Service) DeletePipelineStageNode(ctx context.Context, userId, pipelineId, stageId string) (pipelinedto.PipelineDetail, error) {
	pipeline, err := s.loadPipelineForUser(ctx, userId, pipelineId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if pipeline.Kind == model.PipelineKindTemplate {
		references, err := s.store.TemplatePipelineStageReferences(ctx, pipeline.Id)
		if err != nil {
			return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage references", err)
		}
		stageName, dependentStageName, err := templateStageDeletionDependency(references, stageId)
		if err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		if stageName == "" {
			return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindNotFound, "Pipeline stage "+stageId+" not found")
		}
		if dependentStageName != "" {
			return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "Cannot delete stage "+stageName+" because stage "+dependentStageName+" depends on it")
		}
		remaining, found := removeTemplateReference(references, stageId)
		if !found {
			return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindNotFound, "Pipeline stage "+stageId+" not found")
		}
		if err := validateTemplatePipelineConfiguration(pipeline, remaining); err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		pipeline.Version++
		if err := s.store.UpdateTemplatePipelineWithReferences(ctx, pipeline, remaining); err != nil {
			return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to delete pipeline stage", err)
		}
		return s.pipelineDetail(ctx, pipeline)
	}
	stages, err := s.store.ApplicationPipelineStages(ctx, pipeline.Id)
	if err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load application pipeline stages", err)
	}
	stageName, dependentStageName, err := applicationStageDeletionDependency(stages, stageId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if stageName == "" {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindNotFound, "Pipeline stage "+stageId+" not found")
	}
	if dependentStageName != "" {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "Cannot delete stage "+stageName+" because stage "+dependentStageName+" depends on it")
	}
	variableName, err := pipelineVariableScopedToStage(pipeline.VariableDeclarations, stageId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if variableName != "" {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindValidation, "Cannot delete stage "+stageName+" because pipeline variable "+variableName+" is scoped to it")
	}
	remaining, found := removeApplicationStage(stages, stageId)
	if !found {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindNotFound, "Pipeline stage "+stageId+" not found")
	}
	mappings, err := componentMappingsForStages(remaining)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if len(mappings) == 0 {
		pipeline.VersionForkStrategy, pipeline.FixedVersionId, pipeline.FixedVersionLabel = nil, nil, nil
	}
	if err := s.validatePipelineConfiguration(ctx, pipeline, remaining); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	pipeline.Version++
	if err := s.store.UpdateApplicationPipelineWithStages(ctx, pipeline, remaining); err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to delete pipeline stage", err)
	}
	return s.pipelineDetail(ctx, pipeline)
}

func pipelineVariableScopedToStage(value, stageId string) (string, error) {
	variables, err := pipelinevariable.PipelineVariables(value)
	if err != nil {
		return "", err
	}
	variables, err = pipelinevariable.NormalizePipelineVariables(variables)
	if err != nil {
		return "", err
	}
	for _, variable := range variables {
		variableStageId, _ := variable["stage_id"].(string)
		if strings.TrimSpace(variableStageId) != stageId {
			continue
		}
		name, _ := variable["name"].(string)
		return strings.TrimSpace(name), nil
	}
	return "", nil
}

func templateStageDeletionDependency(references []model.PipelineStageReference, stageId string) (string, string, error) {
	targetId := strings.TrimSpace(stageId)
	targetName := ""
	for _, reference := range references {
		if reference.Id == targetId {
			targetName = reference.Name
			break
		}
	}
	if targetName == "" {
		return "", "", nil
	}
	for _, reference := range references {
		if reference.Id == targetId {
			continue
		}
		dependsOn, err := dependsOnFromJSON(reference.DependsOn)
		if err != nil {
			return "", "", err
		}
		for _, dependency := range dependsOn {
			if dependency == targetId {
				return targetName, reference.Name, nil
			}
		}
	}
	return targetName, "", nil
}

func applicationStageDeletionDependency(stages []model.PipelineStage, stageId string) (string, string, error) {
	targetId := strings.TrimSpace(stageId)
	targetName := ""
	for _, stage := range stages {
		if stage.Id == targetId {
			targetName = stage.Name
			break
		}
	}
	if targetName == "" {
		return "", "", nil
	}
	for _, stage := range stages {
		if stage.Id == targetId {
			continue
		}
		dependsOn, err := stageDependsOn(stage)
		if err != nil {
			return "", "", err
		}
		for _, dependency := range dependsOn {
			if dependency == targetId {
				return targetName, stage.Name, nil
			}
		}
	}
	return targetName, "", nil
}

func (s Service) PipelineStageTemplateUpdatePreview(ctx context.Context, userId, pipelineId, stageId string) (pipelinedto.PipelineStageTemplateUpdatePreview, error) {
	pipeline, err := s.loadPipelineForUser(ctx, userId, pipelineId)
	if err != nil {
		return pipelinedto.PipelineStageTemplateUpdatePreview{}, err
	}
	node, err := s.pipelineStageNode(ctx, pipeline, stageId)
	if err != nil {
		return pipelinedto.PipelineStageTemplateUpdatePreview{}, err
	}
	template, err := s.store.PipelineStageTemplate(ctx, node.Node.SourceTemplateStageId)
	if errors.Is(err, repository.ErrNotFound) {
		return pipelinedto.PipelineStageTemplateUpdatePreview{Available: false, Node: node}, nil
	}
	if err != nil {
		return pipelinedto.PipelineStageTemplateUpdatePreview{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage template", err)
	}
	if template.ProjectId != pipelineProjectId(pipeline) || template.Version == nil || *template.Version <= node.Node.SourceTemplateStageVersion {
		return pipelinedto.PipelineStageTemplateUpdatePreview{Available: false, Node: node, CurrentTemplate: template}, nil
	}
	differences := templateDifferences(node.Node, template)
	return pipelinedto.PipelineStageTemplateUpdatePreview{Available: true, Node: node, CurrentTemplate: template, ExpectedSourceTemplateVersion: node.Node.SourceTemplateStageVersion, TargetTemplateVersion: *template.Version, Differences: differences}, nil
}

func (s Service) ApplyPipelineStageTemplateUpdate(ctx context.Context, userId, pipelineId, stageId string, input pipelinedto.PipelineStageTemplateApplyUpdateInput) (pipelinedto.PipelineDetail, error) {
	pipeline, err := s.loadPipelineForUser(ctx, userId, pipelineId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	node, err := s.pipelineStageNode(ctx, pipeline, stageId)
	if err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	if node.Node.SourceTemplateStageVersion != input.ExpectedSourceTemplateStageVersion {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindConflict, "Pipeline stage source version changed; reload the template update preview")
	}
	template, err := s.store.PipelineStageTemplate(ctx, node.Node.SourceTemplateStageId)
	if errors.Is(err, repository.ErrNotFound) {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindNotFound, "Pipeline stage template not found")
	}
	if err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage template", err)
	}
	if template.ProjectId != pipelineProjectId(pipeline) || template.Version == nil || *template.Version != input.TargetTemplateStageVersion || *template.Version <= node.Node.SourceTemplateStageVersion {
		return pipelinedto.PipelineDetail{}, apperror.New(apperror.KindConflict, "Pipeline stage template version changed; reload the template update preview")
	}
	if pipeline.Kind == model.PipelineKindTemplate {
		references, err := s.store.TemplatePipelineStageReferences(ctx, pipeline.Id)
		if err != nil {
			return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage references", err)
		}
		for index := range references {
			if references[index].Id == stageId {
				references[index].Image, references[index].Script = template.Image, template.Script
				artifacts, err := templateStageArtifactsJSON(template)
				if err != nil {
					return pipelinedto.PipelineDetail{}, err
				}
				references[index].Artifacts = artifacts
				references[index].SourceTemplateStageName = template.Name
				references[index].SourceTemplateStageDescription = template.Description
				references[index].SourceTemplateStageVersion = *template.Version
			}
		}
		if err := validateTemplatePipelineConfiguration(pipeline, references); err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		pipeline.Version++
		if err := s.store.UpdateTemplatePipelineWithReferences(ctx, pipeline, references); err != nil {
			return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline stage template", err)
		}
		return s.pipelineDetail(ctx, pipeline)
	}
	stages, err := s.store.ApplicationPipelineStages(ctx, pipeline.Id)
	if err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load application pipeline stages", err)
	}
	for index := range stages {
		if stages[index].Id != stageId {
			continue
		}
		stages[index].Image, stages[index].Script = template.Image, template.Script
		artifacts, err := mergeTemplateStageArtifacts(template, stages[index].Artifacts)
		if err != nil {
			return pipelinedto.PipelineDetail{}, err
		}
		stages[index].Artifacts = artifacts
		stages[index].SourceTemplateStageName = stageTemplateStringPointer(template.Name)
		stages[index].SourceTemplateStageDescription = stageTemplateStringPointer(template.Description)
		version := *template.Version
		stages[index].SourceTemplateStageVersion = &version
	}
	if err := s.validatePipelineConfiguration(ctx, pipeline, stages); err != nil {
		return pipelinedto.PipelineDetail{}, err
	}
	pipeline.Version++
	if err := s.store.UpdateApplicationPipelineWithStages(ctx, pipeline, stages); err != nil {
		return pipelinedto.PipelineDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline stage template", err)
	}
	return s.pipelineDetail(ctx, pipeline)
}

func (s Service) pipelineStageTemplateForUser(ctx context.Context, userId, stageId string) (model.PipelineStage, error) {
	stageId = strings.TrimSpace(stageId)
	stage, err := s.store.PipelineStageTemplate(ctx, stageId)
	if errors.Is(err, repository.ErrNotFound) {
		return model.PipelineStage{}, apperror.New(apperror.KindNotFound, "Pipeline stage "+stageId+" not found")
	}
	if err != nil {
		return model.PipelineStage{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage", err)
	}
	if err := s.ensureProjectMembership(ctx, stage.ProjectId, userId); err != nil {
		return model.PipelineStage{}, err
	}
	if err := validatePipelineStageTemplate(stage); err != nil {
		return model.PipelineStage{}, err
	}
	return stage, nil
}

func (s Service) ensurePipelineStageTemplateNameAvailable(ctx context.Context, projectId, name, currentID string) error {
	existing, err := s.store.PipelineStageTemplateByName(ctx, projectId, name)
	if err == nil && existing.Id != currentID {
		return apperror.New(apperror.KindConflict, "Pipeline stage '"+name+"' already exists")
	}
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check pipeline stage name", err)
	}
	return nil
}

func pipelineStageTemplateFromInput(projectId, name, image, script, description string, inputArtifacts []pipelinedto.ArtifactConfig) (model.PipelineStage, error) {
	name, image, script = strings.TrimSpace(name), strings.TrimSpace(image), strings.TrimSpace(script)
	if name == "" || image == "" {
		return model.PipelineStage{}, apperror.New(apperror.KindValidation, "name and image are required")
	}
	version := 1
	artifacts, err := marshalTemplateStageArtifacts(inputArtifacts)
	if err != nil {
		return model.PipelineStage{}, err
	}
	stage := model.PipelineStage{Id: idutil.NewId(), ProjectId: projectId, Kind: model.PipelineStageKindTemplate, Name: name, Image: image, Script: script, Description: description, Artifacts: artifacts, Version: &version}
	if err := validatePipelineStageTemplate(stage); err != nil {
		return model.PipelineStage{}, err
	}
	return stage, nil
}

func pipelineStageTemplateDetail(stage model.PipelineStage) (pipelinedto.PipelineStageTemplateDetail, error) {
	artifacts, err := templateStageArtifacts(stage)
	if err != nil {
		return pipelinedto.PipelineStageTemplateDetail{}, err
	}
	items := make([]pipelinedto.ArtifactConfig, 0, len(artifacts))
	for _, artifact := range artifacts {
		items = append(items, pipelinedto.ArtifactConfig{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format})
	}
	return pipelinedto.PipelineStageTemplateDetail{Stage: stage, Artifacts: items}, nil
}

func marshalTemplateStageArtifacts(input []pipelinedto.ArtifactConfig) (*string, error) {
	for _, artifact := range input {
		if artifact.ComponentName != nil {
			return nil, apperror.New(apperror.KindValidation, "pipeline stage template artifacts cannot contain component_name")
		}
	}
	return marshalPipelineStageArtifacts(input)
}

func templateStageArtifacts(stage model.PipelineStage) ([]model.ArtifactConfig, error) {
	if stage.Artifacts == nil {
		return []model.ArtifactConfig{}, nil
	}
	artifacts, err := pipelineStageArtifacts(model.PipelineStage{Artifacts: stage.Artifacts})
	if err != nil {
		return nil, err
	}
	for _, artifact := range artifacts {
		if artifact.ComponentName != nil {
			return nil, apperror.New(apperror.KindValidation, "pipeline stage template artifacts cannot contain component_name")
		}
	}
	return artifacts, nil
}

func templateStageArtifactsJSON(stage model.PipelineStage) (string, error) {
	if _, err := templateStageArtifacts(stage); err != nil {
		return "", err
	}
	if stage.Artifacts == nil {
		return "[]", nil
	}
	return *stage.Artifacts, nil
}

func mergeTemplateStageArtifacts(template model.PipelineStage, existing *string) (*string, error) {
	artifacts, err := templateStageArtifacts(template)
	if err != nil {
		return nil, err
	}
	existingArtifacts, err := pipelineStageArtifacts(model.PipelineStage{Artifacts: existing})
	if err != nil {
		return nil, err
	}
	mappings := make(map[string]*string, len(existingArtifacts))
	for _, artifact := range existingArtifacts {
		if artifact.Collector == "docker_image" && artifact.ComponentName != nil {
			mappings[artifact.Name] = artifact.ComponentName
		}
	}
	items := make([]pipelinedto.ArtifactConfig, 0, len(artifacts))
	for _, artifact := range artifacts {
		componentName := mappings[artifact.Name]
		items = append(items, pipelinedto.ArtifactConfig{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format, ComponentName: componentName})
	}
	data, err := marshalPipelineStageArtifacts(items)
	if err != nil {
		return nil, err
	}
	if data == nil {
		empty := "[]"
		return &empty, nil
	}
	return data, nil
}

func importNodeFields(template model.PipelineStage, input pipelinedto.PipelineStageImportInput) (string, string, string, error) {
	name, description := template.Name, template.Description
	if input.Name != nil {
		name = strings.TrimSpace(*input.Name)
	}
	if input.Description != nil {
		description = *input.Description
	}
	if name == "" || input.SortOrder < 0 {
		return "", "", "", apperror.New(apperror.KindValidation, "invalid pipeline stage node fields")
	}
	dependsOn, err := marshalDependsOn(input.DependsOn)
	if err != nil {
		return "", "", "", err
	}
	return name, description, dependsOn, nil
}

func requiredStageVersion(stage model.PipelineStage) int {
	if stage.Version == nil {
		return 0
	}
	return *stage.Version
}

func stageTemplateStringPointer(value string) *string { return &value }

func (s Service) validateStoredPipelineConfiguration(ctx context.Context, pipeline model.Pipeline) error {
	if pipeline.Kind == model.PipelineKindTemplate {
		references, err := s.store.TemplatePipelineStageReferences(ctx, pipeline.Id)
		if err != nil {
			return apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage references", err)
		}
		return validateTemplatePipelineConfiguration(pipeline, references)
	}
	stages, err := s.store.ApplicationPipelineStages(ctx, pipeline.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load application pipeline stages", err)
	}
	return s.validatePipelineConfiguration(ctx, pipeline, stages)
}

func validateTemplatePipelineConfiguration(pipeline model.Pipeline, references []model.PipelineStageReference) error {
	if pipeline.Kind != model.PipelineKindTemplate || pipeline.ApplicationId != nil || pipeline.RepositoryId != nil || pipeline.SourcePipelineId != nil || pipeline.VersionForkStrategy != nil || pipeline.FixedVersionId != nil {
		return apperror.New(apperror.KindValidation, "template pipeline cannot have application bindings")
	}
	definitions := make([]model.StageDefinition, 0, len(references))
	for _, reference := range references {
		if err := validatePipelineStageReference(pipeline, reference); err != nil {
			return err
		}
		artifacts := reference.Artifacts
		artifactDefinitions, err := templateStageArtifacts(model.PipelineStage{Artifacts: &artifacts})
		if err != nil {
			return err
		}
		dependsOn, err := dependsOnFromJSON(reference.DependsOn)
		if err != nil {
			return err
		}
		definitions = append(definitions, model.StageDefinition{Id: reference.Id, Name: reference.Name, Image: reference.Image, Script: reference.Script, Description: reference.Description, Artifacts: artifactDefinitions, DependsOn: dependsOn, SortOrder: reference.SortOrder})
	}
	if err := validatePipelineDAG(definitions); err != nil {
		return apperror.New(apperror.KindValidation, err.Error())
	}
	return nil
}

func (s Service) pipelineStageNodes(ctx context.Context, pipeline model.Pipeline) ([]pipelinedto.PipelineStageNodeDetail, []model.PipelineStage, error) {
	if pipeline.Kind == model.PipelineKindTemplate {
		references, err := s.store.TemplatePipelineStageReferences(ctx, pipeline.Id)
		if err != nil {
			return nil, nil, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage references", err)
		}
		nodes := make([]pipelinedto.PipelineStageNodeDetail, 0, len(references))
		stageViews := make([]model.PipelineStage, 0, len(references))
		for _, reference := range references {
			node, err := s.referenceNodeDetail(ctx, reference)
			if err != nil {
				return nil, nil, err
			}
			nodes = append(nodes, node)
			artifacts := reference.Artifacts
			stageViews = append(stageViews, model.PipelineStage{Id: reference.Id, Name: reference.Name, Image: reference.Image, Script: reference.Script, Artifacts: &artifacts})
		}
		return nodes, stageViews, nil
	}
	stages, err := s.store.ApplicationPipelineStages(ctx, pipeline.Id)
	if err != nil {
		return nil, nil, apperror.Wrap(apperror.KindInternal, "Failed to load application pipeline stages", err)
	}
	nodes := make([]pipelinedto.PipelineStageNodeDetail, 0, len(stages))
	for _, stage := range stages {
		node, err := s.applicationNodeDetail(ctx, stage)
		if err != nil {
			return nil, nil, err
		}
		nodes = append(nodes, node)
	}
	return nodes, stages, nil
}

func (s Service) pipelineStageNode(ctx context.Context, pipeline model.Pipeline, stageId string) (pipelinedto.PipelineStageNodeDetail, error) {
	nodes, _, err := s.pipelineStageNodes(ctx, pipeline)
	if err != nil {
		return pipelinedto.PipelineStageNodeDetail{}, err
	}
	for _, node := range nodes {
		if node.Node.Id == strings.TrimSpace(stageId) {
			return node, nil
		}
	}
	return pipelinedto.PipelineStageNodeDetail{}, apperror.New(apperror.KindNotFound, "Pipeline stage "+stageId+" not found")
}

func (s Service) referenceNodeDetail(ctx context.Context, reference model.PipelineStageReference) (pipelinedto.PipelineStageNodeDetail, error) {
	dependsOn, err := dependsOnFromJSON(reference.DependsOn)
	if err != nil {
		return pipelinedto.PipelineStageNodeDetail{}, err
	}
	artifactData := reference.Artifacts
	artifacts, err := pipelineStageArtifacts(model.PipelineStage{Artifacts: &artifactData})
	if err != nil {
		return pipelinedto.PipelineStageNodeDetail{}, err
	}
	artifactViews := make([]pipelinedto.ArtifactConfig, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactViews = append(artifactViews, pipelinedto.ArtifactConfig{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format})
	}
	node := model.PipelineStageNode{Id: reference.Id, NodeType: model.PipelineStageNodeTypeTemplateReference, PipelineId: reference.PipelineId, Name: reference.Name, Image: reference.Image, Script: reference.Script, Description: reference.Description, DependsOn: reference.DependsOn, SortOrder: reference.SortOrder, SourceTemplateStageId: reference.SourceTemplateStageId, SourceTemplateStageName: reference.SourceTemplateStageName, SourceTemplateStageVersion: reference.SourceTemplateStageVersion, SourceTemplateStageDescription: reference.SourceTemplateStageDescription, Artifacts: &artifactData, CreatedAt: reference.CreatedAt, UpdatedAt: reference.UpdatedAt}
	return s.withLatestTemplateStageVersion(ctx, pipelinedto.PipelineStageNodeDetail{Node: node, Artifacts: artifactViews, DependsOn: dependsOn})
}

func (s Service) applicationNodeDetail(ctx context.Context, stage model.PipelineStage) (pipelinedto.PipelineStageNodeDetail, error) {
	if stage.PipelineId == nil || stage.SourceTemplateStageId == nil || stage.SourceTemplateStageName == nil || stage.SourceTemplateStageVersion == nil || stage.SourceTemplateStageDescription == nil || stage.DependsOn == nil || stage.SortOrder == nil {
		return pipelinedto.PipelineStageNodeDetail{}, apperror.New(apperror.KindValidation, "application pipeline stage source snapshot is required")
	}
	dependsOn, err := dependsOnFromJSON(*stage.DependsOn)
	if err != nil {
		return pipelinedto.PipelineStageNodeDetail{}, err
	}
	artifacts, err := pipelineStageArtifacts(stage)
	if err != nil {
		return pipelinedto.PipelineStageNodeDetail{}, err
	}
	artifactViews := make([]pipelinedto.ArtifactConfig, 0, len(artifacts))
	for _, artifact := range artifacts {
		artifactViews = append(artifactViews, pipelinedto.ArtifactConfig{Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format, ComponentName: artifact.ComponentName})
	}
	node := model.PipelineStageNode{Id: stage.Id, NodeType: model.PipelineStageNodeTypeApplication, PipelineId: *stage.PipelineId, Name: stage.Name, Image: stage.Image, Script: stage.Script, Description: stage.Description, DependsOn: *stage.DependsOn, SortOrder: *stage.SortOrder, SourceTemplateStageId: *stage.SourceTemplateStageId, SourceTemplateStageName: *stage.SourceTemplateStageName, SourceTemplateStageVersion: *stage.SourceTemplateStageVersion, SourceTemplateStageDescription: *stage.SourceTemplateStageDescription, Artifacts: stage.Artifacts, CreatedAt: stage.CreatedAt, UpdatedAt: stage.UpdatedAt}
	return s.withLatestTemplateStageVersion(ctx, pipelinedto.PipelineStageNodeDetail{Node: node, Artifacts: artifactViews, DependsOn: dependsOn})
}

func (s Service) withLatestTemplateStageVersion(ctx context.Context, node pipelinedto.PipelineStageNodeDetail) (pipelinedto.PipelineStageNodeDetail, error) {
	template, err := s.store.PipelineStageTemplate(ctx, node.Node.SourceTemplateStageId)
	if errors.Is(err, repository.ErrNotFound) {
		return node, nil
	}
	if err != nil {
		return pipelinedto.PipelineStageNodeDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage template", err)
	}
	if template.Version != nil && *template.Version > node.Node.SourceTemplateStageVersion {
		version := *template.Version
		node.LatestTemplateStageVersion = &version
	}
	return node, nil
}

func templateDifferences(node model.PipelineStageNode, template model.PipelineStage) []pipelinedto.PipelineStageTemplateFieldDifference {
	candidates := []pipelinedto.PipelineStageTemplateFieldDifference{
		{Field: "name", Current: node.SourceTemplateStageName, Target: template.Name},
		{Field: "image", Current: node.Image, Target: template.Image},
		{Field: "script", Current: node.Script, Target: template.Script},
		{Field: "description", Current: node.SourceTemplateStageDescription, Target: template.Description},
		{Field: "artifacts", Current: artifactJSON(node.Artifacts), Target: artifactJSON(template.Artifacts)},
	}
	differences := make([]pipelinedto.PipelineStageTemplateFieldDifference, 0, len(candidates))
	for _, candidate := range candidates {
		if candidate.Current != candidate.Target {
			differences = append(differences, candidate)
		}
	}
	return differences
}

func artifactJSON(value *string) string {
	if value == nil {
		return "[]"
	}
	return *value
}

func dependsOnFromJSON(value string) ([]string, error) {
	if err := validateJSONArray(value, "depends_on"); err != nil {
		return nil, err
	}
	var result []string
	if err := json.Unmarshal([]byte(value), &result); err != nil {
		return nil, apperror.New(apperror.KindValidation, "Invalid pipeline stage dependencies")
	}
	return result, nil
}

func validatePipelineStageReference(pipeline model.Pipeline, reference model.PipelineStageReference) error {
	if strings.TrimSpace(reference.Id) == "" || reference.PipelineId != pipeline.Id || strings.TrimSpace(reference.SourceTemplateStageId) == "" || strings.TrimSpace(reference.SourceTemplateStageName) == "" || reference.SourceTemplateStageVersion <= 0 || strings.TrimSpace(reference.Name) == "" || strings.TrimSpace(reference.Image) == "" || strings.TrimSpace(reference.Artifacts) == "" || reference.SortOrder < 0 {
		return apperror.New(apperror.KindValidation, "invalid template pipeline stage reference")
	}
	if err := validateJSONArray(reference.DependsOn, "depends_on"); err != nil {
		return err
	}
	artifacts := reference.Artifacts
	_, err := templateStageArtifacts(model.PipelineStage{Artifacts: &artifacts})
	return err
}

func applyTemplateReferenceUpdate(references []model.PipelineStageReference, stageId string, input pipelinedto.PipelineStageNodeUpdateInput) (bool, bool, error) {
	for index := range references {
		if references[index].Id != strings.TrimSpace(stageId) {
			continue
		}
		changed := false
		if input.Name != nil {
			name := strings.TrimSpace(*input.Name)
			if name == "" {
				return false, true, apperror.New(apperror.KindValidation, "name is required")
			}
			if references[index].Name != name {
				references[index].Name, changed = name, true
			}
		}
		if input.Description != nil && references[index].Description != *input.Description {
			references[index].Description, changed = *input.Description, true
		}
		if input.DependsOn != nil {
			dependsOn, err := marshalDependsOn(*input.DependsOn)
			if err != nil {
				return false, true, err
			}
			if references[index].DependsOn != dependsOn {
				references[index].DependsOn, changed = dependsOn, true
			}
		}
		if input.SortOrder != nil {
			if *input.SortOrder < 0 {
				return false, true, apperror.New(apperror.KindValidation, "sort_order must not be negative")
			}
			if references[index].SortOrder != *input.SortOrder {
				references[index].SortOrder, changed = *input.SortOrder, true
			}
		}
		return changed, true, nil
	}
	return false, false, nil
}

func applyApplicationStageUpdate(stages []model.PipelineStage, stageId string, input pipelinedto.PipelineStageNodeUpdateInput) (bool, bool, error) {
	for index := range stages {
		if stages[index].Id != strings.TrimSpace(stageId) {
			continue
		}
		changed := false
		if input.Name != nil {
			name := strings.TrimSpace(*input.Name)
			if name == "" {
				return false, true, apperror.New(apperror.KindValidation, "name is required")
			}
			if stages[index].Name != name {
				stages[index].Name, changed = name, true
			}
		}
		if input.Image != nil {
			image := strings.TrimSpace(*input.Image)
			if image == "" {
				return false, true, apperror.New(apperror.KindValidation, "image is required")
			}
			if stages[index].Image != image {
				stages[index].Image, changed = image, true
			}
		}
		if input.Script != nil {
			script := strings.TrimSpace(*input.Script)
			if stages[index].Script != script {
				stages[index].Script, changed = script, true
			}
		}
		if input.Description != nil && stages[index].Description != *input.Description {
			stages[index].Description, changed = *input.Description, true
		}
		if input.DependsOn != nil {
			dependsOn, err := marshalDependsOn(*input.DependsOn)
			if err != nil {
				return false, true, err
			}
			if stages[index].DependsOn == nil || *stages[index].DependsOn != dependsOn {
				stages[index].DependsOn, changed = &dependsOn, true
			}
		}
		if input.SortOrder != nil {
			if *input.SortOrder < 0 {
				return false, true, apperror.New(apperror.KindValidation, "sort_order must not be negative")
			}
			if stages[index].SortOrder == nil || *stages[index].SortOrder != *input.SortOrder {
				value := *input.SortOrder
				stages[index].SortOrder, changed = &value, true
			}
		}
		return changed, true, nil
	}
	return false, false, nil
}

func removeTemplateReference(references []model.PipelineStageReference, stageId string) ([]model.PipelineStageReference, bool) {
	remaining := make([]model.PipelineStageReference, 0, len(references))
	found := false
	for _, reference := range references {
		if reference.Id == strings.TrimSpace(stageId) {
			found = true
			continue
		}
		remaining = append(remaining, reference)
	}
	return remaining, found
}

func removeApplicationStage(stages []model.PipelineStage, stageId string) ([]model.PipelineStage, bool) {
	remaining := make([]model.PipelineStage, 0, len(stages))
	found := false
	for _, stage := range stages {
		if stage.Id == strings.TrimSpace(stageId) {
			found = true
			continue
		}
		remaining = append(remaining, stage)
	}
	return remaining, found
}

func clonePipelineStageReferences(references []model.PipelineStageReference, pipelineId, projectId string) ([]model.PipelineStage, error) {
	idMap := make(map[string]string, len(references))
	for _, reference := range references {
		if reference.Id == "" || reference.SourceTemplateStageId == "" || reference.SourceTemplateStageVersion <= 0 {
			return nil, apperror.New(apperror.KindValidation, "template pipeline stage reference source snapshot is required")
		}
		idMap[reference.Id] = idutil.NewId()
	}
	result := make([]model.PipelineStage, 0, len(references))
	for _, reference := range references {
		dependsOn, err := dependsOnFromJSON(reference.DependsOn)
		if err != nil {
			return nil, err
		}
		for index, dependency := range dependsOn {
			mapped, exists := idMap[dependency]
			if !exists {
				return nil, apperror.New(apperror.KindValidation, "Template stage dependency does not exist")
			}
			dependsOn[index] = mapped
		}
		dependsOnData, err := marshalDependsOn(dependsOn)
		if err != nil {
			return nil, err
		}
		id, sourceID, sourceName, sourceDescription := idMap[reference.Id], reference.SourceTemplateStageId, reference.SourceTemplateStageName, reference.SourceTemplateStageDescription
		version, sortOrder := reference.SourceTemplateStageVersion, reference.SortOrder
		artifacts := reference.Artifacts
		if strings.TrimSpace(artifacts) == "" {
			return nil, apperror.New(apperror.KindValidation, "template pipeline stage reference artifacts are required")
		}
		pipelineIDCopy := pipelineId
		result = append(result, model.PipelineStage{Id: id, ProjectId: projectId, Kind: model.PipelineStageKindApplication, PipelineId: &pipelineIDCopy, Name: reference.Name, Image: reference.Image, Script: reference.Script, Description: reference.Description, SourceTemplateStageId: &sourceID, SourceTemplateStageName: &sourceName, SourceTemplateStageVersion: &version, SourceTemplateStageDescription: &sourceDescription, Artifacts: &artifacts, DependsOn: &dependsOnData, SortOrder: &sortOrder})
	}
	return result, nil
}

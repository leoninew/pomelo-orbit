package pipelinesvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	pipelinedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/dto"
	pipelinevariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/rule/pipelinevariable"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var pipelineTemplateCopyPattern = regexp.MustCompile(` copy( [0-9]+)?$`)

func (s Service) ListPipelineTemplates(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[pipelinedto.PipelineTemplateDetail], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[pipelinedto.PipelineTemplateDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[pipelinedto.PipelineTemplateDetail]{}, err
	}
	items, err := s.store.ListPipelineTemplates(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[pipelinedto.PipelineTemplateDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list pipeline templates", err)
	}
	responses := make([]pipelinedto.PipelineTemplateDetail, 0, len(items.Items))
	for _, item := range items.Items {
		detail, err := s.pipelineTemplateDetail(ctx, item)
		if err != nil {
			return repository.Page[pipelinedto.PipelineTemplateDetail]{}, err
		}
		responses = append(responses, detail)
	}
	return repository.Page[pipelinedto.PipelineTemplateDetail]{Items: responses, Total: items.Total, Page: items.Page, PerPage: items.PerPage}, nil
}

func (s Service) CreatePipelineTemplate(ctx context.Context, userId string, input pipelinedto.PipelineTemplateCreateInput) (pipelinedto.PipelineTemplateDetail, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return pipelinedto.PipelineTemplateDetail{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return pipelinedto.PipelineTemplateDetail{}, apperror.New(apperror.KindValidation, "Invalid pipeline template fields")
	}
	if err := s.ensurePipelineTemplateNameAvailable(ctx, projectId, name, ""); err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	variables, err := marshalPipelineTemplateVariables(pipelinevariable.SanitizePipelineTemplateVariables(input.VariableDeclarations))
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	template := model.PipelineTemplate{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Description: input.Description, VariableDeclarations: variables, Version: 1}
	if err := s.store.CreatePipelineTemplate(ctx, template); err != nil {
		return pipelinedto.PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline template", err)
	}
	created, err := s.store.PipelineTemplate(ctx, template.Id)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	return s.pipelineTemplateDetail(ctx, created)
}

func (s Service) PipelineTemplateForUser(ctx context.Context, userId string, templateId string) (pipelinedto.PipelineTemplateDetail, error) {
	template, err := s.loadPipelineTemplateForUser(ctx, userId, templateId)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	return s.pipelineTemplateDetail(ctx, template)
}

func (s Service) UpdatePipelineTemplate(ctx context.Context, userId string, templateId string, input pipelinedto.PipelineTemplateUpdateInput) (pipelinedto.PipelineTemplateDetail, error) {
	template, err := s.loadPipelineTemplateForUser(ctx, userId, templateId)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	stages, orchestrationProvided, err := s.applyPipelineTemplateUpdateInput(ctx, &template, input)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	if orchestrationProvided {
		if err := s.store.UpdatePipelineTemplateWithStages(ctx, template, stages); err != nil {
			return pipelinedto.PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline template", err)
		}
	} else if err := s.store.UpdatePipelineTemplate(ctx, template); err != nil {
		return pipelinedto.PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline template", err)
	}
	updated, err := s.store.PipelineTemplate(ctx, template.Id)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	return s.pipelineTemplateDetail(ctx, updated)
}

func (s Service) DeletePipelineTemplate(ctx context.Context, userId string, templateId string) error {
	template, err := s.loadPipelineTemplateForUser(ctx, userId, templateId)
	if err != nil {
		return err
	}
	referenced, err := s.store.PipelineTemplateReferencedByWebhooks(ctx, template.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check pipeline template references", err)
	}
	if referenced {
		return apperror.New(apperror.KindConflict, "Template is referenced by webhooks, cannot delete")
	}
	if err := s.store.DeletePipelineTemplate(ctx, template.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete pipeline template", err)
	}
	return nil
}

func (s Service) DuplicatePipelineTemplate(ctx context.Context, userId string, templateId string) (pipelinedto.PipelineTemplateDetail, error) {
	template, err := s.loadPipelineTemplateForUser(ctx, userId, templateId)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	projectId := pipelineTemplateProjectId(template)
	name, err := s.nextPipelineTemplateCopyName(ctx, projectId, template.Name)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	existingStages, err := s.store.PipelineTemplateStages(ctx, template.Id)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template stages", err)
	}
	duplicated := model.PipelineTemplate{Id: idutil.NewId(), ProjectId: template.ProjectId, Name: name, Description: template.Description, VariableDeclarations: template.VariableDeclarations, Version: 1}
	stages := make([]model.PipelineTemplateStage, 0, len(existingStages))
	for _, stage := range existingStages {
		stage.Id = idutil.NewId()
		stage.TemplateId = duplicated.Id
		stages = append(stages, stage)
	}
	if err := s.store.DuplicatePipelineTemplate(ctx, duplicated, stages); err != nil {
		return pipelinedto.PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to duplicate pipeline template", err)
	}
	created, err := s.store.PipelineTemplate(ctx, duplicated.Id)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	return s.pipelineTemplateDetail(ctx, created)
}

func (s Service) ResolvePipelineTemplateVariables(ctx context.Context, userId string, input pipelinedto.PipelineTemplateResolveInput) ([]map[string]any, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return nil, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return nil, err
	}
	stages, err := s.loadPipelineStagesForOrchestration(ctx, projectId, input.Orchestration)
	if err != nil {
		return nil, err
	}
	return pipelinevariable.ResolveTemplateVariables(stages, pipelinevariable.SanitizePipelineTemplateVariables(input.VariableDeclarations)), nil
}

func (s Service) loadPipelineTemplateForUser(ctx context.Context, userId string, templateId string) (model.PipelineTemplate, error) {
	templateId = strings.TrimSpace(templateId)
	template, err := s.store.PipelineTemplate(ctx, templateId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.PipelineTemplate{}, apperror.New(apperror.KindNotFound, "Pipeline template "+templateId+" not found")
		}
		return model.PipelineTemplate{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	if err := s.ensureProjectMembership(ctx, pipelineTemplateProjectId(template), userId); err != nil {
		return model.PipelineTemplate{}, err
	}
	return template, nil
}

func (s Service) pipelineTemplateDetail(ctx context.Context, template model.PipelineTemplate) (pipelinedto.PipelineTemplateDetail, error) {
	orchestration, err := s.pipelineTemplateOrchestration(ctx, template.Id)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	stages, err := s.pipelineTemplateStagesResponse(ctx, pipelineTemplateProjectId(template), orchestration)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	variables, err := pipelinevariable.PipelineTemplateVariables(template.VariableDeclarations)
	if err != nil {
		return pipelinedto.PipelineTemplateDetail{}, err
	}
	return pipelinedto.PipelineTemplateDetail{Template: template, Orchestration: orchestration, Stages: stages, VariableDeclarations: resolveTemplateVariablesFromResponses(stages, variables)}, nil
}

func (s Service) pipelineTemplateOrchestration(ctx context.Context, templateId string) ([]pipelinedto.StageOrchestration, error) {
	rows, err := s.store.PipelineTemplateStages(ctx, templateId)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template stages", err)
	}
	orchestration := make([]pipelinedto.StageOrchestration, 0, len(rows))
	for _, row := range rows {
		dependsOn, err := pipelineTemplateDependsOn(row.DependsOn)
		if err != nil {
			return nil, err
		}
		orchestration = append(orchestration, pipelinedto.StageOrchestration{StageId: row.StageId, StageName: row.StageName, StageVersion: row.StageVersion, DependsOn: dependsOn, SortOrder: row.SortOrder})
	}
	return orchestration, nil
}

func (s Service) pipelineTemplateStagesResponse(ctx context.Context, projectId string, orchestration []pipelinedto.StageOrchestration) ([]pipelinedto.PipelineStageDetail, error) {
	stageIds := make([]string, 0, len(orchestration))
	for _, item := range orchestration {
		stageIds = append(stageIds, item.StageId)
	}
	stages, err := s.store.PipelineStagesByIds(ctx, projectId, stageIds)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stages", err)
	}
	stageMap := make(map[string]model.PipelineStage, len(stages))
	for _, stage := range stages {
		stageMap[stage.Id] = stage
	}
	responses := make([]pipelinedto.PipelineStageDetail, 0, len(orchestration))
	for _, item := range orchestration {
		stage, exists := stageMap[item.StageId]
		if !exists {
			return nil, apperror.New(apperror.KindNotFound, "Stage "+item.StageId+" not found")
		}
		responses = append(responses, pipelineStageDetail(stage))
	}
	return responses, nil
}

func (s Service) ensurePipelineTemplateNameAvailable(ctx context.Context, projectId string, name string, currentTemplateId string) error {
	existing, err := s.store.PipelineTemplateByName(ctx, projectId, name)
	if err == nil {
		if existing.Id != currentTemplateId {
			return apperror.New(apperror.KindConflict, "Template '"+name+"' already exists")
		}
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check pipeline template name", err)
	}
	return nil
}

func (s Service) nextPipelineTemplateCopyName(ctx context.Context, projectId string, name string) (string, error) {
	baseName := pipelineTemplateCopyPattern.ReplaceAllString(name, "")
	for i := 1; ; i++ {
		candidate := baseName + " copy"
		if i > 1 {
			candidate = fmt.Sprintf("%s copy %d", baseName, i)
		}
		_, err := s.store.PipelineTemplateByName(ctx, projectId, candidate)
		if errors.Is(err, repository.ErrNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", apperror.Wrap(apperror.KindInternal, "Failed to check pipeline template name", err)
		}
	}
}

func (s Service) applyPipelineTemplateUpdateInput(ctx context.Context, template *model.PipelineTemplate, req pipelinedto.PipelineTemplateUpdateInput) ([]model.PipelineTemplateStage, bool, error) {
	projectId := pipelineTemplateProjectId(*template)
	versionChanged := false
	orchestrationProvided := false
	stages := []model.PipelineTemplateStage(nil)
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" {
			return nil, false, apperror.New(apperror.KindValidation, "Invalid pipeline template fields")
		}
		if err := s.ensurePipelineTemplateNameAvailable(ctx, projectId, value, template.Id); err != nil {
			return nil, false, err
		}
		if value != template.Name {
			template.Name = value
			versionChanged = true
		}
	}
	if req.Description != nil {
		if *req.Description != template.Description {
			template.Description = *req.Description
			versionChanged = true
		}
	}
	if req.VariableDeclarations != nil {
		value, err := marshalPipelineTemplateVariables(pipelinevariable.SanitizePipelineTemplateVariables(*req.VariableDeclarations))
		if err != nil {
			return nil, false, err
		}
		if value != template.VariableDeclarations {
			template.VariableDeclarations = value
			versionChanged = true
		}
	}
	if req.Orchestration != nil {
		orchestrationProvided = true
		orchestration := *req.Orchestration
		loaded, err := s.loadPipelineStagesForOrchestration(ctx, projectId, orchestration)
		if err != nil {
			return nil, false, err
		}
		stages, err = pipelineTemplateStageRows(template.Id, orchestration, loaded)
		if err != nil {
			return nil, false, err
		}
		versionChanged = true
	}
	if versionChanged {
		template.Version++
	}
	return stages, orchestrationProvided, nil
}

func (s Service) loadPipelineStagesForOrchestration(ctx context.Context, projectId string, orchestration []pipelinedto.StageOrchestration) ([]model.PipelineStage, error) {
	stageIds := make([]string, 0, len(orchestration))
	seen := map[string]struct{}{}
	for _, item := range orchestration {
		stageId := strings.TrimSpace(item.StageId)
		if stageId == "" {
			return nil, apperror.New(apperror.KindValidation, "Invalid pipeline template fields")
		}
		if _, exists := seen[stageId]; !exists {
			stageIds = append(stageIds, stageId)
			seen[stageId] = struct{}{}
		}
	}
	stages, err := s.store.PipelineStagesByIds(ctx, projectId, stageIds)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stages", err)
	}
	if len(stages) != len(stageIds) {
		found := map[string]struct{}{}
		for _, stage := range stages {
			found[stage.Id] = struct{}{}
		}
		missing := make([]string, 0)
		for _, stageId := range stageIds {
			if _, exists := found[stageId]; !exists {
				missing = append(missing, stageId)
			}
		}
		return nil, apperror.New(apperror.KindNotFound, "Stage(s) not found: "+strings.Join(missing, ", "))
	}
	return stages, nil
}

func pipelineTemplateStageRows(templateId string, orchestration []pipelinedto.StageOrchestration, stages []model.PipelineStage) ([]model.PipelineTemplateStage, error) {
	stageMap := make(map[string]model.PipelineStage, len(stages))
	for _, stage := range stages {
		stageMap[stage.Id] = stage
	}
	rows := make([]model.PipelineTemplateStage, 0, len(orchestration))
	definitions := make([]model.StageDefinition, 0, len(orchestration))
	for _, item := range orchestration {
		stageId := strings.TrimSpace(item.StageId)
		dependsOn, err := marshalPipelineTemplateDependsOn(item.DependsOn)
		if err != nil {
			return nil, err
		}
		stage := stageMap[stageId]
		artifacts, err := pipelineStageArtifactConfigs(stage)
		if err != nil {
			return nil, err
		}
		definitions = append(definitions, model.StageDefinition{
			Name: item.StageName, Id: stage.Id, Image: stage.Image, Version: stage.Version,
			DependsOn: item.DependsOn, Script: stage.Script, Artifacts: artifacts,
			BuildVersionBinding: cloneBuildVersionBinding(stage.BuildVersionBinding),
		})
		rows = append(rows, model.PipelineTemplateStage{Id: idutil.NewId(), TemplateId: templateId, StageId: stage.Id, StageName: stage.Name, StageVersion: stage.Version, DependsOn: dependsOn, SortOrder: item.SortOrder})
	}
	if err := validateSourceCommitDependencies(definitions); err != nil {
		return nil, apperror.New(apperror.KindValidation, err.Error())
	}
	return rows, nil
}

func pipelineTemplateProjectId(template model.PipelineTemplate) string {
	if template.ProjectId == nil {
		return ""
	}
	return *template.ProjectId
}

func pipelineStageDetail(stage model.PipelineStage) pipelinedto.PipelineStageDetail {
	artifacts := []pipelinedto.ArtifactConfig(nil)
	if stage.Artifacts != nil && strings.TrimSpace(*stage.Artifacts) != "" {
		_ = json.Unmarshal([]byte(*stage.Artifacts), &artifacts)
	}
	return pipelinedto.PipelineStageDetail{Id: stage.Id, Name: stage.Name, Image: stage.Image, Script: stage.Script, Artifacts: artifacts, BuildVersionBinding: buildVersionBindingDetail(stage.BuildVersionBinding), Description: stage.Description, Version: stage.Version, CreatedAt: stage.CreatedAt.UTC().Format(timeFormatRFC3339), UpdatedAt: stage.UpdatedAt.UTC().Format(timeFormatRFC3339)}
}

func buildVersionBindingDetail(value *model.BuildVersionBinding) *pipelinedto.BuildVersionBinding {
	if value == nil {
		return nil
	}
	result := &pipelinedto.BuildVersionBinding{ApplicationId: value.ApplicationId, ApplicationName: value.ApplicationName, ComponentName: value.ComponentName, ForkStrategy: value.ForkStrategy}
	if value.FixedVersionId != nil {
		fixedVersionId := *value.FixedVersionId
		result.FixedVersionId = &fixedVersionId
	}
	return result
}

const timeFormatRFC3339 = "2006-01-02T15:04:05Z07:00"

func marshalPipelineTemplateVariables(variables []map[string]any) (string, error) {
	if variables == nil {
		variables = []map[string]any{}
	}
	data, err := json.Marshal(variables)
	if err != nil {
		return "", apperror.Wrap(apperror.KindValidation, "Invalid variable_declarations", err)
	}
	return string(data), nil
}

func pipelineTemplateDependsOn(value string) ([]string, error) {
	if strings.TrimSpace(value) == "" {
		return []string{}, nil
	}
	var dependsOn []string
	if err := json.Unmarshal([]byte(value), &dependsOn); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Invalid pipeline template orchestration", err)
	}
	return dependsOn, nil
}

func marshalPipelineTemplateDependsOn(dependsOn []string) (string, error) {
	if dependsOn == nil {
		dependsOn = []string{}
	}
	data, err := json.Marshal(dependsOn)
	if err != nil {
		return "", apperror.Wrap(apperror.KindValidation, "Invalid pipeline template fields", err)
	}
	return string(data), nil
}

func resolveTemplateVariablesFromResponses(stages []pipelinedto.PipelineStageDetail, custom []map[string]any) []map[string]any {
	converted := make([]model.PipelineStage, 0, len(stages))
	for _, stage := range stages {
		artifacts, _ := json.Marshal(stage.Artifacts)
		artifactString := string(artifacts)
		converted = append(converted, model.PipelineStage{Name: stage.Name, Script: stage.Script, Artifacts: &artifactString})
	}
	return pipelinevariable.ResolveTemplateVariables(converted, custom)
}

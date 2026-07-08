package cisvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/common/civariable"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var pipelineTemplateCopyPattern = regexp.MustCompile(` copy( [0-9]+)?$`)

type ArtifactConfig struct {
	Type string `json:"type"`
	Path string `json:"path"`
	Name string `json:"name"`
}

type BuildStageDetail struct {
	Id          string
	Name        string
	Image       string
	Script      string
	Artifacts   []ArtifactConfig
	Description string
	Version     int
	CreatedAt   string
	UpdatedAt   string
}

type StageOrchestration struct {
	StageId      string   `json:"stage_id"`
	StageName    string   `json:"stage_name"`
	StageVersion int      `json:"stage_version"`
	DependsOn    []string `json:"depends_on"`
	SortOrder    int      `json:"sort_order"`
}

type PipelineTemplateCreateInput struct {
	ProjectId            string
	Name                 string
	Description          string
	VariableDeclarations []map[string]any
}

type PipelineTemplateUpdateInput struct {
	Name                 *string
	Description          *string
	Orchestration        *[]StageOrchestration
	VariableDeclarations *[]map[string]any
}

type PipelineTemplateResolveInput struct {
	ProjectId            string
	Orchestration        []StageOrchestration
	VariableDeclarations []map[string]any
}

type PipelineTemplateDetail struct {
	Template             model.PipelineTemplate
	Orchestration        []StageOrchestration
	Stages               []BuildStageDetail
	VariableDeclarations []map[string]any
}

func (s Service) ListPipelineTemplates(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[PipelineTemplateDetail], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[PipelineTemplateDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[PipelineTemplateDetail]{}, err
	}
	items, err := s.store.ListPipelineTemplates(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[PipelineTemplateDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list pipeline templates", err)
	}
	responses := make([]PipelineTemplateDetail, 0, len(items.Items))
	for _, item := range items.Items {
		detail, err := s.pipelineTemplateDetail(ctx, item)
		if err != nil {
			return repository.Page[PipelineTemplateDetail]{}, err
		}
		responses = append(responses, detail)
	}
	return repository.Page[PipelineTemplateDetail]{Items: responses, Total: items.Total, Page: items.Page, PerPage: items.PerPage}, nil
}

func (s Service) CreatePipelineTemplate(ctx context.Context, userId string, input PipelineTemplateCreateInput) (PipelineTemplateDetail, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return PipelineTemplateDetail{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return PipelineTemplateDetail{}, err
	}
	name := strings.TrimSpace(input.Name)
	if name == "" {
		return PipelineTemplateDetail{}, apperror.New(apperror.KindValidation, "Invalid pipeline template fields")
	}
	if err := s.ensurePipelineTemplateNameAvailable(ctx, projectId, name, ""); err != nil {
		return PipelineTemplateDetail{}, err
	}
	variables, err := marshalPipelineTemplateVariables(civariable.SanitizePipelineTemplateVariables(input.VariableDeclarations))
	if err != nil {
		return PipelineTemplateDetail{}, err
	}
	template := model.PipelineTemplate{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Description: input.Description, VariableDeclarations: variables, Version: 1}
	if err := s.store.CreatePipelineTemplate(ctx, template); err != nil {
		return PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline template", err)
	}
	created, err := s.store.PipelineTemplate(ctx, template.Id)
	if err != nil {
		return PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	return s.pipelineTemplateDetail(ctx, created)
}

func (s Service) PipelineTemplateForUser(ctx context.Context, userId string, templateId string) (PipelineTemplateDetail, error) {
	template, err := s.loadPipelineTemplateForUser(ctx, userId, templateId)
	if err != nil {
		return PipelineTemplateDetail{}, err
	}
	return s.pipelineTemplateDetail(ctx, template)
}

func (s Service) UpdatePipelineTemplate(ctx context.Context, userId string, templateId string, input PipelineTemplateUpdateInput) (PipelineTemplateDetail, error) {
	template, err := s.loadPipelineTemplateForUser(ctx, userId, templateId)
	if err != nil {
		return PipelineTemplateDetail{}, err
	}
	stages, orchestrationProvided, err := s.applyPipelineTemplateUpdateInput(ctx, &template, input)
	if err != nil {
		return PipelineTemplateDetail{}, err
	}
	if orchestrationProvided {
		if err := s.store.UpdatePipelineTemplateWithStages(ctx, template, stages); err != nil {
			return PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline template", err)
		}
	} else if err := s.store.UpdatePipelineTemplate(ctx, template); err != nil {
		return PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline template", err)
	}
	updated, err := s.store.PipelineTemplate(ctx, template.Id)
	if err != nil {
		return PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
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

func (s Service) DuplicatePipelineTemplate(ctx context.Context, userId string, templateId string) (PipelineTemplateDetail, error) {
	template, err := s.loadPipelineTemplateForUser(ctx, userId, templateId)
	if err != nil {
		return PipelineTemplateDetail{}, err
	}
	projectId := pipelineTemplateProjectId(template)
	name, err := s.nextPipelineTemplateCopyName(ctx, projectId, template.Name)
	if err != nil {
		return PipelineTemplateDetail{}, err
	}
	existingStages, err := s.store.PipelineTemplateStages(ctx, template.Id)
	if err != nil {
		return PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template stages", err)
	}
	duplicated := model.PipelineTemplate{Id: idutil.NewId(), ProjectId: template.ProjectId, Name: name, Description: template.Description, VariableDeclarations: template.VariableDeclarations, Version: 1}
	stages := make([]model.PipelineTemplateStage, 0, len(existingStages))
	for _, stage := range existingStages {
		stage.Id = idutil.NewId()
		stage.TemplateId = duplicated.Id
		stages = append(stages, stage)
	}
	if err := s.store.DuplicatePipelineTemplate(ctx, duplicated, stages); err != nil {
		return PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to duplicate pipeline template", err)
	}
	created, err := s.store.PipelineTemplate(ctx, duplicated.Id)
	if err != nil {
		return PipelineTemplateDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	return s.pipelineTemplateDetail(ctx, created)
}

func (s Service) ResolvePipelineTemplateVariables(ctx context.Context, userId string, input PipelineTemplateResolveInput) ([]map[string]any, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return nil, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return nil, err
	}
	stages, err := s.loadBuildStagesForOrchestration(ctx, projectId, input.Orchestration)
	if err != nil {
		return nil, err
	}
	return civariable.ResolveTemplateVariables(stages, civariable.SanitizePipelineTemplateVariables(input.VariableDeclarations)), nil
}

func (s Service) loadPipelineTemplateForUser(ctx context.Context, userId string, templateId string) (model.PipelineTemplate, error) {
	templateId = strings.TrimSpace(templateId)
	template, err := s.store.PipelineTemplate(ctx, templateId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.PipelineTemplate{}, apperror.New(apperror.KindNotFound, "Pipeline template "+templateId+" not found")
		}
		return model.PipelineTemplate{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	if err := s.ensureProjectMembership(ctx, pipelineTemplateProjectId(template), userId); err != nil {
		return model.PipelineTemplate{}, err
	}
	return template, nil
}

func (s Service) pipelineTemplateDetail(ctx context.Context, template model.PipelineTemplate) (PipelineTemplateDetail, error) {
	orchestration, err := s.pipelineTemplateOrchestration(ctx, template.Id)
	if err != nil {
		return PipelineTemplateDetail{}, err
	}
	stages, err := s.pipelineTemplateStagesResponse(ctx, pipelineTemplateProjectId(template), orchestration)
	if err != nil {
		return PipelineTemplateDetail{}, err
	}
	variables, err := civariable.PipelineTemplateVariables(template.VariableDeclarations)
	if err != nil {
		return PipelineTemplateDetail{}, err
	}
	return PipelineTemplateDetail{Template: template, Orchestration: orchestration, Stages: stages, VariableDeclarations: resolveTemplateVariablesFromResponses(stages, variables)}, nil
}

func (s Service) pipelineTemplateOrchestration(ctx context.Context, templateId string) ([]StageOrchestration, error) {
	rows, err := s.store.PipelineTemplateStages(ctx, templateId)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template stages", err)
	}
	orchestration := make([]StageOrchestration, 0, len(rows))
	for _, row := range rows {
		dependsOn, err := pipelineTemplateDependsOn(row.DependsOn)
		if err != nil {
			return nil, err
		}
		orchestration = append(orchestration, StageOrchestration{StageId: row.StageId, StageName: row.StageName, StageVersion: row.StageVersion, DependsOn: dependsOn, SortOrder: row.SortOrder})
	}
	return orchestration, nil
}

func (s Service) pipelineTemplateStagesResponse(ctx context.Context, projectId string, orchestration []StageOrchestration) ([]BuildStageDetail, error) {
	stageIds := make([]string, 0, len(orchestration))
	for _, item := range orchestration {
		stageIds = append(stageIds, item.StageId)
	}
	stages, err := s.store.BuildStagesByIds(ctx, projectId, stageIds)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load build stages", err)
	}
	stageMap := make(map[string]model.BuildStage, len(stages))
	for _, stage := range stages {
		stageMap[stage.Id] = stage
	}
	responses := make([]BuildStageDetail, 0, len(orchestration))
	for _, item := range orchestration {
		stage, exists := stageMap[item.StageId]
		if !exists {
			return nil, apperror.New(apperror.KindNotFound, "Stage "+item.StageId+" not found")
		}
		responses = append(responses, buildStageDetail(stage))
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
	if !errors.Is(err, sql.ErrNoRows) {
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
		if errors.Is(err, sql.ErrNoRows) {
			return candidate, nil
		}
		if err != nil {
			return "", apperror.Wrap(apperror.KindInternal, "Failed to check pipeline template name", err)
		}
	}
}

func (s Service) applyPipelineTemplateUpdateInput(ctx context.Context, template *model.PipelineTemplate, req PipelineTemplateUpdateInput) ([]model.PipelineTemplateStage, bool, error) {
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
		value, err := marshalPipelineTemplateVariables(civariable.SanitizePipelineTemplateVariables(*req.VariableDeclarations))
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
		loaded, err := s.loadBuildStagesForOrchestration(ctx, projectId, orchestration)
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

func (s Service) loadBuildStagesForOrchestration(ctx context.Context, projectId string, orchestration []StageOrchestration) ([]model.BuildStage, error) {
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
	stages, err := s.store.BuildStagesByIds(ctx, projectId, stageIds)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load build stages", err)
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

func pipelineTemplateStageRows(templateId string, orchestration []StageOrchestration, stages []model.BuildStage) ([]model.PipelineTemplateStage, error) {
	stageMap := make(map[string]model.BuildStage, len(stages))
	for _, stage := range stages {
		stageMap[stage.Id] = stage
	}
	rows := make([]model.PipelineTemplateStage, 0, len(orchestration))
	for _, item := range orchestration {
		stageId := strings.TrimSpace(item.StageId)
		dependsOn, err := marshalPipelineTemplateDependsOn(item.DependsOn)
		if err != nil {
			return nil, err
		}
		stage := stageMap[stageId]
		rows = append(rows, model.PipelineTemplateStage{Id: idutil.NewId(), TemplateId: templateId, StageId: stage.Id, StageName: stage.Name, StageVersion: stage.Version, DependsOn: dependsOn, SortOrder: item.SortOrder})
	}
	return rows, nil
}

func pipelineTemplateProjectId(template model.PipelineTemplate) string {
	if template.ProjectId == nil {
		return ""
	}
	return *template.ProjectId
}

func buildStageDetail(stage model.BuildStage) BuildStageDetail {
	artifacts := []ArtifactConfig(nil)
	if stage.Artifacts != nil && strings.TrimSpace(*stage.Artifacts) != "" {
		_ = json.Unmarshal([]byte(*stage.Artifacts), &artifacts)
	}
	return BuildStageDetail{Id: stage.Id, Name: stage.Name, Image: stage.Image, Script: stage.Script, Artifacts: artifacts, Description: stage.Description, Version: stage.Version, CreatedAt: stage.CreatedAt.UTC().Format(timeFormatRFC3339), UpdatedAt: stage.UpdatedAt.UTC().Format(timeFormatRFC3339)}
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

func resolveTemplateVariablesFromResponses(stages []BuildStageDetail, custom []map[string]any) []map[string]any {
	converted := make([]model.BuildStage, 0, len(stages))
	for _, stage := range stages {
		artifacts, _ := json.Marshal(stage.Artifacts)
		artifactString := string(artifacts)
		converted = append(converted, model.BuildStage{Name: stage.Name, Script: stage.Script, Artifacts: &artifactString})
	}
	return civariable.ResolveTemplateVariables(converted, custom)
}

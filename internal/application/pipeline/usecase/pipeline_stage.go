package pipelinesvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	pipelinedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/dto"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var pipelineStageCopyPattern = regexp.MustCompile(` copy( [0-9]+)?$`)

func (s Service) ListPipelineStages(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[pipelinedto.PipelineStageDetail], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[pipelinedto.PipelineStageDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[pipelinedto.PipelineStageDetail]{}, err
	}
	items, err := s.store.ListPipelineStages(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[pipelinedto.PipelineStageDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list pipeline stages", err)
	}
	responses := make([]pipelinedto.PipelineStageDetail, 0, len(items.Items))
	for _, item := range items.Items {
		responses = append(responses, pipelineStageDetail(item))
	}
	return repository.Page[pipelinedto.PipelineStageDetail]{Items: responses, Total: items.Total, Page: items.Page, PerPage: items.PerPage}, nil
}

func (s Service) CreatePipelineStage(ctx context.Context, userId string, input pipelinedto.PipelineStageCreateInput) (pipelinedto.PipelineStageDetail, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return pipelinedto.PipelineStageDetail{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	name, image, script, err := normalizePipelineStageFields(input.Name, input.Image, input.Script)
	if err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	if err := validatePipelineStageArtifacts(input.Artifacts); err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	binding, err := s.buildVersionBinding(ctx, projectId, artifactConfigsFromDTO(input.Artifacts), input.BuildVersionBinding)
	if err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	if err := s.ensurePipelineStageNameAvailable(ctx, projectId, name, ""); err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	artifacts, err := marshalPipelineStageArtifacts(input.Artifacts)
	if err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	stage := model.PipelineStage{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Image: image, Script: script, Artifacts: artifacts, BuildVersionBinding: binding, Description: input.Description, Version: 1}
	if err := s.store.CreatePipelineStage(ctx, stage); err != nil {
		return pipelinedto.PipelineStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline stage", err)
	}
	created, err := s.store.PipelineStage(ctx, stage.Id)
	if err != nil {
		return pipelinedto.PipelineStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage", err)
	}
	return pipelineStageDetail(created), nil
}

func (s Service) PipelineStageForUser(ctx context.Context, userId string, stageId string) (pipelinedto.PipelineStageDetail, error) {
	stage, err := s.loadPipelineStageForUser(ctx, userId, stageId)
	if err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	return pipelineStageDetail(stage), nil
}

func (s Service) UpdatePipelineStage(ctx context.Context, userId string, stageId string, input pipelinedto.PipelineStageUpdateInput) (pipelinedto.PipelineStageDetail, error) {
	stage, err := s.loadPipelineStageForUser(ctx, userId, stageId)
	if err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	if err := s.applyPipelineStageUpdateInput(ctx, &stage, input); err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	if err := s.store.UpdatePipelineStage(ctx, stage); err != nil {
		return pipelinedto.PipelineStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update pipeline stage", err)
	}
	updated, err := s.store.PipelineStage(ctx, stage.Id)
	if err != nil {
		return pipelinedto.PipelineStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage", err)
	}
	return pipelineStageDetail(updated), nil
}

func (s Service) DeletePipelineStage(ctx context.Context, userId string, stageId string) error {
	stage, err := s.loadPipelineStageForUser(ctx, userId, stageId)
	if err != nil {
		return err
	}
	projectId := pipelineStageProjectId(stage)
	referenced, err := s.store.PipelineStageReferencedByTemplates(ctx, projectId, stage.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check pipeline stage references", err)
	}
	if referenced {
		return apperror.New(apperror.KindConflict, "Stage is referenced by templates, cannot delete")
	}
	if err := s.store.DeletePipelineStage(ctx, stage.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete pipeline stage", err)
	}
	return nil
}

func (s Service) DuplicatePipelineStage(ctx context.Context, userId string, stageId string) (pipelinedto.PipelineStageDetail, error) {
	stage, err := s.loadPipelineStageForUser(ctx, userId, stageId)
	if err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	projectId := pipelineStageProjectId(stage)
	name, err := s.nextPipelineStageCopyName(ctx, projectId, stage.Name)
	if err != nil {
		return pipelinedto.PipelineStageDetail{}, err
	}
	duplicated := model.PipelineStage{Id: idutil.NewId(), ProjectId: stage.ProjectId, Name: name, Image: stage.Image, Script: stage.Script, Artifacts: stage.Artifacts, BuildVersionBinding: cloneBuildVersionBinding(stage.BuildVersionBinding), Description: stage.Description, Version: 1}
	if err := s.store.CreatePipelineStage(ctx, duplicated); err != nil {
		return pipelinedto.PipelineStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to duplicate pipeline stage", err)
	}
	created, err := s.store.PipelineStage(ctx, duplicated.Id)
	if err != nil {
		return pipelinedto.PipelineStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage", err)
	}
	return pipelineStageDetail(created), nil
}

func (s Service) loadPipelineStageForUser(ctx context.Context, userId string, stageId string) (model.PipelineStage, error) {
	stageId = strings.TrimSpace(stageId)
	stage, err := s.store.PipelineStage(ctx, stageId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.PipelineStage{}, apperror.New(apperror.KindNotFound, "Stage "+stageId+" not found")
		}
		return model.PipelineStage{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline stage", err)
	}
	if err := s.ensureProjectMembership(ctx, pipelineStageProjectId(stage), userId); err != nil {
		return model.PipelineStage{}, err
	}
	return stage, nil
}

func (s Service) ensurePipelineStageNameAvailable(ctx context.Context, projectId string, name string, currentStageId string) error {
	existing, err := s.store.PipelineStageByName(ctx, projectId, name)
	if err == nil {
		if existing.Id != currentStageId {
			return apperror.New(apperror.KindConflict, "Stage '"+name+"' already exists")
		}
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check pipeline stage name", err)
	}
	return nil
}

func (s Service) nextPipelineStageCopyName(ctx context.Context, projectId string, name string) (string, error) {
	baseName := pipelineStageCopyPattern.ReplaceAllString(name, "")
	for i := 1; ; i++ {
		candidate := baseName + " copy"
		if i > 1 {
			candidate = fmt.Sprintf("%s copy %d", baseName, i)
		}
		_, err := s.store.PipelineStageByName(ctx, projectId, candidate)
		if errors.Is(err, repository.ErrNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", apperror.Wrap(apperror.KindInternal, "Failed to check pipeline stage name", err)
		}
	}
}

func (s Service) applyPipelineStageUpdateInput(ctx context.Context, stage *model.PipelineStage, req pipelinedto.PipelineStageUpdateInput) error {
	projectId := pipelineStageProjectId(*stage)
	versionChanged := false
	if req.ClearBuildVersionBinding && req.BuildVersionBinding != nil {
		return apperror.New(apperror.KindValidation, "Cannot set and clear build version binding together")
	}
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" {
			return apperror.New(apperror.KindValidation, "Invalid pipeline stage fields")
		}
		if err := s.ensurePipelineStageNameAvailable(ctx, projectId, value, stage.Id); err != nil {
			return err
		}
		stage.Name = value
	}
	if req.Image != nil {
		value := strings.TrimSpace(*req.Image)
		if value == "" {
			return apperror.New(apperror.KindValidation, "Invalid pipeline stage fields")
		}
		if value != stage.Image {
			stage.Image = value
			versionChanged = true
		}
	}
	if req.Script != nil {
		value := strings.TrimSpace(*req.Script)
		if value == "" {
			return apperror.New(apperror.KindValidation, "Invalid pipeline stage fields")
		}
		if value != stage.Script {
			stage.Script = value
			versionChanged = true
		}
	}
	if req.Artifacts != nil {
		artifactJSON, err := marshalPipelineStageArtifacts(*req.Artifacts)
		if err != nil {
			return err
		}
		if optionalStringValue(stage.Artifacts) != optionalStringValue(artifactJSON) {
			stage.Artifacts = artifactJSON
			versionChanged = true
		}
	}
	if req.ClearBuildVersionBinding {
		if stage.BuildVersionBinding != nil {
			stage.BuildVersionBinding = nil
			versionChanged = true
		}
	} else if req.BuildVersionBinding != nil {
		artifacts, err := pipelineStageArtifactConfigs(*stage)
		if err != nil {
			return err
		}
		binding, err := s.buildVersionBinding(ctx, projectId, artifacts, req.BuildVersionBinding)
		if err != nil {
			return err
		}
		if !buildVersionBindingEqual(stage.BuildVersionBinding, binding) {
			stage.BuildVersionBinding = binding
			versionChanged = true
		}
	}
	if stage.BuildVersionBinding != nil {
		artifacts, err := pipelineStageArtifactConfigs(*stage)
		if err != nil {
			return err
		}
		if dockerImageArtifactCount(artifacts) != 1 {
			return apperror.New(apperror.KindValidation, "Build version binding requires exactly one docker_image artifact")
		}
	}
	if req.Description != nil {
		stage.Description = *req.Description
	}
	if versionChanged {
		stage.Version++
	}
	return nil
}

func normalizePipelineStageFields(name string, image string, script string) (string, string, string, error) {
	name = strings.TrimSpace(name)
	image = strings.TrimSpace(image)
	script = strings.TrimSpace(script)
	if name == "" || image == "" || script == "" {
		return "", "", "", apperror.New(apperror.KindValidation, "Invalid pipeline stage fields")
	}
	return name, image, script, nil
}

func marshalPipelineStageArtifacts(artifacts []pipelinedto.ArtifactConfig) (*string, error) {
	if err := validatePipelineStageArtifacts(artifacts); err != nil {
		return nil, err
	}
	if len(artifacts) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(artifacts)
	if err != nil {
		return nil, apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
	}
	value := string(data)
	return &value, nil
}

func validatePipelineStageArtifacts(artifacts []pipelinedto.ArtifactConfig) error {
	names := make(map[string]struct{}, len(artifacts))
	for _, artifact := range artifacts {
		artifact.Name = strings.TrimSpace(artifact.Name)
		artifact.Collector = strings.TrimSpace(artifact.Collector)
		artifact.Reference = strings.TrimSpace(artifact.Reference)
		artifact.Command = strings.TrimSpace(artifact.Command)
		artifact.Format = strings.TrimSpace(artifact.Format)
		if artifact.Name == "" || artifact.Collector == "" {
			return apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
		}
		if _, exists := names[artifact.Name]; exists {
			return apperror.New(apperror.KindValidation, "Pipeline stage artifact names must be unique")
		}
		names[artifact.Name] = struct{}{}
		switch artifact.Collector {
		case "file", "docker_image":
			if artifact.Reference == "" || artifact.Command != "" || artifact.Format != "" {
				return apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
			}
		case "command":
			if artifact.Reference != "" || artifact.Command == "" || (artifact.Format != "text" && artifact.Format != "git_object_id") {
				return apperror.New(apperror.KindValidation, "Invalid pipeline stage artifacts")
			}
		default:
			return apperror.New(apperror.KindValidation, "Unknown artifact collector")
		}
	}
	return nil
}

func pipelineStageProjectId(stage model.PipelineStage) string {
	if stage.ProjectId == nil {
		return ""
	}
	return *stage.ProjectId
}

func optionalStringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

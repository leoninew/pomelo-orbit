package cisvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var buildStageCopyPattern = regexp.MustCompile(` copy( [0-9]+)?$`)

func (s Service) ListBuildStages(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[cidto.BuildStageDetail], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[cidto.BuildStageDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[cidto.BuildStageDetail]{}, err
	}
	items, err := s.store.ListBuildStages(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[cidto.BuildStageDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list build stages", err)
	}
	responses := make([]cidto.BuildStageDetail, 0, len(items.Items))
	for _, item := range items.Items {
		responses = append(responses, buildStageDetail(item))
	}
	return repository.Page[cidto.BuildStageDetail]{Items: responses, Total: items.Total, Page: items.Page, PerPage: items.PerPage}, nil
}

func (s Service) CreateBuildStage(ctx context.Context, userId string, input cidto.BuildStageCreateInput) (cidto.BuildStageDetail, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return cidto.BuildStageDetail{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return cidto.BuildStageDetail{}, err
	}
	name, image, script, err := normalizeBuildStageFields(input.Name, input.Image, input.Script)
	if err != nil {
		return cidto.BuildStageDetail{}, err
	}
	if err := validateBuildStageArtifacts(input.Artifacts); err != nil {
		return cidto.BuildStageDetail{}, err
	}
	if err := s.ensureBuildStageNameAvailable(ctx, projectId, name, ""); err != nil {
		return cidto.BuildStageDetail{}, err
	}
	artifacts, err := marshalBuildStageArtifacts(input.Artifacts)
	if err != nil {
		return cidto.BuildStageDetail{}, err
	}
	stage := model.BuildStage{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Image: image, Script: script, Artifacts: artifacts, Description: input.Description, Version: 1}
	if err := s.store.CreateBuildStage(ctx, stage); err != nil {
		return cidto.BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create build stage", err)
	}
	created, err := s.store.BuildStage(ctx, stage.Id)
	if err != nil {
		return cidto.BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load build stage", err)
	}
	return buildStageDetail(created), nil
}

func (s Service) BuildStageForUser(ctx context.Context, userId string, stageId string) (cidto.BuildStageDetail, error) {
	stage, err := s.loadBuildStageForUser(ctx, userId, stageId)
	if err != nil {
		return cidto.BuildStageDetail{}, err
	}
	return buildStageDetail(stage), nil
}

func (s Service) UpdateBuildStage(ctx context.Context, userId string, stageId string, input cidto.BuildStageUpdateInput) (cidto.BuildStageDetail, error) {
	stage, err := s.loadBuildStageForUser(ctx, userId, stageId)
	if err != nil {
		return cidto.BuildStageDetail{}, err
	}
	if err := s.applyBuildStageUpdateInput(ctx, &stage, input); err != nil {
		return cidto.BuildStageDetail{}, err
	}
	if err := s.store.UpdateBuildStage(ctx, stage); err != nil {
		return cidto.BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update build stage", err)
	}
	updated, err := s.store.BuildStage(ctx, stage.Id)
	if err != nil {
		return cidto.BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load build stage", err)
	}
	return buildStageDetail(updated), nil
}

func (s Service) DeleteBuildStage(ctx context.Context, userId string, stageId string) error {
	stage, err := s.loadBuildStageForUser(ctx, userId, stageId)
	if err != nil {
		return err
	}
	projectId := buildStageProjectId(stage)
	referenced, err := s.store.BuildStageReferencedByTemplates(ctx, projectId, stage.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check build stage references", err)
	}
	if referenced {
		return apperror.New(apperror.KindConflict, "Stage is referenced by templates, cannot delete")
	}
	if err := s.store.DeleteBuildStage(ctx, stage.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete build stage", err)
	}
	return nil
}

func (s Service) DuplicateBuildStage(ctx context.Context, userId string, stageId string) (cidto.BuildStageDetail, error) {
	stage, err := s.loadBuildStageForUser(ctx, userId, stageId)
	if err != nil {
		return cidto.BuildStageDetail{}, err
	}
	projectId := buildStageProjectId(stage)
	name, err := s.nextBuildStageCopyName(ctx, projectId, stage.Name)
	if err != nil {
		return cidto.BuildStageDetail{}, err
	}
	duplicated := model.BuildStage{Id: idutil.NewId(), ProjectId: stage.ProjectId, Name: name, Image: stage.Image, Script: stage.Script, Artifacts: stage.Artifacts, Description: stage.Description, Version: 1}
	if err := s.store.CreateBuildStage(ctx, duplicated); err != nil {
		return cidto.BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to duplicate build stage", err)
	}
	created, err := s.store.BuildStage(ctx, duplicated.Id)
	if err != nil {
		return cidto.BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load build stage", err)
	}
	return buildStageDetail(created), nil
}

func (s Service) loadBuildStageForUser(ctx context.Context, userId string, stageId string) (model.BuildStage, error) {
	stageId = strings.TrimSpace(stageId)
	stage, err := s.store.BuildStage(ctx, stageId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.BuildStage{}, apperror.New(apperror.KindNotFound, "Stage "+stageId+" not found")
		}
		return model.BuildStage{}, apperror.Wrap(apperror.KindInternal, "Failed to load build stage", err)
	}
	if err := s.ensureProjectMembership(ctx, buildStageProjectId(stage), userId); err != nil {
		return model.BuildStage{}, err
	}
	return stage, nil
}

func (s Service) ensureBuildStageNameAvailable(ctx context.Context, projectId string, name string, currentStageId string) error {
	existing, err := s.store.BuildStageByName(ctx, projectId, name)
	if err == nil {
		if existing.Id != currentStageId {
			return apperror.New(apperror.KindConflict, "Stage '"+name+"' already exists")
		}
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check build stage name", err)
	}
	return nil
}

func (s Service) nextBuildStageCopyName(ctx context.Context, projectId string, name string) (string, error) {
	baseName := buildStageCopyPattern.ReplaceAllString(name, "")
	for i := 1; ; i++ {
		candidate := baseName + " copy"
		if i > 1 {
			candidate = fmt.Sprintf("%s copy %d", baseName, i)
		}
		_, err := s.store.BuildStageByName(ctx, projectId, candidate)
		if errors.Is(err, repository.ErrNotFound) {
			return candidate, nil
		}
		if err != nil {
			return "", apperror.Wrap(apperror.KindInternal, "Failed to check build stage name", err)
		}
	}
}

func (s Service) applyBuildStageUpdateInput(ctx context.Context, stage *model.BuildStage, req cidto.BuildStageUpdateInput) error {
	projectId := buildStageProjectId(*stage)
	versionChanged := false
	if req.Name != nil {
		value := strings.TrimSpace(*req.Name)
		if value == "" {
			return apperror.New(apperror.KindValidation, "Invalid build stage fields")
		}
		if err := s.ensureBuildStageNameAvailable(ctx, projectId, value, stage.Id); err != nil {
			return err
		}
		stage.Name = value
	}
	if req.Image != nil {
		value := strings.TrimSpace(*req.Image)
		if value == "" {
			return apperror.New(apperror.KindValidation, "Invalid build stage fields")
		}
		if value != stage.Image {
			stage.Image = value
			versionChanged = true
		}
	}
	if req.Script != nil {
		value := strings.TrimSpace(*req.Script)
		if value == "" {
			return apperror.New(apperror.KindValidation, "Invalid build stage fields")
		}
		if value != stage.Script {
			stage.Script = value
			versionChanged = true
		}
	}
	if req.Artifacts != nil {
		artifactJSON, err := marshalBuildStageArtifacts(*req.Artifacts)
		if err != nil {
			return err
		}
		if optionalStringValue(stage.Artifacts) != optionalStringValue(artifactJSON) {
			stage.Artifacts = artifactJSON
			versionChanged = true
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

func normalizeBuildStageFields(name string, image string, script string) (string, string, string, error) {
	name = strings.TrimSpace(name)
	image = strings.TrimSpace(image)
	script = strings.TrimSpace(script)
	if name == "" || image == "" || script == "" {
		return "", "", "", apperror.New(apperror.KindValidation, "Invalid build stage fields")
	}
	return name, image, script, nil
}

func marshalBuildStageArtifacts(artifacts []cidto.ArtifactConfig) (*string, error) {
	if err := validateBuildStageArtifacts(artifacts); err != nil {
		return nil, err
	}
	if len(artifacts) == 0 {
		return nil, nil
	}
	data, err := json.Marshal(artifacts)
	if err != nil {
		return nil, apperror.New(apperror.KindValidation, "Invalid build stage artifacts")
	}
	value := string(data)
	return &value, nil
}

func validateBuildStageArtifacts(artifacts []cidto.ArtifactConfig) error {
	for _, artifact := range artifacts {
		if strings.TrimSpace(artifact.Type) == "" || strings.TrimSpace(artifact.Path) == "" || strings.TrimSpace(artifact.Name) == "" {
			return apperror.New(apperror.KindValidation, "Invalid build stage artifacts")
		}
	}
	return nil
}

func buildStageProjectId(stage model.BuildStage) string {
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

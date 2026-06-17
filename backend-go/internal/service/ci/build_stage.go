package cisvc

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"

	"backend/internal/apperror"
	"backend/internal/repository"
	"backend/internal/repository/model"
)

var buildStageCopyPattern = regexp.MustCompile(` copy( [0-9]+)?$`)

type BuildStageCreateInput struct {
	ProjectId   string
	Name        string
	Image       string
	Script      string
	Artifacts   []ArtifactConfig
	Description string
}

type BuildStageUpdateInput struct {
	Fields map[string]json.RawMessage
}

func (s Service) ListBuildStages(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[BuildStageDetail], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[BuildStageDetail]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[BuildStageDetail]{}, err
	}
	items, err := s.store.ListBuildStages(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[BuildStageDetail]{}, apperror.Wrap(apperror.KindInternal, "Failed to list build stages", err)
	}
	responses := make([]BuildStageDetail, 0, len(items.Items))
	for _, item := range items.Items {
		responses = append(responses, buildStageDetail(item))
	}
	return repository.Page[BuildStageDetail]{Items: responses, Total: items.Total, Page: items.Page, PerPage: items.PerPage}, nil
}

func (s Service) CreateBuildStage(ctx context.Context, userId string, input BuildStageCreateInput) (BuildStageDetail, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return BuildStageDetail{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return BuildStageDetail{}, err
	}
	name, image, script, err := normalizeBuildStageFields(input.Name, input.Image, input.Script)
	if err != nil {
		return BuildStageDetail{}, err
	}
	if err := validateBuildStageArtifacts(input.Artifacts); err != nil {
		return BuildStageDetail{}, err
	}
	if err := s.ensureBuildStageNameAvailable(ctx, projectId, name, ""); err != nil {
		return BuildStageDetail{}, err
	}
	artifacts, err := marshalBuildStageArtifacts(input.Artifacts)
	if err != nil {
		return BuildStageDetail{}, err
	}
	stage := model.BuildStage{Id: repository.NewId(), ProjectId: &projectId, Name: name, Image: image, Script: script, Artifacts: artifacts, Description: input.Description, Version: 1}
	if err := s.store.CreateBuildStage(ctx, stage); err != nil {
		return BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create build stage", err)
	}
	created, err := s.store.BuildStage(ctx, stage.Id)
	if err != nil {
		return BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load build stage", err)
	}
	return buildStageDetail(created), nil
}

func (s Service) BuildStageForUser(ctx context.Context, userId string, stageId string) (BuildStageDetail, error) {
	stage, err := s.loadBuildStageForUser(ctx, userId, stageId)
	if err != nil {
		return BuildStageDetail{}, err
	}
	return buildStageDetail(stage), nil
}

func (s Service) UpdateBuildStage(ctx context.Context, userId string, stageId string, input BuildStageUpdateInput) (BuildStageDetail, error) {
	stage, err := s.loadBuildStageForUser(ctx, userId, stageId)
	if err != nil {
		return BuildStageDetail{}, err
	}
	if err := s.applyBuildStageUpdateInput(ctx, &stage, input.Fields); err != nil {
		return BuildStageDetail{}, err
	}
	if err := s.store.UpdateBuildStage(ctx, stage); err != nil {
		return BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update build stage", err)
	}
	updated, err := s.store.BuildStage(ctx, stage.Id)
	if err != nil {
		return BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load build stage", err)
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

func (s Service) DuplicateBuildStage(ctx context.Context, userId string, stageId string) (BuildStageDetail, error) {
	stage, err := s.loadBuildStageForUser(ctx, userId, stageId)
	if err != nil {
		return BuildStageDetail{}, err
	}
	projectId := buildStageProjectId(stage)
	name, err := s.nextBuildStageCopyName(ctx, projectId, stage.Name)
	if err != nil {
		return BuildStageDetail{}, err
	}
	duplicated := model.BuildStage{Id: repository.NewId(), ProjectId: stage.ProjectId, Name: name, Image: stage.Image, Script: stage.Script, Artifacts: stage.Artifacts, Description: stage.Description, Version: 1}
	if err := s.store.CreateBuildStage(ctx, duplicated); err != nil {
		return BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to duplicate build stage", err)
	}
	created, err := s.store.BuildStage(ctx, duplicated.Id)
	if err != nil {
		return BuildStageDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load build stage", err)
	}
	return buildStageDetail(created), nil
}

func (s Service) loadBuildStageForUser(ctx context.Context, userId string, stageId string) (model.BuildStage, error) {
	stageId = strings.TrimSpace(stageId)
	stage, err := s.store.BuildStage(ctx, stageId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	if !errors.Is(err, sql.ErrNoRows) {
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
		if errors.Is(err, sql.ErrNoRows) {
			return candidate, nil
		}
		if err != nil {
			return "", apperror.Wrap(apperror.KindInternal, "Failed to check build stage name", err)
		}
	}
}

func (s Service) applyBuildStageUpdateInput(ctx context.Context, stage *model.BuildStage, req map[string]json.RawMessage) error {
	projectId := buildStageProjectId(*stage)
	versionChanged := false
	if raw, exists := req["name"]; exists {
		value, err := decodeRequiredString(raw, "Invalid build stage fields")
		if err != nil {
			return err
		}
		if err := s.ensureBuildStageNameAvailable(ctx, projectId, value, stage.Id); err != nil {
			return err
		}
		stage.Name = value
	}
	if raw, exists := req["image"]; exists {
		value, err := decodeRequiredString(raw, "Invalid build stage fields")
		if err != nil {
			return err
		}
		if value != stage.Image {
			stage.Image = value
			versionChanged = true
		}
	}
	if raw, exists := req["script"]; exists {
		value, err := decodeRequiredString(raw, "Invalid build stage fields")
		if err != nil {
			return err
		}
		if value != stage.Script {
			stage.Script = value
			versionChanged = true
		}
	}
	if raw, exists := req["artifacts"]; exists && string(raw) != "null" {
		var artifacts []ArtifactConfig
		if err := json.Unmarshal(raw, &artifacts); err != nil {
			return apperror.New(apperror.KindValidation, "Invalid build stage fields")
		}
		artifactJSON, err := marshalBuildStageArtifacts(artifacts)
		if err != nil {
			return err
		}
		if optionalStringValue(stage.Artifacts) != optionalStringValue(artifactJSON) {
			stage.Artifacts = artifactJSON
			versionChanged = true
		}
	}
	if raw, exists := req["description"]; exists {
		var value *string
		if string(raw) != "null" {
			decoded := ""
			if err := json.Unmarshal(raw, &decoded); err != nil {
				return apperror.New(apperror.KindValidation, "Invalid build stage fields")
			}
			value = &decoded
		}
		if value != nil {
			stage.Description = *value
		}
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

func marshalBuildStageArtifacts(artifacts []ArtifactConfig) (*string, error) {
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

func validateBuildStageArtifacts(artifacts []ArtifactConfig) error {
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

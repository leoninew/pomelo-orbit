package pipelinesvc

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"

	pipelinedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var componentNamePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

func (s Service) buildVersionBinding(ctx context.Context, projectId string, artifacts []model.ArtifactConfig, input *pipelinedto.BuildVersionBinding) (*model.BuildVersionBinding, error) {
	if input == nil {
		return nil, nil
	}
	applicationId := strings.TrimSpace(input.ApplicationId)
	componentName := strings.TrimSpace(input.ComponentName)
	strategy := strings.TrimSpace(input.ForkStrategy)
	if applicationId == "" || !componentNamePattern.MatchString(componentName) {
		return nil, apperror.New(apperror.KindValidation, "Invalid build version binding")
	}
	app, err := s.store.Application(ctx, applicationId)
	if errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.New(apperror.KindNotFound, "Application "+applicationId+" not found")
	}
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	if app.ProjectId == nil || *app.ProjectId != projectId {
		return nil, apperror.New(apperror.KindNotFound, "Application "+applicationId+" not found")
	}
	if dockerImageArtifactCount(artifacts) != 1 {
		return nil, apperror.New(apperror.KindValidation, "Build version binding requires exactly one docker_image artifact")
	}
	binding := &model.BuildVersionBinding{ApplicationId: applicationId, ApplicationName: app.Name, ComponentName: componentName, ForkStrategy: strategy}
	switch strategy {
	case "latest":
		if input.FixedVersionId != nil && strings.TrimSpace(*input.FixedVersionId) != "" {
			return nil, apperror.New(apperror.KindValidation, "latest fork strategy cannot set fixed_version_id")
		}
	case "fixed":
		if input.FixedVersionId == nil || strings.TrimSpace(*input.FixedVersionId) == "" {
			return nil, apperror.New(apperror.KindValidation, "fixed fork strategy requires fixed_version_id")
		}
		fixedVersionId := strings.TrimSpace(*input.FixedVersionId)
		version, err := s.store.Version(ctx, fixedVersionId)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(apperror.KindNotFound, "Version "+fixedVersionId+" not found")
		}
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to load fixed version", err)
		}
		if version.ApplicationId != applicationId {
			return nil, apperror.New(apperror.KindValidation, "fixed_version_id must belong to application_id")
		}
		components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to load fixed version components", err)
		}
		if !versionHasComponent(components, componentName) {
			return nil, apperror.New(apperror.KindValidation, "fixed version does not contain component "+componentName)
		}
		binding.FixedVersionId = &fixedVersionId
	default:
		return nil, apperror.New(apperror.KindValidation, "fork_strategy must be latest or fixed")
	}
	return binding, nil
}

func artifactConfigsFromDTO(artifacts []pipelinedto.ArtifactConfig) []model.ArtifactConfig {
	result := make([]model.ArtifactConfig, 0, len(artifacts))
	for _, artifact := range artifacts {
		result = append(result, model.ArtifactConfig{
			Name: artifact.Name, Collector: artifact.Collector, Reference: artifact.Reference, Command: artifact.Command, Format: artifact.Format,
		})
	}
	return result
}

func dockerImageArtifactCount(artifacts []model.ArtifactConfig) int {
	count := 0
	for _, artifact := range artifacts {
		if artifact.Collector == "docker_image" {
			count++
		}
	}
	return count
}

func validateSourceCommitDependencies(stages []model.StageDefinition) error {
	byId := make(map[string]model.StageDefinition, len(stages))
	for _, stage := range stages {
		byId[stage.Id] = stage
	}
	for _, stage := range stages {
		if !stageProducesDockerImage(stage) {
			continue
		}
		if _, err := sourceCommitArtifactForStage(stage, byId); err != nil {
			return fmt.Errorf("stage %s: %w", stage.Name, err)
		}
	}
	return nil
}

func stageProducesDockerImage(stage model.StageDefinition) bool {
	for _, artifact := range stage.Artifacts {
		if artifact.Collector == "docker_image" {
			return true
		}
	}
	return false
}

func sourceCommitArtifactForStage(stage model.StageDefinition, stages map[string]model.StageDefinition) (model.ArtifactConfig, error) {
	seen := map[string]bool{}
	artifacts := make([]model.ArtifactConfig, 0, 1)
	var visit func(string) error
	visit = func(stageId string) error {
		if seen[stageId] {
			return nil
		}
		seen[stageId] = true
		ancestor, ok := stages[stageId]
		if !ok {
			return fmt.Errorf("dependency %s does not exist", stageId)
		}
		for _, artifact := range ancestor.Artifacts {
			if artifact.Collector == "command" && artifact.Format == "git_object_id" {
				artifacts = append(artifacts, artifact)
			}
		}
		for _, dependency := range ancestor.DependsOn {
			if err := visit(dependency); err != nil {
				return err
			}
		}
		return nil
	}
	for _, dependency := range stage.DependsOn {
		if err := visit(dependency); err != nil {
			return model.ArtifactConfig{}, err
		}
	}
	if len(artifacts) != 1 {
		return model.ArtifactConfig{}, fmt.Errorf("requires exactly one transitive source_commit artifact, found %d", len(artifacts))
	}
	return artifacts[0], nil
}

func versionHasComponent(components []model.VersionComponent, name string) bool {
	for _, component := range components {
		if component.Name == name {
			return true
		}
	}
	return false
}

func cloneBuildVersionBinding(value *model.BuildVersionBinding) *model.BuildVersionBinding {
	if value == nil {
		return nil
	}
	copy := *value
	if value.FixedVersionId != nil {
		fixedVersionId := *value.FixedVersionId
		copy.FixedVersionId = &fixedVersionId
	}
	return &copy
}

func buildVersionBindingEqual(left *model.BuildVersionBinding, right *model.BuildVersionBinding) bool {
	if left == nil || right == nil {
		return left == right
	}
	if left.ApplicationId != right.ApplicationId || left.ApplicationName != right.ApplicationName || left.ComponentName != right.ComponentName || left.ForkStrategy != right.ForkStrategy {
		return false
	}
	if left.FixedVersionId == nil || right.FixedVersionId == nil {
		return left.FixedVersionId == right.FixedVersionId
	}
	return *left.FixedVersionId == *right.FixedVersionId
}

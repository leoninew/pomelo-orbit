package pipelinerunsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) ensureRepositoryHasNoRunningPipelineRun(ctx context.Context, repositoryId string) error {
	running, err := s.store.RepositoryHasRunningPipelineRun(ctx, repositoryId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check running pipeline runs", err)
	}
	if running {
		return apperror.New(apperror.KindValidation, "Repository already has a running pipeline run")
	}
	return nil
}

func (s Service) resolvePipelineRunBuildVersionBindings(ctx context.Context, projectId *string, snapshot model.PipelineSnapshot) ([]model.PipelineRunBuildVersionBinding, error) {
	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(snapshot.StagesSnapshot), &stages); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid pipeline snapshot stages")
	}
	if err := validateSourceCommitDependencies(stages); err != nil {
		return nil, apperror.New(apperror.KindValidation, err.Error())
	}
	bindings := make([]model.PipelineRunBuildVersionBinding, 0)
	for _, stage := range stages {
		binding := stage.BuildVersionBinding
		if binding == nil {
			continue
		}
		app, err := s.store.Application(ctx, binding.ApplicationId)
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(apperror.KindNotFound, "Application "+binding.ApplicationId+" not found")
		}
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
		}
		if app.ProjectId == nil || projectId == nil || *app.ProjectId != *projectId {
			return nil, apperror.New(apperror.KindValidation, "Build version binding application must belong to repository project")
		}
		version, err := s.sourceVersionForBuildBinding(ctx, *binding)
		if err != nil {
			return nil, err
		}
		components, err := s.store.VersionComponentsByVersion(ctx, version.Id)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to load source version components", err)
		}
		if !versionContainsComponent(components, binding.ComponentName) {
			return nil, apperror.New(apperror.KindValidation, "Source version does not contain component "+binding.ComponentName)
		}
		bindings = append(bindings, model.PipelineRunBuildVersionBinding{
			PipelineStageId: stage.Id, ApplicationId: app.Id, ApplicationName: app.Name, ComponentName: binding.ComponentName,
			SourceVersionId: version.Id, SourceVersionLabel: version.Label,
		})
	}
	return bindings, nil
}

func (s Service) sourceVersionForBuildBinding(ctx context.Context, binding model.BuildVersionBinding) (model.Version, error) {
	switch binding.ForkStrategy {
	case "latest":
		version, err := s.store.LatestVersionByApplication(ctx, binding.ApplicationId)
		if errors.Is(err, repository.ErrNotFound) {
			return model.Version{}, apperror.New(apperror.KindValidation, "Application has no version to fork")
		}
		if err != nil {
			return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to load latest application version", err)
		}
		return version, nil
	case "fixed":
		if binding.FixedVersionId == nil || strings.TrimSpace(*binding.FixedVersionId) == "" {
			return model.Version{}, apperror.New(apperror.KindValidation, "fixed fork strategy requires fixed_version_id")
		}
		version, err := s.store.Version(ctx, strings.TrimSpace(*binding.FixedVersionId))
		if errors.Is(err, repository.ErrNotFound) {
			return model.Version{}, apperror.New(apperror.KindNotFound, "Fixed version not found")
		}
		if err != nil {
			return model.Version{}, apperror.Wrap(apperror.KindInternal, "Failed to load fixed version", err)
		}
		if version.ApplicationId != binding.ApplicationId {
			return model.Version{}, apperror.New(apperror.KindValidation, "fixed_version_id must belong to application_id")
		}
		return version, nil
	default:
		return model.Version{}, apperror.New(apperror.KindValidation, "fork_strategy must be latest or fixed")
	}
}

func versionContainsComponent(components []model.VersionComponent, name string) bool {
	for _, component := range components {
		if component.Name == name {
			return true
		}
	}
	return false
}

func validateSourceCommitDependencies(stages []model.StageDefinition) error {
	byID := make(map[string]model.StageDefinition, len(stages))
	for _, stage := range stages {
		byID[stage.Id] = stage
	}
	for _, stage := range stages {
		if !stageProducesDockerImage(stage) {
			continue
		}
		if _, err := sourceCommitArtifactForStage(stage, byID); err != nil {
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

type sourceCommitArtifact struct {
	ProducerStageId string
	Artifact        model.ArtifactConfig
}

func sourceCommitArtifactForStage(stage model.StageDefinition, stages map[string]model.StageDefinition) (sourceCommitArtifact, error) {
	seen := map[string]bool{}
	artifacts := make([]sourceCommitArtifact, 0, 1)
	var visit func(string) error
	visit = func(stageID string) error {
		if seen[stageID] {
			return nil
		}
		seen[stageID] = true
		ancestor, ok := stages[stageID]
		if !ok {
			return fmt.Errorf("dependency %s does not exist", stageID)
		}
		for _, artifact := range ancestor.Artifacts {
			if artifact.Collector == "command" && artifact.Format == "git_object_id" {
				artifacts = append(artifacts, sourceCommitArtifact{ProducerStageId: ancestor.Id, Artifact: artifact})
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
			return sourceCommitArtifact{}, err
		}
	}
	if len(artifacts) != 1 {
		return sourceCommitArtifact{}, fmt.Errorf("requires exactly one transitive git_object_id artifact, found %d", len(artifacts))
	}
	return artifacts[0], nil
}

package pipelinerunsvc

import (
	"context"
	"errors"
	"strings"

	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func (s Service) ListArtifacts(ctx context.Context, userId string, input pipelinerundto.ArtifactListInput) (repository.Page[model.Artifact], error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Page[model.Artifact]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Artifact]{}, err
	}
	if err := s.ensureRunRepositoryFilter(ctx, projectId, input.RepositoryId); err != nil {
		return repository.Page[model.Artifact]{}, err
	}
	if err := s.ensureRunPipelineFilter(ctx, projectId, input.PipelineId); err != nil {
		return repository.Page[model.Artifact]{}, err
	}
	items, err := s.store.ListArtifacts(ctx, projectId, input.RepositoryId, input.PipelineId, input.Page, input.PerPage, input.Search)
	if err != nil {
		return repository.Page[model.Artifact]{}, apperror.Wrap(apperror.KindInternal, "Failed to list artifacts", err)
	}
	return items, nil
}

func (s Service) ArtifactForUser(ctx context.Context, userId string, artifactId string) (model.Artifact, error) {
	artifactId = strings.TrimSpace(artifactId)
	item, err := s.store.Artifact(ctx, artifactId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Artifact{}, apperror.New(apperror.KindNotFound, "Artifact "+artifactId+" not found")
		}
		return model.Artifact{}, apperror.Wrap(apperror.KindInternal, "Failed to load artifact", err)
	}
	projectId, err := requiredProjectID(item.ProjectId, "Artifact")
	if err != nil {
		return model.Artifact{}, err
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Artifact{}, err
	}
	return item, nil
}

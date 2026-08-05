package pipelinerunsvc

import (
	"context"
	"errors"
	"strings"

	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) ListArtifacts(ctx context.Context, userId string, input pipelinerundto.ArtifactListInput) (repository.Page[model.Artifact], error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Page[model.Artifact]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Artifact]{}, err
	}
	if err := s.ensurePipelineRunRepositoryFilter(ctx, projectId, input.RepositoryId); err != nil {
		return repository.Page[model.Artifact]{}, err
	}
	if err := s.ensurePipelineRunTemplateFilter(ctx, projectId, input.TemplateId); err != nil {
		return repository.Page[model.Artifact]{}, err
	}
	items, err := s.store.ListArtifacts(ctx, projectId, input.RepositoryId, input.TemplateId, input.Page, input.PerPage, input.Search)
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
	if item.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *item.ProjectId, userId); err != nil {
			return model.Artifact{}, err
		}
	}
	return item, nil
}

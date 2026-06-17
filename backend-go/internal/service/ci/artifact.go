package cisvc

import (
	"context"
	"strings"

	"backend/internal/apperror"
	"backend/internal/repository"
)

type ArtifactListInput struct {
	ProjectId    string
	RepositoryId string
	TemplateId   string
	Search       string
	Page         int
	PerPage      int
}

func (s Service) ListArtifacts(ctx context.Context, userId string, input ArtifactListInput) (repository.Page[repository.Artifact], error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return repository.Page[repository.Artifact]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[repository.Artifact]{}, err
	}
	if err := s.ensurePipelineRunRepositoryFilter(ctx, projectId, input.RepositoryId); err != nil {
		return repository.Page[repository.Artifact]{}, err
	}
	if err := s.ensurePipelineRunTemplateFilter(ctx, projectId, input.TemplateId); err != nil {
		return repository.Page[repository.Artifact]{}, err
	}
	items, err := s.store.ListArtifacts(ctx, projectId, input.RepositoryId, input.TemplateId, input.Page, input.PerPage, input.Search)
	if err != nil {
		return repository.Page[repository.Artifact]{}, apperror.Wrap(apperror.KindInternal, "Failed to list artifacts", err)
	}
	return items, nil
}

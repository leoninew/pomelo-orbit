package pipelinerunsvc

import (
	"context"
	"errors"
	"strings"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func (s Service) loadRepositoryForUser(ctx context.Context, userId string, repositoryId string) (model.Repository, error) {
	repositoryId = strings.TrimSpace(repositoryId)
	repo, err := s.store.Repository(ctx, repositoryId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Repository{}, apperror.New(apperror.KindNotFound, "Repository "+repositoryId+" not found")
		}
		return model.Repository{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	if repo.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *repo.ProjectId, userId); err != nil {
			return model.Repository{}, err
		}
	}
	return repo, nil
}

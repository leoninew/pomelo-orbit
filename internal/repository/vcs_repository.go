package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// RepositoryStore persists source repositories. Webhook delivery is not part of
// the current CI model.
type RepositoryStore interface {
	ListRepositories(ctx context.Context, projectId *string, page int, perPage int, search string) (Page[model.Repository], error)
	Repository(ctx context.Context, id string) (model.Repository, error)
	RepositoryByCode(ctx context.Context, projectId *string, code string) (model.Repository, error)
	CreateRepository(ctx context.Context, repo model.Repository) error
	UpdateRepository(ctx context.Context, repo model.Repository) error
	DeleteRepository(ctx context.Context, id string) error
	RepositoryHasRunningPipelines(ctx context.Context, repositoryId string) (bool, error)
}

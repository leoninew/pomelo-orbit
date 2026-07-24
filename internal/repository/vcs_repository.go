package repository

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// RepositoryStore persists source repositories and webhooks.
type RepositoryStore interface {
	ListRepositories(ctx context.Context, projectId *string, page int, perPage int, search string) (Page[model.Repository], error)
	Repository(ctx context.Context, id string) (model.Repository, error)
	RepositoryByCode(ctx context.Context, projectId *string, code string) (model.Repository, error)
	CreateRepository(ctx context.Context, repo model.Repository) error
	UpdateRepository(ctx context.Context, repo model.Repository) error
	DeleteRepository(ctx context.Context, id string) error
	RepositoryHasRunningPipelines(ctx context.Context, repositoryId string) (bool, error)
	ListRepositoryWebhooks(ctx context.Context, repositoryId string) ([]model.RepositoryWebhook, error)
	RepositoryWebhook(ctx context.Context, id string) (model.RepositoryWebhook, error)
	CreateRepositoryWebhook(ctx context.Context, webhook model.RepositoryWebhook) error
	UpdateRepositoryWebhook(ctx context.Context, webhook model.RepositoryWebhook) error
	DeleteRepositoryWebhook(ctx context.Context, id string) error
}

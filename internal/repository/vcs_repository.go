package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// RepositoryProjectCounter provides Project lifecycle facts owned by the
// Repository domain.
type RepositoryProjectCounter interface {
	CountRepositoriesByProject(ctx context.Context, projectId string) (int, error)
}

// RepositoryStore persists source repositories. Webhook delivery is not part of
// the current CI model.
type RepositoryStore interface {
	ListRepositories(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.Repository], error)
	Repository(ctx context.Context, projectId string, id string) (model.Repository, error)
	RepositoryByCode(ctx context.Context, projectId string, code string) (model.Repository, error)
	CreateRepository(ctx context.Context, repo model.Repository) error
	UpdateRepository(ctx context.Context, projectId string, repo model.Repository) error
	DeleteRepository(ctx context.Context, projectId string, id string) error
	RepositoryReferencesCredential(ctx context.Context, projectId string, credentialId string) (bool, error)
}

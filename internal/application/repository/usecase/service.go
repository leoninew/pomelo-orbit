package repositorysvc

import (
	"context"
	"log/slog"

	repositoryport "github.com/leoninew/pomelo-orbit/internal/application/repository/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type Service struct {
	store       stores
	localSource repositoryport.LocalDirectorySource
	logger      *slog.Logger
}
type stores struct {
	project    repository.ProjectReader
	credential repository.CredentialStore
	repository repository.RepositoryStore
}

func New(project repository.ProjectReader, credential repository.CredentialStore, repos repository.RepositoryStore, localSource repositoryport.LocalDirectorySource, logger *slog.Logger) Service {
	return Service{store: stores{project: project, credential: credential, repository: repos}, localSource: localSource, logger: logger}
}
func (s stores) Project(ctx context.Context, id string) (model.Project, error) {
	return s.project.Project(ctx, id)
}
func (s stores) IsProjectMember(ctx context.Context, projectID, userID string) (bool, error) {
	return s.project.IsProjectMember(ctx, projectID, userID)
}
func (s stores) ListRepositories(ctx context.Context, projectID *string, page, perPage int, search string) (repository.Page[model.Repository], error) {
	return s.repository.ListRepositories(ctx, projectID, page, perPage, search)
}
func (s stores) Repository(ctx context.Context, id string) (model.Repository, error) {
	return s.repository.Repository(ctx, id)
}
func (s stores) RepositoryByCode(ctx context.Context, projectID *string, code string) (model.Repository, error) {
	return s.repository.RepositoryByCode(ctx, projectID, code)
}
func (s stores) CreateRepository(ctx context.Context, item model.Repository) error {
	return s.repository.CreateRepository(ctx, item)
}
func (s stores) UpdateRepository(ctx context.Context, item model.Repository) error {
	return s.repository.UpdateRepository(ctx, item)
}
func (s stores) DeleteRepository(ctx context.Context, id string) error {
	return s.repository.DeleteRepository(ctx, id)
}
func (s stores) RepositoryHasRunningPipelines(ctx context.Context, id string) (bool, error) {
	return s.repository.RepositoryHasRunningPipelines(ctx, id)
}
func (s stores) CredentialExists(ctx context.Context, id string) (bool, error) {
	return s.credential.CredentialExists(ctx, id)
}
func (s stores) CredentialName(ctx context.Context, id string) (*string, error) {
	return s.credential.CredentialName(ctx, id)
}

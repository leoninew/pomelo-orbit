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
	project     repository.ProjectReader
	credential  repository.CredentialStore
	repository  repository.RepositoryStore
	pipelineRun repository.PipelineRunStore
}

func New(project repository.ProjectReader, credential repository.CredentialStore, repos repository.RepositoryStore, pipelineRun repository.PipelineRunStore, localSource repositoryport.LocalDirectorySource, logger *slog.Logger) Service {
	return Service{store: stores{project: project, credential: credential, repository: repos, pipelineRun: pipelineRun}, localSource: localSource, logger: logger}
}
func (s stores) Project(ctx context.Context, id string) (model.Project, error) {
	return s.project.Project(ctx, id)
}
func (s stores) IsProjectMember(ctx context.Context, projectId, userId string) (bool, error) {
	return s.project.IsProjectMember(ctx, projectId, userId)
}
func (s stores) ListRepositories(ctx context.Context, projectId string, page, perPage int, search string) (repository.Page[model.Repository], error) {
	return s.repository.ListRepositories(ctx, projectId, page, perPage, search)
}
func (s stores) Repository(ctx context.Context, projectId string, id string) (model.Repository, error) {
	return s.repository.Repository(ctx, projectId, id)
}
func (s stores) RepositoryByCode(ctx context.Context, projectId string, code string) (model.Repository, error) {
	return s.repository.RepositoryByCode(ctx, projectId, code)
}
func (s stores) CreateRepository(ctx context.Context, item model.Repository) error {
	return s.repository.CreateRepository(ctx, item)
}
func (s stores) UpdateRepository(ctx context.Context, projectId string, item model.Repository) error {
	return s.repository.UpdateRepository(ctx, projectId, item)
}
func (s stores) DeleteRepository(ctx context.Context, projectId string, id string) error {
	return s.repository.DeleteRepository(ctx, projectId, id)
}
func (s stores) RepositoryHasActivePipelineRun(ctx context.Context, projectId string, id string) (bool, error) {
	return s.pipelineRun.RepositoryHasActivePipelineRun(ctx, projectId, id)
}
func (s stores) CredentialExists(ctx context.Context, projectId string, id string) (bool, error) {
	return s.credential.CredentialExists(ctx, projectId, id)
}
func (s stores) CredentialName(ctx context.Context, projectId string, id string) (*string, error) {
	return s.credential.CredentialName(ctx, projectId, id)
}

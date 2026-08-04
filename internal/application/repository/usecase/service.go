package repositorysvc

import (
	"context"
	"log/slog"
	"time"

	pipelinerunport "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/port"
	repositoryport "gitee.com/leoninew/PomeloOrbit-go/internal/application/repository/port"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type Service struct {
	store       stores
	dispatcher  pipelinerunport.PipelineRunDispatcher
	localSource repositoryport.LocalDirectorySource
	logger      *slog.Logger
}

type stores struct {
	project     repository.ProjectReader
	credential  repository.CredentialStore
	repository  repository.RepositoryStore
	pipeline    repository.PipelineStore
	pipelineRun repository.PipelineRunStore
}

func New(
	project repository.ProjectReader,
	credential repository.CredentialStore,
	repoStore repository.RepositoryStore,
	pipeline repository.PipelineStore,
	pipelineRun repository.PipelineRunStore,
	dispatcher pipelinerunport.PipelineRunDispatcher,
	localSource repositoryport.LocalDirectorySource,
	logger *slog.Logger,
) Service {
	s := stores{
		project: project, credential: credential, repository: repoStore,
		pipeline: pipeline, pipelineRun: pipelineRun,
	}
	return Service{store: s, dispatcher: dispatcher, localSource: localSource, logger: logger}
}

func (s stores) Project(ctx context.Context, id string) (model.Project, error) {
	return s.project.Project(ctx, id)
}
func (s stores) IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error) {
	return s.project.IsProjectMember(ctx, projectId, userId)
}
func (s stores) ListRepositories(ctx context.Context, projectId *string, page int, perPage int, search string) (repository.Page[model.Repository], error) {
	return s.repository.ListRepositories(ctx, projectId, page, perPage, search)
}
func (s stores) Repository(ctx context.Context, id string) (model.Repository, error) {
	return s.repository.Repository(ctx, id)
}
func (s stores) RepositoryByCode(ctx context.Context, projectId *string, code string) (model.Repository, error) {
	return s.repository.RepositoryByCode(ctx, projectId, code)
}
func (s stores) CreateRepository(ctx context.Context, repo model.Repository) error {
	return s.repository.CreateRepository(ctx, repo)
}
func (s stores) UpdateRepository(ctx context.Context, repo model.Repository) error {
	return s.repository.UpdateRepository(ctx, repo)
}
func (s stores) DeleteRepository(ctx context.Context, id string) error {
	return s.repository.DeleteRepository(ctx, id)
}
func (s stores) RepositoryHasRunningPipelines(ctx context.Context, repositoryId string) (bool, error) {
	return s.repository.RepositoryHasRunningPipelines(ctx, repositoryId)
}
func (s stores) ListRepositoryWebhooks(ctx context.Context, repositoryId string) ([]model.RepositoryWebhook, error) {
	return s.repository.ListRepositoryWebhooks(ctx, repositoryId)
}
func (s stores) RepositoryWebhook(ctx context.Context, id string) (model.RepositoryWebhook, error) {
	return s.repository.RepositoryWebhook(ctx, id)
}
func (s stores) CreateRepositoryWebhook(ctx context.Context, webhook model.RepositoryWebhook) error {
	return s.repository.CreateRepositoryWebhook(ctx, webhook)
}
func (s stores) UpdateRepositoryWebhook(ctx context.Context, webhook model.RepositoryWebhook) error {
	return s.repository.UpdateRepositoryWebhook(ctx, webhook)
}
func (s stores) DeleteRepositoryWebhook(ctx context.Context, id string) error {
	return s.repository.DeleteRepositoryWebhook(ctx, id)
}
func (s stores) CredentialExists(ctx context.Context, id string) (bool, error) {
	return s.credential.CredentialExists(ctx, id)
}
func (s stores) CredentialName(ctx context.Context, id string) (*string, error) {
	return s.credential.CredentialName(ctx, id)
}
func (s stores) PipelineTemplate(ctx context.Context, id string) (model.PipelineTemplate, error) {
	return s.pipeline.PipelineTemplate(ctx, id)
}
func (s stores) LatestPipelineSnapshot(ctx context.Context, templateId string) (model.PipelineSnapshot, error) {
	return s.pipeline.LatestPipelineSnapshot(ctx, templateId)
}
func (s stores) PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error) {
	return s.pipeline.PipelineSnapshot(ctx, id)
}
func (s stores) CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error {
	return s.pipeline.CreatePipelineSnapshot(ctx, snapshot)
}
func (s stores) PipelineTemplateStages(ctx context.Context, templateId string) ([]model.PipelineTemplateStage, error) {
	return s.pipeline.PipelineTemplateStages(ctx, templateId)
}
func (s stores) PipelineStagesByIds(ctx context.Context, projectId string, ids []string) ([]model.PipelineStage, error) {
	return s.pipeline.PipelineStagesByIds(ctx, projectId, ids)
}
func (s stores) CreatePipelineRun(ctx context.Context, run model.PipelineRun) error {
	return s.pipelineRun.CreatePipelineRun(ctx, run)
}
func (s stores) ListPipelineRunsByRepository(ctx context.Context, repositoryId string, page int, perPage int) (repository.Page[model.PipelineRun], error) {
	return s.pipelineRun.ListPipelineRunsByRepository(ctx, repositoryId, page, perPage)
}
func (s stores) ListPipelineRuns(ctx context.Context, projectId string, repositoryId string, templateId string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[model.PipelineRun], error) {
	return s.pipelineRun.ListPipelineRuns(ctx, projectId, repositoryId, templateId, dateFrom, dateTo, page, perPage)
}

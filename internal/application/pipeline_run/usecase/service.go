package pipelinerunsvc

import (
	"context"
	"log/slog"
	"time"

	pipelinerunport "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/port"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type Service struct {
	store             stores
	executionStore    pipelineExecutionStore
	dispatcher        pipelinerunport.PipelineRunDispatcher
	workspace         pipelinerunport.Workspace
	logStore          pipelinerunport.LogReader
	executionLogStore pipelinerunport.ExecutionLogStore
	secretKey         string
	logger            *slog.Logger
	runner            pipelinerunport.ContainerRunner
}

type pipelineExecutionStore interface {
	PipelineRun(ctx context.Context, id string) (model.PipelineRun, error)
	Repository(ctx context.Context, id string) (model.Repository, error)
	Credential(ctx context.Context, id string) (model.Credential, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	PipelineTemplate(ctx context.Context, id string) (model.PipelineTemplate, error)
	MarkPipelineRunRunning(ctx context.Context, id string) error
	CompletePipelineRun(ctx context.Context, id string, status string, message string) error
	InsertPipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error
	UpdatePipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error
	InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error
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
	workspace pipelinerunport.Workspace,
	secretKey string,
	logger *slog.Logger,
	runner pipelinerunport.ContainerRunner,
	logStore pipelinerunport.LogReader,
) Service {
	s := stores{
		project: project, credential: credential, repository: repoStore,
		pipeline: pipeline, pipelineRun: pipelineRun,
	}
	return Service{
		store: s, executionStore: s, dispatcher: dispatcher, workspace: workspace,
		logStore: logStore, secretKey: secretKey, logger: logger, runner: runner,
	}
}

func NewExecutionService(
	project repository.ProjectReader,
	credential repository.CredentialStore,
	repoStore repository.RepositoryStore,
	pipeline repository.PipelineStore,
	pipelineRun repository.PipelineRunStore,
	workspace pipelinerunport.Workspace,
	secretKey string,
	logger *slog.Logger,
	runner pipelinerunport.ContainerRunner,
	logStore pipelinerunport.ExecutionLogStore,
) Service {
	s := stores{
		project: project, credential: credential, repository: repoStore,
		pipeline: pipeline, pipelineRun: pipelineRun,
	}
	return Service{
		store: s, executionStore: s, workspace: workspace, logStore: logStore,
		executionLogStore: logStore, secretKey: secretKey, logger: logger, runner: runner,
	}
}

func (s stores) Project(ctx context.Context, id string) (model.Project, error) {
	return s.project.Project(ctx, id)
}
func (s stores) IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error) {
	return s.project.IsProjectMember(ctx, projectId, userId)
}
func (s stores) Repository(ctx context.Context, id string) (model.Repository, error) {
	return s.repository.Repository(ctx, id)
}
func (s stores) Credential(ctx context.Context, id string) (model.Credential, error) {
	return s.credential.Credential(ctx, id)
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
func (s stores) ListPipelineRuns(ctx context.Context, projectId string, repositoryId string, templateId string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[model.PipelineRun], error) {
	return s.pipelineRun.ListPipelineRuns(ctx, projectId, repositoryId, templateId, dateFrom, dateTo, page, perPage)
}
func (s stores) ListPipelineRunsByRepository(ctx context.Context, repositoryId string, page int, perPage int) (repository.Page[model.PipelineRun], error) {
	return s.pipelineRun.ListPipelineRunsByRepository(ctx, repositoryId, page, perPage)
}
func (s stores) PipelineRun(ctx context.Context, id string) (model.PipelineRun, error) {
	return s.pipelineRun.PipelineRun(ctx, id)
}
func (s stores) ListPipelineStageRuns(ctx context.Context, runId string) ([]model.PipelineStageRun, error) {
	return s.pipelineRun.ListPipelineStageRuns(ctx, runId)
}
func (s stores) PipelineStageRun(ctx context.Context, id string) (model.PipelineStageRun, error) {
	return s.pipelineRun.PipelineStageRun(ctx, id)
}
func (s stores) ListArtifacts(ctx context.Context, projectId string, repositoryId string, templateId string, page int, perPage int, search string) (repository.Page[model.Artifact], error) {
	return s.pipelineRun.ListArtifacts(ctx, projectId, repositoryId, templateId, page, perPage, search)
}
func (s stores) ListArtifactsByRun(ctx context.Context, projectId *string, runId string) ([]model.Artifact, error) {
	return s.pipelineRun.ListArtifactsByRun(ctx, projectId, runId)
}
func (s stores) CreatePipelineRun(ctx context.Context, run model.PipelineRun) error {
	return s.pipelineRun.CreatePipelineRun(ctx, run)
}
func (s stores) CancelPipelineRun(ctx context.Context, id string) error {
	return s.pipelineRun.CancelPipelineRun(ctx, id)
}
func (s stores) MarkPipelineRunRunning(ctx context.Context, id string) error {
	return s.pipelineRun.MarkPipelineRunRunning(ctx, id)
}
func (s stores) CompletePipelineRun(ctx context.Context, id string, status string, message string) error {
	return s.pipelineRun.CompletePipelineRun(ctx, id, status, message)
}
func (s stores) InsertPipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error {
	return s.pipelineRun.InsertPipelineStageRun(ctx, stage)
}
func (s stores) UpdatePipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error {
	return s.pipelineRun.UpdatePipelineStageRun(ctx, stage)
}
func (s stores) InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error {
	return s.pipelineRun.InsertArtifact(ctx, projectId, run, stageName, artifact, path)
}

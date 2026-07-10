package repository

import (
	"context"
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// CIStore persists CI configuration and execution records.
type CIStore interface {
	Project(ctx context.Context, id string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
	ListRepositories(ctx context.Context, projectId *string, page int, perPage int, search string) (Page[model.Repository], error)
	Repository(ctx context.Context, id string) (model.Repository, error)
	RepositoryByCode(ctx context.Context, projectId *string, code string) (model.Repository, error)
	ListCredentials(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.Credential], error)
	Credential(ctx context.Context, id string) (model.Credential, error)
	CredentialByName(ctx context.Context, projectId string, name string) (model.Credential, error)
	CredentialExists(ctx context.Context, id string) (bool, error)
	CredentialName(ctx context.Context, id string) (*string, error)
	CreateCredential(ctx context.Context, credential model.Credential) error
	UpdateCredential(ctx context.Context, credential model.Credential) error
	DeleteCredential(ctx context.Context, id string) error
	CredentialReferencedByRepositories(ctx context.Context, projectId string, credentialId string) (bool, error)
	CreateRepository(ctx context.Context, repo model.Repository) error
	UpdateRepository(ctx context.Context, repo model.Repository) error
	DeleteRepository(ctx context.Context, id string) error
	RepositoryHasRunningPipelines(ctx context.Context, repositoryId string) (bool, error)
	ListRepositoryWebhooks(ctx context.Context, repositoryId string) ([]model.RepositoryWebhook, error)
	RepositoryWebhook(ctx context.Context, id string) (model.RepositoryWebhook, error)
	CreateRepositoryWebhook(ctx context.Context, webhook model.RepositoryWebhook) error
	UpdateRepositoryWebhook(ctx context.Context, webhook model.RepositoryWebhook) error
	DeleteRepositoryWebhook(ctx context.Context, id string) error
	PipelineTemplate(ctx context.Context, id string) (model.PipelineTemplate, error)
	ListPipelineTemplates(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.PipelineTemplate], error)
	PipelineTemplateByName(ctx context.Context, projectId string, name string) (model.PipelineTemplate, error)
	PipelineTemplateStages(ctx context.Context, templateId string) ([]model.PipelineTemplateStage, error)
	ListBuildStages(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.BuildStage], error)
	BuildStage(ctx context.Context, id string) (model.BuildStage, error)
	BuildStageByName(ctx context.Context, projectId string, name string) (model.BuildStage, error)
	BuildStagesByIds(ctx context.Context, projectId string, ids []string) ([]model.BuildStage, error)
	CreateBuildStage(ctx context.Context, stage model.BuildStage) error
	UpdateBuildStage(ctx context.Context, stage model.BuildStage) error
	DeleteBuildStage(ctx context.Context, id string) error
	BuildStageReferencedByTemplates(ctx context.Context, projectId string, stageId string) (bool, error)
	CreatePipelineTemplate(ctx context.Context, template model.PipelineTemplate) error
	UpdatePipelineTemplate(ctx context.Context, template model.PipelineTemplate) error
	UpdatePipelineTemplateWithStages(ctx context.Context, template model.PipelineTemplate, stages []model.PipelineTemplateStage) error
	DuplicatePipelineTemplate(ctx context.Context, template model.PipelineTemplate, stages []model.PipelineTemplateStage) error
	PipelineTemplateReferencedByWebhooks(ctx context.Context, templateId string) (bool, error)
	DeletePipelineTemplate(ctx context.Context, id string) error
	LatestPipelineSnapshot(ctx context.Context, templateId string) (model.PipelineSnapshot, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error
	ListPipelineRuns(ctx context.Context, projectId string, repositoryId string, templateId string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (Page[model.PipelineRun], error)
	ListPipelineRunsByRepository(ctx context.Context, repositoryId string, page int, perPage int) (Page[model.PipelineRun], error)
	PipelineRun(ctx context.Context, id string) (model.PipelineRun, error)
	ListStageRuns(ctx context.Context, runId string) ([]model.StageRun, error)
	StageRun(ctx context.Context, id string) (model.StageRun, error)
	ListArtifacts(ctx context.Context, projectId string, repositoryId string, templateId string, page int, perPage int, search string) (Page[model.Artifact], error)
	ListArtifactsByRun(ctx context.Context, projectId *string, runId string) ([]model.Artifact, error)
	CreatePipelineRun(ctx context.Context, run model.PipelineRun) error
	CancelPipelineRun(ctx context.Context, id string) error
}

// PipelineExecutionStore persists state transitions and output of a pipeline execution.
type PipelineExecutionStore interface {
	PipelineRun(ctx context.Context, id string) (model.PipelineRun, error)
	Repository(ctx context.Context, id string) (model.Repository, error)
	Credential(ctx context.Context, id string) (model.Credential, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	PipelineTemplate(ctx context.Context, id string) (model.PipelineTemplate, error)
	MarkPipelineRunRunning(ctx context.Context, id string) error
	CompletePipelineRun(ctx context.Context, id string, status string, message string) error
	InsertStageRun(ctx context.Context, stage model.StageRun) error
	UpdateStageRun(ctx context.Context, stage model.StageRun) error
	InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error
}

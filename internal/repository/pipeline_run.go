package repository

import (
	"context"
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// PipelineRunStore persists pipeline runs, pipeline stage runs, and artifacts.
type PipelineRunStore interface {
	ListPipelineRuns(ctx context.Context, projectId string, repositoryId string, templateId string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (Page[model.PipelineRun], error)
	ListPipelineRunsByRepository(ctx context.Context, repositoryId string, page int, perPage int) (Page[model.PipelineRun], error)
	PipelineRun(ctx context.Context, id string) (model.PipelineRun, error)
	ListPipelineStageRuns(ctx context.Context, runId string) ([]model.PipelineStageRun, error)
	PipelineStageRun(ctx context.Context, id string) (model.PipelineStageRun, error)
	ListArtifacts(ctx context.Context, projectId string, repositoryId string, templateId string, page int, perPage int, search string) (Page[model.Artifact], error)
	ListArtifactsByRun(ctx context.Context, projectId *string, runId string) ([]model.Artifact, error)
	Artifact(ctx context.Context, artifactId string) (model.Artifact, error)
	CreatePipelineRun(ctx context.Context, run model.PipelineRun) error
	CreatePipelineRunWithBuildVersionBindings(ctx context.Context, run model.PipelineRun, bindings []model.PipelineRunBuildVersionBinding) error
	RepositoryHasRunningPipelineRun(ctx context.Context, repositoryId string) (bool, error)
	PipelineRunBuildVersionBinding(ctx context.Context, pipelineRunId string, pipelineStageId string) (model.PipelineRunBuildVersionBinding, error)
	RunInTransaction(ctx context.Context, fn func(context.Context) error) error
	CancelPipelineRun(ctx context.Context, id string) error
	MarkPipelineRunRunning(ctx context.Context, id string) error
	CompletePipelineRun(ctx context.Context, id string, status string, message string) error
	InsertPipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error
	UpdatePipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error
	CreateArtifact(ctx context.Context, artifact model.Artifact) error
	CommandArtifactByRunStageAndName(ctx context.Context, pipelineRunId string, pipelineStageId string, name string) (model.Artifact, error)
	CompletePipelineRunBuildVersionBinding(ctx context.Context, pipelineRunId string, pipelineStageId string, generatedVersionId string, generatedVersionLabel string, artifactId string) error
}

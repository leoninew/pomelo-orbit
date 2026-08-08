package repository

import (
	"context"
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// PipelineRunStore persists immutable PipelineRun execution history.
type PipelineRunStore interface {
	ListPipelineRuns(ctx context.Context, projectId string, repositoryId string, pipelineId string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (Page[model.PipelineRun], error)
	ListPipelineRunsByPipeline(ctx context.Context, pipelineId string, page int, perPage int) (Page[model.PipelineRun], error)
	PipelineRun(ctx context.Context, id string) (model.PipelineRun, error)
	ListPipelineStageRuns(ctx context.Context, runId string) ([]model.PipelineStageRun, error)
	PipelineStageRun(ctx context.Context, id string) (model.PipelineStageRun, error)
	ListArtifacts(ctx context.Context, projectId string, repositoryId string, pipelineId string, page int, perPage int, search string) (Page[model.Artifact], error)
	ListArtifactsByRun(ctx context.Context, projectId *string, runId string) ([]model.Artifact, error)
	Artifact(ctx context.Context, artifactId string) (model.Artifact, error)
	CreatePipelineRun(ctx context.Context, run model.PipelineRun, binding *model.PipelineRunVersionBinding, stageRuns []model.PipelineStageRun) error
	RepositoryHasActivePipelineRun(ctx context.Context, repositoryId string) (bool, error)
	PipelineRunVersionBinding(ctx context.Context, pipelineRunId string) (model.PipelineRunVersionBinding, error)
	CancelPipelineRun(ctx context.Context, id string) (bool, error)
	CancelRunningPipelineStageRuns(ctx context.Context, runId string) error
	BeginPipelineRun(ctx context.Context, id string) (bool, error)
	CompletePipelineRun(ctx context.Context, id string, status string, message string) (bool, error)
	BeginPipelineStageRun(ctx context.Context, id string) (bool, error)
	CompletePipelineStageRun(ctx context.Context, stage model.PipelineStageRun) (bool, error)
	CreateArtifact(ctx context.Context, artifact model.Artifact) error
	CommandArtifactByRunStageAndName(ctx context.Context, pipelineRunId string, pipelineStageId string, name string) (model.Artifact, error)
	CompletePipelineRunVersionBinding(ctx context.Context, pipelineRunId string, generatedVersionId string, generatedVersionLabel string) error
}

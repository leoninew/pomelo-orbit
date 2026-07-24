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
	CreatePipelineRun(ctx context.Context, run model.PipelineRun) error
	CancelPipelineRun(ctx context.Context, id string) error
	MarkPipelineRunRunning(ctx context.Context, id string) error
	CompletePipelineRun(ctx context.Context, id string, status string, message string) error
	InsertPipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error
	UpdatePipelineStageRun(ctx context.Context, stage model.PipelineStageRun) error
	InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error
}

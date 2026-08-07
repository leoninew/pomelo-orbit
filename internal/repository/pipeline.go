package repository

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// PipelineStore persists Pipeline aggregates and their immutable snapshots.
// Application, Repository, Version, and source-template references are
// logical references; only stages are owned by the aggregate.
type PipelineStore interface {
	Pipeline(ctx context.Context, id string) (model.Pipeline, error)
	PipelineByName(ctx context.Context, projectId string, name string) (model.Pipeline, error)
	ListPipelines(ctx context.Context, projectId string, kind string, page int, perPage int, search string) (Page[model.Pipeline], error)
	CreatePipeline(ctx context.Context, pipeline model.Pipeline) error
	UpdatePipeline(ctx context.Context, pipeline model.Pipeline) error
	DeletePipeline(ctx context.Context, id string) error

	PipelineStages(ctx context.Context, pipelineId string) ([]model.PipelineStage, error)
	PipelineStage(ctx context.Context, id string) (model.PipelineStage, error)
	CreatePipelineStage(ctx context.Context, stage model.PipelineStage) error
	UpdatePipelineStage(ctx context.Context, stage model.PipelineStage) error
	DeletePipelineStage(ctx context.Context, id string) error
	UpdatePipelineWithStages(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error
	CreatePipelineWithStages(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error

	LatestPipelineSnapshot(ctx context.Context, pipelineId string) (model.PipelineSnapshot, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error
}

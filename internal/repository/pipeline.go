package repository

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// PipelineStore persists Pipeline aggregates, project stage templates, and
// their immutable snapshots. A Template Pipeline owns references; an
// Application Pipeline owns executable stages.
type PipelineStore interface {
	Pipeline(ctx context.Context, id string) (model.Pipeline, error)
	PipelineByName(ctx context.Context, projectId string, name string) (model.Pipeline, error)
	ListPipelines(ctx context.Context, projectId string, kind string, page int, perPage int, search string) (Page[model.Pipeline], error)
	CreatePipeline(ctx context.Context, pipeline model.Pipeline) error
	UpdatePipeline(ctx context.Context, pipeline model.Pipeline) error
	DeletePipeline(ctx context.Context, id string) error

	ListPipelineStageTemplates(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.PipelineStage], error)
	PipelineStageTemplate(ctx context.Context, id string) (model.PipelineStage, error)
	PipelineStageTemplateByName(ctx context.Context, projectId string, name string) (model.PipelineStage, error)
	CreatePipelineStageTemplate(ctx context.Context, stage model.PipelineStage) error
	UpdatePipelineStageTemplate(ctx context.Context, stage model.PipelineStage) error
	DeletePipelineStageTemplate(ctx context.Context, id string) error

	TemplatePipelineStageReferences(ctx context.Context, pipelineId string) ([]model.PipelineStageReference, error)
	ApplicationPipelineStages(ctx context.Context, pipelineId string) ([]model.PipelineStage, error)
	UpdateTemplatePipelineWithReferences(ctx context.Context, pipeline model.Pipeline, references []model.PipelineStageReference) error
	UpdateApplicationPipelineWithStages(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error
	CreateApplicationPipelineWithStages(ctx context.Context, pipeline model.Pipeline, stages []model.PipelineStage) error

	LatestPipelineSnapshot(ctx context.Context, pipelineId string) (model.PipelineSnapshot, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error
}

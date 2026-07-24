package repository

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// PipelineStore persists pipeline stages, templates, and snapshots.
type PipelineStore interface {
	PipelineTemplate(ctx context.Context, id string) (model.PipelineTemplate, error)
	ListPipelineTemplates(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.PipelineTemplate], error)
	PipelineTemplateByName(ctx context.Context, projectId string, name string) (model.PipelineTemplate, error)
	PipelineTemplateStages(ctx context.Context, templateId string) ([]model.PipelineTemplateStage, error)
	ListPipelineStages(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.PipelineStage], error)
	PipelineStage(ctx context.Context, id string) (model.PipelineStage, error)
	PipelineStageByName(ctx context.Context, projectId string, name string) (model.PipelineStage, error)
	PipelineStagesByIds(ctx context.Context, projectId string, ids []string) ([]model.PipelineStage, error)
	CreatePipelineStage(ctx context.Context, stage model.PipelineStage) error
	UpdatePipelineStage(ctx context.Context, stage model.PipelineStage) error
	DeletePipelineStage(ctx context.Context, id string) error
	PipelineStageReferencedByTemplates(ctx context.Context, projectId string, stageId string) (bool, error)
	CreatePipelineTemplate(ctx context.Context, template model.PipelineTemplate) error
	UpdatePipelineTemplate(ctx context.Context, template model.PipelineTemplate) error
	UpdatePipelineTemplateWithStages(ctx context.Context, template model.PipelineTemplate, stages []model.PipelineTemplateStage) error
	DuplicatePipelineTemplate(ctx context.Context, template model.PipelineTemplate, stages []model.PipelineTemplateStage) error
	PipelineTemplateReferencedByWebhooks(ctx context.Context, templateId string) (bool, error)
	DeletePipelineTemplate(ctx context.Context, id string) error
	LatestPipelineSnapshot(ctx context.Context, templateId string) (model.PipelineSnapshot, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error
}

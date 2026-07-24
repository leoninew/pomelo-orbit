package pipelinesvc

import (
	"context"
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type Service struct {
	project  repository.ProjectReader
	pipeline repository.PipelineStore
	logger   *slog.Logger
	store    stores
}

type stores struct {
	project  repository.ProjectReader
	pipeline repository.PipelineStore
}

func New(project repository.ProjectReader, pipeline repository.PipelineStore, logger *slog.Logger) Service {
	s := stores{project: project, pipeline: pipeline}
	return Service{project: project, pipeline: pipeline, logger: logger, store: s}
}

func (s stores) Project(ctx context.Context, id string) (model.Project, error) {
	return s.project.Project(ctx, id)
}
func (s stores) IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error) {
	return s.project.IsProjectMember(ctx, projectId, userId)
}
func (s stores) PipelineTemplate(ctx context.Context, id string) (model.PipelineTemplate, error) {
	return s.pipeline.PipelineTemplate(ctx, id)
}
func (s stores) ListPipelineTemplates(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.PipelineTemplate], error) {
	return s.pipeline.ListPipelineTemplates(ctx, projectId, page, perPage, search)
}
func (s stores) PipelineTemplateByName(ctx context.Context, projectId string, name string) (model.PipelineTemplate, error) {
	return s.pipeline.PipelineTemplateByName(ctx, projectId, name)
}
func (s stores) PipelineTemplateStages(ctx context.Context, templateId string) ([]model.PipelineTemplateStage, error) {
	return s.pipeline.PipelineTemplateStages(ctx, templateId)
}
func (s stores) ListPipelineStages(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.PipelineStage], error) {
	return s.pipeline.ListPipelineStages(ctx, projectId, page, perPage, search)
}
func (s stores) PipelineStage(ctx context.Context, id string) (model.PipelineStage, error) {
	return s.pipeline.PipelineStage(ctx, id)
}
func (s stores) PipelineStageByName(ctx context.Context, projectId string, name string) (model.PipelineStage, error) {
	return s.pipeline.PipelineStageByName(ctx, projectId, name)
}
func (s stores) PipelineStagesByIds(ctx context.Context, projectId string, ids []string) ([]model.PipelineStage, error) {
	return s.pipeline.PipelineStagesByIds(ctx, projectId, ids)
}
func (s stores) CreatePipelineStage(ctx context.Context, stage model.PipelineStage) error {
	return s.pipeline.CreatePipelineStage(ctx, stage)
}
func (s stores) UpdatePipelineStage(ctx context.Context, stage model.PipelineStage) error {
	return s.pipeline.UpdatePipelineStage(ctx, stage)
}
func (s stores) DeletePipelineStage(ctx context.Context, id string) error {
	return s.pipeline.DeletePipelineStage(ctx, id)
}
func (s stores) PipelineStageReferencedByTemplates(ctx context.Context, projectId string, stageId string) (bool, error) {
	return s.pipeline.PipelineStageReferencedByTemplates(ctx, projectId, stageId)
}
func (s stores) CreatePipelineTemplate(ctx context.Context, template model.PipelineTemplate) error {
	return s.pipeline.CreatePipelineTemplate(ctx, template)
}
func (s stores) UpdatePipelineTemplate(ctx context.Context, template model.PipelineTemplate) error {
	return s.pipeline.UpdatePipelineTemplate(ctx, template)
}
func (s stores) UpdatePipelineTemplateWithStages(ctx context.Context, template model.PipelineTemplate, stages []model.PipelineTemplateStage) error {
	return s.pipeline.UpdatePipelineTemplateWithStages(ctx, template, stages)
}
func (s stores) DuplicatePipelineTemplate(ctx context.Context, template model.PipelineTemplate, stages []model.PipelineTemplateStage) error {
	return s.pipeline.DuplicatePipelineTemplate(ctx, template, stages)
}
func (s stores) PipelineTemplateReferencedByWebhooks(ctx context.Context, templateId string) (bool, error) {
	return s.pipeline.PipelineTemplateReferencedByWebhooks(ctx, templateId)
}
func (s stores) DeletePipelineTemplate(ctx context.Context, id string) error {
	return s.pipeline.DeletePipelineTemplate(ctx, id)
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

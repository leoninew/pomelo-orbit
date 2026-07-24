package pipelinerepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	pipelinesqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/pipeline"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.PipelineStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *pipelinesqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DBTX) *pipelinesqlc.Queries {
		return pipelinesqlc.New(dbtx)
	})
}

func (r Repository) ListPipelineStages(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.PipelineStage], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	projectNS := sql.NullString{String: strings.TrimSpace(projectId), Valid: true}
	q := r.q(ctx)
	total, err := q.CountPipelineStages(ctx, pipelinesqlc.CountPipelineStagesParams{ProjectID: projectNS, Column2: raw, Name: pattern})
	if err != nil {
		return repository.Page[model.PipelineStage]{}, fmt.Errorf("count pipeline stages: %w", err)
	}
	rows, err := q.ListPipelineStages(ctx, pipelinesqlc.ListPipelineStagesParams{
		ProjectID: projectNS, Column2: raw, Name: pattern, Limit: int64(perPage), Offset: int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.PipelineStage]{}, fmt.Errorf("list pipeline stages: %w", err)
	}
	items := make([]model.PipelineStage, 0, len(rows))
	for _, row := range rows {
		items = append(items, pipelineStageFromRow(row.ID, row.ProjectID, row.Name, row.Image, row.Script, row.Artifacts, row.Description, row.Version, row.CreatedAt, row.UpdatedAt))
	}
	return repository.Page[model.PipelineStage]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) PipelineStage(ctx context.Context, id string) (model.PipelineStage, error) {
	row, err := r.q(ctx).PipelineStageByID(ctx, id)
	if err != nil {
		return model.PipelineStage{}, fmt.Errorf("load pipeline stage %s: %w", id, sqlcommon.TranslateError(err))
	}
	return pipelineStageFromRow(row.ID, row.ProjectID, row.Name, row.Image, row.Script, row.Artifacts, row.Description, row.Version, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) PipelineStageByName(ctx context.Context, projectId string, name string) (model.PipelineStage, error) {
	row, err := r.q(ctx).PipelineStageByName(ctx, pipelinesqlc.PipelineStageByNameParams{
		ProjectID: sql.NullString{String: projectId, Valid: true}, Name: name,
	})
	if err != nil {
		return model.PipelineStage{}, fmt.Errorf("load pipeline stage by name %s: %w", name, sqlcommon.TranslateError(err))
	}
	return pipelineStageFromRow(row.ID, row.ProjectID, row.Name, row.Image, row.Script, row.Artifacts, row.Description, row.Version, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) PipelineStagesByIds(ctx context.Context, projectId string, ids []string) ([]model.PipelineStage, error) {
	if len(ids) == 0 {
		return []model.PipelineStage{}, nil
	}
	rows, err := r.q(ctx).PipelineStagesByIds(ctx, pipelinesqlc.PipelineStagesByIdsParams{
		ProjectID: sql.NullString{String: projectId, Valid: true}, Ids: ids,
	})
	if err != nil {
		return nil, fmt.Errorf("load pipeline stages by ids: %w", err)
	}
	items := make([]model.PipelineStage, 0, len(rows))
	for _, row := range rows {
		items = append(items, pipelineStageFromRow(row.ID, row.ProjectID, row.Name, row.Image, row.Script, row.Artifacts, row.Description, row.Version, row.CreatedAt, row.UpdatedAt))
	}
	return items, nil
}

func (r Repository) CreatePipelineStage(ctx context.Context, stage model.PipelineStage) error {
	now := time.Now().UTC()
	err := r.q(ctx).CreatePipelineStage(ctx, pipelinesqlc.CreatePipelineStageParams{
		ID: stage.Id, ProjectID: dbmodel.NullString(stage.ProjectId), Name: stage.Name, Image: stage.Image,
		Script: stage.Script, Artifacts: dbmodel.NullString(stage.Artifacts), Description: stage.Description,
		Version: int64(stage.Version), CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("create pipeline stage %s: %w", stage.Name, err)
	}
	return nil
}

func (r Repository) UpdatePipelineStage(ctx context.Context, stage model.PipelineStage) error {
	err := r.q(ctx).UpdatePipelineStage(ctx, pipelinesqlc.UpdatePipelineStageParams{
		Name: stage.Name, Image: stage.Image, Script: stage.Script, Artifacts: dbmodel.NullString(stage.Artifacts),
		Description: stage.Description, Version: int64(stage.Version), UpdatedAt: time.Now().UTC(), ID: stage.Id,
	})
	if err != nil {
		return fmt.Errorf("update pipeline stage %s: %w", stage.Id, err)
	}
	return nil
}

func (r Repository) DeletePipelineStage(ctx context.Context, id string) error {
	if err := r.q(ctx).DeletePipelineStage(ctx, id); err != nil {
		return fmt.Errorf("delete pipeline stage %s: %w", id, err)
	}
	return nil
}

func (r Repository) PipelineStageReferencedByTemplates(ctx context.Context, projectId string, stageId string) (bool, error) {
	count, err := r.q(ctx).PipelineStageReferencedByTemplates(ctx, pipelinesqlc.PipelineStageReferencedByTemplatesParams{
		ProjectID: sql.NullString{String: projectId, Valid: true}, StageID: stageId,
	})
	if err != nil {
		return false, fmt.Errorf("count pipeline stage template references %s: %w", stageId, err)
	}
	return count > 0, nil
}

func (r Repository) ListPipelineTemplates(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.PipelineTemplate], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	projectNS := sql.NullString{String: strings.TrimSpace(projectId), Valid: true}
	q := r.q(ctx)
	total, err := q.CountPipelineTemplates(ctx, pipelinesqlc.CountPipelineTemplatesParams{ProjectID: projectNS, Column2: raw, Name: pattern})
	if err != nil {
		return repository.Page[model.PipelineTemplate]{}, fmt.Errorf("count pipeline templates: %w", err)
	}
	rows, err := q.ListPipelineTemplates(ctx, pipelinesqlc.ListPipelineTemplatesParams{
		ProjectID: projectNS, Column2: raw, Name: pattern, Limit: int64(perPage), Offset: int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.PipelineTemplate]{}, fmt.Errorf("list pipeline templates: %w", err)
	}
	items := make([]model.PipelineTemplate, 0, len(rows))
	for _, row := range rows {
		items = append(items, templateFrom(row.ID, row.ProjectID, row.Name, row.Description, row.VariableDeclarations, row.Version, row.CreatedAt, row.UpdatedAt))
	}
	return repository.Page[model.PipelineTemplate]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) PipelineTemplate(ctx context.Context, id string) (model.PipelineTemplate, error) {
	row, err := r.q(ctx).PipelineTemplateByID(ctx, id)
	if err != nil {
		return model.PipelineTemplate{}, fmt.Errorf("load pipeline template %s: %w", id, sqlcommon.TranslateError(err))
	}
	return templateFrom(row.ID, row.ProjectID, row.Name, row.Description, row.VariableDeclarations, row.Version, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) PipelineTemplateByName(ctx context.Context, projectId string, name string) (model.PipelineTemplate, error) {
	row, err := r.q(ctx).PipelineTemplateByName(ctx, pipelinesqlc.PipelineTemplateByNameParams{
		ProjectID: sql.NullString{String: projectId, Valid: true}, Name: name,
	})
	if err != nil {
		return model.PipelineTemplate{}, fmt.Errorf("load pipeline template by name %s: %w", name, sqlcommon.TranslateError(err))
	}
	return templateFrom(row.ID, row.ProjectID, row.Name, row.Description, row.VariableDeclarations, row.Version, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) PipelineTemplateStages(ctx context.Context, templateId string) ([]model.PipelineTemplateStage, error) {
	rows, err := r.q(ctx).PipelineTemplateStages(ctx, templateId)
	if err != nil {
		return nil, fmt.Errorf("load pipeline template stages %s: %w", templateId, err)
	}
	items := make([]model.PipelineTemplateStage, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.PipelineTemplateStage{
			Id: row.ID, TemplateId: row.TemplateID, StageId: row.StageID, StageName: row.StageName,
			StageVersion: int(row.StageVersion), DependsOn: row.DependsOn, SortOrder: int(row.SortOrder),
		})
	}
	return items, nil
}

func (r Repository) CreatePipelineTemplate(ctx context.Context, template model.PipelineTemplate) error {
	now := time.Now().UTC()
	err := r.q(ctx).CreatePipelineTemplate(ctx, pipelinesqlc.CreatePipelineTemplateParams{
		ID: template.Id, ProjectID: dbmodel.NullString(template.ProjectId), Name: template.Name,
		Description: template.Description, VariableDeclarations: template.VariableDeclarations,
		Version: int64(template.Version), CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("create pipeline template %s: %w", template.Name, err)
	}
	return nil
}

func (r Repository) UpdatePipelineTemplate(ctx context.Context, template model.PipelineTemplate) error {
	err := r.q(ctx).UpdatePipelineTemplate(ctx, pipelinesqlc.UpdatePipelineTemplateParams{
		Name: template.Name, Description: template.Description, VariableDeclarations: template.VariableDeclarations,
		Version: int64(template.Version), UpdatedAt: time.Now().UTC(), ID: template.Id,
	})
	if err != nil {
		return fmt.Errorf("update pipeline template %s: %w", template.Id, err)
	}
	return nil
}

func (r Repository) UpdatePipelineTemplateWithStages(ctx context.Context, template model.PipelineTemplate, stages []model.PipelineTemplateStage) error {
	q := r.q(ctx)
	if err := r.UpdatePipelineTemplate(ctx, template); err != nil {
		return err
	}
	if err := q.DeletePipelineTemplateStages(ctx, template.Id); err != nil {
		return fmt.Errorf("delete pipeline template stages %s: %w", template.Id, err)
	}
	for _, stage := range stages {
		if err := q.InsertPipelineTemplateStage(ctx, pipelinesqlc.InsertPipelineTemplateStageParams{
			ID: stage.Id, TemplateID: stage.TemplateId, StageID: stage.StageId, StageName: stage.StageName,
			StageVersion: int64(stage.StageVersion), DependsOn: stage.DependsOn, SortOrder: int64(stage.SortOrder),
		}); err != nil {
			return fmt.Errorf("insert pipeline template stage %s: %w", stage.StageId, err)
		}
	}
	return nil
}

func (r Repository) DuplicatePipelineTemplate(ctx context.Context, template model.PipelineTemplate, stages []model.PipelineTemplateStage) error {
	if err := r.CreatePipelineTemplate(ctx, template); err != nil {
		return err
	}
	q := r.q(ctx)
	for _, stage := range stages {
		if err := q.InsertPipelineTemplateStage(ctx, pipelinesqlc.InsertPipelineTemplateStageParams{
			ID: stage.Id, TemplateID: stage.TemplateId, StageID: stage.StageId, StageName: stage.StageName,
			StageVersion: int64(stage.StageVersion), DependsOn: stage.DependsOn, SortOrder: int64(stage.SortOrder),
		}); err != nil {
			return fmt.Errorf("duplicate pipeline template stage %s: %w", stage.StageId, err)
		}
	}
	return nil
}

func (r Repository) PipelineTemplateReferencedByWebhooks(ctx context.Context, templateId string) (bool, error) {
	count, err := r.q(ctx).PipelineTemplateReferencedByWebhooks(ctx, templateId)
	if err != nil {
		return false, fmt.Errorf("count pipeline template webhook references %s: %w", templateId, err)
	}
	return count > 0, nil
}

func (r Repository) DeletePipelineTemplate(ctx context.Context, id string) error {
	if err := r.q(ctx).DeletePipelineTemplate(ctx, id); err != nil {
		return fmt.Errorf("delete pipeline template %s: %w", id, err)
	}
	return nil
}

func (r Repository) PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error) {
	row, err := r.q(ctx).PipelineSnapshotByID(ctx, id)
	if err != nil {
		return model.PipelineSnapshot{}, fmt.Errorf("load pipeline snapshot %s: %w", id, sqlcommon.TranslateError(err))
	}
	return snapshotFrom(row.ID, row.ProjectID, row.TemplateID, row.Version, row.StagesSnapshot, row.VariablesSnapshot, row.CreatedAt), nil
}

func (r Repository) LatestPipelineSnapshot(ctx context.Context, templateId string) (model.PipelineSnapshot, error) {
	row, err := r.q(ctx).LatestPipelineSnapshot(ctx, templateId)
	if err != nil {
		return model.PipelineSnapshot{}, fmt.Errorf("load latest pipeline snapshot %s: %w", templateId, sqlcommon.TranslateError(err))
	}
	return snapshotFrom(row.ID, row.ProjectID, row.TemplateID, row.Version, row.StagesSnapshot, row.VariablesSnapshot, row.CreatedAt), nil
}

func (r Repository) CreatePipelineSnapshot(ctx context.Context, snapshot model.PipelineSnapshot) error {
	err := r.q(ctx).CreatePipelineSnapshot(ctx, pipelinesqlc.CreatePipelineSnapshotParams{
		ID: snapshot.Id, ProjectID: dbmodel.NullString(snapshot.ProjectId), TemplateID: snapshot.TemplateId,
		Version: int64(snapshot.Version), StagesSnapshot: snapshot.StagesSnapshot, VariablesSnapshot: snapshot.VariablesSnapshot,
		CreatedAt: time.Now().UTC(),
	})
	if err != nil {
		return fmt.Errorf("create pipeline snapshot %s: %w", snapshot.Id, err)
	}
	return nil
}

func pipelineStageFromRow(id string, projectID sql.NullString, name, image, script string, artifacts sql.NullString, description string, version int64, createdAt, updatedAt time.Time) model.PipelineStage {
	return model.PipelineStage{
		Id: id, ProjectId: dbmodel.StringPtr(projectID), Name: name, Image: image, Script: script,
		Artifacts: dbmodel.StringPtr(artifacts), Description: description, Version: int(version),
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func templateFrom(id string, projectID sql.NullString, name, description, vars string, version int64, createdAt, updatedAt time.Time) model.PipelineTemplate {
	return model.PipelineTemplate{
		Id: id, ProjectId: dbmodel.StringPtr(projectID), Name: name, Description: description,
		VariableDeclarations: vars, Version: int(version), CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

func snapshotFrom(id string, projectID sql.NullString, templateID string, version int64, stages, vars string, createdAt time.Time) model.PipelineSnapshot {
	return model.PipelineSnapshot{
		Id: id, ProjectId: dbmodel.StringPtr(projectID), TemplateId: templateID, Version: int(version),
		StagesSnapshot: stages, VariablesSnapshot: vars, CreatedAt: createdAt,
	}
}

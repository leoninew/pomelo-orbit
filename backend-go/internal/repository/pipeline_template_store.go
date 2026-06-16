package repository

import (
	"context"
	"fmt"

	"backend/internal/db"

	"github.com/jmoiron/sqlx"
)

func (s Store) ListPipelineTemplates(ctx context.Context, projectId string, page int, perPage int, search string) (Page[PipelineTemplate], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := projectSearchWhere(&projectId, search, []string{"name"})
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM pipeline_template`+where, args...); err != nil {
		return Page[PipelineTemplate]{}, fmt.Errorf("count pipeline templates: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []PipelineTemplate
	err := s.db.SelectContext(ctx, &items, `SELECT id, project_id, name, description, variable_declarations, version, created_at, updated_at
		FROM pipeline_template`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return Page[PipelineTemplate]{}, fmt.Errorf("list pipeline templates: %w", err)
	}
	return Page[PipelineTemplate]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) PipelineTemplateByName(ctx context.Context, projectId string, name string) (PipelineTemplate, error) {
	var template PipelineTemplate
	err := s.db.GetContext(ctx, &template, `SELECT id, project_id, name, description, variable_declarations, version, created_at, updated_at FROM pipeline_template WHERE project_id = ? AND name = ?`, projectId, name)
	if err != nil {
		return PipelineTemplate{}, fmt.Errorf("load pipeline template by name %s: %w", name, err)
	}
	return template, nil
}

func (s Store) PipelineTemplateStages(ctx context.Context, templateId string) ([]PipelineTemplateStage, error) {
	var stages []PipelineTemplateStage
	err := s.db.SelectContext(ctx, &stages, `SELECT id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order FROM pipeline_template_stage WHERE template_id = ? ORDER BY sort_order, id`, templateId)
	if err != nil {
		return nil, fmt.Errorf("load pipeline template stages %s: %w", templateId, err)
	}
	return stages, nil
}

func (s Store) BuildStagesByIds(ctx context.Context, projectId string, ids []string) ([]BuildStage, error) {
	if len(ids) == 0 {
		return []BuildStage{}, nil
	}
	query, args, err := sqlx.In(`SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at FROM build_stage WHERE project_id = ? AND id IN (?)`, projectId, ids)
	if err != nil {
		return nil, fmt.Errorf("build stage ids query: %w", err)
	}
	query = s.db.Rebind(query)
	var stages []BuildStage
	if err := s.db.SelectContext(ctx, &stages, query, args...); err != nil {
		return nil, fmt.Errorf("load build stages by ids: %w", err)
	}
	return stages, nil
}

func (s Store) CreatePipelineTemplate(ctx context.Context, template PipelineTemplate) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO pipeline_template (id, project_id, name, description, variable_declarations, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), template.Id, template.ProjectId, template.Name, template.Description, template.VariableDeclarations, template.Version)
	if err != nil {
		return fmt.Errorf("create pipeline template %s: %w", template.Name, err)
	}
	return nil
}

func (s Store) UpdatePipelineTemplate(ctx context.Context, template PipelineTemplate) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE pipeline_template SET name = ?, description = ?, variable_declarations = ?, version = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), template.Name, template.Description, template.VariableDeclarations, template.Version, template.Id)
	if err != nil {
		return fmt.Errorf("update pipeline template %s: %w", template.Id, err)
	}
	return nil
}

func (s Store) UpdatePipelineTemplateWithStages(ctx context.Context, template PipelineTemplate, stages []PipelineTemplateStage) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin update pipeline template %s: %w", template.Id, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`UPDATE pipeline_template SET name = ?, description = ?, variable_declarations = ?, version = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), template.Name, template.Description, template.VariableDeclarations, template.Version, template.Id); err != nil {
		return fmt.Errorf("update pipeline template %s: %w", template.Id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM pipeline_template_stage WHERE template_id = ?`, template.Id); err != nil {
		return fmt.Errorf("delete pipeline template stages %s: %w", template.Id, err)
	}
	for _, stage := range stages {
		if _, err := tx.ExecContext(ctx, `INSERT INTO pipeline_template_stage (id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)`, stage.Id, stage.TemplateId, stage.StageId, stage.StageName, stage.StageVersion, stage.DependsOn, stage.SortOrder); err != nil {
			return fmt.Errorf("insert pipeline template stage %s: %w", stage.StageId, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit update pipeline template %s: %w", template.Id, err)
	}
	committed = true
	return nil
}

func (s Store) DuplicatePipelineTemplate(ctx context.Context, template PipelineTemplate, stages []PipelineTemplateStage) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin duplicate pipeline template %s: %w", template.Id, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO pipeline_template (id, project_id, name, description, variable_declarations, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), template.Id, template.ProjectId, template.Name, template.Description, template.VariableDeclarations, template.Version); err != nil {
		return fmt.Errorf("duplicate pipeline template %s: %w", template.Name, err)
	}
	for _, stage := range stages {
		if _, err := tx.ExecContext(ctx, `INSERT INTO pipeline_template_stage (id, template_id, stage_id, stage_name, stage_version, depends_on, sort_order) VALUES (?, ?, ?, ?, ?, ?, ?)`, stage.Id, stage.TemplateId, stage.StageId, stage.StageName, stage.StageVersion, stage.DependsOn, stage.SortOrder); err != nil {
			return fmt.Errorf("duplicate pipeline template stage %s: %w", stage.StageId, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit duplicate pipeline template %s: %w", template.Id, err)
	}
	committed = true
	return nil
}

func (s Store) PipelineTemplateReferencedByWebhooks(ctx context.Context, templateId string) (bool, error) {
	var count int
	err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM repository_webhook WHERE template_id = ?`, templateId)
	if err != nil {
		return false, fmt.Errorf("count pipeline template webhook references %s: %w", templateId, err)
	}
	return count > 0, nil
}

func (s Store) DeletePipelineTemplate(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM pipeline_template WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete pipeline template %s: %w", id, err)
	}
	return nil
}

package orbit

import (
	"context"
	"fmt"

	"backend/internal/db"
)

func (s Store) ListBuildStages(ctx context.Context, projectId string, page int, perPage int, search string) (Page[BuildStage], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := projectSearchWhere(&projectId, search, []string{"name"})
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM build_stage`+where, args...); err != nil {
		return Page[BuildStage]{}, fmt.Errorf("count build stages: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []BuildStage
	err := s.db.SelectContext(ctx, &items, `SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at
		FROM build_stage`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return Page[BuildStage]{}, fmt.Errorf("list build stages: %w", err)
	}
	return Page[BuildStage]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) BuildStage(ctx context.Context, id string) (BuildStage, error) {
	var stage BuildStage
	err := s.db.GetContext(ctx, &stage, `SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at FROM build_stage WHERE id = ?`, id)
	if err != nil {
		return BuildStage{}, fmt.Errorf("load build stage %s: %w", id, err)
	}
	return stage, nil
}

func (s Store) BuildStageByName(ctx context.Context, projectId string, name string) (BuildStage, error) {
	var stage BuildStage
	err := s.db.GetContext(ctx, &stage, `SELECT id, project_id, name, image, script, artifacts, description, version, created_at, updated_at FROM build_stage WHERE project_id = ? AND name = ?`, projectId, name)
	if err != nil {
		return BuildStage{}, fmt.Errorf("load build stage by name %s: %w", name, err)
	}
	return stage, nil
}

func (s Store) CreateBuildStage(ctx context.Context, stage BuildStage) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO build_stage (id, project_id, name, image, script, artifacts, description, version, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), stage.Id, stage.ProjectId, stage.Name, stage.Image, stage.Script, stage.Artifacts, stage.Description, stage.Version)
	if err != nil {
		return fmt.Errorf("create build stage %s: %w", stage.Name, err)
	}
	return nil
}

func (s Store) UpdateBuildStage(ctx context.Context, stage BuildStage) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE build_stage SET name = ?, image = ?, script = ?, artifacts = ?, description = ?, version = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), stage.Name, stage.Image, stage.Script, stage.Artifacts, stage.Description, stage.Version, stage.Id)
	if err != nil {
		return fmt.Errorf("update build stage %s: %w", stage.Id, err)
	}
	return nil
}

func (s Store) DeleteBuildStage(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM build_stage WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete build stage %s: %w", id, err)
	}
	return nil
}

func (s Store) BuildStageReferencedByTemplates(ctx context.Context, projectId string, stageId string) (bool, error) {
	var count int
	err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM pipeline_template_stage JOIN pipeline_template ON pipeline_template.id = pipeline_template_stage.template_id WHERE pipeline_template.project_id = ? AND pipeline_template_stage.stage_id = ?`, projectId, stageId)
	if err != nil {
		return false, fmt.Errorf("count build stage template references %s: %w", stageId, err)
	}
	return count > 0, nil
}

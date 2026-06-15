package orbit

import (
	"context"
	"fmt"

	"backend/internal/db"
)

func (s Store) ApplicationByName(ctx context.Context, name string) (Application, error) {
	var app Application
	err := s.db.GetContext(ctx, &app, `SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at FROM application WHERE name = ?`, name)
	if err != nil {
		return Application{}, fmt.Errorf("load application by name %s: %w", name, err)
	}
	return app, nil
}

func (s Store) CreateApplication(ctx context.Context, app Application) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application (id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), app.Id, app.ProjectId, app.Name, app.Code, app.ImagePullPolicy, app.Status, app.RouteManaged)
	if err != nil {
		return fmt.Errorf("create application %s: %w", app.Code, err)
	}
	return nil
}

func (s Store) UpdateApplication(ctx context.Context, app Application) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application SET name = ?, code = ?, image_pull_policy = ?, route_managed = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)),
		app.Name, app.Code, app.ImagePullPolicy, app.RouteManaged, app.Id)
	if err != nil {
		return fmt.Errorf("update application %s: %w", app.Id, err)
	}
	return nil
}

func (s Store) DeleteApplication(ctx context.Context, id string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin delete application %s: %w", id, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM application_config_file WHERE application_id = ?`, id); err != nil {
		return fmt.Errorf("delete application config files %s: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM application_route WHERE application_id = ?`, id); err != nil {
		return fmt.Errorf("delete application routes %s: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM application_service WHERE application_id = ?`, id); err != nil {
		return fmt.Errorf("delete application service configs %s: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM application WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete application %s: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete application %s: %w", id, err)
	}
	return nil
}

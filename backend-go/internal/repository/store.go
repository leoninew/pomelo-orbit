package repository

import (
	"context"
	"fmt"

	"github.com/jmoiron/sqlx"

	"backend/internal/db"
)

type Store struct {
	db     *sqlx.DB
	driver string
}

func NewStore(db *sqlx.DB, driver string) Store {
	return Store{db: db, driver: driver}
}

func (s Store) DB() *sqlx.DB {
	return s.db
}

func (s Store) Driver() string {
	return s.driver
}

func (s Store) Application(ctx context.Context, id string) (Application, error) {
	var app Application
	err := s.db.GetContext(ctx, &app, `SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at FROM application WHERE id = ?`, id)
	if err != nil {
		return Application{}, fmt.Errorf("load application %s: %w", id, err)
	}
	return app, nil
}

func (s Store) Deployment(ctx context.Context, id string) (Deployment, error) {
	var deployment Deployment
	err := s.db.GetContext(ctx, &deployment, `SELECT id, project_id, application_id, application_name, operation_type, trigger_type, status,
		started_at, finished_at, duration_ms, log_text, error_message, is_rollback, rollback_from_deployment_id FROM deployment WHERE id = ?`, id)
	if err != nil {
		return Deployment{}, fmt.Errorf("load deployment %s: %w", id, err)
	}
	return deployment, nil
}

func (s Store) ConfigFiles(ctx context.Context, applicationId string) ([]ApplicationConfigFile, error) {
	var files []ApplicationConfigFile
	err := s.db.SelectContext(ctx, &files, `SELECT id, application_id, path, content, created_at, updated_at FROM application_config_file WHERE application_id = ? ORDER BY path`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("load config files for application %s: %w", applicationId, err)
	}
	return files, nil
}

func (s Store) ServiceConfigs(ctx context.Context, applicationId string) ([]ApplicationServiceConfig, error) {
	var configs []ApplicationServiceConfig
	err := s.db.SelectContext(ctx, &configs, `SELECT id, application_id, service_name, image, environment, volumes, created_at, updated_at FROM application_service WHERE application_id = ?`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("load service configs for application %s: %w", applicationId, err)
	}
	return configs, nil
}

func (s Store) Routes(ctx context.Context, applicationId string) ([]ApplicationRoute, error) {
	var routes []ApplicationRoute
	err := s.db.SelectContext(ctx, &routes, `SELECT id, application_id, service_name, domain, port, created_at, updated_at FROM application_route WHERE application_id = ?`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("load routes for application %s: %w", applicationId, err)
	}
	return routes, nil
}

func (s Store) MarkApplicationStatus(ctx context.Context, id string, status string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application SET status = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), status, id)
	if err != nil {
		return fmt.Errorf("mark application %s status: %w", id, err)
	}
	return nil
}

func (s Store) MarkDeploymentRunning(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE deployment SET status = ?, started_at = %s, error_message = NULL WHERE id = ?`, db.NowExpr(s.driver)), WorkStatusRunning, id)
	if err != nil {
		return fmt.Errorf("mark deployment running %s: %w", id, err)
	}
	return nil
}

func (s Store) CompleteDeployment(ctx context.Context, id string, status string, message string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE deployment SET status = ?, finished_at = %s, duration_ms = %s, error_message = NULLIF(?, '') WHERE id = ?`, db.NowExpr(s.driver), db.DurationMillisExpr(s.driver, "started_at")), status, message, id)
	if err != nil {
		return fmt.Errorf("complete deployment %s: %w", id, err)
	}
	return nil
}

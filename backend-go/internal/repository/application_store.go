package repository

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

func (s Store) ApplicationByCode(ctx context.Context, code string) (Application, error) {
	var app Application
	err := s.db.GetContext(ctx, &app, `SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at FROM application WHERE code = ?`, code)
	if err != nil {
		return Application{}, fmt.Errorf("load application by code %s: %w", code, err)
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

func (s Store) CreateApplicationBundle(ctx context.Context, app Application, files []ApplicationConfigFile, serviceConfigs []ApplicationServiceConfig, routes []ApplicationRoute) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create application bundle %s: %w", app.Code, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application (id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), app.Id, app.ProjectId, app.Name, app.Code, app.ImagePullPolicy, app.Status, app.RouteManaged); err != nil {
		return fmt.Errorf("create application %s: %w", app.Code, err)
	}
	for _, file := range files {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
			VALUES (?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), file.Id, file.ApplicationId, file.Path, file.Content); err != nil {
			return fmt.Errorf("create application config file %s: %w", file.Path, err)
		}
	}
	for _, config := range serviceConfigs {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_service (id, application_id, service_name, image, environment, volumes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), config.Id, config.ApplicationId, config.ServiceName, config.Image, config.Environment, config.Volumes); err != nil {
			return fmt.Errorf("create application service config %s: %w", config.ServiceName, err)
		}
	}
	for _, route := range routes {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_route (id, application_id, service_name, domain, port, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), route.Id, route.ApplicationId, route.ServiceName, route.Domain, route.Port); err != nil {
			return fmt.Errorf("create application route %s: %w", route.Domain, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create application bundle %s: %w", app.Code, err)
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

func (s Store) ConfigFile(ctx context.Context, id string) (ApplicationConfigFile, error) {
	var file ApplicationConfigFile
	err := s.db.GetContext(ctx, &file, `SELECT id, application_id, path, content, created_at, updated_at FROM application_config_file WHERE id = ?`, id)
	if err != nil {
		return ApplicationConfigFile{}, fmt.Errorf("load config file %s: %w", id, err)
	}
	return file, nil
}

func (s Store) CreateConfigFile(ctx context.Context, file ApplicationConfigFile) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
		VALUES (?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), file.Id, file.ApplicationId, file.Path, file.Content)
	if err != nil {
		return fmt.Errorf("create config file %s: %w", file.Path, err)
	}
	return nil
}

func (s Store) UpdateConfigFile(ctx context.Context, file ApplicationConfigFile) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application_config_file SET path = ?, content = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), file.Path, file.Content, file.Id)
	if err != nil {
		return fmt.Errorf("update config file %s: %w", file.Id, err)
	}
	return nil
}

func (s Store) DeleteConfigFile(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM application_config_file WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete config file %s: %w", id, err)
	}
	return nil
}

func (s Store) ApplicationRoute(ctx context.Context, id string) (ApplicationRoute, error) {
	var route ApplicationRoute
	err := s.db.GetContext(ctx, &route, `SELECT id, application_id, service_name, domain, port, created_at, updated_at FROM application_route WHERE id = ?`, id)
	if err != nil {
		return ApplicationRoute{}, fmt.Errorf("load application route %s: %w", id, err)
	}
	return route, nil
}

func (s Store) CreateApplicationRoute(ctx context.Context, route ApplicationRoute) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_route (id, application_id, service_name, domain, port, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), route.Id, route.ApplicationId, route.ServiceName, route.Domain, route.Port)
	if err != nil {
		return fmt.Errorf("create application route %s: %w", route.Domain, err)
	}
	return nil
}

func (s Store) UpdateApplicationRoute(ctx context.Context, route ApplicationRoute) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application_route SET service_name = ?, domain = ?, port = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), route.ServiceName, route.Domain, route.Port, route.Id)
	if err != nil {
		return fmt.Errorf("update application route %s: %w", route.Id, err)
	}
	return nil
}

func (s Store) DeleteApplicationRoute(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM application_route WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete application route %s: %w", id, err)
	}
	return nil
}

func (s Store) ApplicationServiceConfig(ctx context.Context, applicationId string, serviceName string) (ApplicationServiceConfig, error) {
	var config ApplicationServiceConfig
	err := s.db.GetContext(ctx, &config, `SELECT id, application_id, service_name, image, environment, volumes, created_at, updated_at FROM application_service WHERE application_id = ? AND service_name = ?`, applicationId, serviceName)
	if err != nil {
		return ApplicationServiceConfig{}, fmt.Errorf("load application service config %s/%s: %w", applicationId, serviceName, err)
	}
	return config, nil
}

func (s Store) UpsertApplicationServiceConfig(ctx context.Context, config ApplicationServiceConfig) error {
	var count int
	if err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM application_service WHERE id = ?`, config.Id); err != nil {
		return fmt.Errorf("count application service config %s: %w", config.Id, err)
	}
	if count == 0 {
		_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_service (id, application_id, service_name, image, environment, volumes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), config.Id, config.ApplicationId, config.ServiceName, config.Image, config.Environment, config.Volumes)
		if err != nil {
			return fmt.Errorf("create application service config %s: %w", config.ServiceName, err)
		}
		return nil
	}
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application_service SET image = ?, environment = ?, volumes = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), config.Image, config.Environment, config.Volumes, config.Id)
	if err != nil {
		return fmt.Errorf("update application service config %s: %w", config.Id, err)
	}
	return nil
}

func (s Store) CancelDeployment(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE deployment SET status = ?, finished_at = %s, duration_ms = %s, error_message = ? WHERE id = ?`, db.NowExpr(s.driver), db.DurationMillisExpr(s.driver, "started_at")), WorkStatusCanceled, "Cancelled by user", id)
	if err != nil {
		return fmt.Errorf("cancel deployment %s: %w", id, err)
	}
	return nil
}

func (s Store) CreateDeployment(ctx context.Context, deployment Deployment) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO deployment (id, project_id, application_id, application_name, operation_type, trigger_type, status, started_at, is_rollback, rollback_from_deployment_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, %s, ?, ?)`, db.NowExpr(s.driver)), deployment.Id, deployment.ProjectId, deployment.ApplicationId, deployment.ApplicationName, deployment.OperationType, deployment.TriggerType, deployment.Status, deployment.IsRollback, deployment.RollbackFromDeploymentId)
	if err != nil {
		return fmt.Errorf("create deployment %s: %w", deployment.Id, err)
	}
	return nil
}

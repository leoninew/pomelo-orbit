package cd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	"gitee.com/leoninew/PomeloOrbit-go/internal/db"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/status"
)

// Repository provides persistence for CD application, deployment, and route use cases.
type Repository struct {
	db     *sqlx.DB
	driver string
}

func NewRepository(db *sqlx.DB, driver string) Repository {
	return Repository{db: db, driver: driver}
}

func (r Repository) Project(ctx context.Context, id string) (model.Project, error) {
	var project model.Project
	err := r.db.GetContext(ctx, &project, `SELECT id, name, code, is_active, created_at, updated_at FROM project WHERE id = ?`, id)
	if err != nil {
		return model.Project{}, fmt.Errorf("load project %s: %w", id, err)
	}
	return project, nil
}

func (r Repository) IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM project_member WHERE project_id = ? AND user_id = ?`, projectId, userId); err != nil {
		return false, fmt.Errorf("check project member %s/%s: %w", projectId, userId, err)
	}
	return count > 0, nil
}

func (r Repository) ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string) (repository.Page[model.Application], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := projectSearchWhere(projectId, search, []string{"name", "code"})
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM application`+where, args...); err != nil {
		return repository.Page[model.Application]{}, fmt.Errorf("count applications: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.Application
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at
		FROM application`+where+` ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.Application]{}, fmt.Errorf("list applications: %w", err)
	}
	return repository.Page[model.Application]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) Application(ctx context.Context, id string) (model.Application, error) {
	var app model.Application
	err := r.db.GetContext(ctx, &app, `SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at FROM application WHERE id = ?`, id)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application %s: %w", id, err)
	}
	return app, nil
}

func (r Repository) ApplicationByName(ctx context.Context, name string) (model.Application, error) {
	var app model.Application
	err := r.db.GetContext(ctx, &app, `SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at FROM application WHERE name = ?`, name)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by name %s: %w", name, err)
	}
	return app, nil
}

func (r Repository) ApplicationByCode(ctx context.Context, code string) (model.Application, error) {
	var app model.Application
	err := r.db.GetContext(ctx, &app, `SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at FROM application WHERE code = ?`, code)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by code %s: %w", code, err)
	}
	return app, nil
}

func (r Repository) CreateApplication(ctx context.Context, app model.Application) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application (id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), app.Id, app.ProjectId, app.Name, app.Code, app.ImagePullPolicy, app.Status, app.RouteManaged)
	if err != nil {
		return fmt.Errorf("create application %s: %w", app.Code, err)
	}
	return nil
}

func (r Repository) CreateApplicationBundle(ctx context.Context, app model.Application, files []model.ApplicationConfigFile, serviceConfigs []model.ApplicationServiceConfig, routes []model.ApplicationRoute) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create application bundle %s: %w", app.Code, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application (id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), app.Id, app.ProjectId, app.Name, app.Code, app.ImagePullPolicy, app.Status, app.RouteManaged); err != nil {
		return fmt.Errorf("create application %s: %w", app.Code, err)
	}
	for _, file := range files {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
			VALUES (?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), file.Id, file.ApplicationId, file.Path, file.Content); err != nil {
			return fmt.Errorf("create application config file %s: %w", file.Path, err)
		}
	}
	for _, config := range serviceConfigs {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_service (id, application_id, service_name, image, environment, volumes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), config.Id, config.ApplicationId, config.ServiceName, config.Image, config.Environment, config.Volumes); err != nil {
			return fmt.Errorf("create application service config %s: %w", config.ServiceName, err)
		}
	}
	for _, route := range routes {
		if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_route (id, application_id, service_name, domain, port, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), route.Id, route.ApplicationId, route.ServiceName, route.Domain, route.Port); err != nil {
			return fmt.Errorf("create application route %s: %w", route.Domain, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create application bundle %s: %w", app.Code, err)
	}
	return nil
}

func (r Repository) UpdateApplication(ctx context.Context, app model.Application) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application SET name = ?, code = ?, image_pull_policy = ?, route_managed = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)),
		app.Name, app.Code, app.ImagePullPolicy, app.RouteManaged, app.Id)
	if err != nil {
		return fmt.Errorf("update application %s: %w", app.Id, err)
	}
	return nil
}

func (r Repository) DeleteApplication(ctx context.Context, id string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
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

func (r Repository) ConfigFiles(ctx context.Context, applicationId string) ([]model.ApplicationConfigFile, error) {
	var files []model.ApplicationConfigFile
	err := r.db.SelectContext(ctx, &files, `SELECT id, application_id, path, content, created_at, updated_at FROM application_config_file WHERE application_id = ? ORDER BY path`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("load config files for application %s: %w", applicationId, err)
	}
	return files, nil
}

func (r Repository) ConfigFile(ctx context.Context, id string) (model.ApplicationConfigFile, error) {
	var file model.ApplicationConfigFile
	err := r.db.GetContext(ctx, &file, `SELECT id, application_id, path, content, created_at, updated_at FROM application_config_file WHERE id = ?`, id)
	if err != nil {
		return model.ApplicationConfigFile{}, fmt.Errorf("load config file %s: %w", id, err)
	}
	return file, nil
}

func (r Repository) CreateConfigFile(ctx context.Context, file model.ApplicationConfigFile) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_config_file (id, application_id, path, content, created_at, updated_at)
		VALUES (?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), file.Id, file.ApplicationId, file.Path, file.Content)
	if err != nil {
		return fmt.Errorf("create config file %s: %w", file.Path, err)
	}
	return nil
}

func (r Repository) UpdateConfigFile(ctx context.Context, file model.ApplicationConfigFile) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application_config_file SET path = ?, content = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), file.Path, file.Content, file.Id)
	if err != nil {
		return fmt.Errorf("update config file %s: %w", file.Id, err)
	}
	return nil
}

func (r Repository) DeleteConfigFile(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM application_config_file WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete config file %s: %w", id, err)
	}
	return nil
}

func (r Repository) ServiceConfigs(ctx context.Context, applicationId string) ([]model.ApplicationServiceConfig, error) {
	var configs []model.ApplicationServiceConfig
	err := r.db.SelectContext(ctx, &configs, `SELECT id, application_id, service_name, image, environment, volumes, created_at, updated_at FROM application_service WHERE application_id = ?`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("load service configs for application %s: %w", applicationId, err)
	}
	return configs, nil
}

func (r Repository) ApplicationServiceConfig(ctx context.Context, applicationId string, serviceName string) (model.ApplicationServiceConfig, error) {
	var config model.ApplicationServiceConfig
	err := r.db.GetContext(ctx, &config, `SELECT id, application_id, service_name, image, environment, volumes, created_at, updated_at FROM application_service WHERE application_id = ? AND service_name = ?`, applicationId, serviceName)
	if err != nil {
		return model.ApplicationServiceConfig{}, fmt.Errorf("load application service config %s/%s: %w", applicationId, serviceName, err)
	}
	return config, nil
}

func (r Repository) UpsertApplicationServiceConfig(ctx context.Context, config model.ApplicationServiceConfig) error {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM application_service WHERE id = ?`, config.Id); err != nil {
		return fmt.Errorf("count application service config %s: %w", config.Id, err)
	}
	if count == 0 {
		_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_service (id, application_id, service_name, image, environment, volumes, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), config.Id, config.ApplicationId, config.ServiceName, config.Image, config.Environment, config.Volumes)
		if err != nil {
			return fmt.Errorf("create application service config %s: %w", config.ServiceName, err)
		}
		return nil
	}
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application_service SET image = ?, environment = ?, volumes = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), config.Image, config.Environment, config.Volumes, config.Id)
	if err != nil {
		return fmt.Errorf("update application service config %s: %w", config.Id, err)
	}
	return nil
}

func (r Repository) Routes(ctx context.Context, applicationId string) ([]model.ApplicationRoute, error) {
	var routes []model.ApplicationRoute
	err := r.db.SelectContext(ctx, &routes, `SELECT id, application_id, service_name, domain, port, created_at, updated_at FROM application_route WHERE application_id = ?`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("load routes for application %s: %w", applicationId, err)
	}
	return routes, nil
}

func (r Repository) ApplicationRoute(ctx context.Context, id string) (model.ApplicationRoute, error) {
	var route model.ApplicationRoute
	err := r.db.GetContext(ctx, &route, `SELECT id, application_id, service_name, domain, port, created_at, updated_at FROM application_route WHERE id = ?`, id)
	if err != nil {
		return model.ApplicationRoute{}, fmt.Errorf("load application route %s: %w", id, err)
	}
	return route, nil
}

func (r Repository) CreateApplicationRoute(ctx context.Context, route model.ApplicationRoute) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application_route (id, application_id, service_name, domain, port, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), route.Id, route.ApplicationId, route.ServiceName, route.Domain, route.Port)
	if err != nil {
		return fmt.Errorf("create application route %s: %w", route.Domain, err)
	}
	return nil
}

func (r Repository) UpdateApplicationRoute(ctx context.Context, route model.ApplicationRoute) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application_route SET service_name = ?, domain = ?, port = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), route.ServiceName, route.Domain, route.Port, route.Id)
	if err != nil {
		return fmt.Errorf("update application route %s: %w", route.Id, err)
	}
	return nil
}

func (r Repository) DeleteApplicationRoute(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM application_route WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete application route %s: %w", id, err)
	}
	return nil
}

func (r Repository) ListRoutes(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.Route], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	clauses := []string{"project_id = ?"}
	args := []any{strings.TrimSpace(projectId)}
	search = strings.TrimSpace(search)
	if search != "" {
		like := "%" + search + "%"
		clauses = append(clauses, "(name LIKE ? OR domain LIKE ? OR target_url LIKE ?)")
		args = append(args, like, like, like)
	}
	where := " WHERE " + strings.Join(clauses, " AND ")
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM route`+where, args...); err != nil {
		return repository.Page[model.Route]{}, fmt.Errorf("count routes: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.Route
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at
		FROM route`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.Route]{}, fmt.Errorf("list routes: %w", err)
	}
	return repository.Page[model.Route]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) ListAllRoutes(ctx context.Context, projectId string) ([]model.Route, error) {
	var items []model.Route
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at
		FROM route WHERE project_id = ? ORDER BY id DESC`, strings.TrimSpace(projectId))
	if err != nil {
		return nil, fmt.Errorf("list all routes: %w", err)
	}
	return items, nil
}

func (r Repository) Route(ctx context.Context, id string) (model.Route, error) {
	var route model.Route
	err := r.db.GetContext(ctx, &route, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at FROM route WHERE id = ?`, id)
	if err != nil {
		return model.Route{}, fmt.Errorf("load route %s: %w", id, err)
	}
	return route, nil
}

func (r Repository) RouteByDomain(ctx context.Context, domain string) (model.Route, error) {
	var route model.Route
	err := r.db.GetContext(ctx, &route, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at FROM route WHERE domain = ?`, strings.TrimSpace(domain))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return model.Route{}, sql.ErrNoRows
		}
		return model.Route{}, fmt.Errorf("load route by domain %s: %w", domain, err)
	}
	return route, nil
}

func (r Repository) CreateRoute(ctx context.Context, route model.Route) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO route (id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), route.Id, route.ProjectId, route.Name, route.Domain, route.PathPrefix, route.TargetURL, route.Enabled, route.HTTPSEnabled, route.CertPEM, route.CertKey, route.CertType)
	if err != nil {
		return fmt.Errorf("create route %s: %w", route.Name, err)
	}
	return nil
}

func (r Repository) UpdateRoute(ctx context.Context, route model.Route) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE route SET name = ?, domain = ?, path_prefix = ?, target_url = ?, enabled = ?, https_enabled = ?, cert_pem = ?, cert_key = ?, cert_type = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), route.Name, route.Domain, route.PathPrefix, route.TargetURL, route.Enabled, route.HTTPSEnabled, route.CertPEM, route.CertKey, route.CertType, route.Id)
	if err != nil {
		return fmt.Errorf("update route %s: %w", route.Id, err)
	}
	return nil
}

func (r Repository) DeleteRoute(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM route WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete route %s: %w", id, err)
	}
	return nil
}

func (r Repository) CreateDeployment(ctx context.Context, deployment model.Deployment) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO deployment (id, project_id, application_id, application_name, operation_type, trigger_type, command_text, status, started_at, is_rollback, rollback_from_deployment_id)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, %s, ?, ?)`, db.NowExpr(r.driver)), deployment.Id, deployment.ProjectId, deployment.ApplicationId, deployment.ApplicationName, deployment.OperationType, deployment.TriggerType, deployment.CommandText, deployment.Status, deployment.IsRollback, deployment.RollbackFromDeploymentId)
	if err != nil {
		return fmt.Errorf("create deployment %s: %w", deployment.Id, err)
	}
	return nil
}

func (r Repository) ListDeployments(ctx context.Context, projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[model.Deployment], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	where, args := deploymentWhere(projectId, applicationId, status, search, dateFrom, dateTo)
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM deployment`+where, args...); err != nil {
		return repository.Page[model.Deployment]{}, fmt.Errorf("count deployments: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.Deployment
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, application_id, application_name, operation_type, trigger_type, command_text,
		status, started_at, finished_at, duration_ms, log_text, error_message, is_rollback, rollback_from_deployment_id
		FROM deployment`+where+` ORDER BY started_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.Deployment]{}, fmt.Errorf("list deployments: %w", err)
	}
	return repository.Page[model.Deployment]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) Deployment(ctx context.Context, id string) (model.Deployment, error) {
	var deployment model.Deployment
	err := r.db.GetContext(ctx, &deployment, `SELECT id, project_id, application_id, application_name, operation_type, trigger_type, command_text, status,
		started_at, finished_at, duration_ms, log_text, error_message, is_rollback, rollback_from_deployment_id FROM deployment WHERE id = ?`, id)
	if err != nil {
		return model.Deployment{}, fmt.Errorf("load deployment %s: %w", id, err)
	}
	return deployment, nil
}

func (r Repository) CancelDeployment(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE deployment SET status = ?, finished_at = %s, duration_ms = %s, error_message = ? WHERE id = ?`, db.NowExpr(r.driver), db.DurationMillisExpr(r.driver, "started_at")), status.WorkStatusCanceled, "Cancelled by user", id)
	if err != nil {
		return fmt.Errorf("cancel deployment %s: %w", id, err)
	}
	return nil
}

func (r Repository) MarkApplicationStatus(ctx context.Context, id string, status string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application SET status = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), status, id)
	if err != nil {
		return fmt.Errorf("mark application %s status: %w", id, err)
	}
	return nil
}

func (r Repository) MarkDeploymentRunning(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE deployment SET status = ?, started_at = %s, error_message = NULL WHERE id = ?`, db.NowExpr(r.driver)), status.WorkStatusRunning, id)
	if err != nil {
		return fmt.Errorf("mark deployment running %s: %w", id, err)
	}
	return nil
}

func (r Repository) CompleteDeployment(ctx context.Context, id string, status string, message string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE deployment SET status = ?, finished_at = %s, duration_ms = %s, error_message = NULLIF(?, '') WHERE id = ?`, db.NowExpr(r.driver), db.DurationMillisExpr(r.driver, "started_at")), status, message, id)
	if err != nil {
		return fmt.Errorf("complete deployment %s: %w", id, err)
	}
	return nil
}

func projectSearchWhere(projectId *string, search string, searchColumns []string) (string, []any) {
	clauses := []string{}
	args := []any{}
	if projectId != nil && strings.TrimSpace(*projectId) != "" {
		clauses = append(clauses, "project_id = ?")
		args = append(args, strings.TrimSpace(*projectId))
	}
	search = strings.TrimSpace(search)
	if search != "" && len(searchColumns) > 0 {
		parts := make([]string, 0, len(searchColumns))
		like := "%" + search + "%"
		for _, column := range searchColumns {
			parts = append(parts, column+" LIKE ?")
			args = append(args, like)
		}
		clauses = append(clauses, "("+strings.Join(parts, " OR ")+")")
	}
	if len(clauses) == 0 {
		return "", args
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func deploymentWhere(projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time) (string, []any) {
	clauses := []string{"project_id = ?"}
	args := []any{strings.TrimSpace(projectId)}
	if strings.TrimSpace(applicationId) != "" {
		clauses = append(clauses, "application_id = ?")
		args = append(args, strings.TrimSpace(applicationId))
	}
	if strings.TrimSpace(status) != "" {
		clauses = append(clauses, "status = ?")
		args = append(args, strings.TrimSpace(status))
	}
	if strings.TrimSpace(search) != "" {
		clauses = append(clauses, "application_name LIKE ?")
		args = append(args, "%"+strings.TrimSpace(search)+"%")
	}
	if dateFrom != nil {
		clauses = append(clauses, "started_at >= ?")
		args = append(args, *dateFrom)
	}
	if dateTo != nil {
		clauses = append(clauses, "started_at < ?")
		args = append(args, *dateTo)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

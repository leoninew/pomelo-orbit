package cd

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/jmoiron/sqlx"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

// Repository provides persistence for CD application, deployment, and route use cases.
var _ repository.CDStore = Repository{}
var _ repository.DeploymentExecutionStore = Repository{}

type Repository struct {
	db     *sqlx.DB
	driver string
}

func NewRepository(db *sqlx.DB, driver string) Repository {
	return Repository{db: db, driver: driver}
}

const applicationColumns = `id, project_id, name, code, kind, image_pull_policy, created_at, updated_at`
const versionColumns = `id, application_id, label, status, env_json, created_from_version_id, note, created_at, updated_at`
const versionComponentColumns = `id, version_id, name, image, command_json, args_json, env_json, ports_json, mounts_json, networks_json, depends_on_json, healthcheck_json, resources_json, pull_policy, created_at, updated_at`
const versionExposeColumns = `id, version_id, component_name, protocol, container_port, path_prefix, created_at, updated_at`
const environmentColumns = `id, project_id, code, name, description, base_domain, domain_template, default_entrypoint, tcp_entrypoint, tls_mode, created_at, updated_at`
const serviceColumns = `id, application_id, environment_id, instance_key, version_id, last_successful_version_id, status, created_at, updated_at`
const deploymentColumns = `id, project_id, application_id, application_name, version_id, service_id, environment_id, options_json, operation_type, trigger_type, command_text, status, started_at, finished_at, duration_ms, log_text, error_message, is_rollback, rollback_from_deployment_id`

func (r Repository) Project(ctx context.Context, id string) (model.Project, error) {
	var project model.Project
	err := r.db.GetContext(ctx, &project, `SELECT id, name, code, is_active, created_at, updated_at FROM project WHERE id = ?`, id)
	if err != nil {
		return model.Project{}, fmt.Errorf("load project %s: %w", id, sqlcommon.TranslateError(err))
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
	err := r.db.SelectContext(ctx, &items, `SELECT `+applicationColumns+`
		FROM application`+where+` ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.Application]{}, fmt.Errorf("list applications: %w", err)
	}
	return repository.Page[model.Application]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) Application(ctx context.Context, id string) (model.Application, error) {
	var app model.Application
	err := r.db.GetContext(ctx, &app, `SELECT `+applicationColumns+` FROM application WHERE id = ?`, id)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application %s: %w", id, sqlcommon.TranslateError(err))
	}
	return app, nil
}

func (r Repository) ApplicationByName(ctx context.Context, name string) (model.Application, error) {
	var app model.Application
	err := r.db.GetContext(ctx, &app, `SELECT `+applicationColumns+` FROM application WHERE name = ?`, name)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by name %s: %w", name, sqlcommon.TranslateError(err))
	}
	return app, nil
}

func (r Repository) ApplicationByCode(ctx context.Context, code string) (model.Application, error) {
	var app model.Application
	err := r.db.GetContext(ctx, &app, `SELECT `+applicationColumns+` FROM application WHERE code = ?`, code)
	if err != nil {
		return model.Application{}, fmt.Errorf("load application by code %s: %w", code, sqlcommon.TranslateError(err))
	}
	return app, nil
}

func (r Repository) CreateApplication(ctx context.Context, app model.Application) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO application (id, project_id, name, code, kind, image_pull_policy, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)), app.Id, app.ProjectId, app.Name, app.Code, app.Kind, app.ImagePullPolicy)
	if err != nil {
		return fmt.Errorf("create application %s: %w", app.Code, err)
	}
	return nil
}

func (r Repository) UpdateApplication(ctx context.Context, app model.Application) error {
	// kind is immutable after create; UPDATE never writes kind.
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE application SET name = ?, code = ?, image_pull_policy = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)),
		app.Name, app.Code, app.ImagePullPolicy, app.Id)
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
	if _, err := tx.ExecContext(ctx, `DELETE FROM service WHERE application_id = ?`, id); err != nil {
		return fmt.Errorf("delete application service %s: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM version WHERE application_id = ?`, id); err != nil {
		return fmt.Errorf("delete application versions %s: %w", id, err)
	}
	if _, err := tx.ExecContext(ctx, `DELETE FROM application WHERE id = ?`, id); err != nil {
		return fmt.Errorf("delete application %s: %w", id, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit delete application %s: %w", id, err)
	}
	return nil
}

func (r Repository) ListVersions(ctx context.Context, applicationId string) ([]model.Version, error) {
	var items []model.Version
	err := r.db.SelectContext(ctx, &items, `SELECT `+versionColumns+` FROM version WHERE application_id = ? ORDER BY created_at DESC, id`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("list versions for application %s: %w", applicationId, err)
	}
	return items, nil
}

func (r Repository) Version(ctx context.Context, id string) (model.Version, error) {
	var version model.Version
	err := r.db.GetContext(ctx, &version, `SELECT `+versionColumns+` FROM version WHERE id = ?`, id)
	if err != nil {
		return model.Version{}, fmt.Errorf("load version %s: %w", id, sqlcommon.TranslateError(err))
	}
	return version, nil
}

func (r Repository) CreateVersion(ctx context.Context, version model.Version) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO version (id, application_id, label, status, env_json, created_from_version_id, note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)),
		version.Id, version.ApplicationId, version.Label, version.Status, version.EnvJSON, version.CreatedFromVersionId, version.Note)
	if err != nil {
		return fmt.Errorf("create version %s: %w", version.Label, err)
	}
	return nil
}

func (r Repository) UpdateVersion(ctx context.Context, version model.Version) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE version SET label = ?, status = ?, env_json = ?, note = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)),
		version.Label, version.Status, version.EnvJSON, version.Note, version.Id)
	if err != nil {
		return fmt.Errorf("update version %s: %w", version.Id, err)
	}
	return nil
}

func (r Repository) VersionComponentsByVersion(ctx context.Context, versionId string) ([]model.VersionComponent, error) {
	var items []model.VersionComponent
	err := r.db.SelectContext(ctx, &items, `SELECT `+versionComponentColumns+` FROM version_component WHERE version_id = ? ORDER BY name`, versionId)
	if err != nil {
		return nil, fmt.Errorf("list components for version %s: %w", versionId, err)
	}
	return items, nil
}

func (r Repository) VersionExposesByVersion(ctx context.Context, versionId string) ([]model.VersionExpose, error) {
	var items []model.VersionExpose
	err := r.db.SelectContext(ctx, &items, `SELECT `+versionExposeColumns+` FROM version_expose WHERE version_id = ? ORDER BY component_name, protocol, container_port`, versionId)
	if err != nil {
		return nil, fmt.Errorf("list exposes for version %s: %w", versionId, err)
	}
	return items, nil
}

func (r Repository) ReplaceVersionComponents(ctx context.Context, versionId string, components []model.VersionComponent) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace components %s: %w", versionId, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM version_component WHERE version_id = ?`, versionId); err != nil {
		return fmt.Errorf("delete components for version %s: %w", versionId, err)
	}
	for _, component := range components {
		if err := insertVersionComponent(ctx, tx, r.driver, component); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace components %s: %w", versionId, err)
	}
	return nil
}

func (r Repository) ReplaceVersionExposes(ctx context.Context, versionId string, exposes []model.VersionExpose) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin replace exposes %s: %w", versionId, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, `DELETE FROM version_expose WHERE version_id = ?`, versionId); err != nil {
		return fmt.Errorf("delete exposes for version %s: %w", versionId, err)
	}
	for _, expose := range exposes {
		if err := insertVersionExpose(ctx, tx, r.driver, expose); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit replace exposes %s: %w", versionId, err)
	}
	return nil
}

func (r Repository) CreateVersionWithVersionComponentsAndExposes(ctx context.Context, version model.Version, components []model.VersionComponent, exposes []model.VersionExpose) error {
	tx, err := r.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create version with components %s: %w", version.Label, err)
	}
	defer func() { _ = tx.Rollback() }()
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO version (id, application_id, label, status, env_json, created_from_version_id, note, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)),
		version.Id, version.ApplicationId, version.Label, version.Status, version.EnvJSON, version.CreatedFromVersionId, version.Note); err != nil {
		return fmt.Errorf("create version %s: %w", version.Label, err)
	}
	for _, component := range components {
		if err := insertVersionComponent(ctx, tx, r.driver, component); err != nil {
			return err
		}
	}
	for _, expose := range exposes {
		if err := insertVersionExpose(ctx, tx, r.driver, expose); err != nil {
			return err
		}
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create version with components %s: %w", version.Label, err)
	}
	return nil
}

func insertVersionComponent(ctx context.Context, tx *sqlx.Tx, driver string, component model.VersionComponent) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO version_component (
		id, version_id, name, image, command_json, args_json, env_json, ports_json, mounts_json, networks_json,
		depends_on_json, healthcheck_json, resources_json, pull_policy, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(driver), db.NowExpr(driver)),
		component.Id, component.VersionId, component.Name, component.Image, component.CommandJSON, component.ArgsJSON,
		component.EnvJSON, component.PortsJSON, component.MountsJSON, component.NetworksJSON, component.DependsOnJSON,
		component.HealthcheckJSON, component.ResourcesJSON, component.PullPolicy)
	if err != nil {
		return fmt.Errorf("create component %s: %w", component.Name, err)
	}
	return nil
}

func insertVersionExpose(ctx context.Context, tx *sqlx.Tx, driver string, expose model.VersionExpose) error {
	_, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO version_expose (
		id, version_id, component_name, protocol, container_port, path_prefix, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(driver), db.NowExpr(driver)),
		expose.Id, expose.VersionId, expose.ComponentName, expose.Protocol, expose.ContainerPort, expose.PathPrefix)
	if err != nil {
		return fmt.Errorf("create expose %s/%s/%d: %w", expose.ComponentName, expose.Protocol, expose.ContainerPort, err)
	}
	return nil
}

func (r Repository) ListEnvironments(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.Environment], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	projectId = strings.TrimSpace(projectId)
	clauses := []string{"project_id = ?"}
	args := []any{projectId}
	search = strings.TrimSpace(search)
	if search != "" {
		like := "%" + search + "%"
		clauses = append(clauses, "(code LIKE ? OR name LIKE ?)")
		args = append(args, like, like)
	}
	where := " WHERE " + strings.Join(clauses, " AND ")
	var total int
	if err := r.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM environment`+where, args...); err != nil {
		return repository.Page[model.Environment]{}, fmt.Errorf("count environments: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []model.Environment
	err := r.db.SelectContext(ctx, &items, `SELECT `+environmentColumns+` FROM environment`+where+` ORDER BY code, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.Environment]{}, fmt.Errorf("list environments: %w", err)
	}
	return repository.Page[model.Environment]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) Environment(ctx context.Context, id string) (model.Environment, error) {
	var env model.Environment
	err := r.db.GetContext(ctx, &env, `SELECT `+environmentColumns+` FROM environment WHERE id = ?`, id)
	if err != nil {
		return model.Environment{}, fmt.Errorf("load environment %s: %w", id, sqlcommon.TranslateError(err))
	}
	return env, nil
}

func (r Repository) EnvironmentByProjectCode(ctx context.Context, projectId string, code string) (model.Environment, error) {
	var env model.Environment
	err := r.db.GetContext(ctx, &env, `SELECT `+environmentColumns+` FROM environment WHERE project_id = ? AND code = ?`, projectId, code)
	if err != nil {
		return model.Environment{}, fmt.Errorf("load environment %s/%s: %w", projectId, code, sqlcommon.TranslateError(err))
	}
	return env, nil
}

func (r Repository) CreateEnvironment(ctx context.Context, env model.Environment) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO environment (
		id, project_id, code, name, description, base_domain, domain_template, default_entrypoint, tcp_entrypoint, tls_mode, created_at, updated_at
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)),
		env.Id, env.ProjectId, env.Code, env.Name, env.Description,
		env.BaseDomain, env.DomainTemplate, env.DefaultEntrypoint, env.TCPEntrypoint, env.TLSMode)
	if err != nil {
		return fmt.Errorf("create environment %s: %w", env.Code, err)
	}
	return nil
}

func (r Repository) UpdateEnvironment(ctx context.Context, env model.Environment) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE environment SET
		name = ?, description = ?, base_domain = ?, domain_template = ?, default_entrypoint = ?, tcp_entrypoint = ?, tls_mode = ?, updated_at = %s
		WHERE id = ?`, db.NowExpr(r.driver)),
		env.Name, env.Description, env.BaseDomain, env.DomainTemplate, env.DefaultEntrypoint, env.TCPEntrypoint, env.TLSMode, env.Id)
	if err != nil {
		return fmt.Errorf("update environment %s: %w", env.Id, err)
	}
	return nil
}

func (r Repository) DeleteEnvironment(ctx context.Context, id string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM environment WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete environment %s: %w", id, err)
	}
	return nil
}

func (r Repository) CountServicesByEnvironment(ctx context.Context, environmentId string) (int, error) {
	var count int
	if err := r.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM service WHERE environment_id = ?`, environmentId); err != nil {
		return 0, fmt.Errorf("count services for environment %s: %w", environmentId, err)
	}
	return count, nil
}

func (r Repository) ListServicesByApplication(ctx context.Context, applicationId string) ([]model.Service, error) {
	var items []model.Service
	err := r.db.SelectContext(ctx, &items, `SELECT `+serviceColumns+` FROM service WHERE application_id = ? ORDER BY environment_id, instance_key`, applicationId)
	if err != nil {
		return nil, fmt.Errorf("list services for application %s: %w", applicationId, err)
	}
	return items, nil
}

func (r Repository) ServiceByKey(ctx context.Context, applicationId string, environmentId string, instanceKey string) (model.Service, error) {
	var svc model.Service
	err := r.db.GetContext(ctx, &svc, `SELECT `+serviceColumns+` FROM service WHERE application_id = ? AND environment_id = ? AND instance_key = ?`, applicationId, environmentId, instanceKey)
	if err != nil {
		return model.Service{}, fmt.Errorf("load service %s/%s/%s: %w", applicationId, environmentId, instanceKey, sqlcommon.TranslateError(err))
	}
	return svc, nil
}

func (r Repository) Service(ctx context.Context, id string) (model.Service, error) {
	var svc model.Service
	err := r.db.GetContext(ctx, &svc, `SELECT `+serviceColumns+` FROM service WHERE id = ?`, id)
	if err != nil {
		return model.Service{}, fmt.Errorf("load service %s: %w", id, sqlcommon.TranslateError(err))
	}
	return svc, nil
}

func (r Repository) UpsertService(ctx context.Context, svc model.Service) error {
	var existingID string
	err := r.db.GetContext(ctx, &existingID, `SELECT id FROM service WHERE application_id = ? AND environment_id = ? AND instance_key = ?`,
		svc.ApplicationId, svc.EnvironmentId, svc.InstanceKey)
	if err != nil {
		if !errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("lookup service for application %s: %w", svc.ApplicationId, err)
		}
		_, err = r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO service (
			id, application_id, environment_id, instance_key, version_id, last_successful_version_id, status, created_at, updated_at
		) VALUES (?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(r.driver), db.NowExpr(r.driver)),
			svc.Id, svc.ApplicationId, svc.EnvironmentId, svc.InstanceKey, svc.VersionId, svc.LastSuccessfulVersionId, svc.Status)
		if err != nil {
			return fmt.Errorf("create service for application %s: %w", svc.ApplicationId, err)
		}
		return nil
	}
	id := existingID
	if strings.TrimSpace(svc.Id) != "" {
		id = svc.Id
	}
	_, err = r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE service SET version_id = ?, last_successful_version_id = ?, status = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)),
		svc.VersionId, svc.LastSuccessfulVersionId, svc.Status, id)
	if err != nil {
		return fmt.Errorf("update service %s: %w", id, err)
	}
	return nil
}

func (r Repository) UpdateServiceStatus(ctx context.Context, id string, status string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE service SET status = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), status, id)
	if err != nil {
		return fmt.Errorf("update service status %s: %w", id, err)
	}
	return nil
}

func (r Repository) UpdateServiceAfterDeploy(ctx context.Context, id string, status string, versionId string, lastSuccessfulVersionId *string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE service SET status = ?, version_id = ?, last_successful_version_id = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)),
		status, versionId, lastSuccessfulVersionId, id)
	if err != nil {
		return fmt.Errorf("update service after deploy %s: %w", id, err)
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

func (r Repository) ListEnabledRoutes(ctx context.Context) ([]model.Route, error) {
	var items []model.Route
	err := r.db.SelectContext(ctx, &items, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at
		FROM route WHERE enabled = ? ORDER BY name ASC, id ASC`, true)
	if err != nil {
		return nil, fmt.Errorf("list enabled routes: %w", err)
	}
	return items, nil
}

func (r Repository) HasActiveGatewayService(ctx context.Context, excludeApplicationId string) (bool, error) {
	query := `SELECT COUNT(*) FROM service s
		INNER JOIN application a ON a.id = s.application_id
		WHERE a.kind = ? AND s.status IN (?, ?)`
	args := []any{status.ApplicationKindGateway, status.ServiceStatusRunning, status.ServiceStatusDeploying}
	excludeApplicationId = strings.TrimSpace(excludeApplicationId)
	if excludeApplicationId != "" {
		query += ` AND s.application_id <> ?`
		args = append(args, excludeApplicationId)
	}
	var count int
	if err := r.db.GetContext(ctx, &count, query, args...); err != nil {
		return false, fmt.Errorf("count active gateway services: %w", err)
	}
	return count > 0, nil
}

func (r Repository) Route(ctx context.Context, id string) (model.Route, error) {
	var route model.Route
	err := r.db.GetContext(ctx, &route, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at FROM route WHERE id = ?`, id)
	if err != nil {
		return model.Route{}, fmt.Errorf("load route %s: %w", id, sqlcommon.TranslateError(err))
	}
	return route, nil
}

func (r Repository) RouteByDomain(ctx context.Context, domain string) (model.Route, error) {
	var route model.Route
	err := r.db.GetContext(ctx, &route, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at FROM route WHERE domain = ?`, strings.TrimSpace(domain))
	if err != nil {
		return model.Route{}, fmt.Errorf("load route by domain %s: %w", domain, sqlcommon.TranslateError(err))
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
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO deployment (
		id, project_id, application_id, application_name, version_id, service_id, environment_id, options_json,
		operation_type, trigger_type, command_text, status, started_at, is_rollback, rollback_from_deployment_id
	) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, %s, ?, ?)`, db.NowExpr(r.driver)),
		deployment.Id, deployment.ProjectId, deployment.ApplicationId, deployment.ApplicationName,
		deployment.VersionId, deployment.ServiceId, deployment.EnvironmentId, deployment.OptionsJSON,
		deployment.OperationType, deployment.TriggerType, deployment.CommandText, deployment.Status,
		deployment.IsRollback, deployment.RollbackFromDeploymentId)
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
	err := r.db.SelectContext(ctx, &items, `SELECT `+deploymentColumns+`
		FROM deployment`+where+` ORDER BY started_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return repository.Page[model.Deployment]{}, fmt.Errorf("list deployments: %w", err)
	}
	return repository.Page[model.Deployment]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (r Repository) Deployment(ctx context.Context, id string) (model.Deployment, error) {
	var deployment model.Deployment
	err := r.db.GetContext(ctx, &deployment, `SELECT `+deploymentColumns+` FROM deployment WHERE id = ?`, id)
	if err != nil {
		return model.Deployment{}, fmt.Errorf("load deployment %s: %w", id, sqlcommon.TranslateError(err))
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

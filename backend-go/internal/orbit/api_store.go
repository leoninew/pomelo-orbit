package orbit

import (
	"context"
	"fmt"
	"strings"
	"time"

	"backend/internal/db"
)

type Page[T any] struct {
	Items   []T
	Total   int
	Page    int
	PerPage int
}

func NormalizePage(page int, perPage int) (int, int) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 {
		perPage = 10
	}
	if perPage > 100 {
		perPage = 100
	}
	return page, perPage
}

func (s Store) UserByUsername(ctx context.Context, username string) (User, error) {
	var user User
	err := s.db.GetContext(ctx, &user, `SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
		email, auth_source, created_at, updated_at, last_login_at FROM user WHERE username = ?`, username)
	if err != nil {
		return User{}, fmt.Errorf("load user by username %s: %w", username, err)
	}
	return user, nil
}

func (s Store) UserById(ctx context.Context, id string) (User, error) {
	var user User
	err := s.db.GetContext(ctx, &user, `SELECT id, username, password_hash, status, oauth_provider, oauth_provider_id,
		email, auth_source, created_at, updated_at, last_login_at FROM user WHERE id = ?`, id)
	if err != nil {
		return User{}, fmt.Errorf("load user %s: %w", id, err)
	}
	return user, nil
}

func (s Store) UserRoles(ctx context.Context, userId string) ([]string, error) {
	var roles []string
	err := s.db.SelectContext(ctx, &roles, `SELECT role.code FROM role
		JOIN user_role ON user_role.role_id = role.id
		WHERE user_role.user_id = ? ORDER BY role.code`, userId)
	if err != nil {
		return nil, fmt.Errorf("load user roles %s: %w", userId, err)
	}
	return roles, nil
}

func (s Store) UserPermissions(ctx context.Context, userId string) ([]string, error) {
	var permissions []string
	err := s.db.SelectContext(ctx, &permissions, `SELECT DISTINCT permission.code FROM permission
		JOIN role_permission ON role_permission.permission_id = permission.id
		JOIN user_role ON user_role.role_id = role_permission.role_id
		WHERE user_role.user_id = ? ORDER BY permission.code`, userId)
	if err != nil {
		return nil, fmt.Errorf("load user permissions %s: %w", userId, err)
	}
	return permissions, nil
}

func (s Store) ListProjects(ctx context.Context) ([]Project, error) {
	var projects []Project
	err := s.db.SelectContext(ctx, &projects, `SELECT id, name, code, is_active, created_at, updated_at
		FROM project ORDER BY created_at DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

func (s Store) Project(ctx context.Context, id string) (Project, error) {
	var project Project
	err := s.db.GetContext(ctx, &project, `SELECT id, name, code, is_active, created_at, updated_at FROM project WHERE id = ?`, id)
	if err != nil {
		return Project{}, fmt.Errorf("load project %s: %w", id, err)
	}
	return project, nil
}

func (s Store) ListRepositories(ctx context.Context, projectId *string, page int, perPage int, search string) (Page[Repository], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := projectSearchWhere(projectId, search, []string{"name", "code", "repository_url"})
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM repository`+where, args...); err != nil {
		return Page[Repository]{}, fmt.Errorf("count repositories: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []Repository
	err := s.db.SelectContext(ctx, &items, `SELECT id, project_id, name, code, repository_url, git_credential_id,
		variable_overrides, default_branch, created_at, updated_at FROM repository`+where+` ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return Page[Repository]{}, fmt.Errorf("list repositories: %w", err)
	}
	return Page[Repository]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) ListPipelineRuns(ctx context.Context, projectId string, repositoryId string, templateId string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (Page[PipelineRun], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := pipelineRunWhere(projectId, repositoryId, templateId, dateFrom, dateTo)
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM pipeline_run`+where, args...); err != nil {
		return Page[PipelineRun]{}, fmt.Errorf("count pipeline runs: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []PipelineRun
	err := s.db.SelectContext(ctx, &items, fmt.Sprintf(`SELECT id, project_id, repository_id, repository_name, snapshot_id, template_id,
		template_name, template_version, %s, trigger_ref, variables_snapshot, status, retry_of,
		started_at, finished_at, error_message, created_at FROM pipeline_run`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, db.QuoteIdent(s.driver, "trigger")), args...)
	if err != nil {
		return Page[PipelineRun]{}, fmt.Errorf("list pipeline runs: %w", err)
	}
	return Page[PipelineRun]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) ListPipelineRunsByRepository(ctx context.Context, repositoryId string, page int, perPage int) (Page[PipelineRun], error) {
	page, perPage = NormalizePage(page, perPage)
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM pipeline_run WHERE repository_id = ?`, repositoryId); err != nil {
		return Page[PipelineRun]{}, fmt.Errorf("count repository pipeline runs %s: %w", repositoryId, err)
	}
	var items []PipelineRun
	err := s.db.SelectContext(ctx, &items, fmt.Sprintf(`SELECT id, project_id, repository_id, repository_name, snapshot_id, template_id,
		template_name, template_version, %s, trigger_ref, variables_snapshot, status, retry_of,
		started_at, finished_at, error_message, created_at FROM pipeline_run WHERE repository_id = ? ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, db.QuoteIdent(s.driver, "trigger")), repositoryId, perPage, (page-1)*perPage)
	if err != nil {
		return Page[PipelineRun]{}, fmt.Errorf("list repository pipeline runs %s: %w", repositoryId, err)
	}
	return Page[PipelineRun]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) ListArtifacts(ctx context.Context, projectId string, repositoryId string, templateId string, page int, perPage int, search string) (Page[Artifact], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := artifactWhere(projectId, repositoryId, templateId, search)
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM artifact`+where, args...); err != nil {
		return Page[Artifact]{}, fmt.Errorf("count artifacts: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []Artifact
	err := s.db.SelectContext(ctx, &items, `SELECT id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at
		FROM artifact`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return Page[Artifact]{}, fmt.Errorf("list artifacts: %w", err)
	}
	return Page[Artifact]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) ListArtifactsByRun(ctx context.Context, projectId *string, runId string) ([]Artifact, error) {
	clauses := []string{"pipeline_run_id = ?"}
	args := []any{runId}
	if projectId != nil && strings.TrimSpace(*projectId) != "" {
		clauses = append(clauses, "project_id = ?")
		args = append(args, strings.TrimSpace(*projectId))
	}
	var items []Artifact
	err := s.db.SelectContext(ctx, &items, `SELECT id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at
		FROM artifact WHERE `+strings.Join(clauses, " AND ")+` ORDER BY created_at, id`, args...)
	if err != nil {
		return nil, fmt.Errorf("list run artifacts %s: %w", runId, err)
	}
	return items, nil
}

func pipelineRunWhere(projectId string, repositoryId string, templateId string, dateFrom *time.Time, dateTo *time.Time) (string, []any) {
	clauses := []string{"project_id = ?"}
	args := []any{strings.TrimSpace(projectId)}
	if strings.TrimSpace(repositoryId) != "" {
		clauses = append(clauses, "repository_id = ?")
		args = append(args, strings.TrimSpace(repositoryId))
	}
	if strings.TrimSpace(templateId) != "" {
		clauses = append(clauses, "template_id = ?")
		args = append(args, strings.TrimSpace(templateId))
	}
	if dateFrom != nil {
		clauses = append(clauses, "created_at >= ?")
		args = append(args, *dateFrom)
	}
	if dateTo != nil {
		clauses = append(clauses, "created_at < ?")
		args = append(args, *dateTo)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func artifactWhere(projectId string, repositoryId string, templateId string, search string) (string, []any) {
	clauses := []string{"project_id = ?"}
	args := []any{strings.TrimSpace(projectId)}
	if strings.TrimSpace(repositoryId) != "" {
		clauses = append(clauses, "repository_id = ?")
		args = append(args, strings.TrimSpace(repositoryId))
	}
	if strings.TrimSpace(templateId) != "" {
		clauses = append(clauses, "template_id = ?")
		args = append(args, strings.TrimSpace(templateId))
	}
	if strings.TrimSpace(search) != "" {
		like := "%" + strings.TrimSpace(search) + "%"
		clauses = append(clauses, "(name LIKE ? OR path LIKE ?)")
		args = append(args, like, like)
	}
	return " WHERE " + strings.Join(clauses, " AND "), args
}

func (s Store) ListApplications(ctx context.Context, projectId *string, page int, perPage int, search string) (Page[Application], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := projectSearchWhere(projectId, search, []string{"name", "code"})
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM application`+where, args...); err != nil {
		return Page[Application]{}, fmt.Errorf("count applications: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []Application
	err := s.db.SelectContext(ctx, &items, `SELECT id, project_id, name, code, image_pull_policy, status, route_managed, created_at, updated_at
		FROM application`+where+` ORDER BY created_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return Page[Application]{}, fmt.Errorf("list applications: %w", err)
	}
	return Page[Application]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) ListDeployments(ctx context.Context, projectId *string, page int, perPage int) (Page[Deployment], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := projectSearchWhere(projectId, "", nil)
	var total int
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM deployment`+where, args...); err != nil {
		return Page[Deployment]{}, fmt.Errorf("count deployments: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []Deployment
	err := s.db.SelectContext(ctx, &items, `SELECT id, project_id, application_id, application_name, operation_type, trigger_type,
		status, started_at, finished_at, duration_ms, log_text, error_message, is_rollback, rollback_from_deployment_id
		FROM deployment`+where+` ORDER BY started_at DESC, id LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return Page[Deployment]{}, fmt.Errorf("list deployments: %w", err)
	}
	return Page[Deployment]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
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

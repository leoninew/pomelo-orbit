package repository

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"backend/internal/db"
	"backend/internal/repository/model"
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

func (s Store) ListProjects(ctx context.Context) ([]model.Project, error) {
	var projects []model.Project
	err := s.db.SelectContext(ctx, &projects, `SELECT id, name, code, is_active, created_at, updated_at
		FROM project ORDER BY created_at DESC, id`)
	if err != nil {
		return nil, fmt.Errorf("list projects: %w", err)
	}
	return projects, nil
}

func (s Store) Project(ctx context.Context, id string) (model.Project, error) {
	var project model.Project
	err := s.db.GetContext(ctx, &project, `SELECT id, name, code, is_active, created_at, updated_at FROM project WHERE id = ?`, id)
	if err != nil {
		return model.Project{}, fmt.Errorf("load project %s: %w", id, err)
	}
	return project, nil
}

func (s Store) IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error) {
	var count int
	if err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM project_member WHERE project_id = ? AND user_id = ?`, projectId, userId); err != nil {
		return false, fmt.Errorf("check project member %s/%s: %w", projectId, userId, err)
	}
	return count > 0, nil
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

func (s Store) ListRoutes(ctx context.Context, projectId string, page int, perPage int, search string) (Page[Route], error) {
	page, perPage = NormalizePage(page, perPage)
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
	if err := s.db.GetContext(ctx, &total, `SELECT COUNT(*) FROM route`+where, args...); err != nil {
		return Page[Route]{}, fmt.Errorf("count routes: %w", err)
	}
	args = append(args, perPage, (page-1)*perPage)
	var items []Route
	err := s.db.SelectContext(ctx, &items, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at
		FROM route`+where+` ORDER BY id DESC LIMIT ? OFFSET ?`, args...)
	if err != nil {
		return Page[Route]{}, fmt.Errorf("list routes: %w", err)
	}
	return Page[Route]{Items: items, Total: total, Page: page, PerPage: perPage}, nil
}

func (s Store) ListAllRoutes(ctx context.Context, projectId string) ([]Route, error) {
	var items []Route
	err := s.db.SelectContext(ctx, &items, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at
		FROM route WHERE project_id = ? ORDER BY id DESC`, strings.TrimSpace(projectId))
	if err != nil {
		return nil, fmt.Errorf("list all routes: %w", err)
	}
	return items, nil
}

func (s Store) Route(ctx context.Context, id string) (Route, error) {
	var route Route
	err := s.db.GetContext(ctx, &route, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at FROM route WHERE id = ?`, id)
	if err != nil {
		return Route{}, fmt.Errorf("load route %s: %w", id, err)
	}
	return route, nil
}

func (s Store) RouteByDomain(ctx context.Context, domain string) (Route, error) {
	var route Route
	err := s.db.GetContext(ctx, &route, `SELECT id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at FROM route WHERE domain = ?`, strings.TrimSpace(domain))
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Route{}, sql.ErrNoRows
		}
		return Route{}, fmt.Errorf("load route by domain %s: %w", domain, err)
	}
	return route, nil
}

func (s Store) CreateRoute(ctx context.Context, route Route) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO route (id, project_id, name, domain, path_prefix, target_url, enabled, https_enabled, cert_pem, cert_key, cert_type, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, %s, %s)`, db.NowExpr(s.driver), db.NowExpr(s.driver)), route.Id, route.ProjectId, route.Name, route.Domain, route.PathPrefix, route.TargetURL, route.Enabled, route.HTTPSEnabled, route.CertPEM, route.CertKey, route.CertType)
	if err != nil {
		return fmt.Errorf("create route %s: %w", route.Name, err)
	}
	return nil
}

func (s Store) UpdateRoute(ctx context.Context, route Route) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE route SET name = ?, domain = ?, path_prefix = ?, target_url = ?, enabled = ?, https_enabled = ?, cert_pem = ?, cert_key = ?, cert_type = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), route.Name, route.Domain, route.PathPrefix, route.TargetURL, route.Enabled, route.HTTPSEnabled, route.CertPEM, route.CertKey, route.CertType, route.Id)
	if err != nil {
		return fmt.Errorf("update route %s: %w", route.Id, err)
	}
	return nil
}

func (s Store) DeleteRoute(ctx context.Context, id string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM route WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete route %s: %w", id, err)
	}
	return nil
}

func (s Store) ListDeployments(ctx context.Context, projectId string, applicationId string, status string, search string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (Page[Deployment], error) {
	page, perPage = NormalizePage(page, perPage)
	where, args := deploymentWhere(projectId, applicationId, status, search, dateFrom, dateTo)
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

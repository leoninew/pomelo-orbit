package repository

import (
	"context"
	"fmt"

	"backend/internal/db"
	"backend/internal/repository/model"
)

func (s Store) ListProjectsByMember(ctx context.Context, userId string) ([]Project, error) {
	var projects []Project
	err := s.db.SelectContext(ctx, &projects, `SELECT project.id, project.name, project.code, project.is_active, project.created_at, project.updated_at FROM project
		JOIN project_member ON project_member.project_id = project.id
		WHERE project_member.user_id = ? ORDER BY project.created_at DESC, project.id`, userId)
	if err != nil {
		return nil, fmt.Errorf("list projects by member %s: %w", userId, err)
	}
	return projects, nil
}

func (s Store) ListActiveProjectsByMember(ctx context.Context, userId string) ([]Project, error) {
	var projects []Project
	err := s.db.SelectContext(ctx, &projects, `SELECT project.id, project.name, project.code, project.is_active, project.created_at, project.updated_at FROM project
		JOIN project_member ON project_member.project_id = project.id
		WHERE project_member.user_id = ? AND project.is_active = 1 ORDER BY project.created_at DESC, project.id`, userId)
	if err != nil {
		return nil, fmt.Errorf("list active projects by member %s: %w", userId, err)
	}
	return projects, nil
}

func (s Store) ProjectByCode(ctx context.Context, code string) (Project, error) {
	var project Project
	err := s.db.GetContext(ctx, &project, `SELECT id, name, code, is_active, created_at, updated_at FROM project WHERE code = ?`, code)
	if err != nil {
		return Project{}, fmt.Errorf("load project by code %s: %w", code, err)
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

func (s Store) CreateProject(ctx context.Context, project Project, userId string) error {
	tx, err := s.db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin create project %s: %w", project.Code, err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = tx.Rollback()
		}
	}()
	if _, err := tx.ExecContext(ctx, `INSERT INTO project (id, name, code, is_active, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`, project.Id, project.Name, project.Code, project.IsActive, project.CreatedAt, project.UpdatedAt); err != nil {
		return fmt.Errorf("create project %s: %w", project.Code, err)
	}
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO project_member (project_id, user_id, created_at) VALUES (?, ?, %s)`, db.NowExpr(s.driver)), project.Id, userId); err != nil {
		return fmt.Errorf("add project creator %s/%s: %w", project.Id, userId, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create project %s: %w", project.Code, err)
	}
	committed = true
	return nil
}

func (s Store) UpdateProject(ctx context.Context, project Project) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE project SET name = ?, code = ?, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), project.Name, project.Code, project.Id)
	if err != nil {
		return fmt.Errorf("update project %s: %w", project.Id, err)
	}
	return nil
}

func (s Store) DeprecateProject(ctx context.Context, projectId string) error {
	_, err := s.db.ExecContext(ctx, fmt.Sprintf(`UPDATE project SET is_active = 0, updated_at = %s WHERE id = ?`, db.NowExpr(s.driver)), projectId)
	if err != nil {
		return fmt.Errorf("deprecate project %s: %w", projectId, err)
	}
	return nil
}

func (s Store) CountProjectRepositories(ctx context.Context, projectId string) (int, error) {
	var count int
	if err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM repository WHERE project_id = ?`, projectId); err != nil {
		return 0, fmt.Errorf("count project repositories %s: %w", projectId, err)
	}
	return count, nil
}

func (s Store) CountProjectApplications(ctx context.Context, projectId string) (int, error) {
	var count int
	if err := s.db.GetContext(ctx, &count, `SELECT COUNT(*) FROM application WHERE project_id = ?`, projectId); err != nil {
		return 0, fmt.Errorf("count project applications %s: %w", projectId, err)
	}
	return count, nil
}

func (s Store) ProjectMembers(ctx context.Context, projectId string) ([]model.User, error) {
	var users []model.User
	err := s.db.SelectContext(ctx, &users, `SELECT user.id, user.username, user.password_hash, user.status, user.oauth_provider, user.oauth_provider_id,
		user.email, user.auth_source, user.created_at, user.updated_at, user.last_login_at FROM user
		JOIN project_member ON project_member.user_id = user.id
		WHERE project_member.project_id = ? ORDER BY user.username`, projectId)
	if err != nil {
		return nil, fmt.Errorf("list project members %s: %w", projectId, err)
	}
	return users, nil
}

func (s Store) AddProjectMember(ctx context.Context, projectId string, userId string) error {
	member, err := s.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return err
	}
	if member {
		return nil
	}
	_, err = s.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO project_member (project_id, user_id, created_at) VALUES (?, ?, %s)`, db.NowExpr(s.driver)), projectId, userId)
	if err != nil {
		return fmt.Errorf("add project member %s/%s: %w", projectId, userId, err)
	}
	return nil
}

func (s Store) RemoveProjectMember(ctx context.Context, projectId string, userId string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM project_member WHERE project_id = ? AND user_id = ?`, projectId, userId)
	if err != nil {
		return fmt.Errorf("remove project member %s/%s: %w", projectId, userId, err)
	}
	return nil
}

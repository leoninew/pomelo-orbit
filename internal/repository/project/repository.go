package projectrepo

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/jmoiron/sqlx"

	"gitee.com/leoninew/PomeloOrbit-go/internal/db"
	dbsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/db/sqlc"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
)

type Repository struct {
	db      *sqlx.DB
	driver  string
	queries *dbsqlc.Queries
}

func NewRepository(db *sqlx.DB, driver string) Repository {
	return Repository{db: db, driver: driver, queries: dbsqlc.New(db)}
}

func (r Repository) Project(ctx context.Context, id string) (model.Project, error) {
	project, err := r.queries.ProjectByID(ctx, id)
	if err != nil {
		return model.Project{}, fmt.Errorf("load project %s: %w", id, err)
	}
	return dbmodel.ProjectFromByID(project), nil
}

func (r Repository) ListProjectsByMember(ctx context.Context, userId string) ([]model.Project, error) {
	rows, err := r.queries.ListProjectsByMember(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("list projects by member %s: %w", userId, err)
	}
	projects := make([]model.Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, dbmodel.ProjectFromList(row))
	}
	return projects, nil
}

func (r Repository) ListActiveProjectsByMember(ctx context.Context, userId string) ([]model.Project, error) {
	rows, err := r.queries.ListActiveProjectsByMember(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("list active projects by member %s: %w", userId, err)
	}
	projects := make([]model.Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, dbmodel.ProjectFromActiveList(row))
	}
	return projects, nil
}

func (r Repository) ProjectByCode(ctx context.Context, code string) (model.Project, error) {
	project, err := r.queries.ProjectByCode(ctx, code)
	if err != nil {
		return model.Project{}, fmt.Errorf("load project by code %s: %w", code, err)
	}
	return dbmodel.ProjectFromByCode(project), nil
}

func (r Repository) IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error) {
	count, err := r.queries.IsProjectMember(ctx, dbsqlc.IsProjectMemberParams{ProjectID: projectId, UserID: userId})
	if err != nil {
		return false, fmt.Errorf("check project member %s/%s: %w", projectId, userId, err)
	}
	return count > 0, nil
}

func (r Repository) CreateProject(ctx context.Context, project model.Project, userId string) error {
	tx, err := r.db.BeginTxx(ctx, nil)
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
	if _, err := tx.ExecContext(ctx, fmt.Sprintf(`INSERT INTO project_member (project_id, user_id, created_at) VALUES (?, ?, %s)`, db.NowExpr(r.driver)), project.Id, userId); err != nil {
		return fmt.Errorf("add project creator %s/%s: %w", project.Id, userId, err)
	}
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit create project %s: %w", project.Code, err)
	}
	committed = true
	return nil
}

func (r Repository) UpdateProject(ctx context.Context, project model.Project) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE project SET name = ?, code = ?, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), project.Name, project.Code, project.Id)
	if err != nil {
		return fmt.Errorf("update project %s: %w", project.Id, err)
	}
	return nil
}

func (r Repository) DeprecateProject(ctx context.Context, projectId string) error {
	_, err := r.db.ExecContext(ctx, fmt.Sprintf(`UPDATE project SET is_active = 0, updated_at = %s WHERE id = ?`, db.NowExpr(r.driver)), projectId)
	if err != nil {
		return fmt.Errorf("deprecate project %s: %w", projectId, err)
	}
	return nil
}

func (r Repository) CountProjectRepositories(ctx context.Context, projectId string) (int, error) {
	count, err := r.queries.CountProjectRepositories(ctx, sql.NullString{String: projectId, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("count project repositories %s: %w", projectId, err)
	}
	return int(count), nil
}

func (r Repository) CountProjectApplications(ctx context.Context, projectId string) (int, error) {
	count, err := r.queries.CountProjectApplications(ctx, sql.NullString{String: projectId, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("count project applications %s: %w", projectId, err)
	}
	return int(count), nil
}

func (r Repository) ProjectMembers(ctx context.Context, projectId string) ([]model.User, error) {
	rows, err := r.queries.ProjectMembers(ctx, projectId)
	if err != nil {
		return nil, fmt.Errorf("list project members %s: %w", projectId, err)
	}
	users := make([]model.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, dbmodel.UserFromProjectMember(row))
	}
	return users, nil
}

func (r Repository) AddProjectMember(ctx context.Context, projectId string, userId string) error {
	member, err := r.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return err
	}
	if member {
		return nil
	}
	_, err = r.db.ExecContext(ctx, fmt.Sprintf(`INSERT INTO project_member (project_id, user_id, created_at) VALUES (?, ?, %s)`, db.NowExpr(r.driver)), projectId, userId)
	if err != nil {
		return fmt.Errorf("add project member %s/%s: %w", projectId, userId, err)
	}
	return nil
}

func (r Repository) RemoveProjectMember(ctx context.Context, projectId string, userId string) error {
	err := r.queries.RemoveProjectMember(ctx, dbsqlc.RemoveProjectMemberParams{ProjectID: projectId, UserID: userId})
	if err != nil {
		return fmt.Errorf("remove project member %s/%s: %w", projectId, userId, err)
	}
	return nil
}

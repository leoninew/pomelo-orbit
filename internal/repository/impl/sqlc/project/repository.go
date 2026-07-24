package projectrepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	projectsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/project"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.ProjectStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *projectsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DBTX) *projectsqlc.Queries {
		return projectsqlc.New(dbtx)
	})
}

func (r Repository) Project(ctx context.Context, id string) (model.Project, error) {
	project, err := r.q(ctx).ProjectByID(ctx, id)
	if err != nil {
		return model.Project{}, fmt.Errorf("load project %s: %w", id, sqlcommon.TranslateError(err))
	}
	return model.Project{
		Id:        project.ID,
		Name:      project.Name,
		Code:      project.Code,
		IsActive:  project.IsActive,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}, nil
}

func (r Repository) ListProjectsByMember(ctx context.Context, userId string) ([]model.Project, error) {
	rows, err := r.q(ctx).ListProjectsByMember(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("list projects by member %s: %w", userId, err)
	}
	projects := make([]model.Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, model.Project{
			Id:        row.ID,
			Name:      row.Name,
			Code:      row.Code,
			IsActive:  row.IsActive,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return projects, nil
}

func (r Repository) ListActiveProjectsByMember(ctx context.Context, userId string) ([]model.Project, error) {
	rows, err := r.q(ctx).ListActiveProjectsByMember(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("list active projects by member %s: %w", userId, err)
	}
	projects := make([]model.Project, 0, len(rows))
	for _, row := range rows {
		projects = append(projects, model.Project{
			Id:        row.ID,
			Name:      row.Name,
			Code:      row.Code,
			IsActive:  row.IsActive,
			CreatedAt: row.CreatedAt,
			UpdatedAt: row.UpdatedAt,
		})
	}
	return projects, nil
}

func (r Repository) ProjectByCode(ctx context.Context, code string) (model.Project, error) {
	project, err := r.q(ctx).ProjectByCode(ctx, code)
	if err != nil {
		return model.Project{}, fmt.Errorf("load project by code %s: %w", code, sqlcommon.TranslateError(err))
	}
	return model.Project{
		Id:        project.ID,
		Name:      project.Name,
		Code:      project.Code,
		IsActive:  project.IsActive,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}, nil
}

func (r Repository) IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error) {
	count, err := r.q(ctx).IsProjectMember(ctx, projectsqlc.IsProjectMemberParams{
		ProjectID: projectId,
		UserID:    userId,
	})
	if err != nil {
		return false, fmt.Errorf("check project member %s/%s: %w", projectId, userId, err)
	}
	return count > 0, nil
}

func (r Repository) CreateProject(ctx context.Context, project model.Project, userId string) error {
	q := r.q(ctx)
	if err := q.CreateProject(ctx, projectsqlc.CreateProjectParams{
		ID:        project.Id,
		Name:      project.Name,
		Code:      project.Code,
		IsActive:  project.IsActive,
		CreatedAt: project.CreatedAt,
		UpdatedAt: project.UpdatedAt,
	}); err != nil {
		return fmt.Errorf("create project %s: %w", project.Code, err)
	}
	if err := q.AddProjectMember(ctx, projectsqlc.AddProjectMemberParams{
		ProjectID: project.Id,
		UserID:    userId,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("add project creator %s/%s: %w", project.Id, userId, err)
	}
	return nil
}

func (r Repository) UpdateProject(ctx context.Context, project model.Project) error {
	err := r.q(ctx).UpdateProject(ctx, projectsqlc.UpdateProjectParams{
		Name:      project.Name,
		Code:      project.Code,
		UpdatedAt: time.Now().UTC(),
		ID:        project.Id,
	})
	if err != nil {
		return fmt.Errorf("update project %s: %w", project.Id, err)
	}
	return nil
}

func (r Repository) DeprecateProject(ctx context.Context, projectId string) error {
	err := r.q(ctx).DeprecateProject(ctx, projectsqlc.DeprecateProjectParams{
		UpdatedAt: time.Now().UTC(),
		ID:        projectId,
	})
	if err != nil {
		return fmt.Errorf("deprecate project %s: %w", projectId, err)
	}
	return nil
}

func (r Repository) CountProjectRepositories(ctx context.Context, projectId string) (int, error) {
	count, err := r.q(ctx).CountProjectRepositories(ctx, sql.NullString{String: projectId, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("count project repositories %s: %w", projectId, err)
	}
	return int(count), nil
}

func (r Repository) CountProjectApplications(ctx context.Context, projectId string) (int, error) {
	count, err := r.q(ctx).CountProjectApplications(ctx, sql.NullString{String: projectId, Valid: true})
	if err != nil {
		return 0, fmt.Errorf("count project applications %s: %w", projectId, err)
	}
	return int(count), nil
}

func (r Repository) ProjectMembers(ctx context.Context, projectId string) ([]model.User, error) {
	rows, err := r.q(ctx).ProjectMembers(ctx, projectId)
	if err != nil {
		return nil, fmt.Errorf("list project members %s: %w", projectId, err)
	}
	users := make([]model.User, 0, len(rows))
	for _, row := range rows {
		users = append(users, model.User{
			Id:              row.ID,
			Username:        row.Username,
			PasswordHash:    row.PasswordHash,
			Status:          row.Status,
			OAuthProvider:   row.OauthProvider,
			OAuthProviderId: row.OauthProviderID,
			Email:           dbmodel.StringPtr(row.Email),
			AuthSource:      row.AuthSource,
			CreatedAt:       row.CreatedAt,
			UpdatedAt:       row.UpdatedAt,
			LastLoginAt:     dbmodel.TimePtr(row.LastLoginAt),
		})
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
	if err := r.q(ctx).AddProjectMember(ctx, projectsqlc.AddProjectMemberParams{
		ProjectID: projectId,
		UserID:    userId,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		return fmt.Errorf("add project member %s/%s: %w", projectId, userId, err)
	}
	return nil
}

func (r Repository) RemoveProjectMember(ctx context.Context, projectId string, userId string) error {
	err := r.q(ctx).RemoveProjectMember(ctx, projectsqlc.RemoveProjectMemberParams{
		ProjectID: projectId,
		UserID:    userId,
	})
	if err != nil {
		return fmt.Errorf("remove project member %s/%s: %w", projectId, userId, err)
	}
	return nil
}

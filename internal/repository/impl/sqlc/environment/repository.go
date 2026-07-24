package environmentrepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	environmentsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/environment"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.EnvironmentStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *environmentsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DBTX) *environmentsqlc.Queries {
		return environmentsqlc.New(dbtx)
	})
}

func (r Repository) ListEnvironments(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.Environment], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	q := r.q(ctx)
	total, err := q.CountEnvironments(ctx, environmentsqlc.CountEnvironmentsParams{
		ProjectID: strings.TrimSpace(projectId), Column2: raw, Code: pattern, Name: pattern,
	})
	if err != nil {
		return repository.Page[model.Environment]{}, fmt.Errorf("count environments: %w", err)
	}
	rows, err := q.ListEnvironments(ctx, environmentsqlc.ListEnvironmentsParams{
		ProjectID: strings.TrimSpace(projectId), Column2: raw, Code: pattern, Name: pattern,
		Limit: int64(perPage), Offset: int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Environment]{}, fmt.Errorf("list environments: %w", err)
	}
	items := make([]model.Environment, 0, len(rows))
	for _, row := range rows {
		items = append(items, envFrom(row.ID, row.ProjectID, row.Code, row.Name, row.Description, row.CreatedAt, row.UpdatedAt))
	}
	return repository.Page[model.Environment]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Environment(ctx context.Context, id string) (model.Environment, error) {
	row, err := r.q(ctx).EnvironmentByID(ctx, id)
	if err != nil {
		return model.Environment{}, fmt.Errorf("load environment %s: %w", id, sqlcommon.TranslateError(err))
	}
	return envFrom(row.ID, row.ProjectID, row.Code, row.Name, row.Description, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) EnvironmentByProjectCode(ctx context.Context, projectId string, code string) (model.Environment, error) {
	row, err := r.q(ctx).EnvironmentByProjectCode(ctx, environmentsqlc.EnvironmentByProjectCodeParams{
		ProjectID: projectId, Code: code,
	})
	if err != nil {
		return model.Environment{}, fmt.Errorf("load environment %s/%s: %w", projectId, code, sqlcommon.TranslateError(err))
	}
	return envFrom(row.ID, row.ProjectID, row.Code, row.Name, row.Description, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) CreateEnvironment(ctx context.Context, env model.Environment) error {
	now := time.Now().UTC()
	err := r.q(ctx).CreateEnvironment(ctx, environmentsqlc.CreateEnvironmentParams{
		ID: env.Id, ProjectID: env.ProjectId, Code: env.Code, Name: env.Name,
		Description: dbmodel.NullString(env.Description), CreatedAt: now, UpdatedAt: now,
	})
	if err != nil {
		return fmt.Errorf("create environment %s: %w", env.Code, err)
	}
	return nil
}

func (r Repository) UpdateEnvironment(ctx context.Context, env model.Environment) error {
	err := r.q(ctx).UpdateEnvironment(ctx, environmentsqlc.UpdateEnvironmentParams{
		Name: env.Name, Description: dbmodel.NullString(env.Description), UpdatedAt: time.Now().UTC(), ID: env.Id,
	})
	if err != nil {
		return fmt.Errorf("update environment %s: %w", env.Id, err)
	}
	return nil
}

func (r Repository) DeleteEnvironment(ctx context.Context, id string) error {
	if err := r.q(ctx).DeleteEnvironment(ctx, id); err != nil {
		return fmt.Errorf("delete environment %s: %w", id, err)
	}
	return nil
}

func (r Repository) CountServicesByEnvironment(ctx context.Context, environmentId string) (int, error) {
	count, err := r.q(ctx).CountServicesByEnvironment(ctx, environmentId)
	if err != nil {
		return 0, fmt.Errorf("count services for environment %s: %w", environmentId, err)
	}
	return int(count), nil
}

func envFrom(id, projectID, code, name string, description sql.NullString, createdAt, updatedAt time.Time) model.Environment {
	return model.Environment{
		Id: id, ProjectId: projectID, Code: code, Name: name, Description: dbmodel.StringPtr(description),
		CreatedAt: createdAt, UpdatedAt: updatedAt,
	}
}

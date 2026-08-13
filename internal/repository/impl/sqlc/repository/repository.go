package vcsrepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	reposqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/repository"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.RepositoryStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *reposqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *reposqlc.Queries {
		return reposqlc.New(dbtx)
	})
}

func (r Repository) ListRepositories(ctx context.Context, projectId *string, page int, perPage int, search string) (repository.Page[model.Repository], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	searchPattern := sql.NullString{String: pattern, Valid: raw != ""}
	q := r.q(ctx)
	params := reposqlc.CountRepositoriesParams{
		ProjectID:     optionalNarg(projectId),
		SearchPattern: searchPattern,
	}
	total, err := q.CountRepositories(ctx, params)
	if err != nil {
		return repository.Page[model.Repository]{}, fmt.Errorf("count repositories: %w", err)
	}
	rows, err := q.ListRepositories(ctx, reposqlc.ListRepositoriesParams{
		ProjectID:     optionalNarg(projectId),
		SearchPattern: searchPattern,
		Offset:        int64((page - 1) * perPage),
		Limit:         int64(perPage),
	})
	if err != nil {
		return repository.Page[model.Repository]{}, fmt.Errorf("list repositories: %w", err)
	}
	items := make([]model.Repository, 0, len(rows))
	for _, row := range rows {
		items = append(items, repositoryFrom(row.ID, row.ProjectID, row.Name, row.Code, row.RepositoryType, row.RepositoryUrl, row.GitCredentialID, row.VariableOverrides, row.DefaultBranch, row.CreatedAt, row.UpdatedAt))
	}
	return repository.Page[model.Repository]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Repository(ctx context.Context, id string) (model.Repository, error) {
	row, err := r.q(ctx).RepositoryByID(ctx, id)
	if err != nil {
		return model.Repository{}, fmt.Errorf("load repository %s: %w", id, sqlcommon.TranslateError(err))
	}
	return repositoryFrom(row.ID, row.ProjectID, row.Name, row.Code, row.RepositoryType, row.RepositoryUrl, row.GitCredentialID, row.VariableOverrides, row.DefaultBranch, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) RepositoryByCode(ctx context.Context, projectId *string, code string) (model.Repository, error) {
	row, err := r.q(ctx).RepositoryByCode(ctx, reposqlc.RepositoryByCodeParams{
		Code:      code,
		ProjectID: optionalNarg(projectId),
	})
	if err != nil {
		return model.Repository{}, fmt.Errorf("load repository by code %s: %w", code, sqlcommon.TranslateError(err))
	}
	return repositoryFrom(row.ID, row.ProjectID, row.Name, row.Code, row.RepositoryType, row.RepositoryUrl, row.GitCredentialID, row.VariableOverrides, row.DefaultBranch, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) CreateRepository(ctx context.Context, repo model.Repository) error {
	now := time.Now().UTC()
	createdAt, updatedAt := repo.CreatedAt, repo.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	err := r.q(ctx).CreateRepository(ctx, reposqlc.CreateRepositoryParams{
		ID:                repo.Id,
		ProjectID:         dbmodel.NullString(repo.ProjectId),
		Name:              repo.Name,
		Code:              repo.Code,
		RepositoryType:    repo.RepositoryType,
		RepositoryUrl:     repo.RepositoryUrl,
		GitCredentialID:   dbmodel.NullString(repo.GitCredentialId),
		VariableOverrides: repo.VariableOverrides,
		DefaultBranch:     repo.DefaultBranch,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	})
	if err != nil {
		return fmt.Errorf("create repository %s: %w", repo.Code, err)
	}
	return nil
}

func (r Repository) UpdateRepository(ctx context.Context, repo model.Repository) error {
	err := r.q(ctx).UpdateRepository(ctx, reposqlc.UpdateRepositoryParams{
		Name:              repo.Name,
		RepositoryType:    repo.RepositoryType,
		RepositoryUrl:     repo.RepositoryUrl,
		GitCredentialID:   dbmodel.NullString(repo.GitCredentialId),
		VariableOverrides: repo.VariableOverrides,
		DefaultBranch:     repo.DefaultBranch,
		UpdatedAt:         time.Now().UTC(),
		ID:                repo.Id,
	})
	if err != nil {
		return fmt.Errorf("update repository %s: %w", repo.Id, err)
	}
	return nil
}

func (r Repository) DeleteRepository(ctx context.Context, id string) error {
	if err := r.q(ctx).DeleteRepository(ctx, id); err != nil {
		return fmt.Errorf("delete repository %s: %w", id, err)
	}
	return nil
}

func (r Repository) RepositoryHasRunningPipelines(ctx context.Context, repositoryId string) (bool, error) {
	count, err := r.q(ctx).RepositoryHasRunningPipelines(ctx, reposqlc.RepositoryHasRunningPipelinesParams{
		RepositoryID: repositoryId,
		Status:       status.WorkStatusWaitingToRun,
		Status_2:     status.WorkStatusRunning,
	})
	if err != nil {
		return false, fmt.Errorf("count running repository pipelines %s: %w", repositoryId, err)
	}
	return count > 0, nil
}

func optionalNarg(value *string) sql.NullString {
	if value == nil {
		return sql.NullString{}
	}
	trimmed := *value
	if trimmed == "" {
		return sql.NullString{}
	}
	return sql.NullString{String: trimmed, Valid: true}
}

func repositoryFrom(
	id string,
	projectId sql.NullString,
	name, code, repositoryType, repositoryUrl string,
	gitCredentialId sql.NullString,
	variableOverrides, defaultBranch string,
	createdAt, updatedAt time.Time,
) model.Repository {
	return model.Repository{
		Id:                id,
		ProjectId:         dbmodel.StringPtr(projectId),
		Name:              name,
		Code:              code,
		RepositoryType:    repositoryType,
		RepositoryUrl:     repositoryUrl,
		GitCredentialId:   dbmodel.StringPtr(gitCredentialId),
		VariableOverrides: variableOverrides,
		DefaultBranch:     defaultBranch,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}
}

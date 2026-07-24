package vcsrepo

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	reposqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.RepositoryStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *reposqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DBTX) *reposqlc.Queries {
		return reposqlc.New(dbtx)
	})
}

func (r Repository) ListRepositories(ctx context.Context, projectId *string, page int, perPage int, search string) (repository.Page[model.Repository], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	q := r.q(ctx)
	params := reposqlc.CountRepositoriesParams{
		ProjectID: optionalNarg(projectId),
		Search:    raw,
		Pattern:   pattern,
	}
	total, err := q.CountRepositories(ctx, params)
	if err != nil {
		return repository.Page[model.Repository]{}, fmt.Errorf("count repositories: %w", err)
	}
	rows, err := q.ListRepositories(ctx, reposqlc.ListRepositoriesParams{
		ProjectID: optionalNarg(projectId),
		Search:    raw,
		Pattern:   pattern,
		Offset:    int64((page - 1) * perPage),
		Limit:     int64(perPage),
	})
	if err != nil {
		return repository.Page[model.Repository]{}, fmt.Errorf("list repositories: %w", err)
	}
	items := make([]model.Repository, 0, len(rows))
	for _, row := range rows {
		items = append(items, repositoryFrom(row.ID, row.ProjectID, row.Name, row.Code, row.RepositoryUrl, row.GitCredentialID, row.VariableOverrides, row.DefaultBranch, row.CreatedAt, row.UpdatedAt))
	}
	return repository.Page[model.Repository]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Repository(ctx context.Context, id string) (model.Repository, error) {
	row, err := r.q(ctx).RepositoryByID(ctx, id)
	if err != nil {
		return model.Repository{}, fmt.Errorf("load repository %s: %w", id, sqlcommon.TranslateError(err))
	}
	return repositoryFrom(row.ID, row.ProjectID, row.Name, row.Code, row.RepositoryUrl, row.GitCredentialID, row.VariableOverrides, row.DefaultBranch, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) RepositoryByCode(ctx context.Context, projectId *string, code string) (model.Repository, error) {
	row, err := r.q(ctx).RepositoryByCode(ctx, reposqlc.RepositoryByCodeParams{
		Code:      code,
		ProjectID: optionalNarg(projectId),
	})
	if err != nil {
		return model.Repository{}, fmt.Errorf("load repository by code %s: %w", code, sqlcommon.TranslateError(err))
	}
	return repositoryFrom(row.ID, row.ProjectID, row.Name, row.Code, row.RepositoryUrl, row.GitCredentialID, row.VariableOverrides, row.DefaultBranch, row.CreatedAt, row.UpdatedAt), nil
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
		RepositoryUrl:     repo.RepositoryURL,
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
		RepositoryUrl:     repo.RepositoryURL,
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

func (r Repository) ListRepositoryWebhooks(ctx context.Context, repositoryId string) ([]model.RepositoryWebhook, error) {
	rows, err := r.q(ctx).ListRepositoryWebhooks(ctx, repositoryId)
	if err != nil {
		return nil, fmt.Errorf("list repository webhooks %s: %w", repositoryId, err)
	}
	items := make([]model.RepositoryWebhook, 0, len(rows))
	for _, row := range rows {
		items = append(items, webhookFrom(row))
	}
	return items, nil
}

func (r Repository) RepositoryWebhook(ctx context.Context, id string) (model.RepositoryWebhook, error) {
	row, err := r.q(ctx).RepositoryWebhookByID(ctx, id)
	if err != nil {
		return model.RepositoryWebhook{}, fmt.Errorf("load repository webhook %s: %w", id, sqlcommon.TranslateError(err))
	}
	return webhookFrom(row), nil
}

func (r Repository) CreateRepositoryWebhook(ctx context.Context, webhook model.RepositoryWebhook) error {
	now := time.Now().UTC()
	createdAt, updatedAt := webhook.CreatedAt, webhook.UpdatedAt
	if createdAt.IsZero() {
		createdAt = now
	}
	if updatedAt.IsZero() {
		updatedAt = now
	}
	err := r.q(ctx).CreateRepositoryWebhook(ctx, reposqlc.CreateRepositoryWebhookParams{
		ID:              webhook.Id,
		RepositoryID:    webhook.RepositoryId,
		Name:            webhook.Name,
		TemplateID:      webhook.TemplateId,
		BranchFilter:    dbmodel.NullString(webhook.BranchFilter),
		EncryptedSecret: webhook.EncryptedSecret,
		Enabled:         dbmodel.BoolInt(webhook.Enabled),
		CreatedAt:       createdAt,
		UpdatedAt:       updatedAt,
	})
	if err != nil {
		return fmt.Errorf("create repository webhook %s: %w", webhook.Name, err)
	}
	return nil
}

func (r Repository) UpdateRepositoryWebhook(ctx context.Context, webhook model.RepositoryWebhook) error {
	err := r.q(ctx).UpdateRepositoryWebhook(ctx, reposqlc.UpdateRepositoryWebhookParams{
		Name:            webhook.Name,
		TemplateID:      webhook.TemplateId,
		BranchFilter:    dbmodel.NullString(webhook.BranchFilter),
		EncryptedSecret: webhook.EncryptedSecret,
		Enabled:         dbmodel.BoolInt(webhook.Enabled),
		UpdatedAt:       time.Now().UTC(),
		ID:              webhook.Id,
	})
	if err != nil {
		return fmt.Errorf("update repository webhook %s: %w", webhook.Id, err)
	}
	return nil
}

func (r Repository) DeleteRepositoryWebhook(ctx context.Context, id string) error {
	if err := r.q(ctx).DeleteRepositoryWebhook(ctx, id); err != nil {
		return fmt.Errorf("delete repository webhook %s: %w", id, err)
	}
	return nil
}

func optionalNarg(value *string) interface{} {
	if value == nil {
		return nil
	}
	trimmed := *value
	if trimmed == "" {
		return nil
	}
	return trimmed
}

func repositoryFrom(
	id string,
	projectID sql.NullString,
	name, code, repositoryURL string,
	gitCredentialID sql.NullString,
	variableOverrides, defaultBranch string,
	createdAt, updatedAt time.Time,
) model.Repository {
	return model.Repository{
		Id:                id,
		ProjectId:         dbmodel.StringPtr(projectID),
		Name:              name,
		Code:              code,
		RepositoryURL:     repositoryURL,
		GitCredentialId:   dbmodel.StringPtr(gitCredentialID),
		VariableOverrides: variableOverrides,
		DefaultBranch:     defaultBranch,
		CreatedAt:         createdAt,
		UpdatedAt:         updatedAt,
	}
}

func webhookFrom(row reposqlc.RepositoryWebhook) model.RepositoryWebhook {
	return model.RepositoryWebhook{
		Id:              row.ID,
		RepositoryId:    row.RepositoryID,
		Name:            row.Name,
		TemplateId:      row.TemplateID,
		BranchFilter:    dbmodel.StringPtr(row.BranchFilter),
		EncryptedSecret: row.EncryptedSecret,
		Enabled:         dbmodel.IntBool(row.Enabled),
		CreatedAt:       row.CreatedAt,
		UpdatedAt:       row.UpdatedAt,
	}
}

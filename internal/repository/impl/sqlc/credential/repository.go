package credentialrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	repositorycredentialsqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/repository_credential"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.CredentialStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *repositorycredentialsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *repositorycredentialsqlc.Queries {
		return repositorycredentialsqlc.New(dbtx)
	})
}

func (r Repository) ListCredentials(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.Credential], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	projectId = strings.TrimSpace(projectId)
	searchPattern := sql.NullString{String: pattern, Valid: raw != ""}
	q := r.q(ctx)
	total, err := q.CountRepositoryCredentials(ctx, repositorycredentialsqlc.CountRepositoryCredentialsParams{
		ProjectId:     sql.NullString{String: projectId, Valid: true},
		SearchPattern: searchPattern,
	})
	if err != nil {
		return repository.Page[model.Credential]{}, fmt.Errorf("count credentials: %w", err)
	}
	rows, err := q.ListRepositoryCredentials(ctx, repositorycredentialsqlc.ListRepositoryCredentialsParams{
		ProjectId:     sql.NullString{String: projectId, Valid: true},
		SearchPattern: searchPattern,
		Limit:         int32(perPage),
		Offset:        int32((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Credential]{}, fmt.Errorf("list credentials: %w", err)
	}
	items := make([]model.Credential, 0, len(rows))
	for _, row := range rows {
		items = append(items, credentialFrom(row.Id, row.ProjectId, row.Name, row.Type, row.EncryptedData, row.Revision, row.CreatedAt))
	}
	return repository.Page[model.Credential]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Credential(ctx context.Context, projectId string, id string) (model.Credential, error) {
	row, err := r.q(ctx).RepositoryCredentialById(ctx, repositorycredentialsqlc.RepositoryCredentialByIdParams{
		Id:        id,
		ProjectId: sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
	})
	if err != nil {
		return model.Credential{}, fmt.Errorf("load credential %s: %w", id, sqlcommon.TranslateError(err))
	}
	return credentialFrom(row.Id, row.ProjectId, row.Name, row.Type, row.EncryptedData, row.Revision, row.CreatedAt), nil
}

func (r Repository) CredentialByName(ctx context.Context, projectId string, name string) (model.Credential, error) {
	row, err := r.q(ctx).RepositoryCredentialByName(ctx, repositorycredentialsqlc.RepositoryCredentialByNameParams{
		ProjectId: sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		Name:      strings.TrimSpace(name),
	})
	if err != nil {
		return model.Credential{}, fmt.Errorf("load credential by name %s: %w", name, sqlcommon.TranslateError(err))
	}
	return credentialFrom(row.Id, row.ProjectId, row.Name, row.Type, row.EncryptedData, row.Revision, row.CreatedAt), nil
}

func (r Repository) CredentialExists(ctx context.Context, projectId string, id string) (bool, error) {
	count, err := r.q(ctx).RepositoryCredentialExists(ctx, repositorycredentialsqlc.RepositoryCredentialExistsParams{
		Id:        id,
		ProjectId: sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("check credential exists %s: %w", id, err)
	}
	return count > 0, nil
}

func (r Repository) CredentialName(ctx context.Context, projectId string, id string) (*string, error) {
	name, err := r.q(ctx).RepositoryCredentialName(ctx, repositorycredentialsqlc.RepositoryCredentialNameParams{
		Id:        id,
		ProjectId: sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
	})
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("load credential name %s: %w", id, err)
	}
	return &name, nil
}

func (r Repository) CreateCredential(ctx context.Context, credential model.Credential) error {
	err := r.q(ctx).CreateRepositoryCredential(ctx, repositorycredentialsqlc.CreateRepositoryCredentialParams{
		Id:            credential.Id,
		ProjectId:     dbmodel.NullString(credential.ProjectId),
		Name:          credential.Name,
		Type:          credential.Type,
		EncryptedData: credential.EncryptedData,
		Revision:      credential.Revision,
		CreatedAt:     credential.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("create credential %s: %w", credential.Name, err)
	}
	return nil
}

func (r Repository) UpdateCredential(ctx context.Context, projectId string, credential model.Credential) error {
	err := r.q(ctx).UpdateRepositoryCredential(ctx, repositorycredentialsqlc.UpdateRepositoryCredentialParams{
		Name:          credential.Name,
		EncryptedData: credential.EncryptedData,
		Revision:      credential.Revision,
		Id:            credential.Id,
		ProjectId:     sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
	})
	if err != nil {
		return fmt.Errorf("update credential %s: %w", credential.Id, err)
	}
	return nil
}

func (r Repository) DeleteCredential(ctx context.Context, projectId string, id string) error {
	if err := r.q(ctx).DeleteRepositoryCredential(ctx, repositorycredentialsqlc.DeleteRepositoryCredentialParams{
		Id:        id,
		ProjectId: sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
	}); err != nil {
		return fmt.Errorf("delete credential %s: %w", id, err)
	}
	return nil
}

func credentialFrom(id string, projectId sql.NullString, name, typ, encryptedData string, revision int64, createdAt time.Time) model.Credential {
	return model.Credential{
		Id:            id,
		ProjectId:     dbmodel.StringPtr(projectId),
		Name:          name,
		Type:          typ,
		EncryptedData: encryptedData,
		Revision:      revision,
		CreatedAt:     createdAt,
	}
}

package credentialrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
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

func (r Repository) ListCredentials(ctx context.Context, page int, perPage int, search string) (repository.Page[model.Credential], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	searchPattern := sql.NullString{String: pattern, Valid: raw != ""}
	q := r.q(ctx)
	total, err := q.CountRepositoryCredentials(ctx, repositorycredentialsqlc.CountRepositoryCredentialsParams{
		SearchPattern: searchPattern,
	})
	if err != nil {
		return repository.Page[model.Credential]{}, fmt.Errorf("count credentials: %w", err)
	}
	rows, err := q.ListRepositoryCredentials(ctx, repositorycredentialsqlc.ListRepositoryCredentialsParams{
		SearchPattern: searchPattern,
		Limit:         int32(perPage),
		Offset:        int32((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Credential]{}, fmt.Errorf("list credentials: %w", err)
	}
	items := make([]model.Credential, 0, len(rows))
	for _, row := range rows {
		items = append(items, credentialFrom(row.Id, row.Name, row.Type, row.EncryptedData, row.Revision, row.CreatedAt, row.UpdatedAt))
	}
	return repository.Page[model.Credential]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Credential(ctx context.Context, id string) (model.Credential, error) {
	row, err := r.q(ctx).RepositoryCredentialById(ctx, id)
	if err != nil {
		return model.Credential{}, fmt.Errorf("load credential %s: %w", id, sqlcommon.TranslateError(err))
	}
	return credentialFrom(row.Id, row.Name, row.Type, row.EncryptedData, row.Revision, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) CredentialByName(ctx context.Context, name string) (model.Credential, error) {
	row, err := r.q(ctx).RepositoryCredentialByName(ctx, name)
	if err != nil {
		return model.Credential{}, fmt.Errorf("load credential by name %s: %w", name, sqlcommon.TranslateError(err))
	}
	return credentialFrom(row.Id, row.Name, row.Type, row.EncryptedData, row.Revision, row.CreatedAt, row.UpdatedAt), nil
}

func (r Repository) CredentialExists(ctx context.Context, id string) (bool, error) {
	count, err := r.q(ctx).RepositoryCredentialExists(ctx, id)
	if err != nil {
		return false, fmt.Errorf("check credential exists %s: %w", id, err)
	}
	return count > 0, nil
}

func (r Repository) CredentialName(ctx context.Context, id string) (*string, error) {
	name, err := r.q(ctx).RepositoryCredentialName(ctx, id)
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
		Name:          credential.Name,
		Type:          credential.Type,
		EncryptedData: credential.EncryptedData,
		Revision:      credential.Revision,
		CreatedAt:     credential.CreatedAt,
		UpdatedAt:     credential.UpdatedAt,
	})
	if err != nil {
		return fmt.Errorf("create credential %s: %w", credential.Name, err)
	}
	return nil
}

func (r Repository) UpdateCredential(ctx context.Context, credential model.Credential) error {
	err := r.q(ctx).UpdateRepositoryCredential(ctx, repositorycredentialsqlc.UpdateRepositoryCredentialParams{
		Name:          credential.Name,
		EncryptedData: credential.EncryptedData,
		Revision:      credential.Revision,
		UpdatedAt:     credential.UpdatedAt,
		Id:            credential.Id,
	})
	if err != nil {
		return fmt.Errorf("update credential %s: %w", credential.Id, err)
	}
	return nil
}

func (r Repository) DeleteCredential(ctx context.Context, id string) error {
	if err := r.q(ctx).DeleteRepositoryCredential(ctx, id); err != nil {
		return fmt.Errorf("delete credential %s: %w", id, err)
	}
	return nil
}

func credentialFrom(id string, name, typ, encryptedData string, revision int64, createdAt, updatedAt time.Time) model.Credential {
	return model.Credential{
		Id:            id,
		Name:          name,
		Type:          typ,
		EncryptedData: encryptedData,
		Revision:      revision,
		CreatedAt:     createdAt,
		UpdatedAt:     updatedAt,
	}
}

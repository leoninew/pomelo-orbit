package credentialrepo

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	credentialsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/credential"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlcommon"
)

var _ repository.CredentialStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *credentialsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DBTX) *credentialsqlc.Queries {
		return credentialsqlc.New(dbtx)
	})
}

func (r Repository) ListCredentials(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[model.Credential], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.SearchPattern(search)
	projectNS := sql.NullString{String: strings.TrimSpace(projectId), Valid: true}
	q := r.q(ctx)
	total, err := q.CountCredentials(ctx, credentialsqlc.CountCredentialsParams{
		ProjectID: projectNS,
		Column2:   raw,
		Name:      pattern,
		Type:      pattern,
	})
	if err != nil {
		return repository.Page[model.Credential]{}, fmt.Errorf("count credentials: %w", err)
	}
	rows, err := q.ListCredentials(ctx, credentialsqlc.ListCredentialsParams{
		ProjectID: projectNS,
		Column2:   raw,
		Name:      pattern,
		Type:      pattern,
		Limit:     int64(perPage),
		Offset:    int64((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.Credential]{}, fmt.Errorf("list credentials: %w", err)
	}
	items := make([]model.Credential, 0, len(rows))
	for _, row := range rows {
		items = append(items, credentialFrom(row.ID, row.ProjectID, row.Name, row.Type, row.EncryptedData, row.CreatedAt))
	}
	return repository.Page[model.Credential]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) Credential(ctx context.Context, id string) (model.Credential, error) {
	row, err := r.q(ctx).CredentialByID(ctx, id)
	if err != nil {
		return model.Credential{}, fmt.Errorf("load credential %s: %w", id, sqlcommon.TranslateError(err))
	}
	return credentialFrom(row.ID, row.ProjectID, row.Name, row.Type, row.EncryptedData, row.CreatedAt), nil
}

func (r Repository) CredentialByName(ctx context.Context, projectId string, name string) (model.Credential, error) {
	row, err := r.q(ctx).CredentialByName(ctx, credentialsqlc.CredentialByNameParams{
		ProjectID: sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		Name:      strings.TrimSpace(name),
	})
	if err != nil {
		return model.Credential{}, fmt.Errorf("load credential by name %s: %w", name, sqlcommon.TranslateError(err))
	}
	return credentialFrom(row.ID, row.ProjectID, row.Name, row.Type, row.EncryptedData, row.CreatedAt), nil
}

func (r Repository) CredentialExists(ctx context.Context, id string) (bool, error) {
	count, err := r.q(ctx).CredentialExists(ctx, id)
	if err != nil {
		return false, fmt.Errorf("check credential exists %s: %w", id, err)
	}
	return count > 0, nil
}

func (r Repository) CredentialName(ctx context.Context, id string) (*string, error) {
	name, err := r.q(ctx).CredentialName(ctx, id)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("load credential name %s: %w", id, err)
	}
	return &name, nil
}

func (r Repository) CreateCredential(ctx context.Context, credential model.Credential) error {
	err := r.q(ctx).CreateCredential(ctx, credentialsqlc.CreateCredentialParams{
		ID:            credential.Id,
		ProjectID:     dbmodel.NullString(credential.ProjectId),
		Name:          credential.Name,
		Type:          credential.Type,
		EncryptedData: credential.EncryptedData,
		CreatedAt:     credential.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("create credential %s: %w", credential.Name, err)
	}
	return nil
}

func (r Repository) UpdateCredential(ctx context.Context, credential model.Credential) error {
	err := r.q(ctx).UpdateCredential(ctx, credentialsqlc.UpdateCredentialParams{
		Name:          credential.Name,
		EncryptedData: credential.EncryptedData,
		ID:            credential.Id,
	})
	if err != nil {
		return fmt.Errorf("update credential %s: %w", credential.Id, err)
	}
	return nil
}

func (r Repository) DeleteCredential(ctx context.Context, id string) error {
	if err := r.q(ctx).DeleteCredential(ctx, id); err != nil {
		return fmt.Errorf("delete credential %s: %w", id, err)
	}
	return nil
}

func (r Repository) CredentialReferencedByRepositories(ctx context.Context, projectId string, credentialId string) (bool, error) {
	count, err := r.q(ctx).CredentialReferencedByRepositories(ctx, credentialsqlc.CredentialReferencedByRepositoriesParams{
		ProjectID:       sql.NullString{String: strings.TrimSpace(projectId), Valid: true},
		GitCredentialID: sql.NullString{String: credentialId, Valid: true},
	})
	if err != nil {
		return false, fmt.Errorf("count credential repository refs %s: %w", credentialId, err)
	}
	return count > 0, nil
}

func credentialFrom(id string, projectID sql.NullString, name, typ, encryptedData string, createdAt time.Time) model.Credential {
	return model.Credential{
		Id:            id,
		ProjectId:     dbmodel.StringPtr(projectID),
		Name:          name,
		Type:          typ,
		EncryptedData: encryptedData,
		CreatedAt:     createdAt,
	}
}

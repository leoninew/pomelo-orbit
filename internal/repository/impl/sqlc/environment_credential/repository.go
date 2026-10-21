package environmentcredentialrepo

import (
	"context"
	"database/sql"
	"fmt"
	"strings"

	environmentcredentialsqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/environment_credential"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.EnvironmentCredentialStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *environmentcredentialsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *environmentcredentialsqlc.Queries {
		return environmentcredentialsqlc.New(dbtx)
	})
}

func (r Repository) EnvironmentCredentialByEnvironment(ctx context.Context, environmentId string) (model.EnvironmentCredential, error) {
	row, err := r.q(ctx).EnvironmentCredentialByEnvironment(ctx, strings.TrimSpace(environmentId))
	if err != nil {
		return model.EnvironmentCredential{}, fmt.Errorf("load environment credential for %s: %w", environmentId, sqlcommon.TranslateError(err))
	}
	return environmentCredentialFrom(row), nil
}

func (r Repository) CreateEnvironmentCredential(ctx context.Context, credential model.EnvironmentCredential) error {
	err := r.q(ctx).CreateEnvironmentCredential(ctx, environmentcredentialsqlc.CreateEnvironmentCredentialParams{
		Id:                  credential.Id,
		EnvironmentId:       credential.EnvironmentId,
		PublicKey:           credential.PublicKey,
		EncryptedPrivateKey: credential.EncryptedPrivateKey,
		Revision:            credential.Revision,
		CreatedAt:           credential.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("create environment credential %s: %w", credential.Id, err)
	}
	return nil
}

func (r Repository) DeleteEnvironmentCredential(ctx context.Context, environmentId string) error {
	if err := r.q(ctx).DeleteEnvironmentCredential(ctx, strings.TrimSpace(environmentId)); err != nil {
		return fmt.Errorf("delete environment credential for %s: %w", environmentId, err)
	}
	return nil
}

func environmentCredentialFrom(row environmentcredentialsqlc.EnvironmentCredential) model.EnvironmentCredential {
	return model.EnvironmentCredential{
		Id:                  row.Id,
		EnvironmentId:       row.EnvironmentId,
		PublicKey:           row.PublicKey,
		EncryptedPrivateKey: row.EncryptedPrivateKey,
		Revision:            row.Revision,
		CreatedAt:           row.CreatedAt,
	}
}

package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// CredentialStore persists repository credentials.
type CredentialStore interface {
	ListCredentials(ctx context.Context, page int, perPage int, search string) (Page[model.Credential], error)
	Credential(ctx context.Context, id string) (model.Credential, error)
	CredentialByName(ctx context.Context, name string) (model.Credential, error)
	CredentialExists(ctx context.Context, id string) (bool, error)
	CredentialName(ctx context.Context, id string) (*string, error)
	CreateCredential(ctx context.Context, credential model.Credential) error
	UpdateCredential(ctx context.Context, credential model.Credential) error
	DeleteCredential(ctx context.Context, id string) error
}

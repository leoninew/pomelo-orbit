package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// CredentialStore persists repository credentials.
type CredentialStore interface {
	ListCredentials(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.Credential], error)
	Credential(ctx context.Context, projectId string, id string) (model.Credential, error)
	CredentialByName(ctx context.Context, projectId string, name string) (model.Credential, error)
	CredentialExists(ctx context.Context, projectId string, id string) (bool, error)
	CredentialName(ctx context.Context, projectId string, id string) (*string, error)
	CreateCredential(ctx context.Context, credential model.Credential) error
	UpdateCredential(ctx context.Context, projectId string, credential model.Credential) error
	DeleteCredential(ctx context.Context, projectId string, id string) error
	CredentialReferencedByRepositories(ctx context.Context, projectId string, credentialId string) (bool, error)
}

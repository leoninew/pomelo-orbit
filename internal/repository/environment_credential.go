package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

type EnvironmentCredentialStore interface {
	EnvironmentCredentialByEnvironment(ctx context.Context, environmentId string) (model.EnvironmentCredential, error)
	CreateEnvironmentCredential(ctx context.Context, credential model.EnvironmentCredential) error
	DeleteEnvironmentCredential(ctx context.Context, environmentId string) error
}

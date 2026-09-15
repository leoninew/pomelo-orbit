package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

type EnvironmentCredentialStore interface {
	EnvironmentCredential(ctx context.Context, id string) (model.EnvironmentCredential, error)
	LatestEnvironmentCredentialByProject(ctx context.Context, projectId string) (model.EnvironmentCredential, error)
	CreateEnvironmentCredential(ctx context.Context, credential model.EnvironmentCredential) error
}

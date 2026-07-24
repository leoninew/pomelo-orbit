package repository

import (
	"context"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

// EnvironmentStore persists deployment environments.
type EnvironmentStore interface {
	ListEnvironments(ctx context.Context, projectId string, page int, perPage int, search string) (Page[model.Environment], error)
	Environment(ctx context.Context, id string) (model.Environment, error)
	EnvironmentByProjectCode(ctx context.Context, projectId string, code string) (model.Environment, error)
	CreateEnvironment(ctx context.Context, env model.Environment) error
	UpdateEnvironment(ctx context.Context, env model.Environment) error
	DeleteEnvironment(ctx context.Context, id string) error
	CountServicesByEnvironment(ctx context.Context, environmentId string) (int, error)
}

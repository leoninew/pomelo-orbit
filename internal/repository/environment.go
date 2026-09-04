package repository

import (
	"context"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// EnvironmentStore persists the sole SSH deployment target for each Project.
// The association is a logical foreign key so its lifecycle remains explicit.
type EnvironmentStore interface {
	Environment(ctx context.Context, id string) (model.Environment, error)
	EnvironmentByProject(ctx context.Context, projectId string) (model.Environment, error)
	CreateEnvironment(ctx context.Context, environment model.Environment) error
	UpdateEnvironment(ctx context.Context, environment model.Environment) error
	RecordProbe(ctx context.Context, environmentID string, targetRevision int64, status string, probedAt time.Time, diagnostic string) (bool, error)
	BindGatewayApplication(ctx context.Context, environmentID string, gatewayApplicationID string) (bool, error)
}

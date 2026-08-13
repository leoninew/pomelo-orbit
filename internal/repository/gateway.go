package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// GatewayStore persists gateway configuration and resolution queries.
type GatewayStore interface {
	HasActiveGatewayService(ctx context.Context, excludeApplicationId string) (bool, error)
	GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error)
	ResolveActiveGatewayConfig(ctx context.Context) (model.GatewayConfig, error)
	ListGatewayApplications(ctx context.Context, projectId string) ([]model.Application, error)
	UpsertGatewayConfig(ctx context.Context, cfg model.GatewayConfig) error
}

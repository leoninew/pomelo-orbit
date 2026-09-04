package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// GatewayStore persists gateway configuration and resolution queries.
type GatewayStore interface {
	GatewayConfig(ctx context.Context, applicationId string) (model.GatewayConfig, error)
	GatewayConfigByProject(ctx context.Context, projectId string) (model.GatewayConfig, error)
	ListGatewayApplications(ctx context.Context, projectId string) ([]model.Application, error)
	UpsertGatewayConfig(ctx context.Context, cfg model.GatewayConfig) error
	ReplaceGatewayVersionBindings(ctx context.Context, applicationId string, bindings []model.GatewayVersionBinding) error
}

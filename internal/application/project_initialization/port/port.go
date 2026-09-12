package port

import (
	"context"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type ProjectService interface {
	LoadForUser(ctx context.Context, projectID string, userID string) (model.Project, error)
}

type EnvironmentService interface {
	EnvironmentForUser(ctx context.Context, userID string, projectID string) (environmentdto.View, error)
	SaveInitialization(ctx context.Context, userID string, projectID string, input environmentdto.UpdateInput) (environmentdto.View, error)
	TestSSHReachability(ctx context.Context, userID string, projectID string, input environmentdto.SSHTargetInput) error
	DeploymentSSHPublicKeyForProject(ctx context.Context, userID string, projectID string) (string, error)
	ProbeForUser(ctx context.Context, userID string, projectID string) (environmentdto.View, error)
	InitializeForUser(ctx context.Context, userID string, projectID string, input environmentdto.InitializeInput) (environmentdto.View, error)
}

type GatewayService interface {
	ListGateways(ctx context.Context, userID string, projectID string, page int, perPage int, search string) (repository.Page[gatewaydto.GatewayView], error)
	CreateGateway(ctx context.Context, userID string, input gatewaydto.GatewayCreateInput) (gatewaydto.GatewayView, error)
}

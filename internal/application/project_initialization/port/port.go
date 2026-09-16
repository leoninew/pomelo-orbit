package port

import (
	"context"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type ProjectService interface {
	LoadForUser(ctx context.Context, projectId string, userId string) (model.Project, error)
}

type EnvironmentService interface {
	EnvironmentForUser(ctx context.Context, userId string, projectId string) (environmentdto.View, error)
	SaveInitialization(ctx context.Context, userId string, projectId string, input environmentdto.UpdateInput) (environmentdto.View, error)
	TestSSHReachability(ctx context.Context, userId string, projectId string, input environmentdto.SSHTargetInput) error
	PrepareWindowsEnvironment(ctx context.Context, userId string, projectId string, input environmentdto.SSHTargetInput) (environmentdto.View, string, error)
	ProbeForUser(ctx context.Context, userId string, projectId string) (environmentdto.View, error)
	InitializeForUser(ctx context.Context, userId string, projectId string, input environmentdto.InitializeInput) (environmentdto.View, error)
}

type GatewayService interface {
	ListGateways(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[gatewaydto.GatewayView], error)
	CreateGateway(ctx context.Context, userId string, projectId string, input gatewaydto.GatewayCreateInput) (gatewaydto.GatewayView, error)
}

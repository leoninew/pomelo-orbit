package deploymentsvc

import (
	"context"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func (s Service) gatewayForDeployment(ctx context.Context, app model.Application, plan model.EffectiveServicePlan) (*model.GatewayConfig, error) {
	if s.gatewayCoordinator == nil {
		return nil, apperror.New(apperror.KindInternal, "gateway deployment coordinator is not configured")
	}
	return s.gatewayCoordinator.GatewayForDeployment(ctx, app, plan)
}

func (s Service) selectGatewayVersionForDeployment(ctx context.Context, app model.Application, service model.Service) (model.Service, error) {
	if s.gatewayCoordinator == nil {
		return model.Service{}, apperror.New(apperror.KindInternal, "gateway deployment coordinator is not configured")
	}
	return s.gatewayCoordinator.SelectGatewayDeploymentVersion(ctx, app, service)
}

func isActiveServiceStatus(statusValue string) bool {
	switch statusValue {
	case status.ServiceStatusRunning:
		return true
	default:
		return false
	}
}

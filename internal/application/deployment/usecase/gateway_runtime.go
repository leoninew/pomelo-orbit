package deploymentsvc

import (
	"context"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (s Service) gatewayForDeployment(ctx context.Context, app model.Application, plan model.EffectiveServicePlan) (*model.GatewayConfig, error) {
	if s.gatewayCoordinator == nil {
		return nil, apperror.New(apperror.KindInternal, "gateway deployment coordinator is not configured")
	}
	return s.gatewayCoordinator.GatewayForDeployment(ctx, app, plan)
}

func isActiveServiceStatus(statusValue string) bool {
	switch statusValue {
	case status.ServiceStatusRunning:
		return true
	default:
		return false
	}
}

package deploymentsvc

import (
	"context"
	"strings"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

const (
	entrypointWeb       = "web"
	entrypointWebSecure = "websecure"
)

func (s Service) gatewayForDeployment(ctx context.Context, app model.Application, exposes []model.VersionExpose) (*model.GatewayConfig, error) {
	if s.gatewayCoordinator == nil {
		return nil, apperror.New(apperror.KindInternal, "gateway deployment coordinator is not configured")
	}
	return s.gatewayCoordinator.GatewayForDeployment(ctx, app, exposes)
}

func validGatewayEntrypoint(name string) bool {
	return strings.TrimSpace(name) == entrypointWeb || strings.TrimSpace(name) == entrypointWebSecure
}

func isActiveServiceStatus(statusValue string) bool {
	switch strings.TrimSpace(statusValue) {
	case status.ServiceStatusRunning, status.ServiceStatusDeploying:
		return true
	default:
		return false
	}
}

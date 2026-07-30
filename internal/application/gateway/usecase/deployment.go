package gatewaysvc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

// EnsureGatewayRunning rejects standard application commands until the Gateway
// that owns their required external network has reached the running state.
func (s Service) EnsureGatewayRunning(ctx context.Context, app model.Application) error {
	if strings.TrimSpace(app.Kind) == status.ApplicationKindGateway {
		return nil
	}
	cfg, err := s.config.ResolveActiveGatewayConfig(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindValidation,
				"no gateway configured: create, configure, and deploy a gateway before deploying application "+app.Code)
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to resolve gateway config", err)
	}
	services, err := s.service.ListServicesByApplication(ctx, cfg.ApplicationId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load gateway service state", err)
	}
	for _, item := range services {
		if item.Status == status.ServiceStatusRunning {
			return nil
		}
	}
	gatewayApp, err := s.application.Application(ctx, cfg.ApplicationId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load gateway application", err)
	}
	state := "no service has been deployed"
	if len(services) > 0 {
		states := make([]string, 0, len(services))
		for _, item := range services {
			states = append(states, item.Status)
		}
		state = "service status: " + strings.Join(states, ", ")
	}
	return apperror.New(
		apperror.KindValidation,
		fmt.Sprintf(
			"gateway %q (%s) is configured but not running (%s); deploy the gateway before deploying application %q",
			gatewayApp.Code,
			gatewayApp.Id,
			state,
			app.Code,
		),
	)
}

// GatewayForDeployment resolves the Gateway configuration required to render
// an application's Compose definition.
func (s Service) GatewayForDeployment(ctx context.Context, app model.Application, exposes []model.ServiceExpose) (*model.GatewayConfig, error) {
	if strings.TrimSpace(app.Kind) != status.ApplicationKindGateway && !hasPublicExpose(exposes) {
		return nil, nil
	}
	if strings.TrimSpace(app.Kind) == status.ApplicationKindGateway {
		cfg, err := s.config.GatewayConfig(ctx, app.Id)
		if err == nil {
			return &cfg, nil
		}
		if !errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to load gateway config", err)
		}
	}
	cfg, err := s.config.ResolveActiveGatewayConfig(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(apperror.KindValidation, "no gateway configured: create and configure a gateway first")
		}
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to resolve gateway config", err)
	}
	return &cfg, nil
}

func hasPublicExpose(exposes []model.ServiceExpose) bool {
	for _, expose := range exposes {
		if exposeAccessOf(expose) == exposeAccessPublic {
			return true
		}
	}
	return false
}

package gatewaysvc

import (
	"context"
	"errors"
	"strings"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// GatewayForDeployment resolves the Gateway configuration required to render
// an application's Compose definition.
func (s Service) GatewayForDeployment(ctx context.Context, app model.Application, plan model.EffectiveServicePlan) (*model.GatewayConfig, error) {
	if strings.TrimSpace(app.Kind) != status.ApplicationKindGateway && (!plan.JoinsTraefikNetwork() || !hasGatewayEndpoint(plan)) {
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

func hasGatewayEndpoint(plan model.EffectiveServicePlan) bool {
	for _, component := range plan.Components {
		for _, endpoint := range component.Endpoints {
			if endpoint.Mode == "gateway" {
				return true
			}
		}
	}
	return false
}

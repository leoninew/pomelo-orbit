package gatewaysvc

import (
	"context"
	"errors"
	"strings"

	gatewayport "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

// GatewayForDeployment resolves the Gateway configuration required to render
// an application's Compose definition.
func (s Service) GatewayForDeployment(ctx context.Context, app model.Application, exposes []model.VersionExpose) (*model.GatewayConfig, error) {
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

// PrepareDeployment validates exposure occupancy and recompiles managed
// Gateway listeners when a public TCP deployment changes their port set.
func (s Service) PrepareDeployment(ctx context.Context, app model.Application, exposes []model.VersionExpose) (gatewayport.DeploymentPreparation, error) {
	gateway, err := s.GatewayForDeployment(ctx, app, exposes)
	if err != nil {
		return gatewayport.DeploymentPreparation{}, err
	}
	preparation := gatewayport.DeploymentPreparation{RenderConfig: gateway}
	if strings.TrimSpace(app.Kind) == status.ApplicationKindGateway {
		tcpListens, err := s.ActivePublicTCPListens(ctx, "")
		if err != nil {
			return gatewayport.DeploymentPreparation{}, err
		}
		if _, err := s.CompileGatewayToVersion(ctx, app, *gateway, tcpListens...); err != nil {
			return gatewayport.DeploymentPreparation{}, err
		}
		return preparation, nil
	}

	if err := s.ValidateDeploymentExposureConflicts(ctx, app, exposes, gateway); err != nil {
		return gatewayport.DeploymentPreparation{}, err
	}
	candidateTCP := publicTCPListens(exposes)
	if len(candidateTCP) == 0 {
		return preparation, nil
	}
	activeTCP, err := s.ActivePublicTCPListens(ctx, app.Id)
	if err != nil {
		return gatewayport.DeploymentPreparation{}, err
	}
	targetTCP := normalizeTCPListens(append(activeTCP, candidateTCP...))
	currentTCP, err := s.compiledGatewayTCPListens(ctx, gateway.ApplicationId)
	if err != nil {
		return gatewayport.DeploymentPreparation{}, err
	}
	if TCPListensEqual(currentTCP, targetTCP) {
		return preparation, nil
	}
	gatewayApp, err := s.application.Application(ctx, gateway.ApplicationId)
	if err != nil {
		return gatewayport.DeploymentPreparation{}, err
	}
	if _, err := s.CompileGatewayToVersion(ctx, gatewayApp, *gateway, targetTCP...); err != nil {
		return gatewayport.DeploymentPreparation{}, err
	}
	preparation.RolloutConfig = gateway
	return preparation, nil
}

func (s Service) compiledGatewayTCPListens(ctx context.Context, applicationID string) ([]int, error) {
	versions, err := s.application.ListVersions(ctx, applicationID)
	if err != nil {
		return nil, err
	}
	for _, version := range versions {
		if version.Status != status.VersionStatusUnpublished {
			continue
		}
		components, err := s.application.VersionComponentsByVersion(ctx, version.Id)
		if err != nil {
			return nil, err
		}
		return CompiledTCPListens(components), nil
	}
	return nil, nil
}

func hasPublicExpose(exposes []model.VersionExpose) bool {
	for _, expose := range exposes {
		if exposeAccessOf(expose) == exposeAccessPublic {
			return true
		}
	}
	return false
}

func publicTCPListens(exposes []model.VersionExpose) []int {
	listens := make([]int, 0, len(exposes))
	for _, expose := range exposes {
		if exposeAccessOf(expose) != exposeAccessPublic || strings.ToLower(strings.TrimSpace(expose.Protocol)) != "tcp" {
			continue
		}
		listens = append(listens, effectiveListen(expose))
	}
	return normalizeTCPListens(listens)
}

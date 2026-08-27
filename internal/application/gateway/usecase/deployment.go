package gatewaysvc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// GatewayForDeployment resolves the Gateway configuration required to render
// an application's Compose definition.
func (s Service) GatewayForDeployment(ctx context.Context, app model.Application, plan model.EffectiveServicePlan) (*model.GatewayConfig, error) {
	if s.config == nil {
		return nil, nil
	}
	cfg, err := s.config.GatewayConfig(ctx, app.Id)
	if err == nil {
		return &cfg, nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load gateway config", err)
	}
	if !plan.JoinsTraefikNetwork() {
		return nil, nil
	}
	if !hasGatewayEndpoint(plan) {
		return nil, nil
	}
	cfg, err = s.config.ResolveActiveGatewayConfig(ctx)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return nil, apperror.New(apperror.KindValidation, "no gateway configured: create and configure a gateway first")
		}
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to resolve gateway config", err)
	}
	return &cfg, nil
}

// SelectGatewayDeploymentVersion makes the default Gateway Service point at
// the Version selected by its saved ACME profile before Deployment snapshots
// the Service. Non-Gateway applications are left unchanged.
func (s Service) SelectGatewayDeploymentVersion(ctx context.Context, app model.Application, service model.Service) (model.Service, error) {
	if s.config == nil {
		return service, nil
	}
	cfg, err := s.config.GatewayConfig(ctx, app.Id)
	if errors.Is(err, repository.ErrNotFound) {
		return service, nil
	}
	if err != nil {
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load gateway config", err)
	}
	if service.InstanceKey != "default" {
		return service, nil
	}
	role := cfg.AcmeProfile
	if role == "" {
		role = gatewayVersionRoleBase
	}
	versionID := cfg.VersionIDForProfile(role)
	if versionID == "" {
		return model.Service{}, apperror.New(apperror.KindValidation, "Gateway Version binding is missing for "+role)
	}
	if service.VersionId == versionID {
		return service, nil
	}
	version, err := s.application.Version(ctx, versionID)
	if err != nil {
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load selected Gateway Version", err)
	}
	if version.ApplicationId != app.Id {
		return model.Service{}, apperror.New(apperror.KindValidation, "Gateway Version binding does not belong to the Gateway application")
	}
	declarations, err := s.application.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load selected Gateway Version components", err)
	}
	mappings, err := s.service.ServiceComponentsByService(ctx, service.Id)
	if err != nil {
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load Gateway Service components", err)
	}
	if err := remapGatewayServiceComponents(mappings, declarations); err != nil {
		return model.Service{}, apperror.New(apperror.KindValidation, err.Error())
	}
	service.VersionId = version.Id
	if err := s.service.UpdateServiceConfiguration(ctx, service, mappings); err != nil {
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to select Gateway Version", err)
	}
	return service, nil
}

func remapGatewayServiceComponents(mappings []model.ServiceComponent, declarations []model.VersionComponent) error {
	if len(mappings) != len(declarations) {
		return fmt.Errorf("selected Gateway Version has incompatible component declarations")
	}
	byName := make(map[string]model.VersionComponent, len(declarations))
	for _, declaration := range declarations {
		byName[declaration.Name] = declaration
	}
	for index := range mappings {
		declaration, ok := byName[mappings[index].ComponentName]
		if !ok {
			return fmt.Errorf("selected Gateway Version does not declare component %s", mappings[index].ComponentName)
		}
		if strings.TrimSpace(mappings[index].SourceVersionComponentId) == "" {
			return fmt.Errorf("gateway Service component %s is missing its source Version component", mappings[index].ComponentName)
		}
		mappings[index].SourceVersionComponentId = declaration.Id
	}
	return nil
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

package routesvc

import (
	"context"
	"errors"
	"fmt"
	"strings"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

const routeProtocolHTTP = "http"
const routeProtocolTCP = "tcp"

func (s Service) routeFromCreateInput(ctx context.Context, projectId string, input routedto.RouteCreateInput) (model.Route, error) {
	route := model.Route{
		ProjectId:             &projectId,
		Name:                  strings.TrimSpace(input.Name),
		Protocol:              strings.TrimSpace(input.Protocol),
		Domain:                strings.TrimSpace(input.Domain),
		PathPrefix:            strings.TrimSpace(input.PathPrefix),
		TargetUrl:             strings.TrimSpace(input.TargetUrl),
		ListenPort:            input.ListenPort,
		ServiceId:             optionalString(input.ServiceId),
		ComponentName:         optionalString(input.ComponentName),
		EndpointProtocol:      optionalString(input.EndpointProtocol),
		EndpointContainerPort: input.EndpointContainerPort,
		Enabled:               input.Enabled,
		CertType:              certTypeManual,
		AcmeChallenge:         acmeChallengeHTTP,
	}
	if route.Protocol == routeProtocolHTTP && route.PathPrefix == "" {
		route.PathPrefix = "/"
	}
	if err := s.validateRoute(ctx, projectId, &route, "", true); err != nil {
		return model.Route{}, err
	}
	return route, nil
}

func (s Service) validateRoute(ctx context.Context, projectId string, route *model.Route, excludeId string, resolveManagedTarget bool) error {
	switch route.Protocol {
	case routeProtocolHTTP:
		if !validRouteIdentity(route.Name, route.Domain, route.PathPrefix) {
			return apperror.New(apperror.KindValidation, "Invalid route fields")
		}
		if route.ListenPort != nil {
			return apperror.New(apperror.KindValidation, "HTTP route cannot declare a TCP listen port")
		}
		if hasManagedRouteTarget(*route) {
			if resolveManagedTarget {
				if err := s.resolveManagedRouteTarget(ctx, projectId, route); err != nil {
					return err
				}
			}
			return nil
		}
		if !hasEmptyManagedRouteTarget(*route) || !routeTargetUrlPattern.MatchString(route.TargetUrl) {
			return apperror.New(apperror.KindValidation, "HTTP route requires a managed HTTP endpoint or custom target URL")
		}
	case routeProtocolTCP:
		if !validRouteCode(route.Name) || strings.TrimSpace(route.Domain) == "" || route.ListenPort == nil || *route.ListenPort < 1 || *route.ListenPort > 65535 || route.ServiceId == nil || route.ComponentName == nil || route.EndpointProtocol == nil || route.EndpointContainerPort == nil {
			return apperror.New(apperror.KindValidation, "Invalid TCP route fields")
		}
		if reservedTCPRoutePort(*route.ListenPort) {
			return apperror.New(apperror.KindValidation, fmt.Sprintf("TCP listen port %d is reserved by Gateway", *route.ListenPort))
		}
		if strings.TrimSpace(route.PathPrefix) != "" || route.HTTPSEnabled || route.CertPEM != nil || route.CertKey != nil || route.CertType != "" && route.CertType != certTypeManual {
			return apperror.New(apperror.KindValidation, "TCP route cannot declare HTTP or certificate fields")
		}
		if resolveManagedTarget {
			if err := s.resolveManagedRouteTarget(ctx, projectId, route); err != nil {
				return err
			}
			if route.Enabled {
				if err := s.ensureTCPListenerAvailable(ctx, *route, excludeId); err != nil {
					return err
				}
			}
		}
	default:
		return apperror.New(apperror.KindValidation, "route protocol must be http or tcp")
	}
	return nil
}

func (s Service) resolveManagedRouteTarget(ctx context.Context, projectId string, route *model.Route) error {
	if route.ProjectId == nil || route.ServiceId == nil || route.ComponentName == nil || route.EndpointProtocol == nil || route.EndpointContainerPort == nil {
		return apperror.New(apperror.KindValidation, "managed route target is required")
	}
	service, err := s.service.Service(ctx, projectId, *route.ServiceId)
	if err != nil {
		if isNotFound(err) {
			return apperror.New(apperror.KindValidation, "route target service was not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load route target service", err)
	}
	app, err := s.application.Application(ctx, projectId, service.ApplicationId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load route target application", err)
	}
	components, err := s.serviceEndpointComponents(ctx, projectId, app, service)
	if err != nil {
		return err
	}
	component, endpoint, found := effectiveEndpoint(components, *route.ComponentName, *route.EndpointProtocol, *route.EndpointContainerPort)
	if !found {
		return apperror.New(apperror.KindValidation, "route target endpoint was not found")
	}
	switch route.Protocol {
	case routeProtocolHTTP:
		if endpoint.Protocol != "http" {
			return apperror.New(apperror.KindValidation, "HTTP route target must reference a declared HTTP endpoint")
		}
	case routeProtocolTCP:
		if endpoint.Protocol != "tcp" {
			return apperror.New(apperror.KindValidation, "TCP route target must reference a declared TCP endpoint")
		}
	default:
		return apperror.New(apperror.KindValidation, "route protocol must be http or tcp")
	}
	route.TargetUrl = service.Code + "/" + component.Name + "/" + model.EndpointDisplayName(endpoint.Protocol, endpoint.ContainerPort)
	route.TargetAddress = model.RuntimeContainerName(app.Code, component.Name)
	route.TargetPort = endpoint.ContainerPort
	return nil
}

func (s Service) ensureTCPListenerAvailable(ctx context.Context, candidate model.Route, excludeId string) error {
	if candidate.ProjectId == nil || *candidate.ProjectId == "" {
		return apperror.New(apperror.KindValidation, "Route project is required")
	}
	projectId := *candidate.ProjectId
	routes, err := s.route.ListEnabledRoutesByProject(ctx, projectId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to list enabled routes", err)
	}
	for _, route := range routes {
		if route.Id == excludeId || route.Protocol != routeProtocolTCP || route.ListenPort == nil {
			continue
		}
		if *route.ListenPort == *candidate.ListenPort {
			return apperror.New(apperror.KindConflict, fmt.Sprintf("TCP listen port %d is already used by route %s", *candidate.ListenPort, route.Name))
		}
	}
	conflict, err := s.componentPortConflict(ctx, projectId, *candidate.ListenPort)
	if err != nil {
		return err
	}
	if conflict != "" {
		return apperror.New(apperror.KindConflict, fmt.Sprintf("TCP listen port %d conflicts with component endpoint %s", *candidate.ListenPort, conflict))
	}
	return nil
}

func (s Service) componentPortConflict(ctx context.Context, projectId string, listenPort int) (string, error) {
	if projectId == "" {
		return "", apperror.New(apperror.KindValidation, "project_id is required")
	}
	apps, err := s.application.ListApplications(ctx, projectId, 1, 10000, "", "")
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list applications for TCP port validation", err)
	}
	for _, app := range apps.Items {
		services, err := s.service.ListServicesByApplication(ctx, projectId, app.Id)
		if err != nil {
			return "", apperror.Wrap(apperror.KindInternal, "Failed to list services for TCP port validation", err)
		}
		for _, service := range services {
			if service.Status != status.ServiceStatusRunning {
				continue
			}
			components, err := s.serviceEndpointComponents(ctx, projectId, app, service)
			if err != nil {
				return "", err
			}
			for _, component := range components {
				for _, endpoint := range component.Endpoints {
					if (endpoint.Mode == "local" || endpoint.Mode == "host") && endpoint.ListenPort != nil && *endpoint.ListenPort == listenPort {
						return app.Code + "/" + service.Code + "/" + component.Name + "/" + model.EndpointDisplayName(endpoint.Protocol, endpoint.ContainerPort), nil
					}
				}
			}
		}
	}
	return "", nil
}

func (s Service) serviceEndpointComponents(ctx context.Context, projectId string, app model.Application, service model.Service) ([]model.EffectiveServiceComponent, error) {
	version, err := s.application.Version(ctx, projectId, service.VersionId)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load route target version", err)
	}
	if version.ApplicationId != app.Id {
		return nil, apperror.New(apperror.KindValidation, "route target version does not belong to application")
	}
	declarations, err := s.application.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load route target components", err)
	}
	overlays, err := s.service.ServiceComponentsByService(ctx, projectId, service.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load route target components", err)
	}
	bySource := make(map[string]model.ServiceComponent, len(overlays))
	for _, overlay := range overlays {
		if overlay.ServiceId != service.Id {
			return nil, apperror.New(apperror.KindValidation, "route target component does not belong to service")
		}
		if _, exists := bySource[overlay.SourceVersionComponentId]; exists {
			return nil, apperror.New(apperror.KindValidation, "route target service has duplicate component mappings")
		}
		bySource[overlay.SourceVersionComponentId] = overlay
	}
	components := make([]model.EffectiveServiceComponent, 0, len(declarations))
	for _, declaration := range declarations {
		overlay, exists := bySource[declaration.Id]
		if !exists || overlay.ComponentName != declaration.Name {
			return nil, apperror.New(apperror.KindValidation, "route target component mapping does not match selected version")
		}
		endpoints, err := model.MergeServiceComponentEndpoints(declaration, overlay)
		if err != nil {
			return nil, apperror.New(apperror.KindValidation, "route target service has invalid endpoints: "+err.Error())
		}
		components = append(components, model.EffectiveServiceComponent{Name: declaration.Name, Endpoints: endpoints})
		delete(bySource, declaration.Id)
	}
	if len(bySource) != 0 {
		return nil, apperror.New(apperror.KindValidation, "route target component mapping does not match selected version")
	}
	return components, nil
}

func effectiveEndpoint(components []model.EffectiveServiceComponent, componentName, endpointProtocol string, endpointContainerPort int) (model.EffectiveServiceComponent, model.VersionComponentEndpoint, bool) {
	for _, component := range components {
		if component.Name != componentName {
			continue
		}
		for _, endpoint := range component.Endpoints {
			if endpoint.Protocol == endpointProtocol && endpoint.ContainerPort == endpointContainerPort {
				return component, endpoint, true
			}
		}
	}
	return model.EffectiveServiceComponent{}, model.VersionComponentEndpoint{}, false
}

func optionalString(value string) *string {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	return &value
}

func hasManagedRouteTarget(route model.Route) bool {
	return route.ServiceId != nil || route.ComponentName != nil || route.EndpointProtocol != nil || route.EndpointContainerPort != nil
}

func hasEmptyManagedRouteTarget(route model.Route) bool {
	return route.ServiceId == nil && route.ComponentName == nil && route.EndpointProtocol == nil && route.EndpointContainerPort == nil
}

func reservedTCPRoutePort(port int) bool {
	return port == 80 || port == 443 || port == 8080
}

func isNotFound(err error) bool {
	return errors.Is(err, repository.ErrNotFound)
}

package routesvc

import (
	"context"
	"errors"
	"strconv"

	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"gopkg.in/yaml.v3"
)

func (s Service) CheckRouteDependencies(ctx context.Context, projectId string, plan model.EffectiveServicePlan) error {
	gateway, err := s.gateway.GatewayConfigByProject(ctx, projectId)
	if errors.Is(err, repository.ErrNotFound) {
		return nil
	}
	if err != nil {
		return err
	}
	items, err := s.routePublisher.InspectPublications(ctx, projectId, gateway)
	if err != nil {
		return err
	}
	isGateway := gateway.ApplicationId == plan.Application.Id
	var config struct {
		EntryPoints map[string]any `yaml:"entryPoints"`
		Resolvers   map[string]any `yaml:"certificatesResolvers"`
	}
	if isGateway && len(items) != 0 {
		for _, component := range plan.Components {
			if component.Name != model.GatewayComponentName() {
				continue
			}
			for _, mount := range component.Mounts {
				if mount.Target == "/etc/traefik/traefik.yml" {
					if err := yaml.Unmarshal([]byte(mount.Content), &config); err != nil {
						return err
					}
				}
			}
		}
	}
	for _, item := range items {
		route := item.Route
		dependsOnService := route.ServiceId != nil && *route.ServiceId == plan.Service.Id
		if !isGateway && !dependsOnService {
			continue
		}
		missingGatewayFile := isGateway && item.ActualFingerprint == ""
		if item.Phase != "confirmed" || item.ActualFingerprint != item.Fingerprint && !missingGatewayFile || route.Enabled && item.ActualCertificateRevision != item.CertificateRevision {
			return apperror.NewWithCode(apperror.KindConflict, "route_publication_incomplete", "Resolve incomplete Route publication before deployment")
		}
		if !route.Enabled {
			continue
		}
		if isGateway {
			entrypoint := "web"
			if route.Protocol == routeProtocolTCP && route.ListenPort != nil {
				entrypoint = "tcp" + strconv.Itoa(*route.ListenPort)
			} else if route.HTTPSEnabled {
				entrypoint = "websecure"
			}
			if _, found := config.EntryPoints[entrypoint]; !found {
				return publishedDependencyError(route)
			}
			if route.HTTPSEnabled && route.CertType == certTypeLetsEncrypt {
				resolver := "letsencrypt"
				if route.AcmeChallenge == acmeChallengeDNS {
					resolver = "letsencrypt-dns"
				}
				if _, found := config.Resolvers[resolver]; !found {
					return publishedDependencyError(route)
				}
			}
		}
		if dependsOnService {
			if route.ComponentName == nil || route.EndpointProtocol == nil || route.EndpointContainerPort == nil {
				return publishedDependencyError(route)
			}
			_, _, found := effectiveEndpoint(plan, *route.ComponentName, *route.EndpointProtocol, *route.EndpointContainerPort)
			if !found || !plan.JoinsTraefikNetwork() || route.TargetAddress != model.RuntimeContainerName(plan.Application.Code, *route.ComponentName) {
				return publishedDependencyError(route)
			}
		}
	}
	return nil
}

func publishedDependencyError(route model.Route) error {
	return apperror.NewWithCode(apperror.KindConflict, "route_published_dependency_conflict", "Deployment would remove or rename a dependency used by published Route "+route.Name+". Explicitly sync that Route first.")
}

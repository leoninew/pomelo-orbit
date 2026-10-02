package routesvc

import (
	"context"
	"testing"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestDeploymentPreservesPublishedUpstreamDependencies(t *testing.T) {
	serviceId, component, protocol, port := "service-1", "api", "http", 8080
	route := model.Route{Id: "route-1", Name: "api", Enabled: true, ServiceId: &serviceId, ComponentName: &component, EndpointProtocol: &protocol, EndpointContainerPort: &port, TargetAddress: model.RuntimeContainerName("app", component)}
	publisher := &recordingRoutePublisher{publications: map[string]routeport.Publication{route.Id: {Route: route, Phase: "confirmed"}}}
	s := Service{gateway: gatewayConfigFake{cfg: model.GatewayConfig{ApplicationId: "gateway-1"}}, routePublisher: publisher}
	plan := model.EffectiveServicePlan{Application: model.Application{Id: "app-1", Code: "app"}, Service: model.Service{Id: serviceId}, Components: []model.EffectiveServiceComponent{{Name: component, Endpoints: []model.VersionComponentEndpoint{{Protocol: protocol, ContainerPort: port}}}}}
	if err := s.CheckRouteDependencies(context.Background(), "project-1", plan); err != nil {
		t.Fatal(err)
	}
	withoutNetwork := false
	plan.JoinTraefikNetwork = &withoutNetwork
	if err := s.CheckRouteDependencies(context.Background(), "project-1", plan); err == nil || apperror.Classify(err).Code != "route_published_dependency_conflict" {
		t.Fatalf("network removal err=%v", err)
	}
	plan.JoinTraefikNetwork = nil
	plan.Components = nil
	if err := s.CheckRouteDependencies(context.Background(), "project-1", plan); err == nil || apperror.Classify(err).Code != "route_published_dependency_conflict" {
		t.Fatalf("endpoint removal err=%v", err)
	}
	route.Enabled = false
	publisher.publications[route.Id] = routeport.Publication{Route: route, Phase: "confirmed"}
	if err := s.CheckRouteDependencies(context.Background(), "project-1", plan); err != nil {
		t.Fatalf("withdrawn Route blocked deployment: %v", err)
	}
}

func TestGatewayDeploymentPreservesPublishedResolver(t *testing.T) {
	route := model.Route{Id: "route-1", Name: "secure", Enabled: true, HTTPSEnabled: true, CertType: "letsencrypt", AcmeChallenge: "dns"}
	publisher := &recordingRoutePublisher{publications: map[string]routeport.Publication{route.Id: {Route: route, Phase: "confirmed"}}}
	s := Service{gateway: gatewayConfigFake{cfg: model.GatewayConfig{ApplicationId: "gateway-1"}}, routePublisher: publisher}
	plan := model.EffectiveServicePlan{Application: model.Application{Id: "gateway-1"}, Components: []model.EffectiveServiceComponent{{Name: model.GatewayComponentName(), Mounts: []model.VersionComponentMount{{Target: "/etc/traefik/traefik.yml", Content: "entryPoints:\n  websecure: {}\ncertificatesResolvers:\n  letsencrypt-dns: {}\n"}}}}}
	if err := s.CheckRouteDependencies(context.Background(), "project-1", plan); err != nil {
		t.Fatal(err)
	}
	plan.Components[0].Mounts[0].Content = "entryPoints:\n  websecure: {}\n"
	if err := s.CheckRouteDependencies(context.Background(), "project-1", plan); err == nil || apperror.Classify(err).Code != "route_published_dependency_conflict" {
		t.Fatalf("resolver removal err=%v", err)
	}
}

func TestGatewayDeploymentCanPrepareMissingRouteDirectories(t *testing.T) {
	route := model.Route{Id: "route-1", Name: "api", Enabled: true}
	publication := routeport.Publication{Route: route, Phase: "confirmed", Fingerprint: "published", ActualFingerprint: ""}
	publisher := &recordingRoutePublisher{publications: map[string]routeport.Publication{route.Id: publication}}
	s := Service{gateway: gatewayConfigFake{cfg: model.GatewayConfig{ApplicationId: "gateway-1"}}, routePublisher: publisher}
	plan := model.EffectiveServicePlan{Application: model.Application{Id: "gateway-1"}, Components: []model.EffectiveServiceComponent{{Name: model.GatewayComponentName(), Mounts: []model.VersionComponentMount{{Target: "/etc/traefik/traefik.yml", Content: "entryPoints:\n  web: {}\n"}}}}}
	if err := s.CheckRouteDependencies(context.Background(), "project-1", plan); err != nil {
		t.Fatalf("missing Route file prevented Gateway directory preparation: %v", err)
	}
	publication.ActualFingerprint = "external modification"
	publisher.publications[route.Id] = publication
	if err := s.CheckRouteDependencies(context.Background(), "project-1", plan); err == nil || apperror.Classify(err).Code != "route_publication_incomplete" {
		t.Fatalf("external modification did not block deployment: %v", err)
	}
}

package routesvc

import (
	"context"
	"testing"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestPublishSnapshotWaitsForGatewayAndPublishesRoutes(t *testing.T) {
	publisher := &recordingRoutePublisher{}
	service := Service{
		route:               routeListFake{routes: []model.Route{{Id: "route-1", Name: "api", Protocol: "http", Domain: "api.example.test", PathPrefix: "/", TargetUrl: "http://example:80", Enabled: true}}},
		gateway:             gatewayConfigFake{cfg: model.GatewayConfig{ApplicationId: "gateway-1", RestApiUrl: "http://localhost:8080"}},
		routePublisher:      publisher,
		traefikRouterClient: traefikRouterClientFake{},
	}

	if err := service.PublishSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if publisher.readyWaits != 1 {
		t.Fatalf("PublishSnapshot WaitUntilReady calls = %d, want 1", publisher.readyWaits)
	}
	if len(publisher.snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(publisher.snapshots))
	}
}

type routeListFake struct {
	repository.RouteStore
	routes []model.Route
}

func (f routeListFake) ListEnabledRoutes(context.Context) ([]model.Route, error) {
	return append([]model.Route(nil), f.routes...), nil
}

type gatewayConfigFake struct {
	repository.GatewayStore
	cfg model.GatewayConfig
}

func (f gatewayConfigFake) ResolveActiveGatewayConfig(context.Context) (model.GatewayConfig, error) {
	return f.cfg, nil
}

type traefikRouterClientFake struct{}

func (traefikRouterClientFake) ListRouters(context.Context, string) ([]routeport.TraefikRouter, error) {
	return nil, nil
}

func (traefikRouterClientFake) IsConnectionError(error) bool {
	return false
}

package routesvc

import (
	"context"
	"strings"
	"testing"
	"time"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestPublishSnapshotWaitsForGatewayAndPublishesRoutes(t *testing.T) {
	events := []string{}
	publisher := &snapshotOrderPublisher{events: &events}
	service := Service{
		route:               routeListFake{routes: []model.Route{{Id: "route-1", Name: "api", Protocol: "http", Domain: "api.example.test", PathPrefix: "/", TargetUrl: "http://example:80", Enabled: true}}},
		gateway:             gatewayConfigFake{cfg: model.GatewayConfig{ApplicationId: "gateway-1", RestApiUrl: "http://localhost:8080"}},
		routePublisher:      publisher,
		traefikRouterClient: snapshotOrderRouterClient{events: &events},
	}

	if err := service.PublishSnapshot(context.Background(), "project-1"); err != nil {
		t.Fatal(err)
	}
	if publisher.readyWaits != 1 {
		t.Fatalf("PublishSnapshot WaitUntilReady calls = %d, want 1", publisher.readyWaits)
	}
	if len(publisher.snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(publisher.snapshots))
	}
	if got, want := strings.Join(events, ","), "wait,apply"; got != want {
		t.Fatalf("PublishSnapshot order = %q, want %q", got, want)
	}
}

type routeListFake struct {
	repository.RouteStore
	routes []model.Route
}

func (f routeListFake) ListEnabledRoutesByProject(context.Context, string) ([]model.Route, error) {
	return append([]model.Route(nil), f.routes...), nil
}

type gatewayConfigFake struct {
	repository.GatewayStore
	cfg model.GatewayConfig
}

func (f gatewayConfigFake) GatewayConfigByProject(context.Context, string) (model.GatewayConfig, error) {
	return f.cfg, nil
}

type snapshotOrderPublisher struct {
	recordingRoutePublisher
	events *[]string
}

func (p *snapshotOrderPublisher) WaitUntilReady(ctx context.Context, projectID string, gateway model.GatewayConfig, timeout time.Duration) error {
	*p.events = append(*p.events, "wait")
	return p.recordingRoutePublisher.WaitUntilReady(ctx, projectID, gateway, timeout)
}

func (p *snapshotOrderPublisher) ApplySnapshot(ctx context.Context, projectID string, gateway model.GatewayConfig, routes []model.Route) error {
	*p.events = append(*p.events, "apply")
	return p.recordingRoutePublisher.ApplySnapshot(ctx, projectID, gateway, routes)
}

type snapshotOrderRouterClient struct {
	events *[]string
}

func (c snapshotOrderRouterClient) ListRouters(context.Context, string, model.GatewayConfig) ([]routeport.TraefikRouter, error) {
	*c.events = append(*c.events, "routers")
	return nil, nil
}

func (snapshotOrderRouterClient) ListServices(context.Context, string, model.GatewayConfig) ([]routeport.TraefikService, error) {
	return nil, nil
}

func (snapshotOrderRouterClient) IsConnectionError(error) bool {
	return false
}

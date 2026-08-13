package routesvc

import (
	"context"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestPublishSnapshotDoesNotCompileGatewayListeners(t *testing.T) {
	compiler := &countingGatewayCompiler{}
	publisher := &recordingRoutePublisher{}
	service := Service{
		route:           routeListFake{routes: []model.Route{{Id: "route-1", Name: "api", Protocol: "http", Domain: "api.example.test", PathPrefix: "/", TargetUrl: "http://example:80", Enabled: true}}},
		gateway:         gatewayConfigFake{cfg: model.GatewayConfig{ApplicationId: "gateway-1", RestApiUrl: "http://localhost:8080"}},
		gatewayCompiler: compiler,
		routePublisher:  publisher,
	}

	if err := service.PublishSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if compiler.calls != 0 {
		t.Fatalf("PublishSnapshot compiled gateway listeners %d time(s)", compiler.calls)
	}
	if publisher.readyWaits != 1 {
		t.Fatalf("PublishSnapshot WaitUntilReady calls = %d, want 1", publisher.readyWaits)
	}
	if len(publisher.snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(publisher.snapshots))
	}
}

func TestPublishRouteSnapshotCompilesGatewayListeners(t *testing.T) {
	compiler := &countingGatewayCompiler{}
	publisher := &recordingRoutePublisher{}
	service := Service{
		route:           routeListFake{routes: []model.Route{{Id: "route-1", Name: "api", Protocol: "http", Domain: "api.example.test", PathPrefix: "/", TargetUrl: "http://example:80", Enabled: true}}},
		gateway:         gatewayConfigFake{cfg: model.GatewayConfig{ApplicationId: "gateway-1", RestApiUrl: "http://localhost:8080"}},
		gatewayCompiler: compiler,
		routePublisher:  publisher,
	}

	if err := service.publishRouteSnapshot(context.Background()); err != nil {
		t.Fatal(err)
	}
	if compiler.calls != 1 {
		t.Fatalf("publishRouteSnapshot compiled gateway listeners %d time(s), want 1", compiler.calls)
	}
	if publisher.readyWaits != 0 {
		t.Fatalf("publishRouteSnapshot WaitUntilReady calls = %d, want 0", publisher.readyWaits)
	}
	if len(publisher.snapshots) != 1 {
		t.Fatalf("snapshots = %d, want 1", len(publisher.snapshots))
	}
}

type countingGatewayCompiler struct {
	calls int
}

func (c *countingGatewayCompiler) CompileTCPRouteListeners(context.Context, model.GatewayConfig, []int) error {
	c.calls++
	return nil
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

package routesvc

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestGatewayRecreationWaitsThenVerifiesWithoutReadingEditableRoutes(t *testing.T) {
	events := []string{}
	publisher := &snapshotOrderPublisher{events: &events}
	service := Service{
		route:          routeListFake{},
		gateway:        gatewayConfigFake{cfg: model.GatewayConfig{ApplicationId: "gateway-1", RestApiUrl: model.GatewayRestApiContainerUrl, RestApiHostUrl: model.GatewayRestApiHostUrl}},
		routePublisher: publisher,
	}

	if err := service.VerifyPublishedRoutes(context.Background(), "project-1"); err != nil {
		t.Fatal(err)
	}
	if publisher.readyWaits != 1 {
		t.Fatalf("VerifyPublishedRoutes WaitUntilReady calls = %d, want 1", publisher.readyWaits)
	}
	if len(publisher.published) != 0 {
		t.Fatalf("unexpected publication: %+v", publisher.published)
	}
	if got, want := strings.Join(events, ","), "wait,verify"; got != want {
		t.Fatalf("VerifyPublishedRoutes order = %q, want %q", got, want)
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

func (p *snapshotOrderPublisher) WaitUntilReady(ctx context.Context, projectId string, gateway model.GatewayConfig, timeout time.Duration) error {
	*p.events = append(*p.events, "wait")
	return p.recordingRoutePublisher.WaitUntilReady(ctx, projectId, gateway, timeout)
}

func (p *snapshotOrderPublisher) VerifyPublished(context.Context, string, model.GatewayConfig) error {
	*p.events = append(*p.events, "verify")
	return nil
}

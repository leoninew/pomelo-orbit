package routesvc

import (
	"context"
	"errors"
	"testing"
	"time"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	routeTestUserId    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	routeTestProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"
)

type recordingRoutePublisher struct {
	published    []model.Route
	publications map[string]routeport.Publication
	failures     map[string]error
	attempts     []string
	dependencies string
	inspected    []model.Route
	lists        int
	listedIds    []string
	err          error
}

func (p *recordingRoutePublisher) LockGateway(context.Context, string) (func(), error) {
	return func() {}, nil
}
func (p *recordingRoutePublisher) ValidateGateway(context.Context, string, model.GatewayConfig, []model.Route) (string, error) {
	return p.dependencies, nil
}
func (p *recordingRoutePublisher) OpenSession(ctx context.Context, _ string) (context.Context, func(), error) {
	return ctx, func() {}, nil
}

func (p *recordingRoutePublisher) ListPublicationRouteIds(context.Context, string, model.GatewayConfig) ([]string, error) {
	p.lists++
	ids := append([]string(nil), p.listedIds...)
	for id := range p.publications {
		ids = append(ids, id)
	}
	return ids, nil
}

func (p *recordingRoutePublisher) InspectPublication(_ context.Context, _ string, _ model.GatewayConfig, route model.Route) (*routeport.Publication, error) {
	p.inspected = append(p.inspected, route)
	if item, found := p.publications[route.Id]; found {
		return &item, nil
	}
	return nil, nil
}
func (p *recordingRoutePublisher) PublishRoute(_ context.Context, _ string, _ model.GatewayConfig, route model.Route, _ string) (routeport.PublicationResult, error) {
	p.attempts = append(p.attempts, route.Id)
	result := routeport.PublicationResult{OperationId: "operation-" + route.Id, FileCommit: "committed", ConfigurationMatch: "matched", CertificateVerification: "not_applicable", Recovery: "not_needed", Cleanup: "completed"}
	if route.HTTPSEnabled {
		result.CertificateVerification = "unverified"
	}
	if p.err != nil {
		return result, p.err
	}
	if err := p.failures[route.Id]; err != nil {
		return result, err
	}
	p.published = append(p.published, route)
	if p.publications == nil {
		p.publications = make(map[string]routeport.Publication)
	}
	p.publications[route.Id] = routeport.Publication{Route: route, Phase: "confirmed"}
	return result, nil
}

type recordingCertificateGenerator struct{}

func (recordingCertificateGenerator) Generate(context.Context, string) (string, string, error) {
	return "CERT", "KEY", nil
}

type recordingTraefikClient struct {
	queries            int
	routers            []routeport.TraefikRouter
	services           []routeport.TraefikService
	err                error
	unavailableMessage string
}

func (c *recordingTraefikClient) ListRouters(_ context.Context, _ string, _ model.GatewayConfig) ([]routeport.TraefikRouter, error) {
	c.queries++
	if c.err != nil {
		return nil, c.err
	}
	return c.routers, nil
}

func (c *recordingTraefikClient) ListServices(_ context.Context, _ string, _ model.GatewayConfig) ([]routeport.TraefikService, error) {
	c.queries++
	if c.err != nil {
		return nil, c.err
	}
	return c.services, nil
}

func (c *recordingTraefikClient) TraefikUnavailableMessage(err error) (string, bool) {
	return c.unavailableMessage, c.unavailableMessage != "" && errors.Is(err, c.err)
}

func testRouteSyncTimeouts() routeport.SyncTimeouts {
	return routeport.SyncTimeouts{GatewayLock: 30 * time.Second, StateLoad: 60 * time.Second, FilePublication: 60 * time.Second, ApiRequest: 3 * time.Second, Reload: 5 * time.Second, ConfigurationMatch: 10 * time.Second, Recovery: 10 * time.Second}
}

func selectedPreview(t *testing.T, s Service, ids ...string) routedto.RouteSyncPreview {
	t.Helper()
	preview, err := s.PreviewRouteSync(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "selected", RouteIds: ids})
	if err != nil {
		t.Fatal(err)
	}
	return preview
}

func confirmInput(preview routedto.RouteSyncPreview) routedto.RouteSyncConfirmInput {
	return routedto.RouteSyncConfirmInput{RouteIds: preview.RouteIds, BusinessHash: preview.BusinessHash, PublicationHash: preview.PublicationHash}
}

func confirmPreview(t *testing.T, s Service, preview routedto.RouteSyncPreview) routedto.RouteSyncConfirmResult {
	t.Helper()
	result, err := s.ConfirmRouteSync(context.Background(), routeTestUserId, routeTestProjectId, confirmInput(preview))
	if err != nil {
		t.Fatal(err)
	}
	return result
}

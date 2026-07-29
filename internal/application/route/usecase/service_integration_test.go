package routesvc

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"testing"

	_ "modernc.org/sqlite"

	routedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/route/dto"
	routeport "gitee.com/leoninew/PomeloOrbit-go/internal/application/route/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	applicationrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/application"
	gatewayrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/gateway"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	routerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/route"
)

const (
	routeTestUserId    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	routeTestProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"
)

func TestRouteServicePublishesCertificatesAndTraefikViews(t *testing.T) {
	service, publisher, client, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{Name: "api-route", Domain: "api.example.test", PathPrefix: "/api", TargetUrl: "http://host.docker.internal:8081", Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if route.Id == "" || route.CertType != "manual" || route.HTTPSEnabled {
		t.Fatalf("unexpected created route: %+v", route)
	}
	if err := service.DeleteRoute(ctx, routeTestUserId, route.Id); err != nil {
		t.Fatal(err)
	}

	enabled, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{Name: "enabled-route", Domain: "enabled.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8082", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(publisher.snapshots) == 0 {
		t.Fatal("expected rest snapshot after enabling route")
	}
	lastSnap := publisher.snapshots[len(publisher.snapshots)-1]
	if len(lastSnap) != 1 || lastSnap[0].Id != enabled.Id {
		t.Fatalf("expected enabled route in snapshot, got %+v", lastSnap)
	}
	if err := service.DeleteRoute(ctx, routeTestUserId, enabled.Id); err == nil || apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("expected enabled route delete validation error, got %v", err)
	}
	disabled, err := service.DisableRoute(ctx, routeTestUserId, enabled.Id)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Enabled {
		t.Fatalf("expected disabled route, got %+v", disabled)
	}
	lastSnap = publisher.snapshots[len(publisher.snapshots)-1]
	if len(lastSnap) != 0 {
		t.Fatalf("expected empty snapshot after disable, got %+v", lastSnap)
	}

	certRoute, err := service.UploadRouteCert(ctx, routeTestUserId, disabled.Id, "CERT", "KEY")
	if err != nil {
		t.Fatal(err)
	}
	if !certRoute.HTTPSEnabled || certRoute.CertType != "manual" {
		t.Fatalf("unexpected manual cert route: %+v", certRoute)
	}
	certRoute, err = service.EnableRouteLetsEncrypt(ctx, routeTestUserId, disabled.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !certRoute.HTTPSEnabled || certRoute.CertType != "letsencrypt" || certRoute.CertPEM != nil || certRoute.CertKey != nil {
		t.Fatalf("unexpected letsencrypt route: %+v", certRoute)
	}
	if len(publisher.revokedCerts) == 0 || publisher.revokedCerts[len(publisher.revokedCerts)-1] != "enabled-route" {
		t.Fatalf("expected cert revoke, got %+v", publisher.revokedCerts)
	}

	dashboardProject := routeTestProjectId
	if err := service.route.CreateRoute(ctx, model.Route{Id: "01KTRAETFIKROUTE0000000001", ProjectId: &dashboardProject, Name: "traefik-dashboard", Domain: "traefik.lvh.me", PathPrefix: "/", TargetUrl: "http://traefik:8080", Enabled: true, HTTPSEnabled: true, CertType: "manual"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := service.TraefikRouteConfig(ctx, routeTestUserId, routeTestProjectId)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DashboardDomain != "traefik.lvh.me" || !cfg.HTTPSEnabled {
		t.Fatalf("unexpected traefik config: %+v", cfg)
	}
	client.routers = []routeport.TraefikRouter{{Name: "api@docker", Provider: "docker", Status: "enabled", Rule: "Host(`api.lvh.me`)", Service: "api-service", Entrypoints: []string{"websecure"}, TLS: true}}
	routes, err := service.ListTraefikRoutes(ctx, routeTestUserId, routeTestProjectId)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 || routes[0].Name != "api@docker" || routes[0].Provider != "docker" || routes[0].Status != "enabled" || routes[0].Rule != "Host(`api.lvh.me`)" || routes[0].Service != "api-service" || len(routes[0].Entrypoints) != 1 || routes[0].Entrypoints[0] != "websecure" || !routes[0].TLS {
		t.Fatalf("unexpected traefik routes: %+v", routes)
	}
	client.routers[0].Name = "mutated@docker"
	client.routers[0].Entrypoints[0] = "web"
	if routes[0].Name != "api@docker" || routes[0].Entrypoints[0] != "websecure" {
		t.Fatalf("expected application view to be independent from the port data, got %+v", routes)
	}
	routes[0].Entrypoints[0] = "mutated-response"
	if client.routers[0].Entrypoints[0] != "web" {
		t.Fatalf("expected port data to be independent from the application view, got %+v", client.routers)
	}
	client.err = errors.New("connection refused")
	if _, err := service.ListTraefikRoutes(ctx, routeTestUserId, routeTestProjectId); err == nil || apperror.StatusCode(err) != http.StatusServiceUnavailable || apperror.Classify(err).Message != "Traefik is unavailable." {
		t.Fatalf("expected safe traefik unavailable error, got %v", err)
	}
}

func newRouteIntegrationService(t *testing.T) (Service, *recordingRoutePublisher, *recordingTraefikClient, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}
	cfg.Cert.LetsEncrypt.Enabled = true
	cfg.Cert.LetsEncrypt.Email = "admin@example.test"
	publisher := &recordingRoutePublisher{}
	client := &recordingTraefikClient{}
	service := New(
		projectrepo.NewRepository(database),
		routerepo.NewRepository(database),
		gatewayrepo.NewRepository(database),
		cfg,
		publisher,
		recordingCertificateGenerator{},
		client,
	)
	seedRouteTestGateway(t, database)
	return service, publisher, client, database
}

func seedRouteTestGateway(t *testing.T, database *sql.DB) {
	t.Helper()
	ctx := context.Background()
	appRepo := applicationrepo.NewRepository(database)
	gwRepo := gatewayrepo.NewRepository(database)
	projectId := routeTestProjectId
	app := model.Application{
		Id:              "01KROUTEGATEWAYAPP000000001",
		ProjectId:       &projectId,
		Name:            "Test Gateway",
		Code:            "test-gateway",
		Kind:            status.ApplicationKindGateway,
		ImagePullPolicy: "missing",
	}
	if err := appRepo.CreateApplication(ctx, app); err != nil {
		t.Fatalf("seed gateway app: %v", err)
	}
	img := "traefik:3.6"
	if err := gwRepo.UpsertGatewayConfig(ctx, model.GatewayConfig{
		ApplicationId:     app.Id,
		RestApiUrl:        "http://traefik:8080",
		BaseDomain:        "lvh.me",
		Image:             &img,
		DefaultEntrypoint: "websecure",
		TLSMode:           "optional",
	}); err != nil {
		t.Fatalf("seed gateway config: %v", err)
	}
}

type recordingRoutePublisher struct {
	snapshots    [][]model.Route
	revokedCerts []string
}

func (p *recordingRoutePublisher) ApplySnapshot(_ context.Context, _ string, routes []model.Route) error {
	p.snapshots = append(p.snapshots, append([]model.Route(nil), routes...))
	return nil
}

func (p *recordingRoutePublisher) WriteCertificate(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (p *recordingRoutePublisher) RevokeCertificate(_ context.Context, routeName string) error {
	p.revokedCerts = append(p.revokedCerts, routeName)
	return nil
}

type recordingCertificateGenerator struct{}

func (recordingCertificateGenerator) Generate(context.Context, string) (string, string, error) {
	return "CERT", "KEY", nil
}

type recordingTraefikClient struct {
	routers []routeport.TraefikRouter
	err     error
}

func (c *recordingTraefikClient) ListRouters(_ context.Context, _ string) ([]routeport.TraefikRouter, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.routers, nil
}

func (c *recordingTraefikClient) IsConnectionError(err error) bool {
	return err != nil && errors.Is(err, c.err)
}

package routesvc

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"testing"

	_ "modernc.org/sqlite"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	gatewayrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/gateway"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	routerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/route"
	servicerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/service"
	testseed "github.com/leoninew/pomelo-orbit/internal/testutil/seed"
)

const (
	routeTestUserId    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	routeTestProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"
)

func TestRouteServicePublishesCertificatesAndTraefikViews(t *testing.T) {
	service, publisher, client, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{Name: "api-route", Protocol: "http", Domain: "api.example.test", PathPrefix: "/api", TargetUrl: "http://host.docker.internal:8081", Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if route.Id == "" || route.CertType != "manual" || route.HTTPSEnabled {
		t.Fatalf("unexpected created route: %+v", route)
	}
	if err := service.DeleteRoute(ctx, routeTestUserId, route.Id); err != nil {
		t.Fatal(err)
	}

	enabled, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{Name: "enabled-route", Protocol: "http", Domain: "enabled.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8082", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if !enabled.Enabled {
		t.Fatalf("expected created route to preserve enabled=true, got %+v", enabled)
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
	if err := service.route.CreateRoute(ctx, model.Route{Id: "01KTRAETFIKROUTE0000000001", ProjectId: &dashboardProject, Name: "traefik-dashboard", Protocol: "http", Domain: "traefik.lvh.me", PathPrefix: "/", TargetUrl: "http://traefik:8080", Enabled: true, HTTPSEnabled: true, CertType: "manual"}); err != nil {
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
	testseed.RemoveSQLiteExportedGatewaySeed(t, database)
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}
	cfg.Cert.LetsEncrypt.Enabled = true
	cfg.Cert.LetsEncrypt.Email = "admin@example.test"
	publisher := &recordingRoutePublisher{}
	client := &recordingTraefikClient{}
	service := New(
		projectrepo.NewRepository(database),
		applicationrepo.NewRepository(database),
		servicerepo.NewRepository(database),
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

func TestRouteValidationSeparatesIdentityFromCustomTargetURL(t *testing.T) {
	validTargets := []string{
		"http://host",
		"https://host",
		"http://host:8080",
		"https://api.example.test:443",
	}
	for _, target := range validTargets {
		if !validRouteIdentity("api-route", "api.example.test", "/") || !routeTargetUrlPattern.MatchString(target) {
			t.Errorf("expected target URL to be valid: %s", target)
		}
	}

	invalidTargets := []string{
		"host:8080",
		"http://",
		"http://host/path",
		"ftp://host",
		"http://host:",
	}
	for _, target := range invalidTargets {
		if routeTargetUrlPattern.MatchString(target) {
			t.Errorf("expected target URL to be invalid: %s", target)
		}
	}
}

func TestRouteServiceCreatesManagedHTTPRoute(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	target := seedHTTPRouteTarget(t, database, routeTestProjectId, "01KROUTEHTTPAPP0000000000001", "01KROUTEHTTPVERSION0000000001", "01KROUTEHTTPSERVICE0000000001")

	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "app-route", Protocol: routeProtocolHTTP, Domain: "app.example.test", PathPrefix: "/",
		ServiceId: target.Id, ComponentName: "api", EndpointProtocol: "http", EndpointContainerPort: intPtr(8080), Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if route.TargetUrl != "api-default/api/http8080" || route.ServiceId == nil || *route.ServiceId != target.Id {
		t.Fatalf("unexpected managed HTTP route: %+v", route)
	}
	snapshot := publisher.snapshots[len(publisher.snapshots)-1]
	if len(snapshot) != 1 || snapshot[0].TargetAddress != "api-api" || snapshot[0].TargetPort != 8080 {
		t.Fatalf("unexpected resolved HTTP snapshot: %+v", snapshot)
	}
	updatedName := "app-route-edited"
	emptyTargetURL := ""
	updated, err := service.UpdateRoute(ctx, routeTestUserId, route.Id, routedto.RouteUpdateInput{
		Name:      &updatedName,
		TargetUrl: &emptyTargetURL,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.TargetUrl != "api-default/api/http8080" {
		t.Fatalf("managed HTTP target URL after update = %q, want service-code display address", updated.TargetUrl)
	}

	_, err = service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "invalid-http-route", Protocol: routeProtocolHTTP, Domain: "invalid.example.test", PathPrefix: "/",
		ServiceId: target.Id, ComponentName: "api", EndpointProtocol: "tcp", EndpointContainerPort: intPtr(9090), Enabled: false,
	})
	if err == nil || apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("HTTP target protocol error = %v, want validation", err)
	}

	mixed, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "mixed-http-route", Protocol: routeProtocolHTTP, Domain: "mixed.example.test", PathPrefix: "/", TargetUrl: "http://example.test:8080",
		ServiceId: target.Id, ComponentName: "api", EndpointProtocol: "http", EndpointContainerPort: intPtr(8080), Enabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if mixed.TargetUrl != "api-default/api/http8080" {
		t.Fatalf("managed HTTP target URL = %q, want service-code display address", mixed.TargetUrl)
	}
}

func TestRouteServiceCreatesTCPRouteAndValidatesListeners(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	target := seedTCPRouteTarget(t, database, routeTestProjectId, "01KROUTETARGETAPP00000000001", "01KROUTETARGETVERSION0000001", "01KROUTETARGETSERVICE0000001")
	listenPort := 16379

	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "redis-route", Protocol: routeProtocolTCP, Domain: "redis.example.test", ListenPort: &listenPort,
		ServiceId: target.Id, ComponentName: "redis", EndpointProtocol: "tcp", EndpointContainerPort: intPtr(6379), Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if route.Protocol != routeProtocolTCP || route.ListenPort == nil || *route.ListenPort != listenPort {
		t.Fatalf("unexpected TCP route: %+v", route)
	}
	if route.TargetUrl != "redis-default/redis/tcp6379" {
		t.Fatalf("TCP target URL = %q, want service-code display address", route.TargetUrl)
	}
	if len(publisher.snapshots) == 0 {
		t.Fatal("expected TCP Route snapshot publication")
	}
	snapshot := publisher.snapshots[len(publisher.snapshots)-1]
	if len(snapshot) != 1 || snapshot[0].TargetAddress != "redis-redis" || snapshot[0].TargetPort != 6379 {
		t.Fatalf("unexpected resolved TCP snapshot: %+v", snapshot)
	}
	hostEndpointPort := 16381
	_, err = service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "local-target", Protocol: routeProtocolTCP, Domain: "local-target.example.test", ListenPort: &hostEndpointPort,
		ServiceId: target.Id, ComponentName: "redis", EndpointProtocol: "tcp", EndpointContainerPort: intPtr(6380), Enabled: false,
	})
	if err != nil {
		t.Fatalf("TCP local endpoint target error = %v", err)
	}

	_, err = service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "duplicate-port", Protocol: routeProtocolTCP, Domain: "another.example.test", ListenPort: &listenPort,
		ServiceId: target.Id, ComponentName: "redis", EndpointProtocol: "tcp", EndpointContainerPort: intPtr(6379), Enabled: true,
	})
	if err == nil || apperror.StatusCode(err) != http.StatusConflict {
		t.Fatalf("duplicate TCP listener error = %v, want conflict", err)
	}

	reservedPort := 80
	_, err = service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "reserved-port", Protocol: routeProtocolTCP, Domain: "reserved.example.test", ListenPort: &reservedPort,
		ServiceId: target.Id, ComponentName: "redis", EndpointProtocol: "tcp", EndpointContainerPort: intPtr(6379), Enabled: false,
	})
	if err == nil || apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("reserved TCP listener error = %v, want validation", err)
	}

	conflictingPort := 16380
	conflict, err := service.componentPortConflict(ctx, conflictingPort)
	if err != nil {
		t.Fatal(err)
	}
	if conflict != "" {
		t.Fatalf("stopped service host endpoint conflict = %q, want none", conflict)
	}
	if err := servicerepo.NewRepository(database).UpdateServiceStatus(ctx, target.Id, status.ServiceStatusRunning); err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "host-conflict", Protocol: routeProtocolTCP, Domain: "conflict.example.test", ListenPort: &conflictingPort,
		ServiceId: target.Id, ComponentName: "redis", EndpointProtocol: "tcp", EndpointContainerPort: intPtr(6379), Enabled: true,
	})
	if err == nil || apperror.StatusCode(err) != http.StatusConflict {
		t.Fatalf("host endpoint conflict error = %v, want conflict", err)
	}
}

func seedTCPRouteTarget(t *testing.T, database *sql.DB, projectID, appID, versionID, serviceID string) model.Service {
	t.Helper()
	ctx := context.Background()
	appRepo := applicationrepo.NewRepository(database)
	serviceRepo := servicerepo.NewRepository(database)
	app := model.Application{Id: appID, ProjectId: &projectID, Name: "Redis", Code: "redis", Kind: status.ApplicationKindStandard}
	if err := appRepo.CreateApplication(ctx, app); err != nil {
		t.Fatalf("seed TCP target application: %v", err)
	}
	conflictingPort := 16380
	version := model.Version{Id: versionID, ApplicationId: app.Id, Label: "v1", Status: status.VersionStatusPublished}
	component := model.VersionComponent{
		Id: "01KROUTETARGETCOMPONENT00001", VersionId: version.Id, Name: "redis", Image: "redis:7", PullPolicy: "missing",
		Endpoints: []model.VersionComponentEndpoint{
			{Protocol: "tcp", ContainerPort: 6379, Mode: "internal"},
			{Protocol: "tcp", ContainerPort: 6380, Mode: "local", ListenPort: &conflictingPort},
		},
	}
	if err := appRepo.CreateVersionWithVersionComponents(ctx, version, []model.VersionComponent{component}); err != nil {
		t.Fatalf("seed TCP target version: %v", err)
	}
	target := model.Service{Id: serviceID, ApplicationId: app.Id, VersionId: version.Id, InstanceKey: "default", Code: "redis-default", Status: status.ServiceStatusStopped}
	serviceComponent := model.ServiceComponent{
		Id: "01KROUTETARGETSERVICECOMP01", ServiceId: target.Id,
		SourceVersionComponentId: component.Id, ComponentName: component.Name, Status: "active",
	}
	if err := serviceRepo.CreateServiceWithComponents(ctx, target, []model.ServiceComponent{serviceComponent}); err != nil {
		t.Fatalf("seed TCP target service: %v", err)
	}
	return target
}

func seedHTTPRouteTarget(t *testing.T, database *sql.DB, projectID, appID, versionID, serviceID string) model.Service {
	t.Helper()
	ctx := context.Background()
	appRepo := applicationrepo.NewRepository(database)
	serviceRepo := servicerepo.NewRepository(database)
	app := model.Application{Id: appID, ProjectId: &projectID, Name: "API", Code: "api", Kind: status.ApplicationKindStandard}
	if err := appRepo.CreateApplication(ctx, app); err != nil {
		t.Fatalf("seed HTTP target application: %v", err)
	}
	version := model.Version{Id: versionID, ApplicationId: app.Id, Label: "v1", Status: status.VersionStatusPublished}
	component := model.VersionComponent{
		Id: "01KROUTEHTTPCOMPONENT000001", VersionId: version.Id, Name: "api", Image: "api:test", PullPolicy: "missing",
		Endpoints: []model.VersionComponentEndpoint{
			{Protocol: "http", ContainerPort: 8080, Mode: "internal"},
			{Protocol: "tcp", ContainerPort: 9090, Mode: "internal"},
		},
	}
	if err := appRepo.CreateVersionWithVersionComponents(ctx, version, []model.VersionComponent{component}); err != nil {
		t.Fatalf("seed HTTP target version: %v", err)
	}
	target := model.Service{Id: serviceID, ApplicationId: app.Id, VersionId: version.Id, InstanceKey: "default", Code: "api-default", Status: status.ServiceStatusStopped}
	serviceComponent := model.ServiceComponent{
		Id: "01KROUTEHTTPSERVICECOMP00001", ServiceId: target.Id,
		SourceVersionComponentId: component.Id, ComponentName: component.Name, Status: "active",
	}
	if err := serviceRepo.CreateServiceWithComponents(ctx, target, []model.ServiceComponent{serviceComponent}); err != nil {
		t.Fatalf("seed HTTP target service: %v", err)
	}
	return target
}

func intPtr(value int) *int {
	return &value
}

func seedRouteTestGateway(t *testing.T, database *sql.DB) {
	t.Helper()
	ctx := context.Background()
	appRepo := applicationrepo.NewRepository(database)
	gwRepo := gatewayrepo.NewRepository(database)
	projectId := routeTestProjectId
	app := model.Application{
		Id:        "01KROUTEGATEWAYAPP000000001",
		ProjectId: &projectId,
		Name:      "Test Gateway",
		Code:      "test-gateway",
		Kind:      status.ApplicationKindGateway,
	}
	if err := appRepo.CreateApplication(ctx, app); err != nil {
		t.Fatalf("seed gateway app: %v", err)
	}
	version := model.Version{Id: "01KROUTEGATEWAYVERSION00001", ApplicationId: app.Id, Label: "managed", Status: status.VersionStatusPublished}
	if err := appRepo.CreateVersionWithVersionComponents(ctx, version, []model.VersionComponent{{Id: "01KROUTEGATEWAYCOMPONENT01", VersionId: version.Id, Name: "traefik", Image: "traefik:3.6", PullPolicy: "missing"}}); err != nil {
		t.Fatalf("seed gateway version: %v", err)
	}
	if err := gwRepo.UpsertGatewayConfig(ctx, model.GatewayConfig{
		ApplicationId:     app.Id,
		RestApiUrl:        "http://traefik:8080",
		BaseDomain:        "lvh.me",
		DefaultEntrypoint: "websecure",
		TLSMode:           "optional",
	}); err != nil {
		t.Fatalf("seed gateway config: %v", err)
	}
}

type recordingRoutePublisher struct {
	snapshots    [][]model.Route
	revokedCerts []string
	readyWaits   int
}

func (p *recordingRoutePublisher) WaitUntilReady(context.Context, string) error {
	p.readyWaits++
	return nil
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

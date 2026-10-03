package routesvc

import (
	"context"
	"database/sql"
	"errors"
	"os"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	databasetx "github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	environmentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment"
	gatewayrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/gateway"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	routerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/route"
	servicerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/service"
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
	if err := service.DeleteRoute(ctx, routeTestUserId, routeTestProjectId, route.Id); err != nil {
		t.Fatal(err)
	}

	enabled, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{Name: "enabled-route", Protocol: "http", Domain: "enabled.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8082", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if !enabled.Enabled {
		t.Fatalf("expected created route to preserve enabled=true, got %+v", enabled)
	}
	if len(publisher.published) != 0 {
		t.Fatalf("expected no snapshot publication before explicit sync, got %+v", publisher.published)
	}
	if err := service.DeleteRoute(ctx, routeTestUserId, routeTestProjectId, enabled.Id); err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected enabled route delete validation error, got %v", err)
	}
	disabled, err := service.DisableRoute(ctx, routeTestUserId, routeTestProjectId, enabled.Id)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Enabled {
		t.Fatalf("expected disabled route, got %+v", disabled)
	}
	if len(publisher.published) != 0 {
		t.Fatalf("expected no snapshot publication after disable, got %+v", publisher.published)
	}

	certRoute, err := service.UploadRouteCert(ctx, routeTestUserId, routeTestProjectId, disabled.Id, "CERT", "KEY")
	if err != nil {
		t.Fatal(err)
	}
	if !certRoute.HTTPSEnabled || certRoute.CertType != "manual" {
		t.Fatalf("unexpected manual cert route: %+v", certRoute)
	}
	certRoute, err = service.EnableRouteLetsEncrypt(ctx, routeTestUserId, routeTestProjectId, disabled.Id, acmeChallengeHTTP)
	if err != nil {
		t.Fatal(err)
	}
	if !certRoute.HTTPSEnabled || certRoute.CertType != "letsencrypt" || certRoute.CertPEM != nil || certRoute.CertKey != nil {
		t.Fatalf("unexpected letsencrypt route: %+v", certRoute)
	}
	dashboardProject := routeTestProjectId
	if err := service.route.CreateRoute(ctx, model.Route{Id: "01KTRAETFIKROUTE0000000001", ProjectId: &dashboardProject, Name: "traefik-dashboard", Protocol: "http", Domain: "traefik.lvh.me", PathPrefix: "/", TargetUrl: "http://traefik:8080", Enabled: true, HTTPSEnabled: true, CertType: "manual", AcmeChallenge: acmeChallengeHTTP}); err != nil {
		t.Fatal(err)
	}
	gwRepo := gatewayrepo.NewRepository(database)
	gw, err := gwRepo.GatewayConfigByProject(ctx, routeTestProjectId)
	if err != nil {
		t.Fatal(err)
	}
	gw.ExternalDomain = "public.example.test"
	if err := gwRepo.UpsertGatewayConfig(ctx, gw); err != nil {
		t.Fatal(err)
	}
	cfg, err := service.TraefikRouteConfig(ctx, routeTestUserId, routeTestProjectId)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DashboardDomain != "traefik.lvh.me" || !cfg.HTTPSEnabled || cfg.ExternalDomain != "public.example.test" {
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
	client.unavailableMessage = "Traefik REST API is unavailable on the remote host at http://127.0.0.1:8080."
	if _, err := service.ListTraefikRoutes(ctx, routeTestUserId, routeTestProjectId); err == nil || !apperror.IsKind(err, apperror.KindUnavailable) || apperror.Classify(err).Message != client.unavailableMessage {
		t.Fatalf("expected Traefik unavailable error with access context, got %v", err)
	}
	client.unavailableMessage = ""
	if _, err := service.ListTraefikRoutes(ctx, routeTestUserId, routeTestProjectId); err == nil || !apperror.IsKind(err, apperror.KindInternal) {
		t.Fatalf("expected non-request Traefik error to remain internal, got %v", err)
	}
}

func TestRouteServiceRequiresTokenAndGatewayCapabilityForDNSLetsEncrypt(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{Name: "dns-route", Protocol: "http", Domain: "dns.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8084", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.EnableRouteLetsEncrypt(ctx, routeTestUserId, routeTestProjectId, route.Id, acmeChallengeDNS); err == nil || !strings.Contains(err.Error(), "Gateway DNS API token") {
		t.Fatalf("missing DNS token error = %v", err)
	}
	stored, err := service.route.Route(ctx, routeTestProjectId, route.Id)
	if err != nil || stored.HTTPSEnabled || stored.CertType != certTypeManual {
		t.Fatalf("route changed after token refusal: %+v, err=%v", stored, err)
	}
	gateway, err := service.gateway.GatewayConfigByProject(ctx, routeTestProjectId)
	if err != nil {
		t.Fatal(err)
	}
	gateway.DNSApiToken = "cfat_test_token"
	if err := service.gateway.UpsertGatewayConfig(ctx, gateway); err != nil {
		t.Fatal(err)
	}
	enabled, err := service.EnableRouteLetsEncrypt(ctx, routeTestUserId, routeTestProjectId, route.Id, acmeChallengeDNS)
	if err != nil {
		t.Fatal(err)
	}
	if enabled.AcmeChallenge != acmeChallengeDNS || !enabled.HTTPSEnabled || enabled.CertType != certTypeLetsEncrypt {
		t.Fatalf("DNS route = %+v", enabled)
	}
	if len(publisher.published) != 0 {
		t.Fatalf("expected no snapshot publication after certificate change, got %+v", publisher.published)
	}
	switched, err := service.EnableRouteLetsEncrypt(ctx, routeTestUserId, routeTestProjectId, route.Id, acmeChallengeHTTP)
	if err != nil || switched.AcmeChallenge != acmeChallengeHTTP {
		t.Fatalf("switch Let's Encrypt challenge = %+v, err=%v", switched, err)
	}
}

func TestRouteServiceMutationsDoNotPublishWhenTraefikHasUnmanagedFileRoute(t *testing.T) {
	service, publisher, client, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	managed, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "managed-route", Protocol: "http", Domain: "managed.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8082", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	client.routers = []routeport.TraefikRouter{{Name: "legacy-route@file", Provider: "file", Status: "enabled"}}
	published := len(publisher.published)

	if _, err := service.DisableRoute(ctx, routeTestUserId, routeTestProjectId, managed.Id); err != nil {
		t.Fatalf("expected route change to ignore unmanaged router, got %v", err)
	}
	stored, err := service.RouteForUser(ctx, routeTestUserId, routeTestProjectId, managed.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Enabled {
		t.Fatalf("route was not disabled: %+v", stored)
	}
	if len(publisher.published) != published {
		t.Fatalf("expected no snapshot publication, got %+v", publisher.published[published:])
	}
}

func TestRouteServiceProjectSyncPreservesUnmanagedResources(t *testing.T) {
	service, publisher, client, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	client.routers = []routeport.TraefikRouter{{Name: "legacy-route@file", Provider: "file", Status: "enabled"}}

	created, err := service.CreateRoute(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "legacy", Protocol: "http", Domain: "legacy.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8090", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !created.Enabled {
		t.Fatalf("expected adopted route to be enabled: %+v", created)
	}
	if len(publisher.published) != 0 {
		t.Fatalf("expected no snapshot before explicit sync, got %+v", publisher.published)
	}
	preview, err := service.PreviewRouteSync(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 || preview.Items[0].RouteId != created.Id || preview.Items[0].Action != "publish" {
		t.Fatalf("publication list=%+v", preview.Items)
	}
	if _, err := service.ConfirmRouteSync(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
		BusinessHash: preview.BusinessHash,
		RouteIds:     preview.RouteIds, PublicationHash: preview.PublicationHash,
	}); err != nil {
		t.Fatal(err)
	}
	if len(publisher.published) != 1 || publisher.published[0].Id != created.Id {
		t.Fatalf("expected explicit sync to replace the snapshot, got %+v", publisher.published)
	}
}

func TestRouteServiceSyncAppliesPendingEnableChange(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "pending-route", Protocol: "http", Domain: "pending.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8091", Enabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	preview, err := service.PreviewRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "selected", RouteIds: []string{route.Id}, Changes: []routedto.RouteSyncChange{{RouteId: route.Id, Enabled: &enabled}}})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 || preview.Items[0].Action != "publish" || preview.Items[0].Rule.Protocol != "http" || preview.Items[0].Rule.Match != "Host(`pending.example.test`)" || preview.Items[0].Rule.Target != "http://host.docker.internal:8091" {
		t.Fatalf("publication list=%+v", preview.Items)
	}
	if stored, err := service.route.Route(ctx, routeTestProjectId, route.Id); err != nil || stored.Enabled {
		t.Fatalf("preview changed business state: route=%+v err=%v", stored, err)
	}
	if _, err := service.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
		Changes:      []routedto.RouteSyncChange{{RouteId: route.Id, Enabled: &enabled}},
		BusinessHash: preview.BusinessHash,
		RouteIds:     preview.RouteIds, PublicationHash: preview.PublicationHash,
	}); err != nil {
		t.Fatal(err)
	}
	stored, err := service.route.Route(ctx, routeTestProjectId, route.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !stored.Enabled || len(publisher.published) != 1 || publisher.published[0].Id != route.Id {
		t.Fatalf("sync result = route=%+v snapshots=%+v", stored, publisher.published)
	}
}

func TestEnabledRouteCertificateChangePublishesOnlyOnSync(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "live-route", Protocol: "http", Domain: "live.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8093", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	initial, err := service.PreviewRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
		BusinessHash: initial.BusinessHash, RouteIds: initial.RouteIds, PublicationHash: initial.PublicationHash,
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.EnableRouteMkcert(ctx, routeTestUserId, routeTestProjectId, route.Id); err != nil {
		t.Fatal(err)
	}
	if len(publisher.published) != 1 {
		t.Fatalf("certificate save published route: %+v", publisher.published)
	}
	preview, err := service.PreviewRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
		BusinessHash: preview.BusinessHash, RouteIds: preview.RouteIds, PublicationHash: preview.PublicationHash,
	}); err != nil {
		t.Fatal(err)
	}
	if len(publisher.published) != 2 || !publisher.published[1].HTTPSEnabled {
		t.Fatalf("updated certificate was not published: %+v", publisher.published)
	}
}

func TestRouteServiceSyncEnablesRouteWithSavedCertificate(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "secure-route", Protocol: "http", Domain: "secure.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8093", Enabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	if _, err := service.EnableRouteMkcert(ctx, routeTestUserId, routeTestProjectId, route.Id); err != nil {
		t.Fatal(err)
	}
	changes := []routedto.RouteSyncChange{{RouteId: route.Id, Enabled: &enabled}}
	preview, err := service.PreviewRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "project", Changes: changes})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 {
		t.Fatalf("preview = %+v", preview)
	}
	stored, err := service.route.Route(ctx, routeTestProjectId, route.Id)
	if err != nil || stored.Enabled || !stored.HTTPSEnabled || len(publisher.published) != 0 {
		t.Fatalf("preview changed state: route=%+v snapshots=%+v err=%v", stored, publisher.published, err)
	}
	if _, err := service.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
		Changes: changes, BusinessHash: preview.BusinessHash, RouteIds: preview.RouteIds, PublicationHash: preview.PublicationHash,
	}); err != nil {
		t.Fatal(err)
	}
	stored, err = service.route.Route(ctx, routeTestProjectId, route.Id)
	if err != nil || !stored.Enabled || !stored.HTTPSEnabled || stored.CertType != certTypeMkcert || stored.CertPEM == nil || *stored.CertPEM != "CERT" {
		t.Fatalf("stored route = %+v, err=%v", stored, err)
	}
	if len(publisher.published) != 1 || !publisher.published[0].HTTPSEnabled {
		t.Fatalf("published routes = %+v", publisher.published)
	}
}

func TestRouteServiceSyncSavesDisabledCertificateWithoutPublishingRoute(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "offline-route", Protocol: "http", Domain: "offline.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8094", Enabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.EnableRouteLetsEncrypt(ctx, routeTestUserId, routeTestProjectId, route.Id, acmeChallengeHTTP); err != nil {
		t.Fatal(err)
	}
	stored, err := service.route.Route(ctx, routeTestProjectId, route.Id)
	if err != nil || stored.Enabled || !stored.HTTPSEnabled || stored.CertType != certTypeLetsEncrypt {
		t.Fatalf("disabled route = %+v, err=%v", stored, err)
	}
	if len(publisher.published) != 0 {
		t.Fatalf("disabled certificate change published a snapshot: %+v", publisher.published)
	}
}

func TestRouteServiceRejectsSyncAfterCertificateChangeExpiresPreview(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "stale-cert-route", Protocol: "http", Domain: "stale.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8095", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	preview, err := service.PreviewRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.UploadRouteCert(ctx, routeTestUserId, routeTestProjectId, route.Id, "CERT", "KEY"); err != nil {
		t.Fatal(err)
	}
	_, err = service.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
		BusinessHash: preview.BusinessHash, RouteIds: preview.RouteIds, PublicationHash: preview.PublicationHash,
	})
	if err == nil || apperror.Classify(err).Code != routeSyncPreviewExpiredCode || len(publisher.published) != 0 {
		t.Fatalf("expired preview result = %v, snapshots=%+v", err, publisher.published)
	}
}

func TestRouteServiceReportsSavedStateWhenPublishFails(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "retry-route", Protocol: "http", Domain: "retry.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8096", Enabled: false,
	})
	if err != nil {
		t.Fatal(err)
	}
	enabled := true
	changes := []routedto.RouteSyncChange{{RouteId: route.Id, Enabled: &enabled}}
	preview, err := service.PreviewRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "selected", RouteIds: []string{route.Id}, Changes: changes})
	if err != nil {
		t.Fatal(err)
	}
	publisher.err = os.ErrPermission
	result, err := service.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
		Changes: changes, BusinessHash: preview.BusinessHash, RouteIds: preview.RouteIds, PublicationHash: preview.PublicationHash,
	})
	if err != nil || result.Code != "route_sync_incomplete" || len(result.Results) != 1 || result.Results[0].Code != routeSyncPermissionDeniedCode || result.Results[0].BusinessSave != "saved" {
		t.Fatalf("publication result = %+v, err=%v", result, err)
	}
	stored, err := service.route.Route(ctx, routeTestProjectId, route.Id)
	if err != nil || !stored.Enabled {
		t.Fatalf("saved route = %+v, err=%v", stored, err)
	}
	publisher.err = nil
	preview, err = service.PreviewRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "selected", RouteIds: []string{route.Id}})
	if err != nil {
		t.Fatal(err)
	}
	result, err = service.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
		BusinessHash: preview.BusinessHash, RouteIds: preview.RouteIds, PublicationHash: preview.PublicationHash,
	})
	if err != nil || result.Code != "route_sync_completed" || len(publisher.published) != 1 || publisher.published[0].Id != route.Id {
		t.Fatalf("retry = %+v, err=%v", result, err)
	}
}
func TestRouteRuntimeChangeDoesNotInvalidatePublicationList(t *testing.T) {
	service, publisher, client, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	route := createSyncTestRoute(t, service, "runtime")
	preview := selectedPreview(t, service, route.Id)
	client.err = errors.New("runtime query is unavailable")
	result := confirmPreview(t, service, preview)
	if result.Code != "route_sync_completed" || len(publisher.published) != 1 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRouteServiceRenameWaitsForFullSync(t *testing.T) {
	service, publisher, client, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	route, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "managed-route", Protocol: "http", Domain: "managed.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8082", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	client.routers = []routeport.TraefikRouter{{Name: "renamed-route@file", Provider: "file", Status: "enabled"}}
	published := len(publisher.published)
	newName := "renamed-route"

	if _, err := service.UpdateRoute(ctx, routeTestUserId, routeTestProjectId, route.Id, routedto.RouteUpdateInput{Name: &newName}); err != nil {
		t.Fatalf("expected rename to ignore unmanaged router, got %v", err)
	}
	stored, err := service.RouteForUser(ctx, routeTestUserId, routeTestProjectId, route.Id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Name != newName {
		t.Fatalf("route was not renamed: %+v", stored)
	}
	if len(publisher.published) != published {
		t.Fatalf("expected no snapshot publication, got %+v", publisher.published[published:])
	}
}

func TestRouteServiceKeepsEnabledCustomTargetWhenAnotherRouteIsDisabled(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	_, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "custom-route", Protocol: "http", Domain: "custom.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8090", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	other, err := service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "other-route", Protocol: "http", Domain: "other.example.test", PathPrefix: "/", TargetUrl: "http://host.docker.internal:8091", Enabled: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.DisableRoute(ctx, routeTestUserId, routeTestProjectId, other.Id); err != nil {
		t.Fatal(err)
	}
	if len(publisher.published) != 0 {
		t.Fatalf("expected no snapshot before explicit sync, got %+v", publisher.published)
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
	clearRouteGatewaySeed(t, database)
	publisher := &recordingRoutePublisher{}
	client := &recordingTraefikClient{}
	service := New(
		projectrepo.NewRepository(database),
		applicationrepo.NewRepository(database),
		servicerepo.NewRepository(database),
		routerepo.NewRepository(database),
		gatewayrepo.NewRepository(database),
		publisher,
		recordingCertificateGenerator{},
		client,
		databasetx.NewTransactionRunner(database),
		testRouteSyncTimeouts(),
	)
	seedRouteTestGateway(t, database)
	return service, publisher, client, database
}

func TestRouteValidationSeparatesIdentityFromCustomTargetUrl(t *testing.T) {
	if !validRouteCode("a"+strings.Repeat("b", 31)) || !validRouteCode("api1") || validRouteCode("1-api") || validRouteCode("123") || validRouteCode(strings.Repeat("a", 33)) || validRouteCode("bad_name") || validRouteCode("bad.name") || validRouteCode("bad-") {
		t.Fatal("route code DNS label validation is inconsistent")
	}
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
	if len(publisher.published) != 0 {
		t.Fatalf("expected no HTTP snapshot before explicit sync, got %+v", publisher.published)
	}
	preview, err := service.PreviewRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
		BusinessHash: preview.BusinessHash,
		RouteIds:     preview.RouteIds, PublicationHash: preview.PublicationHash,
	}); err != nil {
		t.Fatal(err)
	}
	publishedRoute := publisher.published[len(publisher.published)-1]
	if publishedRoute.TargetAddress != "api-api" || publishedRoute.TargetPort != 8080 {
		t.Fatalf("unexpected resolved HTTP snapshot: %+v", publishedRoute)
	}
	updatedName := "app-route-edited"
	emptyTargetUrl := ""
	updated, err := service.UpdateRoute(ctx, routeTestUserId, routeTestProjectId, route.Id, routedto.RouteUpdateInput{
		Name:      &updatedName,
		TargetUrl: &emptyTargetUrl,
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
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) {
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
	if len(publisher.published) != 0 {
		t.Fatalf("expected no TCP snapshot before explicit sync, got %+v", publisher.published)
	}
	preview, err := service.PreviewRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
		BusinessHash: preview.BusinessHash,
		RouteIds:     preview.RouteIds, PublicationHash: preview.PublicationHash,
	}); err != nil {
		t.Fatal(err)
	}
	publishedRoute := publisher.published[len(publisher.published)-1]
	if publishedRoute.TargetAddress != "redis-redis" || publishedRoute.TargetPort != 6379 {
		t.Fatalf("unexpected resolved TCP snapshot: %+v", publishedRoute)
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
	if err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("duplicate TCP listener error = %v, want conflict", err)
	}

	reservedPort := 80
	_, err = service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "reserved-port", Protocol: routeProtocolTCP, Domain: "reserved.example.test", ListenPort: &reservedPort,
		ServiceId: target.Id, ComponentName: "redis", EndpointProtocol: "tcp", EndpointContainerPort: intPtr(6379), Enabled: false,
	})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("reserved TCP listener error = %v, want validation", err)
	}

	conflictingPort := 16380
	conflict, err := service.componentPortConflict(ctx, routeTestProjectId, conflictingPort)
	if err != nil {
		t.Fatal(err)
	}
	if conflict != "" {
		t.Fatalf("stopped service host endpoint conflict = %q, want none", conflict)
	}
	if err := servicerepo.NewRepository(database).UpdateServiceStatus(ctx, routeTestProjectId, target.Id, status.ServiceStatusRunning); err != nil {
		t.Fatal(err)
	}
	_, err = service.CreateRoute(ctx, routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{
		Name: "host-conflict", Protocol: routeProtocolTCP, Domain: "conflict.example.test", ListenPort: &conflictingPort,
		ServiceId: target.Id, ComponentName: "redis", EndpointProtocol: "tcp", EndpointContainerPort: intPtr(6379), Enabled: true,
	})
	if err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("host endpoint conflict error = %v, want conflict", err)
	}
}

func seedTCPRouteTarget(t *testing.T, database *sql.DB, projectId, appId, versionId, serviceId string) model.Service {
	t.Helper()
	ctx := context.Background()
	appRepo := applicationrepo.NewRepository(database)
	serviceRepo := servicerepo.NewRepository(database)
	app := model.Application{Id: appId, ProjectId: &projectId, Name: "Redis", Code: "redis", Kind: status.ApplicationKindStandard}
	if err := appRepo.CreateApplication(ctx, app); err != nil {
		t.Fatalf("seed TCP target application: %v", err)
	}
	conflictingPort := 16380
	version := model.Version{Id: versionId, ApplicationId: app.Id, Label: "v1", Status: status.VersionStatusPublished}
	component := model.VersionComponent{
		Id: "01KROUTETARGETCOMPONENT00001", VersionId: version.Id, Name: "redis", Image: "redis:7", PullPolicy: "missing",
		Endpoints: []model.VersionComponentEndpoint{
			{Protocol: "tcp", ContainerPort: 6379, Mode: "internal"},
			{Protocol: "tcp", ContainerPort: 6380, Mode: "local", ListenPort: &conflictingPort},
		},
	}
	if err := appRepo.CreateVersionWithVersionComponents(ctx, projectId, version, []model.VersionComponent{component}); err != nil {
		t.Fatalf("seed TCP target version: %v", err)
	}
	target := model.Service{Id: serviceId, ProjectId: projectId, ApplicationId: app.Id, VersionId: version.Id, Code: "redis-default", Status: status.ServiceStatusStopped}
	serviceComponent := model.ServiceComponent{
		Id: "01KROUTETARGETSERVICECOMP01", ServiceId: target.Id,
		SourceVersionComponentId: component.Id, ComponentName: component.Name, Status: "active",
	}
	if err := serviceRepo.CreateServiceWithComponents(ctx, projectId, target, []model.ServiceComponent{serviceComponent}); err != nil {
		t.Fatalf("seed TCP target service: %v", err)
	}
	return target
}

func seedHTTPRouteTarget(t *testing.T, database *sql.DB, projectId, appId, versionId, serviceId string) model.Service {
	t.Helper()
	ctx := context.Background()
	appRepo := applicationrepo.NewRepository(database)
	serviceRepo := servicerepo.NewRepository(database)
	app := model.Application{Id: appId, ProjectId: &projectId, Name: "API", Code: "api", Kind: status.ApplicationKindStandard}
	if err := appRepo.CreateApplication(ctx, app); err != nil {
		t.Fatalf("seed HTTP target application: %v", err)
	}
	version := model.Version{Id: versionId, ApplicationId: app.Id, Label: "v1", Status: status.VersionStatusPublished}
	component := model.VersionComponent{
		Id: "01KROUTEHTTPCOMPONENT000001", VersionId: version.Id, Name: "api", Image: "api:test", PullPolicy: "missing",
		Endpoints: []model.VersionComponentEndpoint{
			{Protocol: "http", ContainerPort: 8080, Mode: "internal"},
			{Protocol: "tcp", ContainerPort: 9090, Mode: "internal"},
		},
	}
	if err := appRepo.CreateVersionWithVersionComponents(ctx, projectId, version, []model.VersionComponent{component}); err != nil {
		t.Fatalf("seed HTTP target version: %v", err)
	}
	target := model.Service{Id: serviceId, ProjectId: projectId, ApplicationId: app.Id, VersionId: version.Id, Code: "api-default", Status: status.ServiceStatusStopped}
	serviceComponent := model.ServiceComponent{
		Id: "01KROUTEHTTPSERVICECOMP00001", ServiceId: target.Id,
		SourceVersionComponentId: component.Id, ComponentName: component.Name, Status: "active",
	}
	if err := serviceRepo.CreateServiceWithComponents(ctx, projectId, target, []model.ServiceComponent{serviceComponent}); err != nil {
		t.Fatalf("seed HTTP target service: %v", err)
	}
	return target
}

func intPtr(value int) *int {
	return &value
}

func clearRouteGatewaySeed(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, err := database.Exec("DELETE FROM route WHERE id = '01M01ZNW6CPQCB7P5PN669HWJ9'"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM service_component WHERE service_id = '01M01RHDXW3ZXC7YKNT54RWM1M'"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM service WHERE application_id = '01M01MP0950ECGK2DS1FWYNC0B'"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM application WHERE id = '01M01MP0950ECGK2DS1FWYNC0B'"); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM environment WHERE project_id = ?", routeTestProjectId); err != nil {
		t.Fatal(err)
	}
}

func seedRouteTestGateway(t *testing.T, database *sql.DB) {
	t.Helper()
	ctx := context.Background()
	appRepo := applicationrepo.NewRepository(database)
	environmentRepo := environmentrepo.NewRepository(database)
	gwRepo := gatewayrepo.NewRepository(database)
	serviceRepo := servicerepo.NewRepository(database)
	projectId := routeTestProjectId
	app := model.Application{
		Id:        "01KROUTEGATEWAYAPP000000001",
		ProjectId: &projectId,
		Name:      "Test Gateway",
		Code:      "test-gateway",
		Kind:      status.ApplicationKindStandard,
	}
	if err := appRepo.CreateApplication(ctx, app); err != nil {
		t.Fatalf("seed gateway app: %v", err)
	}
	version := model.Version{Id: "01KROUTEGATEWAYVERSION00001", ApplicationId: app.Id, Label: "managed", Status: status.VersionStatusPublished}
	component := model.VersionComponent{Id: "01KROUTEGATEWAYCOMPONENT01", VersionId: version.Id, Name: "traefik", Image: "traefik:3.6", PullPolicy: "missing"}
	if err := appRepo.CreateVersionWithVersionComponents(ctx, projectId, version, []model.VersionComponent{component}); err != nil {
		t.Fatalf("seed gateway version: %v", err)
	}
	gatewayService := model.Service{Id: "01KROUTEGATEWAYSERVICE0000001", ProjectId: projectId, ApplicationId: app.Id, VersionId: version.Id, Code: "test-gateway-default", Status: status.ServiceStatusStopped}
	if err := serviceRepo.CreateServiceWithComponents(ctx, projectId, gatewayService, []model.ServiceComponent{{Id: "01KROUTEGATEWAYSERVICECOMP0001", ServiceId: gatewayService.Id, SourceVersionComponentId: component.Id, ComponentName: component.Name, Status: "active"}}); err != nil {
		t.Fatalf("seed gateway service: %v", err)
	}
	if err := gwRepo.UpsertGatewayConfig(ctx, model.GatewayConfig{
		ApplicationId: app.Id,
		RestApiUrl:    model.GatewayRestApiContainerUrl, RestApiHostUrl: model.GatewayRestApiHostUrl, RestReadyTimeoutSeconds: 20,
		InternalDomain: "lvh.me", DefaultEntrypoint: "websecure", TLSMode: "none",
		AcmeProfile: "http-dns", AcmeEmail: "admin@example.test",
	}); err != nil {
		t.Fatalf("seed gateway config: %v", err)
	}
	environment := model.Environment{
		Id:             "01KROUTEENVIRONMENT0000001",
		ProjectId:      projectId,
		Code:           "route-test",
		TargetType:     model.EnvironmentTargetTypeSSH,
		WorkspaceRoot:  "/srv/pomelo-orbit",
		TargetRevision: 1,
		SSH: &model.EnvironmentSSHTarget{
			Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22, Username: "deploy",
			CredentialId: "route-test-credential", CredentialRevision: 1,
			HostKeyFingerprint: "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		},
	}
	if err := environmentRepo.CreateEnvironment(ctx, environment); err != nil {
		t.Fatalf("seed route environment: %v", err)
	}
	bound, err := environmentRepo.BindGatewayApplication(ctx, environment.Id, app.Id)
	if err != nil {
		t.Fatalf("bind route gateway environment: %v", err)
	}
	if !bound {
		t.Fatal("route gateway environment was already bound")
	}
}

type recordingRoutePublisher struct {
	published    []model.Route
	publications map[string]routeport.Publication
	failures     map[string]error
	attempts     []string
	dependencies string
	readyWaits   int
	err          error
}

func (p *recordingRoutePublisher) WaitUntilReady(context.Context, string, model.GatewayConfig, time.Duration) error {
	p.readyWaits++
	return nil
}

func (p *recordingRoutePublisher) LockGateway(context.Context, string) (func(), error) {
	return func() {}, nil
}
func (p *recordingRoutePublisher) ValidateGateway(context.Context, string, model.GatewayConfig, []model.Route) (string, error) {
	return p.dependencies, nil
}
func (p *recordingRoutePublisher) VerifyPublished(context.Context, string, model.GatewayConfig) error {
	return nil
}
func (p *recordingRoutePublisher) InspectPublications(context.Context, string, model.GatewayConfig) ([]routeport.Publication, error) {
	items := []routeport.Publication{}
	for _, item := range p.publications {
		items = append(items, item)
	}
	return items, nil
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
	routers            []routeport.TraefikRouter
	services           []routeport.TraefikService
	err                error
	unavailableMessage string
}

func (c *recordingTraefikClient) ListRouters(_ context.Context, _ string, _ model.GatewayConfig) ([]routeport.TraefikRouter, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.routers, nil
}

func (c *recordingTraefikClient) ListServices(_ context.Context, _ string, _ model.GatewayConfig) ([]routeport.TraefikService, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.services, nil
}

func (c *recordingTraefikClient) TraefikUnavailableMessage(err error) (string, bool) {
	return c.unavailableMessage, c.unavailableMessage != "" && errors.Is(err, c.err)
}

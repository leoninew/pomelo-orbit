package routesvc

import (
	"context"
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

func TestRouteSyncBoundsGatewayLockWait(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service, publisher, _, _ := newRouteRuleService()
		service.syncTimeouts.GatewayLock = 50 * time.Millisecond
		blocked := &deadlineLockPublisher{RouteConfigPublisher: publisher}
		service.routePublisher = blocked
		started := time.Now()
		_, err := service.ConfirmRouteSync(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{BusinessHash: "business", PublicationHash: "publication"})
		if !errors.Is(err, context.DeadlineExceeded) {
			t.Fatalf("expected timeout while waiting for Gateway lock, got %v", err)
		}
		if blocked.deadline.IsZero() || blocked.deadline.After(started.Add(service.syncTimeouts.GatewayLock+time.Millisecond)) {
			t.Fatalf("Gateway lock did not receive its stage deadline: %s", blocked.deadline)
		}
	})
}

func TestBatchSyncContinuesAfterFailureInFrozenOrder(t *testing.T) {
	s, publisher, _, facts := newRouteRuleService("a", "b", "c")
	a, b, c := facts.routes["a"], facts.routes["b"], facts.routes["c"]
	preview := selectedPreview(t, s, a.Id, b.Id, c.Id)
	publisher.failures = map[string]error{b.Id: os.ErrPermission}
	result := confirmPreview(t, s, preview)
	if result.Code != "route_sync_incomplete" || len(result.Results) != 3 || result.Results[0].Code != "route_sync_completed" || result.Results[1].Code != routeSyncPermissionDeniedCode || result.Results[2].Code != "route_sync_completed" {
		t.Fatalf("batch=%+v", result)
	}
	if !reflect.DeepEqual(publisher.attempts, []string{a.Id, b.Id, c.Id}) || len(publisher.published) != 2 {
		t.Fatalf("attempts=%v published=%+v", publisher.attempts, publisher.published)
	}
}

func TestRouteSyncPreservesSafeConfigurationFailureReason(t *testing.T) {
	service, publisher, _, facts := newRouteRuleService("api")
	route := facts.routes["api"]
	preview := selectedPreview(t, service, route.Id)
	message := "Traefik API queries failed or timed out before the Route configuration could be confirmed"
	publisher.failures = map[string]error{route.Id: apperror.WrapWithCode(apperror.KindUnavailable, "route_sync_configuration_unavailable", message, errors.New("private diagnostic"))}
	result := confirmPreview(t, service, preview)
	if result.Code != "route_sync_incomplete" || len(result.Results) != 1 || result.Results[0].Code != "route_sync_configuration_unavailable" || result.Results[0].Error != message {
		t.Fatalf("configuration failure was replaced by a generic publication message: %+v", result)
	}
}

func TestRouteSyncContinuesAfterItemTimeout(t *testing.T) {
	service, publisher, _, facts := newRouteRuleService("a", "b")
	a, b := facts.routes["a"], facts.routes["b"]
	preview := selectedPreview(t, service, a.Id, b.Id)
	publisher.failures = map[string]error{a.Id: errors.Join(context.DeadlineExceeded, errors.New("private SFTP diagnostic"))}
	result := confirmPreview(t, service, preview)
	if result.Code != "route_sync_incomplete" || len(result.Results) != 2 || result.Results[0].Code != "route_sync_publish_timeout" || result.Results[0].Error != "The Route file publication step timed out." || result.Results[1].Code != "route_sync_completed" {
		t.Fatalf("item timeout did not retain its safe cause and continue the batch: %+v", result)
	}
	if !reflect.DeepEqual(publisher.attempts, []string{a.Id, b.Id}) {
		t.Fatalf("item timeout skipped later Routes: %v", publisher.attempts)
	}
}

func TestProjectPreviewSkipsPublicationDirectoriesWithoutRecords(t *testing.T) {
	service, publisher, _, facts := newRouteRuleService("api")
	route := facts.routes["api"]
	publisher.listedIds = []string{"empty-publication-directory"}
	preview, err := service.PreviewRouteSync(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "project"})
	if err != nil || !reflect.DeepEqual(preview.RouteIds, []string{route.Id}) || len(preview.Items) != 1 {
		t.Fatalf("empty publication directory blocked the business scope: preview=%+v err=%v", preview, err)
	}
	if result := confirmPreview(t, service, preview); result.Code != "route_sync_completed" {
		t.Fatalf("project confirmation=%+v", result)
	}
}

func TestTargetChangeInvalidatesFirstPublicationPreview(t *testing.T) {
	s, publisher, _, facts := newRouteRuleService("first")
	route := facts.routes["first"]
	preview := selectedPreview(t, s, route.Id)
	publisher.dependencies = "new-target-revision"
	_, err := s.ConfirmRouteSync(context.Background(), routeTestUserId, routeTestProjectId, confirmInput(preview))
	if err == nil || apperror.Classify(err).Code != routeSyncPreviewExpiredCode || len(publisher.published) != 0 {
		t.Fatalf("err=%v published=%v", err, publisher.published)
	}
}

type deadlineLockPublisher struct {
	routeport.RouteConfigPublisher
	deadline time.Time
}

func (publisher *deadlineLockPublisher) LockGateway(ctx context.Context, _ string) (func(), error) {
	publisher.deadline, _ = ctx.Deadline()
	<-ctx.Done()
	return nil, ctx.Err()
}

func TestRouteRuntimeChangeDoesNotInvalidatePublicationList(t *testing.T) {
	service, publisher, client, facts := newRouteRuleService("runtime")
	route := facts.routes["runtime"]
	preview := selectedPreview(t, service, route.Id)
	client.err = errors.New("runtime query is unavailable")
	result := confirmPreview(t, service, preview)
	if result.Code != "route_sync_completed" || len(publisher.published) != 1 || client.queries != 0 {
		t.Fatalf("result=%+v", result)
	}
}

func TestRouteLetsEncryptRequiresDNSTokenBeforeSavingChallenge(t *testing.T) {
	service, publisher, _, facts := newRouteRuleService("dns-route")
	ctx := context.Background()
	route := facts.routes["dns-route"]
	if _, err := service.EnableRouteLetsEncrypt(ctx, routeTestUserId, routeTestProjectId, route.Id, acmeChallengeDNS); err == nil || !strings.Contains(err.Error(), "Gateway DNS API token") {
		t.Fatalf("missing DNS token error = %v", err)
	}
	stored, err := service.route.Route(ctx, routeTestProjectId, route.Id)
	if err != nil || stored.HTTPSEnabled || stored.CertType != certTypeManual || facts.updates != 0 {
		t.Fatalf("route changed after token refusal: %+v, err=%v", stored, err)
	}
	gateway, err := service.gateway.GatewayConfigByProject(ctx, routeTestProjectId)
	if err != nil {
		t.Fatal(err)
	}
	gateway.DNSApiToken = "cfat_test_token"
	facts.gateway = gateway
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

func TestTraefikViewsCopyPortDataAndMapRequestFailures(t *testing.T) {
	service, _, client, _ := newRouteRuleService()
	ctx := context.Background()
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

func TestSelectedSyncDoesNotCarryAnotherRoutesUnsyncedEdit(t *testing.T) {
	s, publisher, _, facts := newRouteRuleService("a", "b")
	a := facts.routes["a"]
	b := facts.routes["b"]
	preview := selectedPreview(t, s, a.Id)
	target := "http://new-b:8080"
	b.TargetUrl = target
	facts.routes[b.Id] = b
	result := confirmPreview(t, s, preview)
	if result.Code != "route_sync_completed" || len(publisher.published) != 1 || publisher.published[0].Id != a.Id {
		t.Fatalf("result=%+v published=%+v", result, publisher.published)
	}
	if publisher.lists != 0 || len(publisher.inspected) != 2 {
		t.Fatalf("selected synchronization listed publications or repeated inspection: lists=%d inspected=%+v", publisher.lists, publisher.inspected)
	}
	for _, route := range publisher.inspected {
		if route.Id != a.Id || route.Name != a.Name {
			t.Fatalf("selected synchronization inspected another Route or omitted its business code: %+v", route)
		}
	}
}

func TestItemRevisionRejectsEditedRouteWithoutInvalidatingOtherItems(t *testing.T) {
	s, publisher, _, facts := newRouteRuleService("a", "b", "c")
	ctx := context.Background()
	a, b, c := facts.routes["a"], facts.routes["b"], facts.routes["c"]
	preview := selectedPreview(t, s, a.Id, b.Id, c.Id)
	target := "http://unreviewed-b:8080"
	b.TargetUrl = target
	facts.routes[b.Id] = b
	for _, item := range preview.Items {
		result, err := s.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{
			RouteIds: []string{item.RouteId}, BusinessHash: item.BusinessHash, PublicationHash: item.PublicationHash,
		})
		if item.RouteId == b.Id {
			if err == nil || apperror.Classify(err).Code != routeSyncPreviewExpiredCode {
				t.Fatalf("edited Route result=%+v err=%v", result, err)
			}
		} else if err != nil || result.Code != "route_sync_completed" {
			t.Fatalf("item=%+v result=%+v err=%v", item, result, err)
		}
	}
	if !reflect.DeepEqual(publisher.attempts, []string{a.Id, c.Id}) {
		t.Fatalf("attempts=%v", publisher.attempts)
	}
}

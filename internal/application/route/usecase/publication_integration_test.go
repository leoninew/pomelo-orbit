package routesvc

import (
	"context"
	"errors"
	"os"
	"reflect"
	"testing"
	"time"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func testRouteSyncTimeouts() routeport.SyncTimeouts {
	return routeport.SyncTimeouts{Total: 30 * time.Second, ApiRequest: 3 * time.Second, Reload: 5 * time.Second, ConfigurationMatch: 10 * time.Second, Recovery: 10 * time.Second}
}

func TestRouteSyncTotalBudgetIncludesGatewayLock(t *testing.T) {
	service, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	service.syncTimeouts.Total = 50 * time.Millisecond
	blocked := &deadlineLockPublisher{RouteConfigPublisher: publisher}
	service.routePublisher = blocked
	started := time.Now()
	_, err := service.ConfirmRouteSync(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{BusinessHash: "business", PublicationHash: "publication"})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("expected total timeout while waiting for Gateway lock, got %v", err)
	}
	if blocked.deadline.IsZero() || blocked.deadline.After(started.Add(service.syncTimeouts.Total+time.Millisecond)) {
		t.Fatalf("Gateway lock did not receive the total deadline: %s", blocked.deadline)
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

func TestSelectedSyncDoesNotCarryAnotherRoutesUnsyncedEdit(t *testing.T) {
	s, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	a := createSyncTestRoute(t, s, "a")
	b := createSyncTestRoute(t, s, "b")
	preview := selectedPreview(t, s, a.Id)
	target := "http://new-b:8080"
	if _, err := s.UpdateRoute(ctx, routeTestUserId, routeTestProjectId, b.Id, routedto.RouteUpdateInput{TargetUrl: &target}); err != nil {
		t.Fatal(err)
	}
	result := confirmPreview(t, s, preview)
	if result.Code != "route_sync_completed" || len(publisher.published) != 1 || publisher.published[0].Id != a.Id {
		t.Fatalf("result=%+v published=%+v", result, publisher.published)
	}
}

func TestPreviewRuleProtocolDescribesIngressRatherThanUpstream(t *testing.T) {
	for _, item := range []struct {
		protocol string
		https    bool
		target   string
	}{
		{protocol: "http", target: "https://upstream:8443"},
		{protocol: "https", https: true, target: "http://upstream:8080"},
	} {
		t.Run(item.protocol, func(t *testing.T) {
			rule := plannedRouteRule(model.Route{
				Protocol: "http", HTTPSEnabled: item.https, Domain: "api.example.test", PathPrefix: "/api", TargetUrl: item.target,
			})
			if rule.Protocol != item.protocol || rule.Match != "Host(`api.example.test`) && PathPrefix(`/api`)" || rule.Target != item.target {
				t.Fatalf("rule=%+v", rule)
			}
		})
	}
}

func TestBatchSyncContinuesAfterFailureInFrozenOrder(t *testing.T) {
	s, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	a, b, c := createSyncTestRoute(t, s, "a"), createSyncTestRoute(t, s, "b"), createSyncTestRoute(t, s, "c")
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

func TestProjectPreviewCanBeConfirmedOneRouteAtATime(t *testing.T) {
	s, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	a, b, c := createSyncTestRoute(t, s, "a"), createSyncTestRoute(t, s, "b"), createSyncTestRoute(t, s, "c")
	confirmPreview(t, s, selectedPreview(t, s, a.Id, b.Id))
	disabled := false
	changes := []routedto.RouteSyncChange{{RouteId: a.Id, Enabled: &disabled}, {RouteId: c.Id, Enabled: &disabled}}
	preview, err := s.PreviewRouteSync(ctx, routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "project", Changes: changes})
	if err != nil {
		t.Fatal(err)
	}
	actions := map[string]string{a.Id: "withdraw", b.Id: "publish", c.Id: "skip"}
	wantAttempts := []string{a.Id, b.Id}
	for _, item := range preview.Items {
		if item.Action != actions[item.RouteId] {
			t.Fatalf("item=%+v", item)
		}
		if item.Action != "skip" {
			wantAttempts = append(wantAttempts, item.RouteId)
		}
		input := routedto.RouteSyncConfirmInput{RouteIds: []string{item.RouteId}, BusinessHash: item.BusinessHash, PublicationHash: item.PublicationHash, Changes: []routedto.RouteSyncChange{}}
		for _, change := range changes {
			if change.RouteId == item.RouteId {
				input.Changes = append(input.Changes, change)
			}
		}
		result, err := s.ConfirmRouteSync(ctx, routeTestUserId, routeTestProjectId, input)
		if err != nil || result.Code != "route_sync_completed" || len(result.Results) != 1 || result.Results[0].RouteId != item.RouteId {
			t.Fatalf("item=%+v result=%+v err=%v", item, result, err)
		}
	}
	if !reflect.DeepEqual(publisher.attempts, wantAttempts) || len(publisher.published) != 4 {
		t.Fatalf("attempts=%v published=%+v", publisher.attempts, publisher.published)
	}
}

func TestItemRevisionRejectsEditedRouteWithoutInvalidatingOtherItems(t *testing.T) {
	s, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	a, b, c := createSyncTestRoute(t, s, "a"), createSyncTestRoute(t, s, "b"), createSyncTestRoute(t, s, "c")
	preview := selectedPreview(t, s, a.Id, b.Id, c.Id)
	target := "http://unreviewed-b:8080"
	if _, err := s.UpdateRoute(ctx, routeTestUserId, routeTestProjectId, b.Id, routedto.RouteUpdateInput{TargetUrl: &target}); err != nil {
		t.Fatal(err)
	}
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

func TestProjectPreviewWithdrawsDeletedPublishedRoute(t *testing.T) {
	s, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	route := createSyncTestRoute(t, s, "deleted")
	confirmPreview(t, s, selectedPreview(t, s, route.Id))
	if _, err := s.DisableRoute(context.Background(), routeTestUserId, routeTestProjectId, route.Id); err != nil {
		t.Fatal(err)
	}
	if err := s.DeleteRoute(context.Background(), routeTestUserId, routeTestProjectId, route.Id); err != nil {
		t.Fatal(err)
	}
	preview, err := s.PreviewRouteSync(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "project"})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(preview.RouteIds, []string{route.Id}) {
		t.Fatalf("frozen IDs=%v", preview.RouteIds)
	}
	confirmPreview(t, s, preview)
	if publisher.published[1].Enabled {
		t.Fatal("deleted Route was republished")
	}
}

func TestTargetChangeInvalidatesFirstPublicationPreview(t *testing.T) {
	s, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	route := createSyncTestRoute(t, s, "first")
	preview := selectedPreview(t, s, route.Id)
	publisher.dependencies = "new-target-revision"
	_, err := s.ConfirmRouteSync(context.Background(), routeTestUserId, routeTestProjectId, confirmInput(preview))
	if err == nil || apperror.Classify(err).Code != routeSyncPreviewExpiredCode || len(publisher.published) != 0 {
		t.Fatalf("err=%v published=%v", err, publisher.published)
	}
}

func TestDisabledUnpublishedRouteSavesDraftWithoutPublishing(t *testing.T) {
	s, publisher, _, database := newRouteIntegrationService(t)
	defer func() { _ = database.Close() }()
	route := createSyncTestRoute(t, s, "disabled")
	disabled := false
	changes := []routedto.RouteSyncChange{{RouteId: route.Id, Enabled: &disabled}}
	preview, err := s.PreviewRouteSync(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteSyncPreviewInput{Scope: "selected", RouteIds: []string{route.Id}, Changes: changes})
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 1 || preview.Items[0].Action != "skip" {
		t.Fatalf("preview=%+v", preview)
	}
	input := confirmInput(preview)
	input.Changes = changes
	result, err := s.ConfirmRouteSync(context.Background(), routeTestUserId, routeTestProjectId, input)
	if err != nil {
		t.Fatal(err)
	}
	if result.Code != "route_sync_completed" || result.Results[0].BusinessSave != "saved" || result.Results[0].FileCommit != "not_applicable" || len(publisher.attempts) != 0 || len(publisher.publications) != 0 {
		t.Fatalf("result=%+v publisher=%+v", result, publisher)
	}
	saved, err := s.route.Route(context.Background(), routeTestProjectId, route.Id)
	if err != nil || saved.Enabled {
		t.Fatalf("saved=%+v err=%v", saved, err)
	}
}

func createSyncTestRoute(t *testing.T, s Service, name string) model.Route {
	t.Helper()
	route, err := s.CreateRoute(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteCreateInput{Name: name, Protocol: "http", Domain: name + ".example.test", PathPrefix: "/", TargetUrl: "http://" + name + ":8080", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	return route
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

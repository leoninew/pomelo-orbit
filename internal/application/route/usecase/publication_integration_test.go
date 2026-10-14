//go:build !unit

package routesvc

import (
	"context"
	"reflect"
	"testing"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

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

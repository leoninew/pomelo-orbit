package routesvc

import (
	"context"
	"os"
	"reflect"
	"strings"
	"testing"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestRecreatedRouteSyncAutomaticallyWithdrawsDeletedPublication(t *testing.T) {
	for _, scope := range []string{"selected", "project"} {
		for _, enabled := range []bool{true, false} {
			name := scope + "/disabled"
			if enabled {
				name = scope + "/enabled"
			}
			t.Run(name, func(t *testing.T) {
				service, publisher, _, facts := newRouteRuleService("minio")
				current := facts.routes["minio"]
				old := current
				old.Id = "previous-minio"
				facts.routes[old.Id] = old
				delete(facts.routes, current.Id)
				confirmPreview(t, service, selectedPreview(t, service, old.Id))
				delete(facts.routes, old.Id)
				publisher.owners = map[string][]string{current.Name: {old.Id}}
				current.Enabled, current.TargetUrl = enabled, "http://new-minio:9001"
				facts.routes[current.Id] = current
				publisher.attempts, publisher.published = nil, nil
				input := routedto.RouteSyncPreviewInput{Scope: scope}
				if scope == "selected" {
					input.RouteIds = []string{current.Id}
				}
				preview, err := service.PreviewRouteSync(context.Background(), routeTestUserId, routeTestProjectId, input)
				if err != nil {
					t.Fatal(err)
				}
				action := "withdraw"
				wantAttempts := []string{old.Id}
				if enabled {
					action = "publish"
					wantAttempts = append(wantAttempts, current.Id)
				}
				if !reflect.DeepEqual(preview.RouteIds, []string{current.Id}) || len(preview.Items) != 1 || preview.Items[0].Action != action {
					t.Fatalf("recreated Route exposed historical IDs or skipped cleanup: %+v", preview)
				}
				// Project previews must also support confirmation of this one item.
				item := preview.Items[0]
				result, err := service.ConfirmRouteSync(context.Background(), routeTestUserId, routeTestProjectId, routedto.RouteSyncConfirmInput{RouteIds: []string{item.RouteId}, BusinessHash: item.BusinessHash, PublicationHash: item.PublicationHash})
				if err != nil || result.Code != "route_sync_completed" || len(result.Results) != 1 || result.Results[0].RouteId != current.Id || !reflect.DeepEqual(publisher.attempts, wantAttempts) {
					t.Fatalf("result=%+v err=%v attempts=%v", result, err, publisher.attempts)
				}
				if publisher.publications[old.Id].Route.Enabled || enabled && publisher.publications[current.Id].Route.TargetUrl != current.TargetUrl {
					t.Fatalf("publication did not reach the current desired state: %+v", publisher.publications)
				}
			})
		}
	}
}

func TestOrphanPublicationChangeExpiresRecreatedRoutePreview(t *testing.T) {
	for _, change := range []string{"file", "owner restored"} {
		t.Run(change, func(t *testing.T) {
			service, publisher, _, facts := newRouteRuleService("minio")
			old := facts.routes["minio"]
			old.Id = "previous-minio"
			publisher.publications = map[string]routeport.Publication{old.Id: {Route: old, Phase: "confirmed", Fingerprint: "original", ActualFingerprint: "original"}}
			publisher.owners = map[string][]string{"minio": {old.Id}}
			preview := selectedPreview(t, service, "minio")
			if change == "file" {
				publication := publisher.publications[old.Id]
				publication.ActualFingerprint = "external edit"
				publisher.publications[old.Id] = publication
			} else {
				facts.routes[old.Id] = old
			}
			_, err := service.ConfirmRouteSync(context.Background(), routeTestUserId, routeTestProjectId, confirmInput(preview))
			if err == nil || apperror.Classify(err).Code != routeSyncPreviewExpiredCode || len(publisher.attempts) != 0 {
				t.Fatalf("changed orphan state was published: err=%v attempts=%v", err, publisher.attempts)
			}
		})
	}
}

func TestOrphanWithdrawalFailureStopsCurrentRouteAndAllowsRetry(t *testing.T) {
	service, publisher, _, facts := newRouteRuleService("minio", "other")
	old := facts.routes["minio"]
	old.Id = "previous-minio"
	publisher.publications = map[string]routeport.Publication{old.Id: {Route: old, Phase: "confirmed"}}
	publisher.owners = map[string][]string{"minio": {old.Id}}
	publisher.failures = map[string]error{old.Id: os.ErrPermission}
	result := confirmPreview(t, service, selectedPreview(t, service, "minio", "other"))
	if result.Code != "route_sync_incomplete" || result.Results[0].RouteId != "minio" || result.Results[0].Code != routeSyncPermissionDeniedCode || result.Results[1].Code != "route_sync_completed" || !reflect.DeepEqual(publisher.attempts, []string{old.Id, "other"}) {
		t.Fatalf("cleanup failure did not stop publication of its Route: result=%+v attempts=%v", result, publisher.attempts)
	}
	publisher.failures, publisher.attempts = nil, nil
	result = confirmPreview(t, service, selectedPreview(t, service, "minio"))
	if result.Code != "route_sync_completed" || !reflect.DeepEqual(publisher.attempts, []string{old.Id, "minio"}) {
		t.Fatalf("retry=%+v attempts=%v", result, publisher.attempts)
	}
}

func TestRecreatedRouteRetriesAfterOrphanWasWithdrawnButPublishFailed(t *testing.T) {
	service, publisher, _, facts := newRouteRuleService("minio")
	old := facts.routes["minio"]
	old.Id = "previous-minio"
	publisher.publications = map[string]routeport.Publication{old.Id: {Route: old, Phase: "confirmed"}}
	publisher.owners = map[string][]string{"minio": {old.Id}}
	publisher.failures = map[string]error{"minio": os.ErrPermission}
	result := confirmPreview(t, service, selectedPreview(t, service, "minio"))
	if result.Code != "route_sync_incomplete" || publisher.publications[old.Id].Route.Enabled || !reflect.DeepEqual(publisher.attempts, []string{old.Id, "minio"}) {
		t.Fatalf("result=%+v attempts=%v", result, publisher.attempts)
	}
	publisher.failures, publisher.attempts = nil, nil
	result = confirmPreview(t, service, selectedPreview(t, service, "minio"))
	if result.Code != "route_sync_completed" || !reflect.DeepEqual(publisher.attempts, []string{"minio"}) {
		t.Fatalf("retry repeated cleanup or failed to publish: result=%+v attempts=%v", result, publisher.attempts)
	}
}

func TestSelectedRouteDoesNotWithdrawLiveOwnerOrUnrelatedOrphans(t *testing.T) {
	service, publisher, _, facts := newRouteRuleService("minio", "renamed")
	live := facts.routes["renamed"]
	live.Name = "minio" // Its stored publication predates an unsynced business rename.
	unrelated := facts.routes["minio"]
	unrelated.Id, unrelated.Name = "deleted-other", "other"
	publisher.publications = map[string]routeport.Publication{
		live.Id: {Route: live, Phase: "confirmed"}, unrelated.Id: {Route: unrelated, Phase: "confirmed"},
	}
	publisher.owners = map[string][]string{"minio": {live.Id}, "other": {unrelated.Id}}
	confirmPreview(t, service, selectedPreview(t, service, "minio"))
	if !reflect.DeepEqual(publisher.attempts, []string{"minio"}) || !publisher.publications[live.Id].Route.Enabled || !publisher.publications[unrelated.Id].Route.Enabled {
		t.Fatalf("selected sync changed another Route's publication: attempts=%v publications=%+v", publisher.attempts, publisher.publications)
	}
}

type restoreOwnerTransaction struct {
	facts *routeRuleFacts
	owner model.Route
}

func (tx restoreOwnerTransaction) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	tx.facts.routes[tx.owner.Id] = tx.owner
	return fn(ctx)
}

func TestOrphanOwnerIsRecheckedBeforePublication(t *testing.T) {
	service, publisher, client, facts := newRouteRuleService("minio")
	old := facts.routes["minio"]
	old.Id = "previous-minio"
	publisher.publications = map[string]routeport.Publication{old.Id: {Route: old, Phase: "confirmed"}}
	publisher.owners = map[string][]string{"minio": {old.Id}}
	preview := selectedPreview(t, service, "minio")
	service = New(facts, nil, nil, facts, facts, publisher, nil, client, restoreOwnerTransaction{facts: facts, owner: old}, testRouteSyncTimeouts())
	result := confirmPreview(t, service, preview)
	if result.Code != "route_sync_incomplete" || result.Results[0].Code != routeSyncPreviewExpiredCode || len(publisher.attempts) != 0 {
		t.Fatalf("restored owner was withdrawn after state load: result=%+v attempts=%v", result, publisher.attempts)
	}
}

func TestPendingRenameOwnershipIsPartOfEachAffectedRouteSync(t *testing.T) {
	for _, ids := range [][]string{{"api"}, {"new-api"}, {"api", "new-api"}} {
		t.Run(strings.Join(ids, "+"), func(t *testing.T) {
			service, publisher, _, facts := newRouteRuleService("api", "new-api")
			previous := facts.routes["api"]
			previous.Id = "deleted-owner"
			candidate := previous
			candidate.Name = "new-api"
			publisher.publications = map[string]routeport.Publication{previous.Id: {
				Route: candidate, Phase: "pending", Previous: &routeport.Publication{Route: previous, Phase: "confirmed", ActualFingerprint: "original"},
			}}
			publisher.owners = map[string][]string{"api": {previous.Id}, "new-api": {previous.Id}}
			preview := selectedPreview(t, service, ids...)
			if !reflect.DeepEqual(preview.RouteIds, ids) {
				t.Fatalf("pending owner leaked into selected scope: %+v", preview)
			}
			result := confirmPreview(t, service, preview)
			wantAttempts := append([]string{previous.Id}, ids...)
			if result.Code != "route_sync_completed" || !reflect.DeepEqual(publisher.attempts, wantAttempts) {
				t.Fatalf("pending owner was omitted or withdrawn more than once: result=%+v attempts=%v", result, publisher.attempts)
			}
		})
	}
}

func TestPendingRenamePreviousFileChangeExpiresPreview(t *testing.T) {
	service, publisher, _, facts := newRouteRuleService("api")
	previous := facts.routes["api"]
	previous.Id = "deleted-owner"
	candidate := previous
	candidate.Name = "new-api"
	publisher.publications = map[string]routeport.Publication{previous.Id: {
		Route: candidate, Phase: "pending", Previous: &routeport.Publication{Route: previous, Phase: "confirmed", ActualFingerprint: "original"},
	}}
	publisher.owners = map[string][]string{"api": {previous.Id}}
	preview := selectedPreview(t, service, "api")
	publication := publisher.publications[previous.Id]
	publication.Previous.ActualFingerprint = "external edit"
	publisher.publications[previous.Id] = publication
	_, err := service.ConfirmRouteSync(context.Background(), routeTestUserId, routeTestProjectId, confirmInput(preview))
	if err == nil || apperror.Classify(err).Code != routeSyncPreviewExpiredCode || len(publisher.attempts) != 0 {
		t.Fatalf("previous file change was not frozen into the preview: err=%v attempts=%v", err, publisher.attempts)
	}
}

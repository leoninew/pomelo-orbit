package traefik

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

func TestProjectPublicationDiscoveryExcludesOtherProjects(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	current, foreign := publicationTestRoute("current"), publicationTestRoute("foreign")
	publishTestRoute(t, m, current)
	publishTestRoute(t, m, foreign)
	var record routeport.Publication
	if err := json.Unmarshal(runtime.files[publicationRecordPath(foreign.Id)], &record); err != nil {
		t.Fatal(err)
	}
	otherProject := "other-project"
	record.Route.ProjectId, record.GatewayApplicationId = &otherProject, "other-gateway"
	data, err := json.Marshal(record)
	if err != nil {
		t.Fatal(err)
	}
	runtime.files[publicationRecordPath(foreign.Id)] = data
	runtime.reads, runtime.writes = nil, nil
	items, err := inspectTestPublications(m, context.Background(), "project-1", testGateway())
	if err != nil || len(items) != 1 || items[0].Route.Id != current.Id {
		t.Fatalf("Project discovery should include only its publication: items=%+v err=%v", items, err)
	}
	for _, name := range runtime.reads {
		if name == activeTestPath(foreign.Name) {
			t.Fatal("Project discovery inspected another Project's runtime configuration")
		}
	}
	if len(runtime.writes) != 0 {
		t.Fatal("preview modified publication files")
	}
	// The same file must remain a conflict for a real business Route in scope.
	_, err = m.InspectPublication(context.Background(), "project-1", testGateway(), foreign)
	if !apperror.IsKind(err, apperror.KindConflict) || apperror.Classify(err).Code != "route_sync_publication_ownership_conflict" {
		t.Fatalf("expected an actionable ownership conflict, got %v", err)
	}
	if !strings.Contains(err.Error(), otherProject) || strings.Contains(apperror.Classify(err).Message, otherProject) {
		t.Fatalf("ownership details must be retained for logs, with a separate UI message: %v", err)
	}
}

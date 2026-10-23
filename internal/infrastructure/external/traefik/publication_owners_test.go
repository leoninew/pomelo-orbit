package traefik

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"
	"testing"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

type deniedPublicationFileRuntime struct {
	*routeRuntimeFake
	deniedPath string
}

func (r *deniedPublicationFileRuntime) ReadFile(ctx context.Context, target environmentport.Target, name string) ([]byte, error) {
	if name == r.deniedPath {
		r.reads = append(r.reads, name)
		return nil, os.ErrPermission
	}
	return r.routeRuntimeFake.ReadFile(ctx, target, name)
}

func TestPublicationOwnerLookupInspectsOnlyMatchingRouteFiles(t *testing.T) {
	for _, deniedFile := range []string{"configuration", "certificate"} {
		t.Run(deniedFile, func(t *testing.T) {
			runtime := newRouteRuntimeFake()
			runtime.autoAPI = true
			m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
			current, unrelated := publicationTestRoute("current"), publicationTestRoute("unrelated")
			cert, key := generateTestCertificate(t, unrelated.Domain)
			unrelated.HTTPSEnabled, unrelated.CertType, unrelated.CertPEM, unrelated.CertKey = true, "manual", &cert, &key
			publishTestRoute(t, m, current)
			publishTestRoute(t, m, unrelated)
			deniedPath := activeTestPath(unrelated.Name)
			if deniedFile == "certificate" {
				deniedPath = path.Join("/srv/orbit/traefik-default/gateway/certs/route-"+unrelated.Id, certificateRevision(unrelated), "cert.pem")
			}
			m = newRouteManager(routeTargetResolver{}, &deniedPublicationFileRuntime{routeRuntimeFake: runtime, deniedPath: deniedPath}, testRouteTimeouts(), func() bool { return false })
			runtime.reads, runtime.writes = nil, nil
			owners, err := m.FindPublicationOwners(context.Background(), "project-1", testGateway(), []string{current.Name})
			if err != nil || len(owners) != 1 || len(owners[current.Name]) != 1 {
				t.Fatalf("unrelated file blocked owner lookup: owners=%+v err=%v", owners, err)
			}
			owner := owners[current.Name][0]
			if owner.Route.Id != current.Id || owner.ActualFingerprint != digest(runtime.files[activeTestPath(current.Name)]) {
				t.Fatalf("matching publication was not inspected: %+v", owner)
			}
			for _, name := range runtime.reads {
				if name == activeTestPath(unrelated.Name) || strings.Contains(name, "/certs/route-"+unrelated.Id+"/") {
					t.Fatalf("owner lookup read an unrelated runtime file: %s", name)
				}
			}
			if len(runtime.writes) != 0 {
				t.Fatal("owner lookup modified publication files")
			}
			_, err = m.FindPublicationOwners(context.Background(), "project-1", testGateway(), []string{unrelated.Name})
			if !errors.Is(err, os.ErrPermission) {
				t.Fatalf("matching file error was suppressed: %v", err)
			}
		})
	}
}

func TestInterruptedRenameOwnerCanBeWithdrawnBeforeRecreatingPreviousName(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	route := publicationTestRoute("api")
	route.Id = "deleted-owner"
	publishTestRoute(t, m, route)
	previous, err := m.InspectPublication(context.Background(), "project-1", testGateway(), route)
	if err != nil || previous == nil {
		t.Fatalf("previous publication=%+v err=%v", previous, err)
	}
	target, _, base, err := m.publicationWorkspace(context.Background(), "project-1", testGateway())
	if err != nil {
		t.Fatal(err)
	}
	runtime.files[path.Join(base, ".orbit/route-publication", route.Id, "previous.yaml")] = append([]byte(nil), runtime.files[activeTestPath(route.Name)]...)
	candidate := route
	candidate.Name = "new-api"
	body, err := yaml.Marshal(buildRouteSnapshot([]model.Route{candidate}))
	if err != nil {
		t.Fatal(err)
	}
	runtime.files[activeTestPath(candidate.Name)] = body
	pending := routeport.Publication{Route: candidate, Phase: "pending", Fingerprint: digest(body), Previous: previous, PreviousFingerprint: previous.Fingerprint, TargetRevision: previous.TargetRevision, GatewayApplicationId: previous.GatewayApplicationId}
	if err := m.savePublication(context.Background(), target, base, pending); err != nil {
		t.Fatal(err)
	}
	owners, err := m.FindPublicationOwners(context.Background(), "project-1", testGateway(), []string{route.Name, candidate.Name})
	if err != nil || len(owners[route.Name]) != 1 || len(owners[candidate.Name]) != 1 {
		t.Fatalf("interrupted rename did not own both paths: owners=%+v err=%v", owners, err)
	}
	owner := owners[route.Name][0]
	if owner.Previous == nil || owner.Previous.ActualFingerprint != previous.Fingerprint || owner.ActualFingerprint != pending.Fingerprint || publicationFingerprint(owner) != publicationFingerprint(owners[candidate.Name][0]) {
		t.Fatalf("owner lookup did not inspect both paths: %+v", owner)
	}
	withdrawal := owner.Route
	withdrawal.Enabled = false
	if _, err := m.PublishRoute(context.Background(), "project-1", testGateway(), withdrawal, publicationFingerprint(owner)); err != nil {
		t.Fatal(err)
	}
	owners, err = m.FindPublicationOwners(context.Background(), "project-1", testGateway(), []string{route.Name, candidate.Name})
	if err != nil || len(owners) != 0 {
		t.Fatalf("withdrawn publication still claims a name: owners=%+v err=%v", owners, err)
	}
	for _, name := range []string{route.Name, candidate.Name} {
		if _, exists := runtime.files[activeTestPath(name)]; exists {
			t.Fatalf("withdrawal retained an interrupted rename path: %s", name)
		}
	}
	current := route
	current.Id, current.TargetUrl = "recreated-api", "http://new-api:9000"
	result := publishTestRoute(t, m, current)
	if result.ConfigurationMatch != "matched" || !strings.Contains(string(runtime.files[activeTestPath(current.Name)]), current.TargetUrl) {
		t.Fatalf("recreated Route was not published: %+v", result)
	}
}

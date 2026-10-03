package traefik

import (
	"context"
	"errors"
	"os"
	"path"
	"strings"
	"testing"

	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

func TestRouteCodeNamesFilesAndResourcesWhileIdentityOwnsCertificates(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI, runtime.reloadRequired = true, true
	runtime.physicalBase = "D:/orbit/traefik-default"
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	a, b := publicationTestRoute("api"), publicationTestRoute("web")
	a.Id, b.Id = "id-a", "id-b"
	cert, key := generateTestCertificate(t, a.Domain)
	a.HTTPSEnabled, a.CertType, a.CertPEM, a.CertKey = true, "manual", &cert, &key
	publishTestRoute(t, m, a)
	publishTestRoute(t, m, b)
	body := string(runtime.files[activeTestPath("api")])
	if !strings.Contains(body, "route-api-route:") || !strings.Contains(body, "route-api-service:") || strings.Contains(body, "route-id-a-route") {
		t.Fatalf("resource names do not use the code: %s", body)
	}
	certificatePath := path.Join("/srv/orbit/traefik-default/gateway/certs/route-id-a", certificateRevision(a), "key.pem")
	if string(runtime.files[certificatePath]) != key {
		t.Fatal("certificate ownership no longer uses the stable identity")
	}
	beforeB, beforeWrites := string(runtime.files[activeTestPath("web")]), len(runtime.writes)
	a.Name = "new-api"
	publishTestRoute(t, m, a)
	if _, exists := runtime.files[activeTestPath("api")]; exists {
		t.Fatal("rename retained the previous file")
	}
	assertLoadedRouteCode(t, m, a, "api")
	if string(runtime.files[activeTestPath("web")]) != beforeB || string(runtime.files[certificatePath]) != key {
		t.Fatal("rename changed another Route or its own certificate version")
	}
	for _, write := range runtime.writes[beforeWrites:] {
		if strings.Contains(write.Path, "/gateway/certs/") || write.Path == activeTestPath("web") {
			t.Fatalf("rename rewrote unrelated content: %s", write.Path)
		}
	}
	// A business rename can free a code that is still published by another ID.
	b.Name = "api"
	publishTestRoute(t, m, b)
	beforeB = string(runtime.files[activeTestPath("api")])
	a.Name, a.Enabled = "api", false
	publishTestRoute(t, m, a)
	if _, exists := runtime.files[activeTestPath("new-api")]; exists || string(runtime.files[activeTestPath("api")]) != beforeB {
		t.Fatal("withdrawal used the unsynced business code instead of the published code")
	}
	// A later withdrawal of A must not remove B's reuse of A's last published code.
	b.Name = "new-api"
	publishTestRoute(t, m, b)
	beforeB = string(runtime.files[activeTestPath("new-api")])
	publishTestRoute(t, m, a)
	if string(runtime.files[activeTestPath("new-api")]) != beforeB {
		t.Fatal("repeated withdrawal removed a code reused by another Route")
	}
	items, err := inspectTestPublications(m, context.Background(), "project-1", testGateway())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Phase != "confirmed" || item.ActualFingerprint != item.Fingerprint {
			t.Fatalf("publication rejected a legitimately reused code: %+v", item)
		}
	}
	assertLoadedRouteCode(t, m, b, "api")
}

func TestRouteRenameFailureRestoresOriginalPathAndResources(t *testing.T) {
	for _, stage := range []string{"write candidate", "remove previous", "reload candidate"} {
		t.Run(stage, func(t *testing.T) {
			runtime := newRouteRuntimeFake()
			runtime.autoAPI, runtime.reloadRequired = true, true
			runtime.physicalBase = "D:/orbit/traefik-default"
			m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
			a, b := publicationTestRoute("api"), publicationTestRoute("web")
			a.Id, b.Id = "id-a", "id-b"
			publishTestRoute(t, m, a)
			publishTestRoute(t, m, b)
			oldBody, other := string(runtime.files[activeTestPath("api")]), string(runtime.files[activeTestPath("web")])
			failure := errors.New("reload denied")
			switch stage {
			case "write candidate":
				runtime.writeFailures = map[string]error{activeTestPath("new-api"): os.ErrPermission}
			case "remove previous":
				runtime.removeFailures = map[string]error{activeTestPath("api"): os.ErrPermission}
			case "reload candidate":
				runtime.reloadFailures = map[int]error{runtime.reloadAttempts + 1: failure}
			}
			a.Name = "new-api"
			result, err := m.PublishRoute(context.Background(), "project-1", testGateway(), a, testPublicationFingerprint(t, m, a.Id))
			if err == nil || result.Recovery != "restored" || result.FileCommit != "restored" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			if string(runtime.files[activeTestPath("api")]) != oldBody || string(runtime.files[activeTestPath("web")]) != other {
				t.Fatal("recovery changed the previous Route or another Route")
			}
			if _, exists := runtime.files[activeTestPath("new-api")]; exists {
				t.Fatal("recovery retained the candidate file")
			}
			a.Name = "api"
			assertLoadedRouteCode(t, m, a, "new-api")
			runtime.writeFailures, runtime.removeFailures, runtime.reloadFailures = nil, nil, nil
			a.Name = "new-api"
			publishTestRoute(t, m, a)
			assertLoadedRouteCode(t, m, a, "api")
		})
	}
}

func TestInterruptedRenameReconcilesBothPaths(t *testing.T) {
	for _, stage := range []string{"before commit", "both files", "candidate only", "external previous change"} {
		t.Run(stage, func(t *testing.T) {
			runtime := newRouteRuntimeFake()
			runtime.autoAPI, runtime.reloadRequired = true, true
			runtime.physicalBase = "D:/orbit/traefik-default"
			m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
			route := publicationTestRoute("api")
			route.Id = "id-a"
			publishTestRoute(t, m, route)
			items, _ := inspectTestPublications(m, context.Background(), "project-1", testGateway())
			previous := items[0]
			target, files, base, _ := m.publicationWorkspace(context.Background(), "project-1", testGateway())
			runtime.files[path.Join(base, ".orbit/route-publication", route.Id, "previous.yaml")] = append([]byte(nil), runtime.files[activeTestPath("api")]...)
			route.Name = "new-api"
			body, _ := yaml.Marshal(buildRouteSnapshot([]model.Route{route}))
			pending := routeport.Publication{Route: route, Phase: "pending", Fingerprint: digest(body), Previous: &previous, PreviousFingerprint: previous.Fingerprint, TargetRevision: previous.TargetRevision, GatewayApplicationId: previous.GatewayApplicationId}
			if stage != "before commit" {
				runtime.files[activeTestPath("new-api")] = body
			}
			if stage == "candidate only" {
				delete(runtime.files, activeTestPath("api"))
			}
			if stage == "external previous change" {
				runtime.files[activeTestPath("api")] = []byte("external change")
			}
			if err := m.savePublication(context.Background(), target, base, pending); err != nil {
				t.Fatal(err)
			}
			items, err := inspectTestPublications(m, context.Background(), "project-1", testGateway())
			if err != nil {
				t.Fatal(err)
			}
			if stage == "both files" {
				assertLoadedFilesDoNotMatchRename(t, m, route, runtime.files[activeTestPath("new-api")])
			}
			recovered, err := m.reconcilePending(context.Background(), "project-1", testGateway(), target, files, base, items[0])
			if stage == "external previous change" {
				if err == nil || string(runtime.files[activeTestPath("api")]) != "external change" || string(runtime.files[activeTestPath("new-api")]) != string(body) {
					t.Fatalf("recovery did not preserve unknown content: err=%v", err)
				}
				return
			}
			wanted, absent := "api", "new-api"
			if stage == "candidate only" {
				wanted, absent = absent, wanted
			}
			if err != nil || recovered.Phase != "confirmed" || recovered.Route.Name != wanted {
				t.Fatalf("recovered=%+v err=%v", recovered, err)
			}
			if _, exists := runtime.files[activeTestPath(absent)]; exists {
				t.Fatal("reconciliation retained an obsolete path")
			}
			assertLoadedRouteCode(t, m, recovered.Route, absent)
		})
	}
}

func TestRouteCodeOwnershipRejectsPublishedAndUnknownDestinations(t *testing.T) {
	for _, owner := range []string{"confirmed", "pending previous", "unknown file"} {
		t.Run(owner, func(t *testing.T) {
			runtime := newRouteRuntimeFake()
			runtime.autoAPI = true
			m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
			a, b := publicationTestRoute("api"), publicationTestRoute("web")
			a.Id, b.Id = "id-a", "id-b"
			publishTestRoute(t, m, a)
			b.Name = "api"
			if strings.HasPrefix(owner, "pending") {
				items, _ := inspectTestPublications(m, context.Background(), "project-1", testGateway())
				previous, pending := items[0], items[0]
				pending.Phase, pending.Previous, pending.Route.Name = "pending", &previous, "new-api"
				target, _, base, _ := m.publicationWorkspace(context.Background(), "project-1", testGateway())
				if err := m.savePublication(context.Background(), target, base, pending); err != nil {
					t.Fatal(err)
				}
			}
			if owner == "unknown file" {
				b.Name = "external"
				runtime.files[activeTestPath(b.Name)] = []byte("unknown content")
			}
			before, count := string(runtime.files[activeTestPath("api")]), len(runtime.writes)
			_, err := m.PublishRoute(context.Background(), "project-1", testGateway(), b, "")
			if err == nil || len(runtime.writes) != count || string(runtime.files[activeTestPath("api")]) != before {
				t.Fatalf("conflicting publication changed an owned file: %v", err)
			}
			if owner == "unknown file" && string(runtime.files[activeTestPath("external")]) != "unknown content" {
				t.Fatal("conflicting publication overwrote an unknown file")
			}
		})
	}
}

func TestMissingRouteFileCanBeExplicitlyRepublishedOrWithdrawn(t *testing.T) {
	for _, action := range []string{"publish", "rename", "withdraw"} {
		t.Run(action, func(t *testing.T) {
			runtime := newRouteRuntimeFake()
			runtime.autoAPI, runtime.reloadRequired = true, true
			runtime.physicalBase = "D:/orbit/traefik-default"
			m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
			a, b := publicationTestRoute("api"), publicationTestRoute("web")
			a.Id, b.Id = "id-a", "id-b"
			publishTestRoute(t, m, a)
			publishTestRoute(t, m, b)
			other := string(runtime.files[activeTestPath("web")])
			oldPreview := testPublicationFingerprint(t, m, a.Id)
			delete(runtime.files, activeTestPath("api"))
			if _, err := m.PublishRoute(context.Background(), "project-1", testGateway(), a, oldPreview); err == nil {
				t.Fatal("a file deleted after preview did not expire that preview")
			}
			switch action {
			case "rename":
				a.Name = "new-api"
			case "withdraw":
				a.Enabled = false
			}
			publishTestRoute(t, m, a)
			if string(runtime.files[activeTestPath("web")]) != other {
				t.Fatal("missing-file recovery changed another Route")
			}
			if a.Enabled {
				if len(runtime.files[activeTestPath(a.Name)]) == 0 {
					t.Fatal("explicit synchronization did not recreate the missing file")
				}
				absent := "unused"
				if action == "rename" {
					absent = "api"
				}
				assertLoadedRouteCode(t, m, a, absent)
			} else {
				routers, _ := m.ListRouters(context.Background(), "project-1", testGateway())
				services, _ := m.ListServices(context.Background(), "project-1", testGateway())
				if !matchesConfiguration(dynamicConfiguration{}, []routeResource{{code: "api", protocol: "http"}}, routers, services) {
					t.Fatal("withdrawal left the previously loaded Route in memory")
				}
			}
		})
	}
}

func TestMissingFilePublicationFailureRestoresAbsenceAndCanRetry(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI, runtime.reloadRequired = true, true
	runtime.physicalBase = "D:/orbit/traefik-default"
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	route := publicationTestRoute("api")
	route.Id = "id-a"
	publishTestRoute(t, m, route)
	delete(runtime.files, activeTestPath("api"))
	backup := "/srv/orbit/traefik-default/.orbit/route-publication/id-a/previous.yaml"
	runtime.files[backup] = []byte("stale backup")
	route.Name = "new-api"
	runtime.reloadFailures = map[int]error{runtime.reloadAttempts + 1: errors.New("reload denied")}
	result, err := m.PublishRoute(context.Background(), "project-1", testGateway(), route, testPublicationFingerprint(t, m, route.Id))
	if err == nil || result.Recovery != "restored" || string(runtime.files[backup]) != "stale backup" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	for _, code := range []string{"api", "new-api"} {
		if _, exists := runtime.files[activeTestPath(code)]; exists {
			t.Fatal("recovery fabricated a previous configuration or retained the candidate")
		}
	}
	runtime.reloadFailures = nil
	publishTestRoute(t, m, route)
	assertLoadedRouteCode(t, m, route, "api")
}

func assertLoadedRouteCode(t *testing.T, m *RouteManager, route model.Route, absent string) {
	t.Helper()
	body, _ := yaml.Marshal(buildRouteSnapshot([]model.Route{route}))
	var expected dynamicConfiguration
	if err := yaml.Unmarshal(body, &expected); err != nil {
		t.Fatal(err)
	}
	routers, routerErr := m.ListRouters(context.Background(), "project-1", testGateway())
	services, serviceErr := m.ListServices(context.Background(), "project-1", testGateway())
	if routerErr != nil || serviceErr != nil || !matchesConfiguration(expected, []routeResource{{code: absent, protocol: route.Protocol}}, routers, services) {
		t.Fatalf("configuration does not match code %s without %s: routers=%+v services=%+v", route.Name, absent, routers, services)
	}
}

func assertLoadedFilesDoNotMatchRename(t *testing.T, m *RouteManager, route model.Route, body []byte) {
	t.Helper()
	runtime := m.runtime.(*routeRuntimeFake)
	runtime.loadedFiles = map[string][]byte{activeTestPath("api"): runtime.files[activeTestPath("api")], activeTestPath("new-api"): body}
	var expected dynamicConfiguration
	_ = yaml.Unmarshal(body, &expected)
	routers, _ := m.ListRouters(context.Background(), "project-1", testGateway())
	services, _ := m.ListServices(context.Background(), "project-1", testGateway())
	if matchesConfiguration(expected, []routeResource{{code: "api", protocol: route.Protocol}}, routers, services) {
		t.Fatalf("code %s matched while its previous resources remain loaded", route.Name)
	}
}

package traefik

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"path"
	"strings"
	"testing"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

func TestSelectedPublicationReadsOnlySelectedRecordAndFiles(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	a, b := publicationTestRoute("a"), publicationTestRoute("b")
	a.Id, b.Id = "id-a", "id-b"
	aCert, aKey := generateTestCertificate(t, a.Domain)
	a.HTTPSEnabled, a.CertType, a.CertPEM, a.CertKey = true, "manual", &aCert, &aKey
	cert, key := generateTestCertificate(t, b.Domain)
	b.HTTPSEnabled, b.CertType, b.CertPEM, b.CertKey = true, "manual", &cert, &key
	publishTestRoute(t, m, a)
	publishTestRoute(t, m, b)
	runtime.files[publicationRecordPath(b.Id)] = []byte("invalid unrelated record")
	runtime.reads, runtime.directories, runtime.writes, runtime.requests = nil, nil, nil, nil
	ctx, closeSession, err := m.OpenSession(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	defer closeSession()
	selected, err := m.InspectPublication(ctx, "project-1", testGateway(), a)
	if err != nil || selected == nil || selected.Route.Id != a.Id {
		t.Fatalf("item=%+v err=%v", selected, err)
	}
	result, err := m.PublishRoute(ctx, "project-1", testGateway(), a, publicationFingerprint(*selected))
	if err != nil || result.FileCommit != "unchanged" || result.ConfigurationMatch != "matched" || result.CertificateVerification != "unverified" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(runtime.directories) != 0 || len(runtime.writes) != 0 || runtime.reloadAttempts != 0 {
		t.Fatalf("directories=%v writes=%d reloads=%d", runtime.directories, len(runtime.writes), runtime.reloadAttempts)
	}
	counts := map[string]int{}
	for _, name := range runtime.reads {
		counts[name]++
		if name == activeTestPath("b") || name == publicationRecordPath(b.Id) || strings.Contains(name, "/certs/route-"+b.Id+"/") {
			t.Fatalf("selected synchronization inspected unrelated files: %s", name)
		}
	}
	for id, count := range map[string]int{a.Id: 2, b.Id: 0} {
		if counts[publicationRecordPath(id)] != count {
			t.Fatalf("record %s reads=%d want=%d", id, counts[publicationRecordPath(id)], count)
		}
	}
	if len(runtime.requests) != 2 || !strings.HasSuffix(runtime.requests[0], "/api/http/routers") || !strings.HasSuffix(runtime.requests[1], "/api/http/services") {
		t.Fatalf("HTTP synchronization made unnecessary API calls: %v", runtime.requests)
	}
	// The destination file still prevents overwriting another Route's code.
	a.Name = b.Name
	if _, err := m.PublishRoute(ctx, "project-1", testGateway(), a, publicationFingerprint(*selected)); !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("destination file ownership was lost: %v", err)
	}
}

func TestBatchSessionProcessesRoutesWithoutScanningPublicationDirectory(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	a, b := publicationTestRoute("a"), publicationTestRoute("b")
	publishTestRoute(t, m, a)
	publishTestRoute(t, m, b)
	runtime.directories = nil
	ctx, closeSession, err := m.OpenSession(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	defer closeSession()
	fingerprints := map[string]string{}
	for _, route := range []model.Route{a, b} {
		item, err := m.InspectPublication(ctx, "project-1", testGateway(), route)
		if err != nil || item == nil {
			t.Fatalf("inspect %s: item=%+v err=%v", route.Name, item, err)
		}
		fingerprints[route.Id] = publicationFingerprint(*item)
	}
	a.Name, b.Name = "new-a", "a"
	for _, route := range []model.Route{a, b} {
		if _, err := m.PublishRoute(ctx, "project-1", testGateway(), route, fingerprints[route.Id]); err != nil {
			t.Fatalf("publish %s using current batch ownership: %v", route.Id, err)
		}
	}
	scans := 0
	for _, directory := range runtime.directories {
		if strings.HasSuffix(directory, "/.orbit/route-publication") {
			scans++
		}
	}
	if scans != 0 {
		t.Fatalf("batch directory scans=%d", scans)
	}
}

func TestUnchangedPublicationReloadsOnlyWhenLoadedConfigurationDiffers(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI, runtime.reloadRequired, runtime.physicalBase = true, true, "D:/orbit/traefik-default"
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	route := publicationTestRoute("a")
	publishTestRoute(t, m, route)
	runtime.loadedFiles = nil
	runtime.writes, runtime.requests = nil, nil
	beforeReload := runtime.reloadAttempts
	result := publishTestRoute(t, m, route)
	if result.FileCommit != "unchanged" || result.ConfigurationMatch != "matched" || len(runtime.writes) != 0 || runtime.reloadAttempts != beforeReload+1 || len(runtime.requests) != 4 {
		t.Fatalf("result=%+v writes=%d reloads=%d requests=%v", result, len(runtime.writes), runtime.reloadAttempts-beforeReload, runtime.requests)
	}
	// File equality cannot substitute for a successful API observation.
	runtime.autoAPI = false
	runtime.responses["/api/http/routers"] = "[]"
	runtime.failures["/api/http/routers"] = 1
	failure := errors.New("API unavailable")
	runtime.failureErrors["/api/http/routers"] = failure
	result, err := m.PublishRoute(context.Background(), "project-1", testGateway(), route, testPublicationFingerprint(t, m, route.Id))
	if !errors.Is(err, failure) || apperror.Classify(err).Code != "route_sync_configuration_unavailable" || result.ConfigurationMatch == "matched" || len(runtime.writes) != 0 {
		t.Fatalf("unchanged publication hid API failure: result=%+v err=%v", result, err)
	}
}

func TestPendingMatchingDesiredPublicationDoesNotPublishAgain(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI, runtime.reloadRequired, runtime.physicalBase = true, true, "D:/orbit/traefik-default"
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	route := publicationTestRoute("a")
	publishTestRoute(t, m, route)
	items, err := inspectTestPublications(m, context.Background(), "project-1", testGateway())
	if err != nil {
		t.Fatal(err)
	}
	previous := items[0]
	runtime.files[path.Join(path.Dir(publicationRecordPath(route.Id)), "previous.yaml")] = runtime.files[activeTestPath(route.Name)]
	route.Domain = "updated.test"
	body, _ := yaml.Marshal(buildRouteSnapshot([]model.Route{route}))
	pending := previous
	pending.Route, pending.Phase, pending.Previous = route, "pending", &previous
	pending.Fingerprint, pending.PreviousFingerprint = digest(body), previous.Fingerprint
	data, _ := json.Marshal(pending)
	runtime.files[publicationRecordPath(route.Id)] = data
	runtime.files[activeTestPath(route.Name)] = body
	runtime.writes, runtime.requests = nil, nil
	beforeReload := runtime.reloadAttempts
	result := publishTestRoute(t, m, route)
	if result.FileCommit != "unchanged" || result.ConfigurationMatch != "matched" || runtime.reloadAttempts != beforeReload+1 || len(runtime.requests) != 2 {
		t.Fatalf("pending verification was repeated: result=%+v reloads=%d requests=%v", result, runtime.reloadAttempts-beforeReload, runtime.requests)
	}
	if len(runtime.writes) != 1 || runtime.writes[0].Path != publicationRecordPath(route.Id) {
		t.Fatalf("recovered candidate was published again: writes=%v", runtime.writes)
	}
}

func TestProtocolSwitchVerifiesNewAndRetiredResources(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	route := publicationTestRoute("a")
	publishTestRoute(t, m, route)
	oldBody := runtime.files[activeTestPath(route.Name)]
	route.Protocol, route.TargetUrl = "tcp", "database:5432"
	route.TargetAddress, route.TargetPort = "database", 5432
	port := 5432
	route.ListenPort = &port
	runtime.requests = nil
	publishTestRoute(t, m, route)
	if len(runtime.requests) != 4 {
		t.Fatalf("protocol switch API calls=%v", runtime.requests)
	}
	runtime.files[activeTestPath("residual")] = oldBody
	body, _ := yaml.Marshal(buildRouteSnapshot([]model.Route{route}))
	matched, err := m.checkConfiguration(context.Background(), "project-1", testGateway(), body, "tcp", routeResource{code: route.Name, protocol: "http"})
	if err != nil || matched {
		t.Fatalf("old protocol resource was not detected: matched=%t err=%v", matched, err)
	}
}

func TestUnchangedRenamePreservesOldCodeReusedByAnotherProtocol(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	a, b := publicationTestRoute("a"), publicationTestRoute("b")
	publishTestRoute(t, m, a)
	a.Name = "new-a"
	publishTestRoute(t, m, a)
	port := 5432
	b.Name, b.Protocol, b.ListenPort, b.TargetAddress, b.TargetPort = "a", "tcp", &port, "database", port
	publishTestRoute(t, m, b)
	ctx, closeSession, err := m.OpenSession(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	defer closeSession()
	item, err := m.InspectPublication(ctx, "project-1", testGateway(), a)
	if err != nil {
		t.Fatal(err)
	}
	var fingerprint string
	if item != nil {
		fingerprint = publicationFingerprint(*item)
	}
	runtime.reads, runtime.writes = nil, nil
	result, err := m.PublishRoute(ctx, "project-1", testGateway(), a, fingerprint)
	if err != nil || result.FileCommit != "unchanged" || len(runtime.writes) != 0 {
		t.Fatalf("valid cross-protocol reuse was rejected: result=%+v err=%v", result, err)
	}
	for _, name := range runtime.reads {
		if name == activeTestPath(b.Name) {
			t.Fatal("unchanged publication read a file now owned by another Route")
		}
	}
}

type mutableRouteTarget struct {
	target environmentport.Target
}

func (r *mutableRouteTarget) ResolveProjectTarget(context.Context, string) (environmentport.Target, error) {
	return r.target, nil
}

func TestHistoricalTargetRevisionAllowsExplicitRepublish(t *testing.T) {
	target, _ := (routeTargetResolver{targetType: model.EnvironmentTargetTypeSSH}).ResolveProjectTarget(context.Background(), "project-1")
	resolver := &mutableRouteTarget{target: target}
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(resolver, runtime, testRouteTimeouts(), func() bool { return false })
	route := publicationTestRoute("a")
	publishTestRoute(t, m, route)
	oldRecord := string(runtime.files[publicationRecordPath(route.Id)])
	resolver.target.Environment.TargetType, resolver.target.Environment.SSH = model.EnvironmentTargetTypeLocal, nil
	resolver.target.Environment.TargetRevision = 2
	runtime.writes = nil
	ctx, closeSession, err := m.OpenSession(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	defer closeSession()
	item, err := m.InspectPublication(ctx, "project-1", testGateway(), route)
	if err != nil || item == nil || len(runtime.writes) != 0 || string(runtime.files[publicationRecordPath(route.Id)]) != oldRecord {
		t.Fatalf("historical record was rejected or modified on inspection: item=%v err=%v", item, err)
	}
	result, err := m.PublishRoute(ctx, "project-1", testGateway(), route, publicationFingerprint(*item))
	var current routeport.Publication
	_ = json.Unmarshal(runtime.files[publicationRecordPath(route.Id)], &current)
	if err != nil || result.FileCommit != "committed" || current.TargetRevision != "environment-1:2" {
		t.Fatalf("explicit publication did not use the current revision: result=%+v revision=%s err=%v", result, current.TargetRevision, err)
	}
	resolver.target.Environment.TargetRevision++
	if _, err := m.InspectPublication(ctx, "project-1", testGateway(), route); apperror.Classify(err).Code != "route_sync_preview_expired" {
		t.Fatalf("in-flight target change was accepted: %v", err)
	}
}

func TestRoutePublicationLogsReadableCodesAtEveryStage(t *testing.T) {
	target, _ := (routeTargetResolver{}).ResolveProjectTarget(context.Background(), "project-1")
	target.Environment.Code = "demo-project"
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(&mutableRouteTarget{target: target}, runtime, testRouteTimeouts(), func() bool { return false })
	route := publicationTestRoute("api")
	route.Id = "opaque-route-id"
	publishTestRoute(t, m, route)
	var logs bytes.Buffer
	previousLogger := slog.Default()
	slog.SetDefault(slog.New(slog.NewTextHandler(&logs, nil)))
	t.Cleanup(func() { slog.SetDefault(previousLogger) })
	ctx, closeSession, err := m.OpenSession(context.Background(), "project-1")
	if err != nil {
		t.Fatal(err)
	}
	defer closeSession()
	item, err := m.InspectPublication(ctx, "project-1", testGateway(), route)
	if err != nil || item == nil {
		t.Fatalf("item=%+v err=%v", item, err)
	}
	if _, err := m.PublishRoute(ctx, "project-1", testGateway(), route, publicationFingerprint(*item)); err != nil {
		t.Fatal(err)
	}
	for _, phase := range []string{"read_publication_record", "verify_publication", "configuration_match", "request"} {
		found := false
		for _, line := range strings.Split(logs.String(), "\n") {
			if strings.Contains(line, "phase="+phase) {
				found = true
				if !strings.Contains(line, "project_code=demo-project") || !strings.Contains(line, "route_code=api") {
					t.Fatalf("unreadable %s log: %s", phase, line)
				}
			}
		}
		if !found {
			t.Fatalf("missing %s log: %s", phase, logs.String())
		}
	}
	if strings.Contains(logs.String(), "project_id=") || strings.Contains(logs.String(), "route_id=") {
		t.Fatalf("publication logs still identify resources only by ID: %s", logs.String())
	}
}

func publicationRecordPath(id string) string {
	return "/srv/orbit/traefik-default/.orbit/route-publication/" + id + "/state.json"
}

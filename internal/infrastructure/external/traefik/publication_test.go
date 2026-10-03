package traefik

import (
	"context"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/tls"
	"crypto/x509"
	"crypto/x509/pkix"
	"encoding/pem"
	"errors"
	"fmt"
	"math/big"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	routeport "github.com/leoninew/pomelo-orbit/internal/application/route/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"gopkg.in/yaml.v3"
)

func TestFileProviderReloadUsesTargetFilesystem(t *testing.T) {
	for _, tc := range []struct {
		name, targetType, platform, physicalBase string
		inContainer, reload                      bool
	}{
		{name: "local Windows", physicalBase: `D:\orbit\traefik-default`, reload: true},
		{name: "local Linux", physicalBase: "/srv/orbit/traefik-default"},
		{name: "Windows DooD", physicalBase: "D:/orbit/traefik-default", inContainer: true, reload: true},
		{name: "Windows Docker Desktop DooD", physicalBase: "/run/desktop/mnt/host/d/orbit/traefik-default", inContainer: true, reload: true},
		{name: "Linux DooD", physicalBase: "/var/lib/docker/volumes/orbit/_data/traefik-default", inContainer: true},
		{name: "SSH Windows", targetType: model.EnvironmentTargetTypeSSH, platform: model.EnvironmentPlatformWindows, inContainer: true, reload: true},
		{name: "SSH Linux", targetType: model.EnvironmentTargetTypeSSH, platform: model.EnvironmentPlatformLinux, inContainer: true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			runtime := newRouteRuntimeFake()
			runtime.autoAPI, runtime.reloadRequired = true, tc.reload
			runtime.physicalBase, runtime.containerId = tc.physicalBase, "project-gateway-id"
			m := newRouteManager(routeTargetResolver{targetType: tc.targetType, platform: tc.platform}, runtime, testRouteTimeouts(), func() bool { return tc.inContainer })
			publishTestRoute(t, m, publicationTestRoute("a"))
			if (runtime.reloadAttempts != 0) != tc.reload {
				t.Fatalf("reload attempts=%d", runtime.reloadAttempts)
			}
			for _, command := range runtime.commands {
				if command.name != "docker" {
					continue
				}
				if strings.HasPrefix(strings.Join(command.args, " "), "compose ps ") && command.serviceCode != testGateway().RuntimeServiceCode {
					t.Fatal("resolved container outside the Gateway Service workspace")
				}
				if command.args[0] == "kill" && strings.Join(command.args, " ") != "kill --signal=HUP project-gateway-id" {
					t.Fatalf("unexpected signal command: %+v", command)
				}
				if tc.targetType == model.EnvironmentTargetTypeSSH && command.target.Environment.SSH.Platform != tc.platform {
					t.Fatal("reload ignored the SSH target platform")
				}
			}
		})
	}
}

func TestGatewayValidationReportsDockerQueryFailureSafely(t *testing.T) {
	failure := errors.New("Process exited with status 1")
	diagnostic := "error during connect: Docker Desktop engine is unavailable"
	runtime := newRouteRuntimeFake()
	runtime.containerError, runtime.containerDiagnostic = failure, diagnostic
	manager := NewRouteManager(routeTargetResolver{targetType: model.EnvironmentTargetTypeSSH, platform: model.EnvironmentPlatformWindows}, runtime, testRouteTimeouts())
	_, err := manager.ValidateGateway(context.Background(), "project-1", testGateway(), nil)
	if !errors.Is(err, failure) || !strings.Contains(err.Error(), diagnostic) {
		t.Fatalf("Gateway validation lost the Docker command diagnostic: %v", err)
	}
	if !apperror.IsKind(err, apperror.KindUnavailable) || apperror.Classify(err).Code != "route_sync_gateway_unavailable" {
		t.Fatalf("Docker failure was not classified as Gateway unavailability: %v", err)
	}
	message := apperror.Classify(err).Message
	if !strings.Contains(message, "Docker") || strings.Contains(message, diagnostic) || strings.Contains(message, failure.Error()) {
		t.Fatalf("public error did not retain a safe actionable cause: %q", message)
	}
}

func TestGatewayValidationReportsStoppedContainer(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.containerId = "\n"
	manager := NewRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts())
	_, err := manager.ValidateGateway(context.Background(), "project-1", testGateway(), nil)
	if !apperror.IsKind(err, apperror.KindUnavailable) || apperror.Classify(err).Code != "route_sync_gateway_not_running" {
		t.Fatalf("stopped Gateway was not identified: %v", err)
	}
}

func TestWindowsPublicationReloadsUpdatesAndWithdrawalWithoutChangingOtherRoutes(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI, runtime.reloadRequired = true, true
	runtime.physicalBase = "D:/orbit/traefik-default"
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	a, b := publicationTestRoute("a"), publicationTestRoute("b")
	unknown := "/srv/orbit/traefik-default/gateway/dynamic/custom.yaml"
	unknownBody, _ := yaml.Marshal(buildRouteSnapshot([]model.Route{publicationTestRoute("external")}))
	runtime.files[unknown] = unknownBody
	publishTestRoute(t, m, a)
	publishTestRoute(t, m, b)
	beforeB := string(runtime.files[activeTestPath("b")])
	a.Domain = "updated.test"
	publishTestRoute(t, m, a)
	if string(runtime.loadedFiles[activeTestPath("a")]) != string(runtime.files[activeTestPath("a")]) {
		t.Fatal("Traefik still uses the previous Route")
	}
	a.Enabled = false
	publishTestRoute(t, m, a)
	if _, found := runtime.loadedFiles[activeTestPath("a")]; found {
		t.Fatal("Traefik still uses the withdrawn Route")
	}
	if string(runtime.files[activeTestPath("b")]) != beforeB || string(runtime.loadedFiles[activeTestPath("b")]) != beforeB || string(runtime.loadedFiles[unknown]) != string(unknownBody) {
		t.Fatal("reload changed another Route or an unknown file")
	}
}

func TestWindowsReloadFailureReportsRecoveryAndSupportsRetry(t *testing.T) {
	for _, recoveryFails := range []bool{false, true} {
		t.Run(fmt.Sprintf("recovery failure=%t", recoveryFails), func(t *testing.T) {
			runtime := newRouteRuntimeFake()
			runtime.autoAPI, runtime.reloadRequired = true, true
			runtime.physicalBase = "D:/orbit/traefik-default"
			m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
			a, b := publicationTestRoute("a"), publicationTestRoute("b")
			publishTestRoute(t, m, a)
			publishTestRoute(t, m, b)
			previous := string(runtime.files[activeTestPath("a")])
			other := string(runtime.files[activeTestPath("b")])
			failure := errors.New("Docker denied the reload signal")
			runtime.reloadFailures = map[int]error{runtime.reloadAttempts + 1: failure}
			if recoveryFails {
				runtime.reloadFailures[runtime.reloadAttempts+2] = failure
			}
			a.Domain = "updated.test"
			result, err := m.PublishRoute(context.Background(), "project-1", testGateway(), a, testPublicationFingerprint(t, m, "a"))
			classified, ok := apperror.As(err)
			if !ok || classified.Code != "route_sync_reload_failed" || !errors.Is(err, failure) || result.ConfigurationMatch != "unverified" {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			wantRecovery, wantCommit := "restored", "restored"
			if recoveryFails {
				wantRecovery, wantCommit = "failed", "committed"
			}
			if result.Recovery != wantRecovery || result.FileCommit != wantCommit || string(runtime.files[activeTestPath("a")]) != previous || string(runtime.loadedFiles[activeTestPath("a")]) != previous || string(runtime.files[activeTestPath("b")]) != other {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			runtime.reloadFailures = nil
			publishTestRoute(t, m, a)
			if string(runtime.loadedFiles[activeTestPath("a")]) != string(runtime.files[activeTestPath("a")]) || string(runtime.loadedFiles[activeTestPath("b")]) != other {
				t.Fatal("retry did not load the selected Route independently")
			}
		})
	}
}

func TestRoutePublicationUsesConfiguredTimeoutForMatchingAndRecovery(t *testing.T) {
	timeouts := testRouteTimeouts()
	timeouts.ApiRequest, timeouts.Reload = 2*time.Second, 4*time.Second
	runtime := &routeTimeoutRuntime{routeRuntimeFake: newRouteRuntimeFake()}
	runtime.autoAPI, runtime.reloadRequired = true, true
	manager := NewRouteManager(routeTargetResolver{targetType: model.EnvironmentTargetTypeSSH, platform: model.EnvironmentPlatformWindows}, runtime, timeouts)
	route := publicationTestRoute("configured-timeout")
	publishTestRoute(t, manager, route)
	assertTimeouts := func() {
		t.Helper()
		if len(runtime.remainingTimeouts) != 3 {
			t.Fatalf("expected reload and two HTTP API requests, got %d", len(runtime.remainingTimeouts))
		}
		for index, remaining := range runtime.remainingTimeouts {
			timeout := timeouts.ApiRequest
			if index == 0 {
				timeout = timeouts.Reload
			}
			if remaining <= timeout-time.Second || remaining > timeout {
				t.Fatalf("configured timeout %s was not applied: remaining=%s", timeout, remaining)
			}
		}
	}
	assertTimeouts()
	runtime.remainingTimeouts = nil
	runtime.writeFailures = map[string]error{activeTestPath(route.Name): os.ErrPermission}
	route.TargetUrl = "http://updated:8080"
	result, err := manager.PublishRoute(context.Background(), "project-1", testGateway(), route, testPublicationFingerprint(t, manager, route.Id))
	if !errors.Is(err, os.ErrPermission) || result.Recovery != "restored" || result.FileCommit != "restored" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	assertTimeouts()
}

type routeTimeoutRuntime struct {
	*routeRuntimeFake
	remainingTimeouts []time.Duration
	reloadDelay       time.Duration
	blockCommit       bool
	blockReload       bool
	requestFailure    error
	recordReadFailure error
	reloadDeadline    time.Time
}

func (runtime *routeTimeoutRuntime) QueryAtEnvironmentRoot(ctx context.Context, target environmentport.Target, name string, args ...string) (string, error) {
	if name == "curl" || name == "docker" && len(args) > 0 && args[0] == "kill" {
		deadline, found := ctx.Deadline()
		if !found {
			return "", errors.New("Route configuration operation has no deadline")
		}
		runtime.remainingTimeouts = append(runtime.remainingTimeouts, time.Until(deadline))
		if name == "curl" && runtime.requestFailure != nil {
			return "", runtime.requestFailure
		}
		if name == "docker" {
			runtime.reloadDeadline = deadline
			if runtime.blockReload {
				<-ctx.Done()
				return "", ctx.Err()
			}
			if runtime.reloadDelay != 0 {
				timer := time.NewTimer(runtime.reloadDelay)
				defer timer.Stop()
				select {
				case <-ctx.Done():
					return "", ctx.Err()
				case <-timer.C:
				}
			}
		}
	}
	return runtime.routeRuntimeFake.QueryAtEnvironmentRoot(ctx, target, name, args...)
}

func (runtime *routeTimeoutRuntime) SyncFiles(ctx context.Context, target environmentport.Target, directory string, files []deploymentport.WorkspaceFile, pruneSuffix string) error {
	if runtime.blockCommit && strings.Contains(directory, "/gateway/dynamic") {
		<-ctx.Done()
		return ctx.Err()
	}
	return runtime.routeRuntimeFake.SyncFiles(ctx, target, directory, files, pruneSuffix)
}

func (runtime *routeTimeoutRuntime) ReadFile(ctx context.Context, target environmentport.Target, name string) ([]byte, error) {
	if runtime.recordReadFailure != nil && strings.HasSuffix(name, "/state.json") {
		<-ctx.Done()
		return nil, runtime.recordReadFailure
	}
	return runtime.routeRuntimeFake.ReadFile(ctx, target, name)
}

func TestPublicationRecordTimeoutPreservesSFTPFailure(t *testing.T) {
	failure := errors.New("start SFTP client: ssh: unexpected packet in response to channel open: <nil>")
	timeouts := testRouteTimeouts()
	timeouts.StateLoad = 30 * time.Millisecond
	runtime := &routeTimeoutRuntime{routeRuntimeFake: newRouteRuntimeFake(), recordReadFailure: failure}
	runtime.files["/srv/orbit/traefik-default/.orbit/route-publication/existing/state.json"] = []byte("{}")
	manager := NewRouteManager(routeTargetResolver{}, runtime, timeouts)
	result, err := manager.PublishRoute(context.Background(), "project-1", testGateway(), publicationTestRoute("new"), "")
	if !errors.Is(err, failure) || !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("publication inspection lost its timeout or transport cause: %v", err)
	}
	if result.FileCommit != "not_attempted" || result.Recovery != "not_needed" || len(runtime.writes) != 0 {
		t.Fatalf("inspection failure attempted file publication: result=%+v writes=%v", result, runtime.writes)
	}
}

func TestFilePublicationBudgetDoesNotTruncateReload(t *testing.T) {
	timeouts := testRouteTimeouts()
	timeouts.FilePublication, timeouts.Reload = 300*time.Millisecond, 2*time.Second
	runtime := &routeTimeoutRuntime{routeRuntimeFake: newRouteRuntimeFake(), reloadDelay: 10 * time.Millisecond}
	runtime.autoAPI, runtime.reloadRequired = true, true
	manager := NewRouteManager(routeTargetResolver{targetType: model.EnvironmentTargetTypeSSH, platform: model.EnvironmentPlatformWindows}, runtime, timeouts)
	result, err := manager.PublishRoute(context.Background(), "project-1", testGateway(), publicationTestRoute("independent"), "")
	if err != nil || result.FileCommit != "committed" || result.ConfigurationMatch != "matched" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if len(runtime.remainingTimeouts) != 3 || runtime.remainingTimeouts[0] <= time.Second {
		t.Fatalf("reload did not receive its own budget: %v", runtime.remainingTimeouts)
	}
}

func TestConfigurationMatchBudgetStartsAfterReload(t *testing.T) {
	timeouts := testRouteTimeouts()
	timeouts.ApiRequest, timeouts.Reload, timeouts.ConfigurationMatch = 160*time.Millisecond, 100*time.Millisecond, 80*time.Millisecond
	runtime := &routeTimeoutRuntime{routeRuntimeFake: newRouteRuntimeFake(), reloadDelay: 70 * time.Millisecond}
	runtime.autoAPI, runtime.reloadRequired = true, true
	manager := NewRouteManager(routeTargetResolver{targetType: model.EnvironmentTargetTypeSSH, platform: model.EnvironmentPlatformWindows}, runtime, timeouts)
	route := publicationTestRoute("delayed-reload")
	body, err := yaml.Marshal(buildRouteSnapshot([]model.Route{route}))
	if err != nil {
		t.Fatal(err)
	}
	runtime.files[activeTestPath(route.Name)] = body
	target, _ := manager.resolveTarget(context.Background(), "project-1")
	if err := manager.loadFileConfiguration(context.Background(), "project-1", testGateway(), target, body); err != nil {
		t.Fatal(err)
	}
	if len(runtime.remainingTimeouts) != 3 || runtime.remainingTimeouts[1] < 50*time.Millisecond {
		t.Fatalf("reload consumed the configuration matching budget: %v", runtime.remainingTimeouts)
	}
	for _, remaining := range runtime.remainingTimeouts[1:] {
		if remaining <= 0 || remaining > timeouts.ConfigurationMatch {
			t.Fatalf("API request exceeded the configuration matching budget: %s", remaining)
		}
	}
}

func TestRouteRecoveryUsesIndependentBudgetAfterPublicationTimeout(t *testing.T) {
	timeouts := testRouteTimeouts()
	timeouts.FilePublication, timeouts.ApiRequest, timeouts.Reload, timeouts.ConfigurationMatch, timeouts.Recovery = 15*time.Millisecond, 20*time.Millisecond, 40*time.Millisecond, 30*time.Millisecond, 80*time.Millisecond
	runtime := &routeTimeoutRuntime{routeRuntimeFake: newRouteRuntimeFake(), blockCommit: true, blockReload: true}
	runtime.autoAPI, runtime.reloadRequired = true, true
	manager := NewRouteManager(routeTargetResolver{targetType: model.EnvironmentTargetTypeSSH, platform: model.EnvironmentPlatformWindows}, runtime, timeouts)
	deadline := time.Now().Add(30 * time.Millisecond)
	ctx, cancel := context.WithDeadline(context.Background(), deadline)
	defer cancel()
	result, err := manager.PublishRoute(ctx, "project-1", testGateway(), publicationTestRoute("bounded"), "")
	if !errors.Is(err, context.DeadlineExceeded) || result.Recovery != "failed" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if !runtime.reloadDeadline.After(deadline) {
		t.Fatalf("recovery reused the publication deadline: got %s, parent %s", runtime.reloadDeadline, deadline)
	}
}

func TestRecoveryUsesFreshBoundedContext(t *testing.T) {
	manager := NewRouteManager(routeTargetResolver{}, newRouteRuntimeFake(), testRouteTimeouts())
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	started := time.Now()
	recoveryCtx, stop := manager.recoveryContext(ctx)
	defer stop()
	if recoveryCtx.Err() != nil {
		t.Fatalf("recovery reused the canceled context: %v", recoveryCtx.Err())
	}
	if actual, _ := recoveryCtx.Deadline(); actual.Before(started.Add(manager.timeouts.Recovery)) || actual.After(time.Now().Add(manager.timeouts.Recovery)) {
		t.Fatalf("recovery did not receive its own bounded deadline: %s", actual)
	}
}

func TestConfigurationTimeoutPreservesLastAPIError(t *testing.T) {
	failure := errors.New("remote API connection failed")
	timeouts := testRouteTimeouts()
	timeouts.ConfigurationMatch = 15 * time.Millisecond
	runtime := &routeTimeoutRuntime{routeRuntimeFake: newRouteRuntimeFake(), requestFailure: failure}
	manager := NewRouteManager(routeTargetResolver{}, runtime, timeouts)
	body, _ := yaml.Marshal(buildRouteSnapshot([]model.Route{publicationTestRoute("api")}))
	err := manager.waitConfiguration(context.Background(), "project-1", testGateway(), body)
	if !errors.Is(err, context.DeadlineExceeded) || !errors.Is(err, failure) {
		t.Fatalf("configuration timeout lost its cause: %v", err)
	}
	if classified, ok := apperror.As(err); !ok || classified.Code != "route_sync_configuration_unavailable" {
		t.Fatalf("configuration timeout did not identify the failed stage: %v", err)
	}
}

func TestRoutePublicationUpdatesAndWithdrawsOnlyOwnedFiles(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	unknownFiles := map[string]string{
		"/srv/orbit/traefik-default/gateway/dynamic/custom.yaml": "http:\n  routers: {}\n",
		"/srv/orbit/traefik-default/gateway/certs/custom.pem":    "external certificate",
	}
	for name, body := range unknownFiles {
		runtime.files[name] = []byte(body)
	}
	m := newRouteManager(routeTargetResolver{targetType: model.EnvironmentTargetTypeSSH}, runtime, testRouteTimeouts(), func() bool { return true })
	a, b := publicationTestRoute("a"), publicationTestRoute("b")
	cert, key := generateTestCertificate(t, b.Domain)
	b.HTTPSEnabled, b.CertType, b.CertPEM, b.CertKey = true, "manual", &cert, &key
	publishTestRoute(t, m, a)
	publishTestRoute(t, m, b)
	beforeB := string(runtime.files[activeTestPath(b.Id)])
	beforeWrites := len(runtime.writes)
	a.Domain = "updated.test"
	publishTestRoute(t, m, a)
	a.Enabled = false
	publishTestRoute(t, m, a)
	if _, exists := runtime.files[activeTestPath(a.Id)]; exists {
		t.Fatal("disabled Route file still exists")
	}
	if string(runtime.files[activeTestPath(b.Id)]) != beforeB {
		t.Fatal("publication changed another Route")
	}
	for name, body := range unknownFiles {
		if string(runtime.files[name]) != body {
			t.Fatalf("publication changed unknown file %s", name)
		}
	}
	for _, write := range runtime.writes[beforeWrites:] {
		if strings.Contains(write.Path, "/route-b/") || strings.HasSuffix(write.Path, "route-b.yaml") {
			t.Fatalf("rewrote unrelated file %s", write.Path)
		}
	}
	for _, write := range runtime.writes {
		if write.Mode != 0o600 {
			t.Fatalf("managed file mode=%o", write.Mode)
		}
	}
	if runtime.lastTarget.Environment.TargetType != model.EnvironmentTargetTypeSSH {
		t.Fatal("publication ignored SSH target")
	}
}

func TestCertificateRevisionIsReusedWithoutRewritingAndRejectsCorruption(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	route := publicationTestRoute("secure")
	cert, key := generateTestCertificate(t, route.Domain)
	route.HTTPSEnabled, route.CertType, route.CertPEM, route.CertKey = true, "manual", &cert, &key
	publishTestRoute(t, m, route)
	count := len(runtime.writes)
	result := publishTestRoute(t, m, route)
	if result.ConfigurationMatch != "matched" || result.CertificateVerification != "unverified" {
		t.Fatalf("result=%+v", result)
	}
	for _, write := range runtime.writes[count:] {
		if strings.Contains(write.Path, "/gateway/certs/") {
			t.Fatal("immutable certificate was rewritten")
		}
	}
	keyPath := path.Join("/srv/orbit/traefik-default/gateway/certs", "route-secure", certificateRevision(route), "key.pem")
	runtime.files[keyPath] = []byte("corrupt")
	before := string(runtime.files[activeTestPath(route.Id)])
	_, err := m.PublishRoute(context.Background(), "project-1", testGateway(), route, testPublicationFingerprint(t, m, route.Id))
	if err == nil || string(runtime.files[activeTestPath(route.Id)]) != before {
		t.Fatalf("corruption err=%v", err)
	}
}

func TestPublicationFailureRestoresOnlyPreviousRoute(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	a, b := publicationTestRoute("a"), publicationTestRoute("b")
	publishTestRoute(t, m, a)
	publishTestRoute(t, m, b)
	previous := string(runtime.files[activeTestPath(a.Id)])
	other := string(runtime.files[activeTestPath(b.Id)])
	runtime.writeFailures = map[string]error{activeTestPath(a.Id): os.ErrPermission}
	a.TargetUrl = "http://new-a:8080"
	result, err := m.PublishRoute(context.Background(), "project-1", testGateway(), a, testPublicationFingerprint(t, m, a.Id))
	if err == nil || result.Recovery != "restored" || string(runtime.files[activeTestPath(a.Id)]) != previous || string(runtime.files[activeTestPath(b.Id)]) != other {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestInterruptedCandidateIsConfirmedBeforeRetryChangesBackup(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI, runtime.reloadRequired = true, true
	runtime.physicalBase = "D:/orbit/traefik-default"
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	route := publicationTestRoute("a")
	publishTestRoute(t, m, route)
	items, _ := inspectTestPublications(m, context.Background(), "project-1", testGateway())
	previous := items[0]
	backup := "/srv/orbit/traefik-default/.orbit/route-publication/a/previous.yaml"
	runtime.files[backup] = append([]byte(nil), runtime.files[activeTestPath("a")]...)
	route.Domain = "candidate.test"
	body, _ := yaml.Marshal(buildRouteSnapshot([]model.Route{route}))
	runtime.files[activeTestPath("a")] = body
	pending := routeport.Publication{Route: route, Phase: "pending", Fingerprint: digest(body), PreviousFingerprint: previous.Fingerprint, Previous: &previous, TargetRevision: previous.TargetRevision, GatewayApplicationId: previous.GatewayApplicationId}
	target, _, base, _ := m.publicationWorkspace(context.Background(), "project-1", testGateway())
	if err := m.savePublication(context.Background(), target, base, pending); err != nil {
		t.Fatal(err)
	}
	route.Domain = "retry.test"
	publishTestRoute(t, m, route)
	if digest(runtime.files[backup]) != pending.Fingerprint {
		t.Fatal("retry lost the last loaded candidate backup")
	}
}

func TestPendingRecoveryPreservesUnknownExternalFile(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	route := publicationTestRoute("a")
	publishTestRoute(t, m, route)
	items, _ := inspectTestPublications(m, context.Background(), "project-1", testGateway())
	item := items[0]
	item.Phase = "pending"
	item.PreviousFingerprint = item.Fingerprint
	target, _, base, _ := m.publicationWorkspace(context.Background(), "project-1", testGateway())
	if err := m.savePublication(context.Background(), target, base, item); err != nil {
		t.Fatal(err)
	}
	runtime.files[activeTestPath("a")] = []byte("external change")
	_, err := m.PublishRoute(context.Background(), "project-1", testGateway(), route, testPublicationFingerprint(t, m, "a"))
	if err == nil || string(runtime.files[activeTestPath("a")]) != "external change" {
		t.Fatalf("recovery err=%v", err)
	}
}

func TestPendingCertificateRecoveryRequiresIntactCertificateFiles(t *testing.T) {
	for _, missingPrevious := range []bool{false, true} {
		t.Run(map[bool]string{false: "restore previous", true: "preserve recovery material"}[missingPrevious], func(t *testing.T) {
			runtime := newRouteRuntimeFake()
			runtime.autoAPI, runtime.reloadRequired = true, true
			runtime.physicalBase = "D:/orbit/traefik-default"
			m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
			route := publicationTestRoute("secure")
			cert, key := generateTestCertificate(t, route.Domain)
			route.HTTPSEnabled, route.CertType, route.CertPEM, route.CertKey = true, "manual", &cert, &key
			publishTestRoute(t, m, route)
			items, err := inspectTestPublications(m, context.Background(), "project-1", testGateway())
			if err != nil {
				t.Fatal(err)
			}
			previous := items[0]
			previousBody := append([]byte(nil), runtime.files[activeTestPath(route.Id)]...)
			target, files, base, err := m.publicationWorkspace(context.Background(), "project-1", testGateway())
			if err != nil {
				t.Fatal(err)
			}
			runtime.files[path.Join(base, ".orbit/route-publication", route.Id, "previous.yaml")] = previousBody
			cert, key = generateTestCertificate(t, route.Domain)
			body, err := yaml.Marshal(buildRouteSnapshot([]model.Route{route}))
			if err != nil {
				t.Fatal(err)
			}
			runtime.files[activeTestPath(route.Id)] = body
			pending := previous
			pending.Phase, pending.Previous = "pending", &previous
			pending.Fingerprint, pending.ActualFingerprint = digest(body), digest(body)
			pending.PreviousFingerprint = previous.Fingerprint
			pending.CertificateRevision, pending.ActualCertificateRevision = certificateRevision(route), ""
			if missingPrevious {
				delete(runtime.files, path.Join(base, "gateway/certs/route-secure", previous.CertificateRevision, "key.pem"))
			}
			recovered, err := m.reconcilePending(context.Background(), "project-1", testGateway(), target, files, base, pending)
			if missingPrevious {
				if err == nil || string(runtime.files[activeTestPath(route.Id)]) != string(body) {
					t.Fatalf("err=%v recovered=%+v", err, recovered)
				}
				return
			}
			if err != nil || recovered.CertificateRevision != previous.CertificateRevision || string(runtime.files[activeTestPath(route.Id)]) != string(previousBody) {
				t.Fatalf("err=%v recovered=%+v", err, recovered)
			}
			if runtime.reloadAttempts != 2 || string(runtime.loadedFiles[activeTestPath(route.Id)]) != string(previousBody) {
				t.Fatal("recovery did not reload the previous configuration")
			}
		})
	}
}

func TestGatewayPublicationLockIsSharedAcrossManagers(t *testing.T) {
	m1, m2 := &RouteManager{}, &RouteManager{}
	unlock, err := m1.LockGateway(context.Background(), "lock-test-project")
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if _, err := m2.LockGateway(ctx, "lock-test-project"); err == nil {
		t.Fatal("second manager acquired a locked Gateway")
	}
	unlock()
	unlock, err = m2.LockGateway(context.Background(), "lock-test-project")
	if err != nil {
		t.Fatal(err)
	}
	unlock()
}

func TestPublishingCertificateConfigurationDoesNotReadOtherRoutes(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, testRouteTimeouts(), func() bool { return false })
	a := publicationTestRoute("a")
	cert, key := generateTestCertificate(t, a.Domain)
	a.HTTPSEnabled, a.CertType, a.CertPEM, a.CertKey = true, "manual", &cert, &key
	publishTestRoute(t, m, a)
	b := publicationTestRoute("b")
	b.Domain = a.Domain
	b.PathPrefix = "/b"
	b.HTTPSEnabled = true
	b.CertType = "letsencrypt"
	before := string(runtime.files[activeTestPath(a.Name)])
	runtime.reads = nil
	publishTestRoute(t, m, b)
	if string(runtime.files[activeTestPath(a.Name)]) != before {
		t.Fatal("publishing another Route changed the first Route certificate configuration")
	}
	for _, name := range runtime.reads {
		if name == publicationRecordPath(a.Id) || name == activeTestPath(a.Name) || strings.Contains(name, "/certs/route-"+a.Id+"/") {
			t.Fatalf("publishing another Route read unrelated certificate configuration: %s", name)
		}
	}
}

func publicationTestRoute(id string) model.Route {
	projectId := "project-1"
	return model.Route{Id: id, ProjectId: &projectId, Name: id, Protocol: "http", Domain: id + ".test", PathPrefix: "/", TargetUrl: "http://" + id + ":8080", Enabled: true}
}
func activeTestPath(id string) string {
	return "/srv/orbit/traefik-default/gateway/dynamic/route-" + id + ".yaml"
}
func inspectTestPublications(m *RouteManager, ctx context.Context, projectId string, gateway model.GatewayConfig) ([]routeport.Publication, error) {
	ids, err := m.ListPublicationRouteIds(ctx, projectId, gateway)
	if err != nil {
		return nil, err
	}
	items := make([]routeport.Publication, 0, len(ids))
	for _, id := range ids {
		item, err := m.InspectPublication(ctx, projectId, gateway, model.Route{Id: id})
		if err != nil {
			return nil, err
		}
		if item != nil {
			items = append(items, *item)
		}
	}
	return items, nil
}

func testPublicationFingerprint(t *testing.T, m *RouteManager, id string) string {
	t.Helper()
	item, err := m.InspectPublication(context.Background(), "project-1", testGateway(), model.Route{Id: id})
	if err != nil {
		t.Fatal(err)
	}
	if item != nil {
		return publicationFingerprint(*item)
	}
	return ""
}
func publishTestRoute(t *testing.T, m *RouteManager, route model.Route) routeport.PublicationResult {
	t.Helper()
	result, err := m.PublishRoute(context.Background(), "project-1", testGateway(), route, testPublicationFingerprint(t, m, route.Id))
	if err != nil {
		t.Fatalf("publish %s: %v (%+v)", route.Id, err, result)
	}
	return result
}
func generateTestCertificate(t *testing.T, domain string) (string, string) {
	t.Helper()
	key, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	serial, err := rand.Int(rand.Reader, new(big.Int).Lsh(big.NewInt(1), 128))
	if err != nil {
		t.Fatal(err)
	}
	template := x509.Certificate{SerialNumber: serial, Subject: pkix.Name{CommonName: domain}, DNSNames: []string{domain}, NotBefore: time.Now().Add(-time.Hour), NotAfter: time.Now().Add(time.Hour), KeyUsage: x509.KeyUsageDigitalSignature, ExtKeyUsage: []x509.ExtKeyUsage{x509.ExtKeyUsageServerAuth}}
	der, err := x509.CreateCertificate(rand.Reader, &template, &template, &key.PublicKey, key)
	if err != nil {
		t.Fatal(err)
	}
	keyDER, err := x509.MarshalECPrivateKey(key)
	if err != nil {
		t.Fatal(err)
	}
	cert := string(pem.EncodeToMemory(&pem.Block{Type: "CERTIFICATE", Bytes: der}))
	privateKey := string(pem.EncodeToMemory(&pem.Block{Type: "EC PRIVATE KEY", Bytes: keyDER}))
	if _, err := tls.X509KeyPair([]byte(cert), []byte(privateKey)); err != nil {
		t.Fatal(err)
	}
	return cert, privateKey
}

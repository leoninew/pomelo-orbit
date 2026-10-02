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
			m := newRouteManager(routeTargetResolver{targetType: tc.targetType, platform: tc.platform}, runtime, func() bool { return tc.inContainer })
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

func TestWindowsPublicationReloadsUpdatesAndWithdrawalWithoutChangingOtherRoutes(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI, runtime.reloadRequired = true, true
	runtime.physicalBase = "D:/orbit/traefik-default"
	m := newRouteManager(routeTargetResolver{}, runtime, func() bool { return false })
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
			m := newRouteManager(routeTargetResolver{}, runtime, func() bool { return false })
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
	m := newRouteManager(routeTargetResolver{targetType: model.EnvironmentTargetTypeSSH}, runtime, func() bool { return true })
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
	m := newRouteManager(routeTargetResolver{}, runtime, func() bool { return false })
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
	m := newRouteManager(routeTargetResolver{}, runtime, func() bool { return false })
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
	m := newRouteManager(routeTargetResolver{}, runtime, func() bool { return false })
	route := publicationTestRoute("a")
	publishTestRoute(t, m, route)
	items, _ := m.InspectPublications(context.Background(), "project-1", testGateway())
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
	m := newRouteManager(routeTargetResolver{}, runtime, func() bool { return false })
	route := publicationTestRoute("a")
	publishTestRoute(t, m, route)
	items, _ := m.InspectPublications(context.Background(), "project-1", testGateway())
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
			m := newRouteManager(routeTargetResolver{}, runtime, func() bool { return false })
			route := publicationTestRoute("secure")
			cert, key := generateTestCertificate(t, route.Domain)
			route.HTTPSEnabled, route.CertType, route.CertPEM, route.CertKey = true, "manual", &cert, &key
			publishTestRoute(t, m, route)
			items, err := m.InspectPublications(context.Background(), "project-1", testGateway())
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

func TestPublishedManualCertificateConflictsWithCandidateACME(t *testing.T) {
	runtime := newRouteRuntimeFake()
	runtime.autoAPI = true
	m := newRouteManager(routeTargetResolver{}, runtime, func() bool { return false })
	a := publicationTestRoute("a")
	cert, key := generateTestCertificate(t, a.Domain)
	a.HTTPSEnabled, a.CertType, a.CertPEM, a.CertKey = true, "manual", &cert, &key
	publishTestRoute(t, m, a)
	b := publicationTestRoute("b")
	b.Domain = a.Domain
	b.PathPrefix = "/b"
	b.HTTPSEnabled = true
	b.CertType = "letsencrypt"
	_, err := m.PublishRoute(context.Background(), "project-1", testGateway(), b, "")
	if err == nil {
		t.Fatal("ACME candidate replaced a published manual certificate")
	}
	if _, exists := runtime.files[activeTestPath("b")]; exists {
		t.Fatal("conflicting candidate was committed")
	}
}

func publicationTestRoute(id string) model.Route {
	projectId := "project-1"
	return model.Route{Id: id, ProjectId: &projectId, Name: id, Protocol: "http", Domain: id + ".test", PathPrefix: "/", TargetUrl: "http://" + id + ":8080", Enabled: true}
}
func activeTestPath(id string) string {
	return "/srv/orbit/traefik-default/gateway/dynamic/route-" + id + ".yaml"
}
func testPublicationFingerprint(t *testing.T, m *RouteManager, id string) string {
	t.Helper()
	items, err := m.InspectPublications(context.Background(), "project-1", testGateway())
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range items {
		if item.Route.Id == id {
			return publicationFingerprint(item)
		}
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

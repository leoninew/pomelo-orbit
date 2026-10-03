package traefik

import (
	"context"
	"crypto/tls"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	localrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/local"
	runtimepath "github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestFileProviderRestoresRoutesAfterTraefikRestart(t *testing.T) {
	if os.Getenv("POMELO_ORBIT_TRAEFIK_E2E") != "1" {
		t.Skip("set POMELO_ORBIT_TRAEFIK_E2E=1 to run with Docker and traefik:3.6")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	root := t.TempDir()
	runtime := localrunner.NewRuntime(runtimepath.ResolveDockerDaemonPath)
	resolver := routeTargetResolver{workspaceRoot: root}
	manager := newRouteManager(resolver, runtime, testRouteTimeouts(), func() bool { return false })
	gateway := testGateway()
	target, _ := resolver.ResolveProjectTarget(ctx, "project-1")
	base, err := runtime.ServiceDir(target, gateway.RuntimeServiceCode)
	if err != nil {
		t.Fatal(err)
	}
	physicalBase, err := runtime.ComposeMountSourceDir(ctx, target, gateway.RuntimeServiceCode)
	if err != nil {
		t.Fatal(err)
	}
	container := "orbit-route-restart-" + strings.ToLower(idutil.NewId())
	composeFile := filepath.Join(base, "docker-compose.yml")
	compose := fmt.Sprintf("name: %s\nservices:\n  traefik:\n    image: traefik:3.6\n    container_name: %s\n    ports: [\"127.0.0.1::80\", \"127.0.0.1::443\", \"127.0.0.1::8080\"]\n    volumes:\n      - '%s/traefik.yml:/etc/traefik/traefik.yml:ro'\n      - '%s/gateway/dynamic:/etc/traefik/dynamic:ro'\n      - '%s/gateway/certs:/etc/traefik/certs:ro'\n      - '%s/gateway/acme:/letsencrypt'\n", container, container, filepath.ToSlash(physicalBase), filepath.ToSlash(physicalBase), filepath.ToSlash(physicalBase), filepath.ToSlash(physicalBase))
	static := "api:\n  insecure: true\nentryPoints:\n  web:\n    address: :80\n  websecure:\n    address: :443\nproviders:\n  file:\n    directory: /etc/traefik/dynamic\n    watch: true\n"
	if err := runtime.StageWorkspace(ctx, target, deploymentport.Workspace{ServiceCode: gateway.RuntimeServiceCode, Compose: compose, Directories: []string{filepath.Join(base, "gateway", "dynamic"), filepath.Join(base, "gateway", "certs"), filepath.Join(base, "gateway", "acme")}, Files: []deploymentport.WorkspaceFile{{Path: filepath.Join(base, "traefik.yml"), Content: []byte(static), Mode: 0o644}}}); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cleanupCancel()
		if output, err := exec.CommandContext(cleanupCtx, "docker", "compose", "--file", composeFile, "down").CombinedOutput(); err != nil {
			t.Errorf("remove isolated test deployment: %v: %s", err, output)
		}
	})
	runTraefikTestDocker(t, ctx, "compose", "--file", composeFile, "up", "--detach")
	var apiAddress, webAddress, tlsAddress, webURL string
	if runtimepath.IsRunningInContainer() {
		hostname, err := os.Hostname()
		if err != nil {
			t.Fatal(err)
		}
		runTraefikTestDocker(t, ctx, "network", "connect", container+"_default", hostname)
		t.Cleanup(func() { _ = exec.Command("docker", "network", "disconnect", container+"_default", hostname).Run() })
	}
	refreshAddresses := func() {
		if runtimepath.IsRunningInContainer() {
			address := runTraefikTestDocker(t, ctx, "inspect", container, "--format", "{{range .NetworkSettings.Networks}}{{.IPAddress}}{{end}}")
			apiAddress, webAddress, tlsAddress = address+":8080", address+":80", address+":443"
		} else {
			apiAddress = waitForTraefikTestPort(t, ctx, container, "8080/tcp")
			webAddress = waitForTraefikTestPort(t, ctx, container, "80/tcp")
			tlsAddress = waitForTraefikTestPort(t, ctx, container, "443/tcp")
		}
		gateway.RestApiHostUrl = "http://" + apiAddress
		webURL = "http://" + webAddress + "/api/overview"
	}
	refreshAddresses()
	if err := manager.WaitUntilReady(ctx, "project-1", gateway, 15*time.Second); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.ValidateGateway(ctx, "project-1", gateway, nil); err != nil {
		t.Fatal(err)
	}
	a, b := publicationTestRoute("a"), publicationTestRoute("b")
	a.Id, b.Id = "id-a", "id-b"
	a.TargetUrl, b.TargetUrl = "http://127.0.0.1:8080", "http://127.0.0.1:8080"
	cert, key := generateTestCertificate(t, b.Domain)
	b.HTTPSEnabled, b.CertType, b.CertPEM, b.CertKey = true, "manual", &cert, &key
	publish := func(route model.Route) {
		t.Helper()
		startedAt := runTraefikTestDocker(t, ctx, "inspect", container, "--format", "{{.State.StartedAt}}")
		observeCtx, stopObserve := context.WithCancel(ctx)
		observed := make(chan struct{})
		go func() {
			defer close(observed)
			select {
			case <-observeCtx.Done():
				return
			case <-time.After(3 * time.Second):
				routers, routerErr := manager.ListRouters(observeCtx, "project-1", gateway)
				services, serviceErr := manager.ListServices(observeCtx, "project-1", gateway)
				t.Logf("Slow publication: routers=%+v (%v), services=%+v (%v)", routers, routerErr, services, serviceErr)
			}
		}()
		defer func() { stopObserve(); <-observed }()
		item, err := manager.InspectPublication(ctx, "project-1", gateway, route)
		if err != nil {
			t.Fatal(err)
		}
		fingerprint := ""
		if item != nil {
			fingerprint = publicationFingerprint(*item)
		}
		result, err := manager.PublishRoute(ctx, "project-1", gateway, route, fingerprint)
		if err != nil || result.ConfigurationMatch != "matched" {
			logs, _ := exec.CommandContext(ctx, "docker", "logs", container).CombinedOutput()
			t.Logf("Test Traefik logs: %s", logs)
			t.Fatalf("publication=%+v err=%v", result, err)
		}
		if runTraefikTestDocker(t, ctx, "inspect", container, "--format", "{{.State.StartedAt}}") != startedAt {
			t.Fatal("Route publication restarted Traefik")
		}
	}
	publish(a)
	publish(b)
	waitForTraefikTestResponse(t, ctx, webURL, a.Domain, http.StatusOK)
	checkCertificate := func() {
		t.Helper()
		pair, _ := tls.X509KeyPair([]byte(cert), []byte(key))
		dialer := &tls.Dialer{NetDialer: &net.Dialer{Timeout: 3 * time.Second}, Config: &tls.Config{ServerName: b.Domain, InsecureSkipVerify: true}} // Inspect the exact published self-signed leaf.
		waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
		defer cancel()
		for {
			conn, err := dialer.DialContext(waitCtx, "tcp", tlsAddress)
			if err == nil {
				leaf := conn.(*tls.Conn).ConnectionState().PeerCertificates[0]
				_ = conn.Close()
				if digest(leaf.Raw) == digest(pair.Certificate[0]) {
					return
				}
			}
			select {
			case <-waitCtx.Done():
				t.Fatalf("Traefik did not serve the published certificate: %v", err)
			case <-time.After(100 * time.Millisecond):
			}
		}
	}
	checkCertificate()
	runTraefikTestDocker(t, ctx, "restart", container)
	refreshAddresses()
	waitForTraefikTestResponse(t, ctx, webURL, a.Domain, http.StatusOK)
	checkCertificate()
	bPath := filepath.Join(base, "gateway", "dynamic", "route-b.yaml")
	beforeB, err := os.ReadFile(bPath)
	if err != nil {
		t.Fatal(err)
	}
	a.Domain = "updated.test"
	a.Name = "new-a"
	publish(a)
	if _, err := os.Stat(filepath.Join(base, "gateway", "dynamic", "route-a.yaml")); !os.IsNotExist(err) {
		t.Fatalf("rename left the previous Route file: %v", err)
	}
	waitForTraefikTestResponse(t, ctx, webURL, a.Domain, http.StatusOK)
	waitForTraefikTestResponse(t, ctx, webURL, "a.test", http.StatusNotFound)
	a.Enabled = false
	publish(a)
	runTraefikTestDocker(t, ctx, "restart", container)
	refreshAddresses()
	waitForTraefikTestResponse(t, ctx, webURL, a.Domain, http.StatusNotFound)
	checkCertificate()
	afterB, err := os.ReadFile(bPath)
	if err != nil || string(afterB) != string(beforeB) {
		t.Fatal("publishing A modified B")
	}
	cert, key = generateTestCertificate(t, b.Domain)
	b.CertPEM, b.CertKey = &cert, &key
	publish(b)
	checkCertificate()
	runTraefikTestDocker(t, ctx, "compose", "--file", composeFile, "up", "--detach", "--force-recreate")
	refreshAddresses()
	if err := manager.WaitUntilReady(ctx, "project-1", gateway, 15*time.Second); err != nil {
		t.Fatal(err)
	}
	checkCertificate()
	activeConfiguration, err := os.ReadFile(bPath)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.waitConfiguration(ctx, "project-1", gateway, activeConfiguration); err != nil {
		t.Fatal(err)
	}
	// Recreate an intentionally removed test directory, then explicitly publish
	// its missing Route; Gateway staging must not publish editable business data.
	dynamicDirectory := filepath.Join(base, "gateway", "dynamic")
	if err := os.Remove(bPath); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(dynamicDirectory); err != nil {
		t.Fatal(err)
	}
	if err := runtime.StageWorkspace(ctx, target, deploymentport.Workspace{ServiceCode: gateway.RuntimeServiceCode, Compose: compose, Directories: []string{dynamicDirectory}}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dynamicDirectory)
	if err != nil || len(entries) != 0 {
		t.Fatalf("staging did not recreate an empty dynamic directory: entries=%v err=%v", entries, err)
	}
	runTraefikTestDocker(t, ctx, "compose", "--file", composeFile, "up", "--detach", "--force-recreate")
	refreshAddresses()
	if err := manager.WaitUntilReady(ctx, "project-1", gateway, 15*time.Second); err != nil {
		t.Fatal(err)
	}
	waitForTraefikTestResponse(t, ctx, webURL, b.Domain, http.StatusNotFound)
	publish(b)
	checkCertificate()
	waitForTraefikTestResponse(t, ctx, webURL, b.Domain, http.StatusOK)
}

func runTraefikTestDocker(t *testing.T, ctx context.Context, args ...string) string {
	t.Helper()
	output, err := exec.CommandContext(ctx, "docker", args...).CombinedOutput()
	if err != nil {
		t.Fatalf("docker %s: %v: %s", args[0], err, output)
	}
	return strings.TrimSpace(string(output))
}

func waitForTraefikTestPort(t *testing.T, ctx context.Context, container, port string) string {
	t.Helper()
	waitCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()
	for {
		output, err := exec.CommandContext(waitCtx, "docker", "port", container, port).CombinedOutput()
		if err == nil && strings.TrimSpace(string(output)) != "" {
			return strings.TrimSpace(string(output))
		}
		select {
		case <-waitCtx.Done():
			t.Fatalf("Traefik port %s did not become available: %v: %s", port, err, output)
		case <-time.After(100 * time.Millisecond):
		}
	}
}

func waitForTraefikTestResponse(t *testing.T, ctx context.Context, url, domain string, status int) {
	t.Helper()
	waitCtx, cancel := context.WithTimeout(ctx, 15*time.Second)
	defer cancel()
	client := &http.Client{Timeout: 2 * time.Second}
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()
	var lastErr error
	for {
		req, err := http.NewRequestWithContext(waitCtx, http.MethodGet, url, nil)
		if err != nil {
			t.Fatal(err)
		}
		req.Host = domain
		response, err := client.Do(req)
		lastErr = err
		if err == nil {
			_, _ = io.Copy(io.Discard, response.Body)
			_ = response.Body.Close()
			if response.StatusCode == status {
				return
			}
			lastErr = fmt.Errorf("HTTP status %d, want %d", response.StatusCode, status)
		}
		select {
		case <-waitCtx.Done():
			t.Fatalf("route %s did not become ready: %v", domain, lastErr)
		case <-ticker.C:
		}
	}
}

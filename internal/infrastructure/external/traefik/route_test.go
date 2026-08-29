package traefik

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestRouteManagerListRoutersMapsTraefikResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || (request.URL.Path != "/api/http/routers" && request.URL.Path != "/api/tcp/routers") {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/api/tcp/routers" {
			_, _ = writer.Write([]byte(`[{"name":"redis@rest","provider":"rest","status":"enabled","rule":"HostSNI(` + "`*`" + `)","service":"redis-service","entryPoints":["tcp16379"],"tls":null}]`))
			return
		}
		_, _ = writer.Write([]byte(`[{"name":"api@docker","provider":"docker","status":"enabled","rule":"Host(api.example.test)","service":"api-service","entryPoints":["websecure"],"tls":{}},{"name":"dashboard@file","provider":"file","status":"disabled","rule":"Host(dashboard.example.test)","service":"dashboard-service","entryPoints":["web"],"tls":null}]`))
	}))
	defer server.Close()

	manager := NewRouteManager(config.Config{})
	routers, err := manager.ListRouters(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(routers) != 3 {
		t.Fatalf("expected two routers, got %+v", routers)
	}
	if routers[0].Name != "api@docker" || routers[0].Provider != "docker" || routers[0].Status != "enabled" || routers[0].Rule != "Host(api.example.test)" || routers[0].Service != "api-service" || len(routers[0].Entrypoints) != 1 || routers[0].Entrypoints[0] != "websecure" || !routers[0].TLS || routers[0].TLSConfig != "{}" {
		t.Fatalf("unexpected first router: %+v", routers[0])
	}
	if routers[1].Name != "dashboard@file" || routers[1].TLS || len(routers[1].Entrypoints) != 1 || routers[1].Entrypoints[0] != "web" {
		t.Fatalf("unexpected second router: %+v", routers[1])
	}
	if routers[2].Name != "redis@rest" || routers[2].TLS || routers[2].Rule != "HostSNI(`*`)" || routers[2].Entrypoints[0] != "tcp16379" {
		t.Fatalf("unexpected TCP router: %+v", routers[2])
	}
}

func TestRouteManagerListServicesMapsHTTPAndTCPResponses(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || (request.URL.Path != "/api/http/services" && request.URL.Path != "/api/tcp/services") {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		if request.URL.Path == "/api/tcp/services" {
			_, _ = writer.Write([]byte(`[{"name":"redis-service@rest","provider":"rest","status":"enabled","loadBalancer":{"servers":[{"address":"redis:6379"}]}}]`))
			return
		}
		_, _ = writer.Write([]byte(`[{"name":"api-service@rest","provider":"rest","status":"enabled","loadBalancer":{"servers":[{"url":"http://api:8080"}]}},{"name":"docker-service@docker","provider":"docker","status":"enabled","loadBalancer":{"servers":[{"url":"http://docker:8080"}]}}]`))
	}))
	defer server.Close()

	manager := NewRouteManager(config.Config{})
	services, err := manager.ListServices(context.Background(), server.URL)
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 3 {
		t.Fatalf("services = %+v, want 3 items", services)
	}
	if services[0].Name != "api-service@rest" || services[0].Protocol != "http" || len(services[0].Servers) != 1 || services[0].Servers[0] != "http://api:8080" {
		t.Fatalf("unexpected HTTP service: %+v", services[0])
	}
	if services[2].Name != "redis-service@rest" || services[2].Protocol != "tcp" || len(services[2].Servers) != 1 || services[2].Servers[0] != "redis:6379" {
		t.Fatalf("unexpected TCP service: %+v", services[2])
	}
}

func TestRouteManagerListRoutersReturnsErrorForInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	manager := NewRouteManager(config.Config{})
	if _, err := manager.ListRouters(context.Background(), server.URL); err == nil {
		t.Fatal("expected error for non-success Traefik response")
	}
}

func TestRouteManagerWaitUntilReadyPollsUntilApiAccepts(t *testing.T) {
	attempts := 0
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/overview" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		attempts++
		if attempts < 3 {
			writer.WriteHeader(http.StatusBadGateway)
			return
		}
		writer.WriteHeader(http.StatusOK)
		_, _ = writer.Write([]byte(`{}`))
	}))
	defer server.Close()

	manager := NewRouteManager(config.Config{Traefik: config.TraefikConfig{RestReadyTimeout: 5 * time.Second}})
	if err := manager.WaitUntilReady(context.Background(), server.URL, 5*time.Second); err != nil {
		t.Fatal(err)
	}
	if attempts < 3 {
		t.Fatalf("attempts = %d, want at least 3", attempts)
	}
}

func TestRouteManagerApplySnapshotPutsFullRestConfig(t *testing.T) {
	var gotMethod, gotPath, gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		gotMethod = request.Method
		gotPath = request.URL.Path
		raw, _ := io.ReadAll(request.Body)
		gotBody = string(raw)
		if request.Header.Get("Content-Type") != "application/json" {
			t.Fatalf("expected application/json, got %q", request.Header.Get("Content-Type"))
		}
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewRouteManager(config.Config{})
	err := manager.ApplySnapshot(context.Background(), model.GatewayConfig{RestApiUrl: server.URL, RuntimeServiceCode: "traefik-default"}, []model.Route{
		{Name: "api", Protocol: "http", Domain: "api.example.test", PathPrefix: "/v1", TargetUrl: "http://app:8080", Enabled: true},
		{Name: "off", Protocol: "http", Domain: "off.example.test", PathPrefix: "/", TargetUrl: "http://app:8081", Enabled: false},
		{Name: "secure", Protocol: "http", Domain: "secure.example.test", PathPrefix: "/", TargetUrl: "http://app:8082", Enabled: true, HTTPSEnabled: true, CertType: "letsencrypt"},
		{Name: "redis", Protocol: "tcp", Domain: "redis.example.test", ListenPort: intPtr(16379), TargetAddress: "redis-redis", TargetPort: 6379, Enabled: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if gotMethod != http.MethodPut || gotPath != "/api/providers/rest" {
		t.Fatalf("unexpected request: %s %s", gotMethod, gotPath)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatal(err)
	}
	httpCfg := payload["http"].(map[string]any)
	routers := httpCfg["routers"].(map[string]any)
	services := httpCfg["services"].(map[string]any)
	if len(routers) != 2 || len(services) != 2 {
		t.Fatalf("expected 2 enabled routers/services, got routers=%d services=%d body=%s", len(routers), len(services), gotBody)
	}
	apiRouter := routers["api-route"].(map[string]any)
	if apiRouter["rule"] != "Host(`api.example.test`) && PathPrefix(`/v1`)" {
		t.Fatalf("unexpected api rule: %v", apiRouter["rule"])
	}
	secureRouter := routers["secure-route"].(map[string]any)
	tls := secureRouter["tls"].(map[string]any)
	if tls["certResolver"] != "letsencrypt" {
		t.Fatalf("expected letsencrypt certResolver, got %#v", tls)
	}
	tcpCfg := payload["tcp"].(map[string]any)
	tcpRouters := tcpCfg["routers"].(map[string]any)
	tcpServices := tcpCfg["services"].(map[string]any)
	redisRouter := tcpRouters["redis-route"].(map[string]any)
	if redisRouter["rule"] != "HostSNI(`*`)" || redisRouter["entryPoints"].([]any)[0] != "tcp16379" {
		t.Fatalf("unexpected TCP router: %#v", redisRouter)
	}
	address := tcpServices["redis-service"].(map[string]any)["loadBalancer"].(map[string]any)["servers"].([]any)[0].(map[string]any)["address"]
	if address != "redis-redis:6379" {
		t.Fatalf("unexpected TCP server address: %v", address)
	}
}

func intPtr(value int) *int { return &value }

func TestRouteManagerApplySnapshotClearsWithEmptyMaps(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		gotBody = string(raw)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewRouteManager(config.Config{})
	if err := manager.ApplySnapshot(context.Background(), model.GatewayConfig{RestApiUrl: server.URL, RuntimeServiceCode: "traefik-default"}, nil); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatal(err)
	}
	httpCfg := payload["http"].(map[string]any)
	if len(httpCfg["routers"].(map[string]any)) != 0 || len(httpCfg["services"].(map[string]any)) != 0 {
		t.Fatalf("expected empty routers/services maps, got %s", gotBody)
	}
	tcpCfg := payload["tcp"].(map[string]any)
	if len(tcpCfg["routers"].(map[string]any)) != 0 || len(tcpCfg["services"].(map[string]any)) != 0 {
		t.Fatalf("expected empty TCP routers/services maps, got %s", gotBody)
	}
}

func TestRouteManagerApplySnapshotPublishesAndPrunesStoredCertificate(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		gotBody = string(raw)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	deploymentRoot := t.TempDir()
	certificate := "certificate"
	key := "private key"
	manager := NewRouteManager(config.Config{Workspace: config.WorkspaceConfig{Deployment: deploymentRoot}})
	err := manager.ApplySnapshot(context.Background(), model.GatewayConfig{RestApiUrl: server.URL, RuntimeServiceCode: "traefik-default"}, []model.Route{{
		Name: "secure", Protocol: "http", Domain: "secure.example.test", PathPrefix: "/", TargetUrl: "http://app:8080", Enabled: true,
		HTTPSEnabled: true, CertType: "manual", CertPEM: &certificate, CertKey: &key,
	}})
	if err != nil {
		t.Fatal(err)
	}

	certificatePath := filepath.Join(deploymentRoot, "traefik-default", "gateway", "certs", "secure.pem")
	if content, err := os.ReadFile(certificatePath); err != nil || string(content) != certificate {
		t.Fatalf("stored certificate = %q, err=%v", content, err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(gotBody), &payload); err != nil {
		t.Fatal(err)
	}
	certificates := payload["tls"].(map[string]any)["certificates"].([]any)
	if len(certificates) != 1 {
		t.Fatalf("TLS certificates = %#v", certificates)
	}
	entry := certificates[0].(map[string]any)
	if entry["certFile"] != "/etc/traefik/certs/secure.pem" || entry["keyFile"] != "/etc/traefik/certs/secure-key.pem" {
		t.Fatalf("TLS certificate entry = %#v", entry)
	}
	stalePath := filepath.Join(deploymentRoot, "traefik-default", "gateway", "certs", "stale.pem")
	if err := os.WriteFile(stalePath, []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.ApplySnapshot(context.Background(), model.GatewayConfig{RestApiUrl: server.URL, RuntimeServiceCode: "traefik-default"}, nil); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(certificatePath); !os.IsNotExist(err) {
		t.Fatalf("expected secure certificate to be pruned, err=%v", err)
	}
	if _, err := os.Stat(stalePath); !os.IsNotExist(err) {
		t.Fatalf("expected stale certificate to be pruned, err=%v", err)
	}
}

func TestBuildRestSnapshotSkipsDisabled(t *testing.T) {
	snapshot := buildRestSnapshot([]model.Route{
		{Name: "a", Protocol: "http", Domain: "a.test", PathPrefix: "/", TargetUrl: "http://a:1", Enabled: true},
		{Name: "b", Protocol: "http", Domain: "b.test", PathPrefix: "/", TargetUrl: "http://b:1", Enabled: false},
	})
	httpCfg := snapshot["http"].(map[string]any)
	if len(httpCfg["routers"].(map[string]any)) != 1 {
		t.Fatalf("expected one router: %#v", snapshot)
	}
}

func TestBuildRestSnapshotUsesDNSResolverForDNSChallenge(t *testing.T) {
	snapshot := buildRestSnapshot([]model.Route{{
		Name: "dns", Protocol: "http", Domain: "dns.example.test", PathPrefix: "/", TargetUrl: "http://app:8080", Enabled: true,
		HTTPSEnabled: true, CertType: "letsencrypt", AcmeChallenge: "dns",
	}})
	router := snapshot["http"].(map[string]any)["routers"].(map[string]any)["dns-route"].(map[string]any)
	if got := router["tls"].(map[string]any)["certResolver"]; got != "letsencrypt-dns" {
		t.Fatalf("DNS cert resolver = %#v", got)
	}
}

func TestBuildRestSnapshotUsesResolvedManagedHTTPTarget(t *testing.T) {
	snapshot := buildRestSnapshot([]model.Route{{
		Name: "api", Protocol: "http", Domain: "api.test", PathPrefix: "/", TargetAddress: "api-api", TargetPort: 8080, Enabled: true,
	}})
	httpCfg := snapshot["http"].(map[string]any)
	service := httpCfg["services"].(map[string]any)["api-service"].(map[string]any)
	servers := service["loadBalancer"].(map[string]any)["servers"].([]map[string]string)
	if len(servers) != 1 || servers[0]["url"] != "http://api-api:8080" {
		t.Fatalf("unexpected managed HTTP servers: %+v", servers)
	}
}

func TestRouteManagerUsesDeploymentCertificateDirectory(t *testing.T) {
	root := t.TempDir()
	deploymentRoot := filepath.Join(root, "cd")
	manager := NewRouteManager(config.Config{Workspace: config.WorkspaceConfig{Deployment: deploymentRoot}})
	want := filepath.Join(deploymentRoot, "traefik-default", "gateway", "certs")
	if got := manager.routeCertDir(model.GatewayConfig{RuntimeServiceCode: "traefik-default"}); got != want {
		t.Fatalf("certificate directory = %q, want %q", got, want)
	}
}

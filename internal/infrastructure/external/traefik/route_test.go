package traefik

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRouteManagerListRoutersMapsTraefikResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if request.Method != http.MethodGet || request.URL.Path != "/api/http/routers" {
			t.Fatalf("unexpected request: %s %s", request.Method, request.URL.Path)
		}
		writer.Header().Set("Content-Type", "application/json")
		_, _ = writer.Write([]byte(`[{"name":"api@docker","provider":"docker","status":"enabled","rule":"Host(api.example.test)","service":"api-service","entryPoints":["websecure"],"tls":{}},{"name":"dashboard@file","provider":"file","status":"disabled","rule":"Host(dashboard.example.test)","service":"dashboard-service","entryPoints":["web"],"tls":null}]`))
	}))
	defer server.Close()

	manager := NewRouteManager(config.Config{Traefik: config.TraefikConfig{APIURL: server.URL}})
	routers, err := manager.ListRouters(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(routers) != 2 {
		t.Fatalf("expected two routers, got %+v", routers)
	}
	if routers[0].Name != "api@docker" || routers[0].Provider != "docker" || routers[0].Status != "enabled" || routers[0].Rule != "Host(api.example.test)" || routers[0].Service != "api-service" || len(routers[0].Entrypoints) != 1 || routers[0].Entrypoints[0] != "websecure" || !routers[0].TLS {
		t.Fatalf("unexpected first router: %+v", routers[0])
	}
	if routers[1].Name != "dashboard@file" || routers[1].TLS || len(routers[1].Entrypoints) != 1 || routers[1].Entrypoints[0] != "web" {
		t.Fatalf("unexpected second router: %+v", routers[1])
	}
}

func TestRouteManagerListRoutersReturnsErrorForInvalidResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, _ *http.Request) {
		writer.WriteHeader(http.StatusBadGateway)
	}))
	defer server.Close()

	manager := NewRouteManager(config.Config{Traefik: config.TraefikConfig{APIURL: server.URL}})
	if _, err := manager.ListRouters(context.Background()); err == nil {
		t.Fatal("expected error for non-success Traefik response")
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

	manager := NewRouteManager(config.Config{Traefik: config.TraefikConfig{APIURL: server.URL}})
	err := manager.ApplySnapshot(context.Background(), []model.Route{
		{Name: "api", Domain: "api.example.test", PathPrefix: "/v1", TargetURL: "http://app:8080", Enabled: true},
		{Name: "off", Domain: "off.example.test", PathPrefix: "/", TargetURL: "http://app:8081", Enabled: false},
		{Name: "secure", Domain: "secure.example.test", PathPrefix: "/", TargetURL: "http://app:8082", Enabled: true, HTTPSEnabled: true, CertType: "letsencrypt"},
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
}

func TestRouteManagerApplySnapshotClearsWithEmptyMaps(t *testing.T) {
	var gotBody string
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		raw, _ := io.ReadAll(request.Body)
		gotBody = string(raw)
		writer.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	manager := NewRouteManager(config.Config{Traefik: config.TraefikConfig{APIURL: server.URL}})
	if err := manager.ApplySnapshot(context.Background(), nil); err != nil {
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
}

func TestBuildRestSnapshotSkipsDisabled(t *testing.T) {
	snapshot := buildRestSnapshot([]model.Route{
		{Name: "a", Domain: "a.test", PathPrefix: "/", TargetURL: "http://a:1", Enabled: true},
		{Name: "b", Domain: "b.test", PathPrefix: "/", TargetURL: "http://b:1", Enabled: false},
	})
	httpCfg := snapshot["http"].(map[string]any)
	if len(httpCfg["routers"].(map[string]any)) != 1 {
		t.Fatalf("expected one router: %#v", snapshot)
	}
}

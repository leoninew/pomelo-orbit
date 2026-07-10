package traefik

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
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

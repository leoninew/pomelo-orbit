package transporthttp

import (
	"encoding/json"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/response"
)

func TestTraefikRouteEndpointsRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/cd/traefik-route/config?project_id=" + testRouteProjectId},
		{http.MethodGet, "/api/cd/traefik-route?project_id=" + testRouteProjectId},
	}
	for _, route := range routes {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(route.method, route.path, nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected %s %s status 401, got %d: %s", route.method, route.path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestTraefikRouteEndpointsRequireProjectId(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	paths := []string{
		"/api/cd/traefik-route/config",
		"/api/cd/traefik-route",
	}
	for _, path := range paths {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, path, nil, token))
		if recorder.Code != http.StatusBadRequest {
			t.Fatalf("expected %s status 400, got %d: %s", path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestTraefikRouteConfigWithoutDashboardRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.Traefik.DomainSuffix = "lvh.me"
	token := testToken(t, server)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/cd/traefik-route/config?project_id="+testRouteProjectId, nil, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp pomeloorbit.TraefikConfigResp
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.DashboardDomain != "traefik.lvh.me" || resp.HttpsEnabled {
		t.Fatalf("unexpected traefik config response: %+v", &resp)
	}
}

func TestTraefikRouteConfigWithHTTPSDashboardRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.Traefik.DomainSuffix = "lvh.me"
	projectId := testRouteProjectId
	if err := server.cdRepository.CreateRoute(t.Context(), model.Route{
		Id:           "01KTRAETFIKROUTE0000000001",
		ProjectId:    &projectId,
		Name:         "traefik-dashboard",
		Domain:       "traefik.lvh.me",
		PathPrefix:   "/",
		TargetURL:    "http://traefik:8080",
		Enabled:      true,
		HTTPSEnabled: true,
		CertType:     "manual",
	}); err != nil {
		t.Fatal(err)
	}
	token := testToken(t, server)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/cd/traefik-route/config?project_id="+testRouteProjectId, nil, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp pomeloorbit.TraefikConfigResp
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.DashboardDomain != "traefik.lvh.me" || !resp.HttpsEnabled {
		t.Fatalf("unexpected traefik config response: %+v", &resp)
	}
}

func TestTraefikRouteListReturnsRouters(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet {
			t.Fatalf("unexpected method: %s", r.Method)
		}
		if r.URL.Path != "/api/http/routers" {
			t.Fatalf("unexpected path: %s", r.URL.Path)
		}
		transportresponse.JSON(slog.Default(), w, http.StatusOK, []map[string]any{
			{
				"name":        "api@docker",
				"provider":    "docker",
				"status":      "enabled",
				"rule":        "Host(`api.lvh.me`)",
				"service":     "api-service",
				"entryPoints": []string{"websecure"},
				"tls":         map[string]any{},
			},
			{
				"name":        "web@file",
				"provider":    "file",
				"status":      "enabled",
				"rule":        "Host(`web.lvh.me`)",
				"service":     "web-service",
				"entryPoints": []string{"web"},
			},
		})
	}))
	defer upstream.Close()

	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.Traefik.APIURL = upstream.URL
	token := testToken(t, server)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/cd/traefik-route?project_id="+testRouteProjectId, nil, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var resp pomeloorbit.TraefikRouteListResp
	if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
		t.Fatal(err)
	}
	if resp.Total != 2 || len(resp.Items) != 2 {
		t.Fatalf("unexpected traefik route list response: %+v", &resp)
	}
	if resp.Items[0].Name != "api@docker" || resp.Items[0].Entrypoints[0] != "websecure" || !resp.Items[0].Tls {
		t.Fatalf("unexpected first traefik route: %+v", resp.Items[0])
	}
	if resp.Items[1].Tls {
		t.Fatalf("unexpected tls for second traefik route: %+v", resp.Items[1])
	}
}

func TestTraefikRouteListReturnsUnavailableForConnectionFailure(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	apiURL := upstream.URL
	upstream.Close()

	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.Traefik.APIURL = apiURL
	token := testToken(t, server)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/cd/traefik-route?project_id="+testRouteProjectId, nil, token))
	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "无法连接到 Traefik:") {
		t.Fatalf("unexpected error response: %s", recorder.Body.String())
	}
}

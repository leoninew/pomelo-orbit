package transporthttp

import (
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/routes"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

func newServerForServerTest(cfg config.Config) Server {
	if cfg.Server.Host == "" {
		cfg.Server.Host = "127.0.0.1"
	}
	if cfg.Server.ApiPathPrefixes == nil {
		cfg.Server.ApiPathPrefixes = []string{"/api"}
	}
	return New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), routes.Dependencies{})
}

func TestHealth(t *testing.T) {
	server := newServerForServerTest(config.Config{})

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestAddr(t *testing.T) {
	server := newServerForServerTest(config.Config{Server: config.ServerConfig{Host: "127.0.0.1", Port: 8080, ApiPathPrefixes: []string{"/api"}}})

	if server.Addr() != "127.0.0.1:8080" {
		t.Fatalf("unexpected addr: %s", server.Addr())
	}
}

func TestStaticFilesFallbackUsesConfiguredApiPathPrefixes(t *testing.T) {
	server := newServerForServerTest(config.Config{Server: config.ServerConfig{ApiPathPrefixes: []string{"/api", "/graphql"}}})
	withStaticDir(t, "<html><head><!-- __RUNTIME_CONFIG__ --></head><body>app</body></html>", nil)

	apiRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(apiRecorder, httptest.NewRequest(http.MethodGet, "/graphql", nil))
	if apiRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected configured api prefix 404, got %d: %s", apiRecorder.Code, apiRecorder.Body.String())
	}

	nonAPIRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(nonAPIRecorder, httptest.NewRequest(http.MethodGet, "/graphqlx", nil))
	if nonAPIRecorder.Code != http.StatusOK || !strings.Contains(nonAPIRecorder.Body.String(), "<script>window.__CONFIG__ = {};</script>") {
		t.Fatalf("expected non-api spa fallback, got %d: %s", nonAPIRecorder.Code, nonAPIRecorder.Body.String())
	}
}

func TestStaticFilesFallbackServesFrontend(t *testing.T) {
	server := newServerForServerTest(config.Config{})
	withStaticDir(t, "<html><head><!-- __RUNTIME_CONFIG__ --></head><body>app</body></html>", map[string]string{
		"asset.js": "console.log('app')",
	})

	assetRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(assetRecorder, httptest.NewRequest(http.MethodGet, "/asset.js", nil))
	if assetRecorder.Code != http.StatusOK || assetRecorder.Body.String() != "console.log('app')" {
		t.Fatalf("expected static asset, got %d: %s", assetRecorder.Code, assetRecorder.Body.String())
	}

	spaRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(spaRecorder, httptest.NewRequest(http.MethodGet, "/repository", nil))
	if spaRecorder.Code != http.StatusOK || !strings.Contains(spaRecorder.Body.String(), "<script>window.__CONFIG__ = {};</script>") {
		t.Fatalf("expected spa fallback with runtime config, got %d: %s", spaRecorder.Code, spaRecorder.Body.String())
	}

	apiRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(apiRecorder, httptest.NewRequest(http.MethodGet, "/api/missing", nil))
	if apiRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected api 404, got %d: %s", apiRecorder.Code, apiRecorder.Body.String())
	}

	apiRootRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(apiRootRecorder, httptest.NewRequest(http.MethodGet, "/api", nil))
	if apiRootRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected api root 404, got %d: %s", apiRootRecorder.Code, apiRootRecorder.Body.String())
	}

	nonAPIRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(nonAPIRecorder, httptest.NewRequest(http.MethodGet, "/apix/missing", nil))
	if nonAPIRecorder.Code != http.StatusOK || !strings.Contains(nonAPIRecorder.Body.String(), "<script>window.__CONFIG__ = {};</script>") {
		t.Fatalf("expected non-api spa fallback, got %d: %s", nonAPIRecorder.Code, nonAPIRecorder.Body.String())
	}

	postRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(postRecorder, httptest.NewRequest(http.MethodPost, "/repository", nil))
	if postRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected non-get static fallback status 404, got %d: %s", postRecorder.Code, postRecorder.Body.String())
	}
}

func TestApiFallbackWritesNotFoundContract(t *testing.T) {
	server := newServerForServerTest(config.Config{})
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/missing", nil))

	assertServerErrorResponse(t, recorder, http.StatusNotFound, "not_found")
}

func TestApiMethodMismatchWritesContract(t *testing.T) {
	server := newServerForServerTest(config.Config{})
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/api/health", nil))

	assertServerErrorResponse(t, recorder, http.StatusMethodNotAllowed, "method_not_allowed")
	if got := recorder.Header().Get("Allow"); got != http.MethodGet {
		t.Fatalf("expected Allow header %q, got %q", http.MethodGet, got)
	}
}

func TestAllowedHTTPMethodsMatchesParameterizedRoute(t *testing.T) {
	routes := []gin.RouteInfo{
		{Method: http.MethodGet, Path: "/api/projects/:projectId"},
		{Method: http.MethodPut, Path: "/api/projects/:projectId"},
		{Method: http.MethodDelete, Path: "/api/projects/:projectId/members/:userId"},
	}

	methods := allowedHTTPMethods(routes, "/api/projects/project-1")
	if got := strings.Join(methods, ", "); got != "GET, PUT" {
		t.Fatalf("expected parameterized route methods, got %q", got)
	}
}

func TestStaticFilesInjectRuntimeConfigPublicURL(t *testing.T) {
	server := newServerForServerTest(config.Config{Server: config.ServerConfig{PublicURL: "https://orbit-api.preflite.cn"}})
	withStaticDir(t, "<html><head><!-- __RUNTIME_CONFIG__ --></head><body>app</body></html>", nil)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	body := recorder.Body.String()
	if !strings.Contains(body, `<script>window.__CONFIG__ = {"publicUrl":"https://orbit-api.preflite.cn"};</script>`) {
		t.Fatalf("expected runtime config public url, got: %s", body)
	}
	if recorder.Header().Get("Content-Type") != "text/html; charset=utf-8" {
		t.Fatalf("unexpected content type: %s", recorder.Header().Get("Content-Type"))
	}
}

func TestStaticFilesInjectRuntimeConfigEmptyObject(t *testing.T) {
	server := newServerForServerTest(config.Config{})
	withStaticDir(t, "<html><head><!-- __RUNTIME_CONFIG__ --></head><body>app</body></html>", nil)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), "<script>window.__CONFIG__ = {};</script>") {
		t.Fatalf("expected empty runtime config, got: %s", recorder.Body.String())
	}
}

func TestStaticFilesInjectRuntimeConfigFallbackBeforeHeadEnd(t *testing.T) {
	server := newServerForServerTest(config.Config{Server: config.ServerConfig{PublicURL: "https://orbit-api.preflite.cn"}})
	withStaticDir(t, "<html><head><title>Orbit</title></head><body>app</body></html>", nil)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
	want := `<script>window.__CONFIG__ = {"publicUrl":"https://orbit-api.preflite.cn"};</script></head>`
	if !strings.Contains(recorder.Body.String(), want) {
		t.Fatalf("expected runtime config before head end, got: %s", recorder.Body.String())
	}
}

func withStaticDir(t *testing.T, indexHTML string, files map[string]string) {
	t.Helper()
	workingDir := t.TempDir()
	staticDir := filepath.Join(workingDir, "static")
	if err := os.Mkdir(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte(indexHTML), 0o644); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(staticDir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatal(err)
		}
	})
}

func assertServerErrorResponse(t *testing.T, recorder *httptest.ResponseRecorder, statusCode int, code string) {
	t.Helper()
	if recorder.Code != statusCode {
		t.Fatalf("expected status %d, got %d: %s", statusCode, recorder.Code, recorder.Body.String())
	}
	var response transportresponse.ErrorResp
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if response.Code != code || response.RequestId == "" {
		t.Fatalf("unexpected error response: %+v", response)
	}
	if got := recorder.Header().Get("X-Request-ID"); got != response.RequestId {
		t.Fatalf("expected matching request id header and body, got header=%q body=%q", got, response.RequestId)
	}
}

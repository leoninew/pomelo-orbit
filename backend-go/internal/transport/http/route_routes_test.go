package transporthttp

import (
	"bytes"
	"encoding/json"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	cdhandler "backend/internal/transport/http/handler/cd"
)

const testRouteProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"

func TestRouteCRUDAndStatus(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.Orbit.Root = t.TempDir()
	token := testToken(t, server)

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/cd/route?project_id="+testRouteProjectId, bytes.NewBufferString(`{"name":"api-route","domain":"api.example.test","path_prefix":"/api","target_url":"http://host.docker.internal:8081","enabled":false}`), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected route create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created cdhandler.RouteResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Name != "api-route" || created.PathPrefix != "/api" || created.CertType != "manual" || created.HTTPSEnabled {
		t.Fatalf("unexpected created route: %+v", created)
	}

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/cd/route?project_id="+testRouteProjectId+"&search=api", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected route list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var list paginatedResp[cdhandler.RouteResp]
	if err := json.NewDecoder(listRecorder.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Total == 0 || len(list.Items) == 0 {
		t.Fatalf("unexpected route list: %+v", list)
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/cd/route/"+created.Id, bytes.NewBufferString(`{"name":"api-route-2","domain":"api2.example.test","path_prefix":"/v2","target_url":"http://host.docker.internal:8082","enabled":true}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected route update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated cdhandler.RouteResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "api-route-2" || updated.TargetURL != "http://host.docker.internal:8082" || !updated.Enabled {
		t.Fatalf("unexpected updated route: %+v", updated)
	}
	configPath := filepath.Join(routeConfigDir(server), "api-route-2.yml")
	configData, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("expected route config file: %v", err)
	}
	if !strings.Contains(string(configData), "Host(`api2.example.test`) && PathPrefix(`/v2`)") {
		t.Fatalf("unexpected route config: %s", string(configData))
	}

	deleteEnabledRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteEnabledRecorder, authedRequest(http.MethodDelete, "/api/cd/route/"+created.Id, nil, token))
	if deleteEnabledRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected enabled route delete status 400, got %d: %s", deleteEnabledRecorder.Code, deleteEnabledRecorder.Body.String())
	}

	disableRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(disableRecorder, authedRequest(http.MethodPost, "/api/cd/route/"+created.Id+"/disable", nil, token))
	if disableRecorder.Code != http.StatusOK {
		t.Fatalf("expected disable route status 200, got %d: %s", disableRecorder.Code, disableRecorder.Body.String())
	}
	if _, err := os.Stat(configPath); !os.IsNotExist(err) {
		t.Fatalf("expected route config to be removed, err=%v", err)
	}

	enableRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(enableRecorder, authedRequest(http.MethodPost, "/api/cd/route/"+created.Id+"/enable", nil, token))
	if enableRecorder.Code != http.StatusOK {
		t.Fatalf("expected enable route status 200, got %d: %s", enableRecorder.Code, enableRecorder.Body.String())
	}

	disableAgainRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(disableAgainRecorder, authedRequest(http.MethodPost, "/api/cd/route/"+created.Id+"/disable", nil, token))
	if disableAgainRecorder.Code != http.StatusOK {
		t.Fatalf("expected disable route status 200, got %d: %s", disableAgainRecorder.Code, disableAgainRecorder.Body.String())
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/cd/route/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected route delete status 204, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestRouteCertificateOperations(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.Orbit.Root = t.TempDir()
	server.appCfg.Cert.LetsEncrypt.Enabled = true
	server.appCfg.Cert.LetsEncrypt.Email = "admin@example.test"
	token := testToken(t, server)
	route := createRouteForTest(t, server, token, `{"name":"cert-route","domain":"cert.example.test","path_prefix":"/","target_url":"http://host.docker.internal:8081","enabled":true}`)

	certRecorder := httptest.NewRecorder()
	certRequest := multipartRouteCertRequest(t, "/api/cd/route/"+route.Id+"/cert", token, testCombinedPEM())
	server.Handler().ServeHTTP(certRecorder, certRequest)
	if certRecorder.Code != http.StatusOK {
		t.Fatalf("expected cert upload status 200, got %d: %s", certRecorder.Code, certRecorder.Body.String())
	}
	var withCert cdhandler.RouteResp
	if err := json.NewDecoder(certRecorder.Body).Decode(&withCert); err != nil {
		t.Fatal(err)
	}
	if !withCert.HTTPSEnabled || withCert.CertType != "manual" {
		t.Fatalf("unexpected cert route: %+v", withCert)
	}
	if _, err := os.Stat(filepath.Join(routeCertDir(server), "cert-route.pem")); err != nil {
		t.Fatalf("expected cert file: %v", err)
	}

	disableHTTPSRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(disableHTTPSRecorder, authedRequest(http.MethodDelete, "/api/cd/route/"+route.Id+"/https", nil, token))
	if disableHTTPSRecorder.Code != http.StatusOK {
		t.Fatalf("expected disable https status 200, got %d: %s", disableHTTPSRecorder.Code, disableHTTPSRecorder.Body.String())
	}
	if _, err := os.Stat(filepath.Join(routeCertDir(server), "cert-route.pem")); !os.IsNotExist(err) {
		t.Fatalf("expected cert file to be removed, err=%v", err)
	}

	leRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(leRecorder, authedRequest(http.MethodPost, "/api/cd/route/"+route.Id+"/letsencrypt", nil, token))
	if leRecorder.Code != http.StatusOK {
		t.Fatalf("expected letsencrypt status 200, got %d: %s", leRecorder.Code, leRecorder.Body.String())
	}
	var letsEncrypt cdhandler.RouteResp
	if err := json.NewDecoder(leRecorder.Body).Decode(&letsEncrypt); err != nil {
		t.Fatal(err)
	}
	if !letsEncrypt.HTTPSEnabled || letsEncrypt.CertType != "letsencrypt" {
		t.Fatalf("unexpected letsencrypt route: %+v", letsEncrypt)
	}

	syncRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(syncRecorder, authedRequest(http.MethodPost, "/api/cd/route/sync?project_id="+testRouteProjectId, nil, token))
	if syncRecorder.Code != http.StatusOK {
		t.Fatalf("expected route sync status 200, got %d: %s", syncRecorder.Code, syncRecorder.Body.String())
	}
}

func TestRouteEndpointsRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/cd/route?project_id=" + testRouteProjectId},
		{http.MethodPost, "/api/cd/route?project_id=" + testRouteProjectId},
		{http.MethodPost, "/api/cd/route/sync?project_id=" + testRouteProjectId},
		{http.MethodGet, "/api/cd/route/route-1"},
		{http.MethodPut, "/api/cd/route/route-1"},
		{http.MethodDelete, "/api/cd/route/route-1"},
		{http.MethodPost, "/api/cd/route/route-1/enable"},
		{http.MethodPost, "/api/cd/route/route-1/disable"},
		{http.MethodPost, "/api/cd/route/route-1/cert"},
		{http.MethodDelete, "/api/cd/route/route-1/https"},
		{http.MethodPost, "/api/cd/route/route-1/letsencrypt"},
		{http.MethodPost, "/api/cd/route/route-1/mkcert"},
	}
	for _, route := range routes {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(route.method, route.path, nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected %s %s status 401, got %d: %s", route.method, route.path, recorder.Code, recorder.Body.String())
		}
	}
}

func createRouteForTest(t *testing.T, server Server, token string, body string) cdhandler.RouteResp {
	t.Helper()
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/cd/route?project_id="+testRouteProjectId, bytes.NewBufferString(body), token))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected route create status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var route cdhandler.RouteResp
	if err := json.NewDecoder(recorder.Body).Decode(&route); err != nil {
		t.Fatal(err)
	}
	return route
}

func multipartRouteCertRequest(t *testing.T, path string, token string, content string) *http.Request {
	t.Helper()
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)
	part, err := writer.CreateFormFile("pem", "cert.pem")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := part.Write([]byte(content)); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, path, &body)
	request.Header.Set("Authorization", "Bearer "+token)
	request.Header.Set("Content-Type", writer.FormDataContentType())
	return request
}

func testCombinedPEM() string {
	return strings.Join([]string{
		"-----BEGIN CERTIFICATE-----",
		"MIIB",
		"-----END CERTIFICATE-----",
		"-----BEGIN PRIVATE KEY-----",
		"MIIC",
		"-----END PRIVATE KEY-----",
		"",
	}, "\n")
}

func routeConfigDir(server Server) string {
	if strings.TrimSpace(server.appCfg.Traefik.DynamicRouteDir) == "" {
		return filepath.Join(server.appCfg.DataRoot(), "cd", "traefik", "data", "dynamic")
	}
	return cleanTestConfigPath(server.appCfg.OrbitRoot(), server.appCfg.Traefik.DynamicRouteDir)
}

func routeCertDir(server Server) string {
	if strings.TrimSpace(server.appCfg.Traefik.CertDir) == "" {
		return filepath.Join(server.appCfg.DataRoot(), "cd", "traefik", "data", "certs")
	}
	return cleanTestConfigPath(server.appCfg.OrbitRoot(), server.appCfg.Traefik.CertDir)
}

func cleanTestConfigPath(root string, path string) string {
	path = filepath.Clean(strings.TrimSpace(path))
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(root, path)
}

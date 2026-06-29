package transporthttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestCIListRoutesKeepPaginatedResponseShape(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.JWT.SecretKey = credentialRouteFernetKey
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	routes := []string{
		"/api/ci/repository?project_id=" + projectId,
		"/api/ci/template?project_id=" + projectId,
		"/api/ci/build-stage?project_id=" + projectId,
		"/api/ci/run?project_id=" + projectId,
		"/api/ci/artifact?project_id=" + projectId,
		"/api/ci/credential?project_id=" + projectId,
	}
	for _, route := range routes {
		t.Run(route, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, route, nil, token))
			if recorder.Code != http.StatusOK {
				t.Fatalf("expected %s status 200, got %d: %s", route, recorder.Code, recorder.Body.String())
			}
			var body map[string]any
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			assertPaginatedShape(t, body)
		})
	}
}

func TestCIRepositoryResponseKeepsFieldNames(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/ci/repository?project_id="+projectId, bytes.NewBufferString(`{"name":"Repo Compatibility","code":"repo-compatibility","repository_url":"https://example.test/repo.git","default_branch":"main","variable_overrides":[{"name":"FOO","value":"bar"}]}`), token))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected repository create status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var body map[string]any
	if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
		t.Fatal(err)
	}
	assertJSONKeys(t, body, []string{"id", "project_id", "name", "code", "repository_url", "has_credential", "git_credential_id", "variable_declarations", "default_branch", "created_at", "updated_at"})
	if _, ok := body["variable_overrides"]; ok {
		t.Fatalf("repository response must expose variable_declarations, not variable_overrides: %+v", body)
	}
}

func TestCIErrorResponsesKeepDetailShape(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	tests := []struct {
		name   string
		method string
		path   string
		body   *bytes.Buffer
		status int
		detail string
	}{
		{name: "invalid json", method: http.MethodPost, path: "/api/ci/repository?project_id=" + projectId, body: bytes.NewBufferString(`{"name"`), status: http.StatusBadRequest, detail: "Invalid JSON body"},
		{name: "missing project", method: http.MethodGet, path: "/api/ci/template", status: http.StatusBadRequest, detail: "project_id is required"},
		{name: "not found", method: http.MethodGet, path: "/api/ci/repository/missing-repository", status: http.StatusNotFound, detail: "Repository missing-repository not found"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			server.Handler().ServeHTTP(recorder, authedRequest(tt.method, tt.path, tt.body, token))
			if recorder.Code != tt.status {
				t.Fatalf("expected status %d, got %d: %s", tt.status, recorder.Code, recorder.Body.String())
			}
			var body map[string]string
			if err := json.NewDecoder(recorder.Body).Decode(&body); err != nil {
				t.Fatal(err)
			}
			if body["detail"] != tt.detail {
				t.Fatalf("expected detail %q, got %+v", tt.detail, body)
			}
		})
	}
}

func assertPaginatedShape(t *testing.T, body map[string]any) {
	t.Helper()
	assertJSONKeys(t, body, []string{"items", "total", "page", "per_page"})
	if _, ok := body["items"].([]any); !ok {
		t.Fatalf("expected items to be an array, got %+v", body)
	}
	for _, key := range []string{"total", "page", "per_page"} {
		if _, ok := body[key].(float64); !ok {
			t.Fatalf("expected %s to be numeric, got %+v", key, body)
		}
	}
}

func assertJSONKeys(t *testing.T, body map[string]any, keys []string) {
	t.Helper()
	for _, key := range keys {
		if _, ok := body[key]; !ok {
			t.Fatalf("expected JSON key %q in %+v", key, body)
		}
	}
}

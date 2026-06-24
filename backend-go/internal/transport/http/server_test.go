package transporthttp

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/repository"
	taskrepo "backend/internal/repository/task"
	"backend/internal/status"
	authhandler "backend/internal/transport/http/handler/auth"
	projecthandler "backend/internal/transport/http/handler/project"
)

type fakeTurnstileVerifier struct {
	err error
}

func (v fakeTurnstileVerifier) Verify(ctx context.Context, token string, remoteIP string) error {
	return v.err
}

func newTestServer(t *testing.T) (Server, *sqlx.DB) {
	t.Helper()
	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.NewMigrator(database, config.DatabaseDriverSQLite).Up(); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Server: config.ServerConfig{Host: "127.0.0.1", Port: 0}}
	cfg.JWT.SecretKey = credentialRouteFernetKey
	cfg.Turnstile = config.TurnstileConfig{Enabled: true, SiteKey: "test-site-key", SecretKey: "test-secret-key", VerifyURL: "https://turnstile.example.test"}
	server := New(cfg, slog.Default(), repository.NewStore(database, config.DatabaseDriverSQLite), taskrepo.NewRepository(database, config.DatabaseDriverSQLite), 3)
	server.turnstileVerifier = fakeTurnstileVerifier{}
	return server, database
}

func TestHealth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodGet, "/api/health", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", recorder.Code)
	}
}

func TestCreateAndGetTask(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	body := bytes.NewBufferString(`{"id":"task-1","task_type":"ci.pipeline_run.execute","payload":{"pipeline_run_id":"run-1"},"max_attempts":2}`)
	createRecorder := httptest.NewRecorder()
	createRequest := httptest.NewRequest(http.MethodPost, "/api/background/task", body)
	server.Handler().ServeHTTP(createRecorder, createRequest)

	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created taskrepo.Task
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id != "task-1" {
		t.Fatalf("unexpected task id: %s", created.Id)
	}
	if created.Status != status.TaskPending {
		t.Fatalf("unexpected task status: %s", created.Status)
	}
	if created.MaxAttempts != 2 {
		t.Fatalf("unexpected max attempts: %d", created.MaxAttempts)
	}

	getRecorder := httptest.NewRecorder()
	getRequest := httptest.NewRequest(http.MethodGet, "/api/background/task/task-1", nil)
	server.Handler().ServeHTTP(getRecorder, getRequest)
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected status 200, got %d", getRecorder.Code)
	}
}

func TestEnqueueCIPipelineRun(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/background/ci/pipeline-run/run-1/execute", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var created taskrepo.Task
	if err := json.NewDecoder(recorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.TaskType != status.TaskTypeCIPipelineRunExecute {
		t.Fatalf("unexpected task type: %s", created.TaskType)
	}
	if created.PayloadJSON != `{"pipeline_run_id":"run-1"}` {
		t.Fatalf("unexpected payload: %s", created.PayloadJSON)
	}
}

func TestEnqueueCDApplicationDeploy(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/background/cd/application/app-1/deploy/deploy-1", nil)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var created taskrepo.Task
	if err := json.NewDecoder(recorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.TaskType != status.TaskTypeCDApplicationDeploy {
		t.Fatalf("unexpected task type: %s", created.TaskType)
	}
	if created.PayloadJSON != `{"application_id":"app-1","deployment_id":"deploy-1"}` {
		t.Fatalf("unexpected payload: %s", created.PayloadJSON)
	}
}

func TestCreateTaskValidatesPayload(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	body := bytes.NewBufferString(`{"task_type":"ci.pipeline_run.execute","payload_json":"not-json"}`)
	recorder := httptest.NewRecorder()
	request := httptest.NewRequest(http.MethodPost, "/api/background/task", body)
	server.Handler().ServeHTTP(recorder, request)

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status 400, got %d", recorder.Code)
	}
}

func TestAuthLoginAndMe(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	csrfRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(csrfRecorder, httptest.NewRequest(http.MethodGet, "/api/auth/csrf-token", nil))
	if csrfRecorder.Code != http.StatusOK {
		t.Fatalf("expected csrf status 200, got %d", csrfRecorder.Code)
	}

	turnstileConfigRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(turnstileConfigRecorder, httptest.NewRequest(http.MethodGet, "/api/auth/turnstile-config", nil))
	if turnstileConfigRecorder.Code != http.StatusOK {
		t.Fatalf("expected turnstile config status 200, got %d", turnstileConfigRecorder.Code)
	}

	loginBody := bytes.NewBufferString(`{"username":"admin","password":"admin","csrf_token":"csrf","turnstile_token":"turnstile"}`)
	loginRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody))
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d: %s", loginRecorder.Code, loginRecorder.Body.String())
	}
	var token authhandler.TokenResp
	if err := json.NewDecoder(loginRecorder.Body).Decode(&token); err != nil {
		t.Fatal(err)
	}
	if token.AccessToken == "" || token.TokenType != "bearer" {
		t.Fatalf("unexpected token response: %+v", token)
	}

	meRecorder := httptest.NewRecorder()
	meRequest := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	meRequest.Header.Set("Authorization", "Bearer "+token.AccessToken)
	server.Handler().ServeHTTP(meRecorder, meRequest)
	if meRecorder.Code != http.StatusOK {
		t.Fatalf("expected me status 200, got %d: %s", meRecorder.Code, meRecorder.Body.String())
	}
	var me authhandler.UserInfoResp
	if err := json.NewDecoder(meRecorder.Body).Decode(&me); err != nil {
		t.Fatal(err)
	}
	if me.Username != "admin" || len(me.Roles) == 0 || len(me.Permissions) == 0 {
		t.Fatalf("unexpected me response: %+v", me)
	}
}

func TestAuthLoginRequiresTurnstileToken(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	loginBody := bytes.NewBufferString(`{"username":"admin","password":"admin","csrf_token":"csrf"}`)
	loginRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody))
	if loginRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected login status 400, got %d: %s", loginRecorder.Code, loginRecorder.Body.String())
	}
}

func TestAuthLoginRejectsInvalidTurnstileToken(t *testing.T) {
	server, database := newTestServer(t)
	server.turnstileVerifier = fakeTurnstileVerifier{err: errors.New("invalid token")}
	defer func() { _ = database.Close() }()

	loginBody := bytes.NewBufferString(`{"username":"admin","password":"admin","csrf_token":"csrf","turnstile_token":"bad-token"}`)
	loginRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody))
	if loginRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected login status 400, got %d: %s", loginRecorder.Code, loginRecorder.Body.String())
	}
}

func TestAuthLoginSkipsTurnstileWhenDisabled(t *testing.T) {
	server, database := newTestServer(t)
	server.appCfg.Turnstile.Enabled = false
	server.turnstileVerifier = fakeTurnstileVerifier{err: errors.New("should not be called")}
	defer func() { _ = database.Close() }()

	loginBody := bytes.NewBufferString(`{"username":"admin","password":"admin","csrf_token":"csrf"}`)
	loginRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody))
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d: %s", loginRecorder.Code, loginRecorder.Body.String())
	}
}

func TestAuthCaptchaRouteIsRemoved(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/auth/captcha", nil))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected captcha status 404, got %d", recorder.Code)
	}
}

func TestStaticFilesFallbackServesFrontend(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	workingDir := t.TempDir()
	staticDir := filepath.Join(workingDir, "static")
	if err := os.Mkdir(staticDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "index.html"), []byte("<html>app</html>"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(staticDir, "asset.js"), []byte("console.log('app')"), 0o644); err != nil {
		t.Fatal(err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(workingDir); err != nil {
		t.Fatal(err)
	}
	defer func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Fatal(err)
		}
	}()

	assetRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(assetRecorder, httptest.NewRequest(http.MethodGet, "/asset.js", nil))
	if assetRecorder.Code != http.StatusOK || assetRecorder.Body.String() != "console.log('app')" {
		t.Fatalf("expected static asset, got %d: %s", assetRecorder.Code, assetRecorder.Body.String())
	}

	spaRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(spaRecorder, httptest.NewRequest(http.MethodGet, "/ci/repository", nil))
	if spaRecorder.Code != http.StatusOK || spaRecorder.Body.String() != "<html>app</html>" {
		t.Fatalf("expected spa fallback, got %d: %s", spaRecorder.Code, spaRecorder.Body.String())
	}

	apiRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(apiRecorder, httptest.NewRequest(http.MethodGet, "/api/missing", nil))
	if apiRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected api 404, got %d: %s", apiRecorder.Code, apiRecorder.Body.String())
	}

	postRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(postRecorder, httptest.NewRequest(http.MethodPost, "/ci/repository", nil))
	if postRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected non-get static fallback status 404, got %d: %s", postRecorder.Code, postRecorder.Body.String())
	}
}

func TestProjectAndDashboardLists(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	projectRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(projectRecorder, authedRequest(http.MethodGet, "/api/project", nil, token))
	if projectRecorder.Code != http.StatusOK {
		t.Fatalf("expected project status 200, got %d", projectRecorder.Code)
	}
	var projects []projecthandler.ProjectResp
	if err := json.NewDecoder(projectRecorder.Body).Decode(&projects); err != nil {
		t.Fatal(err)
	}
	if len(projects) == 0 || projects[0].Code != "default" {
		t.Fatalf("unexpected projects: %+v", projects)
	}

	paths := []string{"/api/ci/repository", "/api/ci/run", "/api/cd/application", "/api/cd/deployment"}
	for _, path := range paths {
		recorder := httptest.NewRecorder()
		request := httptest.NewRequest(http.MethodGet, path, nil)
		if path == "/api/ci/repository" || path == "/api/ci/run" || path == "/api/cd/application" || path == "/api/cd/deployment" {
			request.Header.Set("Authorization", "Bearer "+token)
		}
		if path == "/api/ci/run" {
			request = httptest.NewRequest(http.MethodGet, path+"?project_id=01KRRKK0K3T519ZQZES3M4QA9Z&per_page=10", nil)
			request.Header.Set("Authorization", "Bearer "+token)
		}
		if path == "/api/cd/deployment" {
			request = httptest.NewRequest(http.MethodGet, path+"?project_id=01KRRKK0K3T519ZQZES3M4QA9Z&per_page=10", nil)
			request.Header.Set("Authorization", "Bearer "+token)
		}
		server.Handler().ServeHTTP(recorder, request)
		if recorder.Code != http.StatusOK {
			t.Fatalf("expected %s status 200, got %d: %s", path, recorder.Code, recorder.Body.String())
		}
		var resp paginatedResp[map[string]any]
		if err := json.NewDecoder(recorder.Body).Decode(&resp); err != nil {
			t.Fatal(err)
		}
		if resp.Items == nil || resp.Page != 1 || resp.PerPage != 10 {
			t.Fatalf("unexpected paginated response for %s: %+v", path, resp)
		}
	}
}

package httpserver

import (
	"bytes"
	"encoding/json"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/status"
	"backend/internal/task"
)

func newTestServer(t *testing.T) (Server, *sqlx.DB) {
	t.Helper()
	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.NewMigrator(database).Up(); err != nil {
		t.Fatal(err)
	}
	server := New(config.ServerConfig{Host: "127.0.0.1", Port: 0}, slog.Default(), task.NewRepository(database), 3)
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
	var created task.Task
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
	var created task.Task
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
	var created task.Task
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

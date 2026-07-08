package transporthttp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
)

func TestPipelineRunDetailRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.Orbit.Root = t.TempDir()
	token := testToken(t, server)
	insertPipelineRunRouteData(t, database, "run-detail-test", "faulted")
	insertStageRunRouteData(t, database, "stage-run-detail-test", "run-detail-test", "faulted")

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/ci/run/run-detail-test", nil, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected pipeline run detail status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var run pomeloorbit.PipelineRunResp
	if err := json.NewDecoder(recorder.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.Id != "run-detail-test" || run.RepositoryId != "01KNNRBH52BQJYT9487B2H8N62" || len(run.VariablesSnapshot) != 1 || len(run.StageRuns) != 1 {
		t.Fatalf("unexpected pipeline run detail: %+v", &run)
	}
}

func TestPipelineRunListFilters(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"
	insertPipelineRunRouteData(t, database, "run-filter-test", "ran_to_completion")

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/ci/run?project_id="+projectId+"&repository_id=01KNNRBH52BQJYT9487B2H8N62&template_id=01KNVEJPWVK757139NMNNNCEFE&date_from=2024-03-15T00:00:00Z&date_to=2024-03-17T00:00:00Z&per_page=20", nil, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected pipeline run list status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var runs pomeloorbit.PipelineRunPaginatedResp
	if err := json.NewDecoder(recorder.Body).Decode(&runs); err != nil {
		t.Fatal(err)
	}
	if runs.Total == 0 || runs.PerPage != 20 {
		t.Fatalf("unexpected pipeline run list: %+v", &runs)
	}
}

func TestPipelineStageLogRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.Orbit.Root = t.TempDir()
	token := testToken(t, server)
	insertPipelineRunRouteData(t, database, "run-log-test", "running")
	insertStageRunRouteData(t, database, "stage-run-log-test", "run-log-test", "running")
	logDir := filepath.Join(server.appCfg.DataRoot(), "ci", "runs", "run-log-test", "stages")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "stage-run-log-test.log"), []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/ci/run/run-log-test/stages/stage-run-log-test/log?offset=6", nil, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected stage log status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var logResp pomeloorbit.PipelineStageLogResp
	if err := json.NewDecoder(recorder.Body).Decode(&logResp); err != nil {
		t.Fatal(err)
	}
	if logResp.Logs != "world\n" || logResp.Offset != 12 || logResp.IsComplete {
		t.Fatalf("unexpected stage log response: %+v", &logResp)
	}
}

func TestPipelineRunCancelRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	insertPipelineRunRouteData(t, database, "run-cancel-test", "running")

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/ci/run/run-cancel-test/cancel", bytes.NewBufferString(`{}`), token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected pipeline run cancel status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var run pomeloorbit.PipelineRunResp
	if err := json.NewDecoder(recorder.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.Status != "canceled" || run.FinishedAt == nil {
		t.Fatalf("unexpected canceled run: %+v", &run)
	}
}

func TestPipelineRunRetryRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	insertPipelineRunRouteData(t, database, "run-retry-test", "faulted")

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/ci/run/run-retry-test/retry", bytes.NewBufferString(`{}`), token))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected pipeline run retry status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var run pomeloorbit.PipelineRunResp
	if err := json.NewDecoder(recorder.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.Id == "run-retry-test" || run.RetryOf == nil || *run.RetryOf != "run-retry-test" || run.Status != "waiting_to_run" {
		t.Fatalf("unexpected retry run: %+v", &run)
	}
}

func TestPipelineRunRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/ci/run/run-detail-test", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected pipeline run get status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func insertPipelineRunRouteData(t *testing.T, database interface {
	Exec(query string, args ...any) (sql.Result, error)
}, runId string, runStatus string) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO pipeline_snapshot (id, project_id, template_id, version, stages_snapshot, variables_snapshot) VALUES ('snapshot-run-route-test', '01KRRKK0K3T519ZQZES3M4QA9Z', '01KNVEJPWVK757139NMNNNCEFE', 8, '[]', '[]') ON CONFLICT(id) DO NOTHING`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO pipeline_run (id, project_id, repository_id, repository_name, snapshot_id, template_id, template_name, template_version, trigger, trigger_ref, variables_snapshot, status, created_at) VALUES (?, '01KRRKK0K3T519ZQZES3M4QA9Z', '01KNNRBH52BQJYT9487B2H8N62', 'golang/example', 'snapshot-run-route-test', '01KNVEJPWVK757139NMNNNCEFE', 'Go 构建流水线', 8, 'manual', 'main', '[{"name":"FOO","value":"bar","description":"","default":null,"secret":false,"source":"runtime","editable":true}]', ?, '2024-03-16T00:00:00Z')`, runId, runStatus); err != nil {
		t.Fatal(err)
	}
}

func insertStageRunRouteData(t *testing.T, database interface {
	Exec(query string, args ...any) (sql.Result, error)
}, stageRunId string, runId string, stageStatus string) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO stage_run (id, pipeline_run_id, stage_id, stage_name, status) VALUES (?, ?, 'stage-build', 'build', ?)`, stageRunId, runId, stageStatus); err != nil {
		t.Fatal(err)
	}
}

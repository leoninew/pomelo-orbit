package transporthttp

import (
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cihandler "backend/internal/transport/http/handler/ci"
)

func TestPipelineSnapshotGetRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	insertPipelineSnapshotTestData(t, database, "snapshot-route-test", `[{"name":"build","id":"stage-build","image":"golang:1.23","version":3,"depends_on":[],"script":"go test ./...","artifacts":[{"type":"file","path":"coverage.out","name":"coverage"}]}]`, `[{"name":"FOO","description":"demo var","default":"bar","value":"baz","secret":false,"source":"template_custom","editable":true}]`)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/ci/snapshot/snapshot-route-test", nil, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected pipeline snapshot get status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var snapshot cihandler.PipelineSnapshotResp
	if err := json.NewDecoder(recorder.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if snapshot.Id != "snapshot-route-test" || snapshot.TemplateId != "01KNVEJPWVK757139NMNNNCEFE" || snapshot.Version != 99 || snapshot.CreatedAt == "" {
		t.Fatalf("unexpected pipeline snapshot: %+v", snapshot)
	}
	if len(snapshot.StagesSnapshot) != 1 || snapshot.StagesSnapshot[0].Name != "build" || len(snapshot.StagesSnapshot[0].Artifacts) != 1 {
		t.Fatalf("unexpected pipeline snapshot stages: %+v", snapshot.StagesSnapshot)
	}
	if len(snapshot.VariablesSnapshot) != 1 || snapshot.VariablesSnapshot[0].Name != "FOO" || snapshot.VariablesSnapshot[0].Value != "baz" {
		t.Fatalf("unexpected pipeline snapshot variables: %+v", snapshot.VariablesSnapshot)
	}
}

func TestPipelineSnapshotGetMissingRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/ci/snapshot/missing-snapshot", nil, token))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected missing pipeline snapshot status 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPipelineSnapshotInvalidJSON(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	insertPipelineSnapshotTestData(t, database, "snapshot-invalid-json", `not-json`, `[]`)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/ci/snapshot/snapshot-invalid-json", nil, token))
	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected invalid pipeline snapshot status 500, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPipelineSnapshotRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/ci/snapshot/snapshot-route-test", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected pipeline snapshot get status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func insertPipelineSnapshotTestData(t *testing.T, database interface {
	Exec(query string, args ...any) (sql.Result, error)
}, snapshotId string, stages string, variables string) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO pipeline_snapshot (id, project_id, template_id, version, stages_snapshot, variables_snapshot, created_at) VALUES (?, '01KRRKK0K3T519ZQZES3M4QA9Z', '01KNVEJPWVK757139NMNNNCEFE', 99, ?, ?, '2024-03-16T00:00:00Z')`, snapshotId, stages, variables); err != nil {
		t.Fatal(err)
	}
}

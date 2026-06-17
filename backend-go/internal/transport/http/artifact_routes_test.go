package transporthttp

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/jmoiron/sqlx"

	cihandler "backend/internal/transport/http/handler/ci"
)

func TestArtifactListRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"
	insertArtifactTestData(t, database)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/ci/artifact?project_id="+projectId+"&page=1&per_page=20", nil, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected artifact list status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var artifacts paginatedResp[artifactResp]
	if err := json.NewDecoder(recorder.Body).Decode(&artifacts); err != nil {
		t.Fatal(err)
	}
	if artifacts.Total != 2 || len(artifacts.Items) != 2 || artifacts.PerPage != 20 {
		t.Fatalf("unexpected artifact list: %+v", artifacts)
	}
	if artifacts.Items[0].Id != "artifact-2" || artifacts.Items[0].RepositoryName != "golang/example" || artifacts.Items[0].Path == nil || *artifacts.Items[0].Path != "dist/test.log" {
		t.Fatalf("unexpected first artifact: %+v", artifacts.Items[0])
	}
}

func TestArtifactListFilters(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"
	insertArtifactTestData(t, database)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/ci/artifact?project_id="+projectId+"&repository_id=01KNNRBH52BQJYT9487B2H8N62&template_id=01KNVEJPWVK757139NMNNNCEFE&search=test.log", nil, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected filtered artifact list status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var artifacts paginatedResp[artifactResp]
	if err := json.NewDecoder(recorder.Body).Decode(&artifacts); err != nil {
		t.Fatal(err)
	}
	if artifacts.Total != 1 || len(artifacts.Items) != 1 || artifacts.Items[0].Id != "artifact-2" {
		t.Fatalf("unexpected filtered artifact list: %+v", artifacts)
	}
}

func TestArtifactListRequiresProject(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/ci/artifact", nil, token))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing project status 400, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestArtifactListValidatesRepositoryAndTemplate(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	repositoryRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(repositoryRecorder, authedRequest(http.MethodGet, "/api/ci/artifact?project_id="+projectId+"&repository_id=missing-repository", nil, token))
	if repositoryRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected missing repository status 404, got %d: %s", repositoryRecorder.Code, repositoryRecorder.Body.String())
	}

	templateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(templateRecorder, authedRequest(http.MethodGet, "/api/ci/artifact?project_id="+projectId+"&template_id=missing-template", nil, token))
	if templateRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected missing template status 404, got %d: %s", templateRecorder.Code, templateRecorder.Body.String())
	}
}

func TestPipelineRunArtifactsRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	insertArtifactTestData(t, database)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodGet, "/api/ci/run/run-artifact-test/artifacts", nil, token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected run artifact list status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var artifacts []cihandler.ArtifactResp
	if err := json.NewDecoder(recorder.Body).Decode(&artifacts); err != nil {
		t.Fatal(err)
	}
	if len(artifacts) != 2 || artifacts[0].Id != "artifact-1" || artifacts[1].Id != "artifact-2" {
		t.Fatalf("unexpected run artifact list: %+v", artifacts)
	}
}

func TestArtifactRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/ci/artifact", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected artifact list status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func insertArtifactTestData(t *testing.T, database *sqlx.DB) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO pipeline_run (id, project_id, repository_id, repository_name, snapshot_id, template_id, template_name, template_version, trigger, trigger_ref, variables_snapshot, status) VALUES ('run-artifact-test', '01KRRKK0K3T519ZQZES3M4QA9Z', '01KNNRBH52BQJYT9487B2H8N62', 'golang/example', 'snapshot-artifact-test', '01KNVEJPWVK757139NMNNNCEFE', 'Go 构建流水线', 7, 'manual', 'main', '{}', 'ran_to_completion')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO artifact (id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at) VALUES ('artifact-1', '01KRRKK0K3T519ZQZES3M4QA9Z', 'run-artifact-test', '01KNNRBH52BQJYT9487B2H8N62', 'golang/example', '01KNVEJPWVK757139NMNNNCEFE', 'Go 构建流水线', 'build', 'binary', 'app', 'dist/app', '2024-03-16T00:00:00Z')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO artifact (id, project_id, pipeline_run_id, repository_id, repository_name, template_id, template_name, stage_name, type, name, path, created_at) VALUES ('artifact-2', '01KRRKK0K3T519ZQZES3M4QA9Z', 'run-artifact-test', '01KNNRBH52BQJYT9487B2H8N62', 'golang/example', '01KNVEJPWVK757139NMNNNCEFE', 'Go 构建流水线', 'test', 'binary', 'test-log', 'dist/test.log', '2024-03-16T00:01:00Z')`); err != nil {
		t.Fatal(err)
	}
}

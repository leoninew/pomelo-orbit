package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestRepositoryRoutes(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/ci/repository?project_id="+projectId, bytes.NewBufferString(`{"name":"Repo One","code":"repo-one","repository_url":"https://example.test/repo.git","default_branch":"main","variable_overrides":[{"name":"FOO","value":"bar"}]}`), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected repository create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created repositoryResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Code != "repo-one" || created.DefaultBranch != "main" || len(created.VariableDeclarations) != 1 {
		t.Fatalf("unexpected created repository: %+v", created)
	}

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/ci/repository?project_id="+projectId+"&page=1&per_page=10", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected repository list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var repositories paginatedResp[repositoryResp]
	if err := json.NewDecoder(listRecorder.Body).Decode(&repositories); err != nil {
		t.Fatal(err)
	}
	if repositories.Total == 0 || repositories.Items == nil {
		t.Fatalf("unexpected repository list: %+v", repositories)
	}

	getRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRecorder, authedRequest(http.MethodGet, "/api/ci/repository/"+created.Id, nil, token))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected repository get status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/ci/repository/"+created.Id, bytes.NewBufferString(`{"name":"Repo One Updated","repository_url":"https://example.test/repo2.git","default_branch":"develop","variable_overrides":[]}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected repository update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated repositoryResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Repo One Updated" || updated.DefaultBranch != "develop" {
		t.Fatalf("unexpected updated repository: %+v", updated)
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/ci/repository/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected repository delete status 204, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestRepositoryTriggerRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	if _, err := database.Exec(`INSERT INTO pipeline_snapshot (id, project_id, template_id, version, stages_snapshot, variables_snapshot) VALUES ('snapshot-test-1', '01KRRKK0K3T519ZQZES3M4QA9Z', '01KNVEJPWVK757139NMNNNCEFE', 8, '[]', '[]')`); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/ci/repository/01KNNRBH52BQJYT9487B2H8N62/trigger", bytes.NewBufferString(`{"template_id":"01KNVEJPWVK757139NMNNNCEFE","trigger_ref":"main","variables":{"FOO":"bar"}}`), token))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected repository trigger status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var run pipelineRunResp
	if err := json.NewDecoder(recorder.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.Id == "" || run.RepositoryId != "01KNNRBH52BQJYT9487B2H8N62" || run.Status != "waiting_to_run" {
		t.Fatalf("unexpected triggered run: %+v", run)
	}
}

func TestRepositoryRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/ci/repository", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected repository list status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

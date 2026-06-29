package transporthttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	cihandler "backend/internal/transport/http/handler/ci"
	transportresponse "backend/internal/transport/http/response"
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
	var created cihandler.RepositoryResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Code != "repo-one" || created.DefaultBranch != "main" || len(created.VariableDeclarations) != 6 {
		t.Fatalf("unexpected created repository: %+v", created)
	}
	if created.VariableDeclarations[0]["name"] != "repository_id" || created.VariableDeclarations[0]["default"] != created.Id {
		t.Fatalf("unexpected repository_id variable: %+v", created.VariableDeclarations[0])
	}
	if created.VariableDeclarations[1]["name"] != "repository_name" || created.VariableDeclarations[1]["default"] != "Repo One" {
		t.Fatalf("unexpected repository_name variable: %+v", created.VariableDeclarations[1])
	}
	if created.VariableDeclarations[5]["name"] != "FOO" || created.VariableDeclarations[5]["source"] != "repository_custom" {
		t.Fatalf("unexpected custom variable: %+v", created.VariableDeclarations[5])
	}

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/ci/repository?project_id="+projectId+"&page=1&per_page=10", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected repository list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var repositories transportresponse.PaginatedResp[cihandler.RepositoryResp]
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

	if _, err := database.Exec(`INSERT INTO pipeline_run (id, project_id, repository_id, repository_name, snapshot_id, template_id, template_name, template_version, trigger, trigger_ref, variables_snapshot, status) VALUES ('run-repo-one', ?, ?, ?, 'snapshot-repo-one', 'template-repo-one', 'Template One', 1, 'manual', 'main', '{}', 'ran_to_completion')`, projectId, created.Id, created.Name); err != nil {
		t.Fatal(err)
	}
	runRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(runRecorder, authedRequest(http.MethodGet, "/api/ci/repository/"+created.Id+"/run?page=1&per_page=10", nil, token))
	if runRecorder.Code != http.StatusOK {
		t.Fatalf("expected repository run list status 200, got %d: %s", runRecorder.Code, runRecorder.Body.String())
	}
	var runs transportresponse.PaginatedResp[cihandler.PipelineRunResp]
	if err := json.NewDecoder(runRecorder.Body).Decode(&runs); err != nil {
		t.Fatal(err)
	}
	if runs.Total != 1 || len(runs.Items) != 1 || runs.Items[0].RepositoryId != created.Id {
		t.Fatalf("unexpected repository run list: %+v", runs)
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/ci/repository/"+created.Id, bytes.NewBufferString(`{"name":"Repo One Updated","repository_url":"https://example.test/repo2.git","default_branch":"develop","variable_overrides":[]}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected repository update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated cihandler.RepositoryResp
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

func TestRepositoryWebhookRoutes(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	repositoryId := "01KNNRBH52BQJYT9487B2H8N62"
	templateId := "01KNVEJPWVK757139NMNNNCEFE"

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/ci/repository/"+repositoryId+"/webhook", bytes.NewBufferString(`{"name":"github","template_id":"`+templateId+`","secret":"webhook-secret","branch_filter":"main"}`), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected webhook create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created cihandler.RepositoryWebhookResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.RepositoryId != repositoryId || created.TemplateId != templateId || !created.Enabled {
		t.Fatalf("unexpected created webhook: %+v", created)
	}

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/ci/repository/"+repositoryId+"/webhook?project_id=01KRRKK0K3T519ZQZES3M4QA9Z", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected webhook list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var webhooks []cihandler.RepositoryWebhookResp
	if err := json.NewDecoder(listRecorder.Body).Decode(&webhooks); err != nil {
		t.Fatal(err)
	}
	if len(webhooks) != 1 || webhooks[0].Id != created.Id {
		t.Fatalf("unexpected webhook list: %+v", webhooks)
	}

	getRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRecorder, authedRequest(http.MethodGet, "/api/ci/webhook/"+created.Id, nil, token))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected webhook get status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/ci/repository/"+repositoryId+"/webhook/"+created.Id, bytes.NewBufferString(`{"name":"github updated","branch_filter":"release/*","enabled":false}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected webhook update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated cihandler.RepositoryWebhookResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "github updated" || updated.BranchFilter == nil || *updated.BranchFilter != "release/*" || updated.Enabled {
		t.Fatalf("unexpected updated webhook: %+v", updated)
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/ci/repository/"+repositoryId+"/webhook/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected webhook delete status 204, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestRepositoryTriggerRouteCreatesSnapshotWhenMissing(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/ci/repository/01KNNRBH52BQJYT9487B2H8N62/trigger", bytes.NewBufferString(`{"template_id":"01KNVEJPWVK757139NMNNNCEFE","trigger_ref":"main","variables":{"FOO":"bar"}}`), token))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected repository trigger status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var run cihandler.PipelineRunResp
	if err := json.NewDecoder(recorder.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.Id == "" || run.RepositoryId != "01KNNRBH52BQJYT9487B2H8N62" || run.Status != "waiting_to_run" {
		t.Fatalf("unexpected triggered run: %+v", run)
	}
	if run.SnapshotId == "" || run.TemplateVersion != 7 {
		t.Fatalf("expected created snapshot on run, got snapshot=%q version=%d", run.SnapshotId, run.TemplateVersion)
	}
	var count int
	if err := database.Get(&count, `SELECT COUNT(*) FROM pipeline_snapshot WHERE template_id = '01KNVEJPWVK757139NMNNNCEFE' AND version = 7`); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one created snapshot, got %d", count)
	}
}

func TestRepositoryTriggerRouteReusesSameVersionSnapshot(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	if _, err := database.Exec(`INSERT INTO pipeline_snapshot (id, project_id, template_id, version, stages_snapshot, variables_snapshot) VALUES ('snapshot-test-1', '01KRRKK0K3T519ZQZES3M4QA9Z', '01KNVEJPWVK757139NMNNNCEFE', 7, '[]', '[]')`); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/ci/repository/01KNNRBH52BQJYT9487B2H8N62/trigger", bytes.NewBufferString(`{"template_id":"01KNVEJPWVK757139NMNNNCEFE","trigger_ref":"main","variables":{}}`), token))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected repository trigger status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var run cihandler.PipelineRunResp
	if err := json.NewDecoder(recorder.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.SnapshotId != "snapshot-test-1" {
		t.Fatalf("expected existing snapshot to be reused, got %q", run.SnapshotId)
	}
	var count int
	if err := database.Get(&count, `SELECT COUNT(*) FROM pipeline_snapshot WHERE template_id = '01KNVEJPWVK757139NMNNNCEFE' AND version = 7`); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected existing snapshot only, got %d", count)
	}
}

func TestRepositoryTriggerRouteCreatesNewSnapshotForTemplateVersion(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	if _, err := database.Exec(`INSERT INTO pipeline_snapshot (id, project_id, template_id, version, stages_snapshot, variables_snapshot) VALUES ('snapshot-test-old', '01KRRKK0K3T519ZQZES3M4QA9Z', '01KNVEJPWVK757139NMNNNCEFE', 6, '[]', '[]')`); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/ci/repository/01KNNRBH52BQJYT9487B2H8N62/trigger", bytes.NewBufferString(`{"template_id":"01KNVEJPWVK757139NMNNNCEFE","trigger_ref":"main","variables":{}}`), token))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected repository trigger status 201, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var run cihandler.PipelineRunResp
	if err := json.NewDecoder(recorder.Body).Decode(&run); err != nil {
		t.Fatal(err)
	}
	if run.SnapshotId == "snapshot-test-old" || run.TemplateVersion != 7 {
		t.Fatalf("expected new current-version snapshot, got snapshot=%q version=%d", run.SnapshotId, run.TemplateVersion)
	}
	var count int
	if err := database.Get(&count, `SELECT COUNT(*) FROM pipeline_snapshot WHERE template_id = '01KNVEJPWVK757139NMNNNCEFE'`); err != nil {
		t.Fatal(err)
	}
	if count != 2 {
		t.Fatalf("expected old and new snapshots, got %d", count)
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

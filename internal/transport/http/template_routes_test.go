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

func TestPipelineTemplateRoutes(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"
	stageId := "01KNRANZDR4PASATAXKTBBTRX9"

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/ci/template?project_id="+projectId, bytes.NewBufferString(`{"name":"Template One","description":"run ci","variable_declarations":[{"name":"CUSTOM_VAR","value":"demo","secret":false,"source":"template_custom"}]}`), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected template create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created cihandler.PipelineTemplateResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Name != "Template One" || created.Version != 1 || len(created.VariableDeclarations) == 0 {
		t.Fatalf("unexpected created template: %+v", created)
	}

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/ci/template?project_id="+projectId+"&page=1&per_page=10&search=Template%20One", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected template list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var templates transportresponse.PaginatedResp[cihandler.PipelineTemplateResp]
	if err := json.NewDecoder(listRecorder.Body).Decode(&templates); err != nil {
		t.Fatal(err)
	}
	if templates.Total != 1 || len(templates.Items) != 1 || templates.Items[0].Id != created.Id {
		t.Fatalf("unexpected template list: %+v", templates)
	}

	getRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRecorder, authedRequest(http.MethodGet, "/api/ci/template/"+created.Id, nil, token))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected template get status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/ci/template/"+created.Id, bytes.NewBufferString(`{"name":"Template One Updated","orchestration":[{"stage_id":"`+stageId+`","stage_name":"ignored","stage_version":1,"depends_on":[],"sort_order":0}],"variable_declarations":[{"name":"working_dir","value":".","secret":false,"source":"template_custom"}]}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected template update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated cihandler.PipelineTemplateResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Template One Updated" || updated.Version != 2 || len(updated.Orchestration) != 1 || updated.Orchestration[0].StageId != stageId || updated.Orchestration[0].StageName != "golang:1.23 test" || len(updated.Stages) != 1 {
		t.Fatalf("unexpected updated template: %+v", updated)
	}

	duplicateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(duplicateRecorder, authedRequest(http.MethodPost, "/api/ci/template/"+created.Id+"/duplicate", bytes.NewBufferString(`{}`), token))
	if duplicateRecorder.Code != http.StatusCreated {
		t.Fatalf("expected template duplicate status 201, got %d: %s", duplicateRecorder.Code, duplicateRecorder.Body.String())
	}
	var duplicated cihandler.PipelineTemplateResp
	if err := json.NewDecoder(duplicateRecorder.Body).Decode(&duplicated); err != nil {
		t.Fatal(err)
	}
	if duplicated.Id == created.Id || duplicated.Name != "Template One Updated copy" || duplicated.Version != 1 || len(duplicated.Orchestration) != 1 {
		t.Fatalf("unexpected duplicated template: %+v", duplicated)
	}

	deleteDuplicateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteDuplicateRecorder, authedRequest(http.MethodDelete, "/api/ci/template/"+duplicated.Id, nil, token))
	if deleteDuplicateRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected duplicated template delete status 204, got %d: %s", deleteDuplicateRecorder.Code, deleteDuplicateRecorder.Body.String())
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/ci/template/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected template delete status 204, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}

	missingRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(missingRecorder, authedRequest(http.MethodGet, "/api/ci/template/"+created.Id, nil, token))
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected deleted template get status 404, got %d: %s", missingRecorder.Code, missingRecorder.Body.String())
	}
}

func TestPipelineTemplateResolveVariablesRoute(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"
	stageId := "01KNRANZDR4PASATAXKTBBTRX9"

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/ci/template/resolve-variables?project_id="+projectId, bytes.NewBufferString(`{"orchestration":[{"stage_id":"`+stageId+`","stage_name":"ignored","stage_version":1,"depends_on":[],"sort_order":0}],"variable_declarations":[{"name":"working_dir","value":"src","secret":false,"source":"template_custom"}]}`), token))
	if recorder.Code != http.StatusOK {
		t.Fatalf("expected resolve variables status 200, got %d: %s", recorder.Code, recorder.Body.String())
	}
	var variables cihandler.TemplateVariableResolveResp
	if err := json.NewDecoder(recorder.Body).Decode(&variables); err != nil {
		t.Fatal(err)
	}
	if !hasVariable(variables.Items, "working_dir") {
		t.Fatalf("expected working_dir variable, got %+v", variables)
	}
}

func TestPipelineTemplateRoutesValidation(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	missingProjectRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(missingProjectRecorder, authedRequest(http.MethodGet, "/api/ci/template", nil, token))
	if missingProjectRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing project status 400, got %d: %s", missingProjectRecorder.Code, missingProjectRecorder.Body.String())
	}

	emptyNameRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(emptyNameRecorder, authedRequest(http.MethodPost, "/api/ci/template?project_id="+projectId, bytes.NewBufferString(`{"name":" "}`), token))
	if emptyNameRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected empty name status 400, got %d: %s", emptyNameRecorder.Code, emptyNameRecorder.Body.String())
	}

	unknownStageRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(unknownStageRecorder, authedRequest(http.MethodPut, "/api/ci/template/01KNVEJPWVK757139NMNNNCEFE", bytes.NewBufferString(`{"orchestration":[{"stage_id":"missing-stage","stage_name":"missing","stage_version":1,"depends_on":[],"sort_order":0}]}`), token))
	if unknownStageRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected unknown stage status 404, got %d: %s", unknownStageRecorder.Code, unknownStageRecorder.Body.String())
	}
}

func TestPipelineTemplateDeleteReferencedByWebhook(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	if _, err := database.Exec(`INSERT INTO repository_webhook (id, repository_id, name, template_id, encrypted_secret) VALUES ('webhook-template-delete-test', '01KNNRBH52BQJYT9487B2H8N62', 'delete guard', '01KNVEJPWVK757139NMNNNCEFE', 'secret')`); err != nil {
		t.Fatal(err)
	}

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodDelete, "/api/ci/template/01KNVEJPWVK757139NMNNNCEFE", nil, token))
	if recorder.Code != http.StatusConflict {
		t.Fatalf("expected referenced template delete status 409, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestPipelineTemplateRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/ci/template", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected template list status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func hasVariable(variables []cihandler.VariableDeclarationResp, name string) bool {
	for _, variable := range variables {
		if variable.Name == name {
			return true
		}
	}
	return false
}

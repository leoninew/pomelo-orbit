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

func TestBuildStageRoutes(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/ci/build-stage?project_id="+projectId, bytes.NewBufferString(`{"name":"go test custom","image":"golang:1.24","script":"go test ./...","artifacts":[{"type":"binary","path":"dist/app","name":"app"}],"description":"run tests"}`), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected build stage create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created cihandler.BuildStageResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Name != "go test custom" || created.Version != 1 || len(created.Artifacts) != 1 {
		t.Fatalf("unexpected created build stage: %+v", &created)
	}

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/ci/build-stage?project_id="+projectId+"&page=1&per_page=10&search=go%20test%20custom", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected build stage list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var stages transportresponse.PaginatedResp[cihandler.BuildStageResp]
	if err := json.NewDecoder(listRecorder.Body).Decode(&stages); err != nil {
		t.Fatal(err)
	}
	if stages.Total != 1 || len(stages.Items) != 1 || stages.Items[0].Id != created.Id {
		t.Fatalf("unexpected build stage list: %+v", stages)
	}

	getRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRecorder, authedRequest(http.MethodGet, "/api/ci/build-stage/"+created.Id, nil, token))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected build stage get status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/ci/build-stage/"+created.Id, bytes.NewBufferString(`{"name":"go test custom updated","script":"go test ./internal/...","artifacts":[],"description":"updated"}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected build stage update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated cihandler.BuildStageResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "go test custom updated" || updated.Script != "go test ./internal/..." || updated.Version != 2 || len(updated.Artifacts) != 0 {
		t.Fatalf("unexpected updated build stage: %+v", &updated)
	}

	duplicateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(duplicateRecorder, authedRequest(http.MethodPost, "/api/ci/build-stage/"+created.Id+"/duplicate", bytes.NewBufferString(`{}`), token))
	if duplicateRecorder.Code != http.StatusCreated {
		t.Fatalf("expected build stage duplicate status 201, got %d: %s", duplicateRecorder.Code, duplicateRecorder.Body.String())
	}
	var duplicated cihandler.BuildStageResp
	if err := json.NewDecoder(duplicateRecorder.Body).Decode(&duplicated); err != nil {
		t.Fatal(err)
	}
	if duplicated.Id == created.Id || duplicated.Name != "go test custom updated copy" || duplicated.Version != 1 {
		t.Fatalf("unexpected duplicated build stage: %+v", &duplicated)
	}

	deleteDuplicateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteDuplicateRecorder, authedRequest(http.MethodDelete, "/api/ci/build-stage/"+duplicated.Id, nil, token))
	if deleteDuplicateRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected duplicated build stage delete status 204, got %d: %s", deleteDuplicateRecorder.Code, deleteDuplicateRecorder.Body.String())
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/ci/build-stage/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected build stage delete status 204, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestBuildStageRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/ci/build-stage", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected build stage list status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

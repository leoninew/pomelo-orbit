package httpserver

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"backend/internal/orbit"
)

func TestApplicationRoutesCRUD(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/cd/application?project_id="+projectId, bytes.NewBufferString(`{"name":"Route App","code":"route-app","image_pull_policy":"missing"}`), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected application create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created applicationResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.ProjectId == nil || *created.ProjectId != projectId || created.Name != "Route App" || created.Code != "route-app" || created.Status != orbit.ApplicationStatusUndeployed || created.ImagePullPolicy != "missing" {
		t.Fatalf("unexpected created application: %+v", created)
	}

	getRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRecorder, authedRequest(http.MethodGet, "/api/cd/application/"+created.Id, nil, token))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected application get status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/cd/application?project_id="+projectId+"&search=Route", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected application list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var list paginatedResp[applicationResp]
	if err := json.NewDecoder(listRecorder.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].Id != created.Id {
		t.Fatalf("unexpected application list: %+v", list)
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/cd/application/"+created.Id, bytes.NewBufferString(`{"name":"Route App 2","code":"route-app-2","image_pull_policy":"always","route_managed":true}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected application update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated applicationResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Route App 2" || updated.Code != "route-app-2" || updated.ImagePullPolicy != "always" || !updated.RouteManaged {
		t.Fatalf("unexpected updated application: %+v", updated)
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/cd/application/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected application delete status 204, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}

	missingRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(missingRecorder, authedRequest(http.MethodGet, "/api/cd/application/"+created.Id, nil, token))
	if missingRecorder.Code != http.StatusNotFound {
		t.Fatalf("expected deleted application status 404, got %d: %s", missingRecorder.Code, missingRecorder.Body.String())
	}
}

func TestApplicationRoutesValidation(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	missingProjectRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(missingProjectRecorder, authedRequest(http.MethodPost, "/api/cd/application", bytes.NewBufferString(`{"name":"No Project","code":"no-project","image_pull_policy":"missing"}`), token))
	if missingProjectRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected missing project status 400, got %d: %s", missingProjectRecorder.Code, missingProjectRecorder.Body.String())
	}

	invalidRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(invalidRecorder, authedRequest(http.MethodPost, "/api/cd/application?project_id="+projectId, bytes.NewBufferString(`{"name":"Bad Code","code":"1bad","image_pull_policy":"missing"}`), token))
	if invalidRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected invalid create status 400, got %d: %s", invalidRecorder.Code, invalidRecorder.Body.String())
	}

	duplicateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(duplicateRecorder, authedRequest(http.MethodPost, "/api/cd/application?project_id="+projectId, bytes.NewBufferString(`{"name":"Traefik","code":"traefik-copy","image_pull_policy":"missing"}`), token))
	if duplicateRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected duplicate name status 400, got %d: %s", duplicateRecorder.Code, duplicateRecorder.Body.String())
	}

	unauthorizedRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(unauthorizedRecorder, httptest.NewRequest(http.MethodGet, "/api/cd/application?project_id="+projectId, nil))
	if unauthorizedRecorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected unauthorized list status 401, got %d: %s", unauthorizedRecorder.Code, unauthorizedRecorder.Body.String())
	}
}

func TestDeleteApplicationRejectsRunningStatus(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/cd/application?project_id="+projectId, bytes.NewBufferString(`{"name":"Running App","code":"running-app","image_pull_policy":"missing"}`), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected application create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created applicationResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`UPDATE application SET status = ? WHERE id = ?`, orbit.ApplicationStatusDeployed, created.Id); err != nil {
		t.Fatal(err)
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/cd/application/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusBadRequest {
		t.Fatalf("expected deployed delete status 400, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

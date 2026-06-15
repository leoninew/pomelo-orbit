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

func TestApplicationRelatedRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/cd/application/import"},
		{http.MethodGet, "/api/cd/application/app-1/export"},
		{http.MethodPost, "/api/cd/application/app-1/compose-preview"},
		{http.MethodGet, "/api/cd/application/app-1/files"},
		{http.MethodPost, "/api/cd/application/app-1/file"},
		{http.MethodGet, "/api/cd/application/app-1/file/file-1"},
		{http.MethodPut, "/api/cd/application/app-1/file/file-1"},
		{http.MethodDelete, "/api/cd/application/app-1/file/file-1"},
		{http.MethodPost, "/api/cd/application/app-1/deploy"},
		{http.MethodPost, "/api/cd/application/app-1/stop"},
		{http.MethodPost, "/api/cd/application/app-1/restart"},
		{http.MethodGet, "/api/cd/application/app-1/status"},
		{http.MethodGet, "/api/cd/application/app-1/logs"},
		{http.MethodGet, "/api/cd/application/app-1/route"},
		{http.MethodPost, "/api/cd/application/app-1/route"},
		{http.MethodPut, "/api/cd/application/app-1/route/route-1"},
		{http.MethodDelete, "/api/cd/application/app-1/route/route-1"},
		{http.MethodGet, "/api/cd/application/app-1/compose-service"},
		{http.MethodGet, "/api/cd/application/app-1/service-config"},
		{http.MethodPut, "/api/cd/application/app-1/service-config/web"},
	}
	for _, route := range routes {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(route.method, route.path, nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected %s %s status 401, got %d: %s", route.method, route.path, recorder.Code, recorder.Body.String())
		}
	}
}

func TestApplicationImportExportFilesRoutesAndServiceConfig(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)
	projectId := "01KRRKK0K3T519ZQZES3M4QA9Z"
	importBody := `{"name":"Imported App","code":"imported-app","image_pull_policy":"missing","route_managed":true,"config_files":[{"path":"docker-compose.yml","content":"services:\n  web:\n    image: nginx\n    ports:\n      - '8080:80'\n"}],"service_configs":[{"service_name":"web","image":"nginx:1.27"}],"routes":[{"service_name":"web","domain":"imported.example.test","port":80}]}`

	importRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(importRecorder, authedRequest(http.MethodPost, "/api/cd/application/import?project_id="+projectId, bytes.NewBufferString(importBody), token))
	if importRecorder.Code != http.StatusCreated {
		t.Fatalf("expected application import status 201, got %d: %s", importRecorder.Code, importRecorder.Body.String())
	}
	var imported applicationResp
	if err := json.NewDecoder(importRecorder.Body).Decode(&imported); err != nil {
		t.Fatal(err)
	}

	exportRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(exportRecorder, authedRequest(http.MethodGet, "/api/cd/application/"+imported.Id+"/export", nil, token))
	if exportRecorder.Code != http.StatusOK {
		t.Fatalf("expected application export status 200, got %d: %s", exportRecorder.Code, exportRecorder.Body.String())
	}
	var exported applicationExportResp
	if err := json.NewDecoder(exportRecorder.Body).Decode(&exported); err != nil {
		t.Fatal(err)
	}
	if exported.Name != "Imported App" || len(exported.ConfigFiles) != 1 || len(exported.ServiceConfigs) != 1 || len(exported.Routes) != 1 {
		t.Fatalf("unexpected exported application: %+v", exported)
	}

	fileRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(fileRecorder, authedRequest(http.MethodPost, "/api/cd/application/"+imported.Id+"/file", bytes.NewBufferString(`{"path":"README.md","content":"hello"}`), token))
	if fileRecorder.Code != http.StatusOK {
		t.Fatalf("expected application file create status 200, got %d: %s", fileRecorder.Code, fileRecorder.Body.String())
	}
	var createdFile configFileResp
	if err := json.NewDecoder(fileRecorder.Body).Decode(&createdFile); err != nil {
		t.Fatal(err)
	}

	readFileRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(readFileRecorder, authedRequest(http.MethodGet, "/api/cd/application/"+imported.Id+"/file/"+createdFile.Id, nil, token))
	if readFileRecorder.Code != http.StatusOK {
		t.Fatalf("expected application file read status 200, got %d: %s", readFileRecorder.Code, readFileRecorder.Body.String())
	}

	serviceRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(serviceRecorder, authedRequest(http.MethodGet, "/api/cd/application/"+imported.Id+"/compose-service", nil, token))
	if serviceRecorder.Code != http.StatusOK {
		t.Fatalf("expected compose service status 200, got %d: %s", serviceRecorder.Code, serviceRecorder.Body.String())
	}
	var services []composeServiceResp
	if err := json.NewDecoder(serviceRecorder.Body).Decode(&services); err != nil {
		t.Fatal(err)
	}
	if len(services) != 1 || services[0].ServiceName != "web" || services[0].DefaultPort != 80 {
		t.Fatalf("unexpected compose services: %+v", services)
	}

	previewRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(previewRecorder, authedRequest(http.MethodPost, "/api/cd/application/"+imported.Id+"/compose-preview", nil, token))
	if previewRecorder.Code != http.StatusOK {
		t.Fatalf("expected compose preview status 200, got %d: %s", previewRecorder.Code, previewRecorder.Body.String())
	}

	updateConfigRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateConfigRecorder, authedRequest(http.MethodPut, "/api/cd/application/"+imported.Id+"/service-config/web", bytes.NewBufferString(`{"image":"nginx:1.28"}`), token))
	if updateConfigRecorder.Code != http.StatusOK {
		t.Fatalf("expected service config update status 200, got %d: %s", updateConfigRecorder.Code, updateConfigRecorder.Body.String())
	}
	var serviceConfig applicationServiceConfigResp
	if err := json.NewDecoder(updateConfigRecorder.Body).Decode(&serviceConfig); err != nil {
		t.Fatal(err)
	}
	if serviceConfig.Image == nil || *serviceConfig.Image != "nginx:1.28" {
		t.Fatalf("unexpected service config: %+v", serviceConfig)
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

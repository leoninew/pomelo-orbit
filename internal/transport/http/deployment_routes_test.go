package transporthttp

import (
	apiv1 "backend/internal/gen/orbit/api/v1"
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

const deploymentRouteProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"
const deploymentRouteApplicationId = "01KN8CG4A5S4VVH6NKNJF4F9NJ"

func TestDeploymentListDetailLogsAndCancelRoutes(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	server.appCfg.Orbit.Root = t.TempDir()
	token := testToken(t, server)
	insertDeploymentRouteData(t, database, "deploy-route-test", "running")

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/cd/deployment?project_id="+deploymentRouteProjectId+"&application_id="+deploymentRouteApplicationId+"&status=running&search=File&date_from=2024-03-15T00:00:00Z&date_to=2024-03-17T00:00:00Z", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected deployment list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var list apiv1.DeploymentPaginatedResp
	if err := json.NewDecoder(listRecorder.Body).Decode(&list); err != nil {
		t.Fatal(err)
	}
	if list.Total != 1 || len(list.Items) != 1 || list.Items[0].Id != "deploy-route-test" {
		t.Fatalf("unexpected deployment list: %+v", &list)
	}

	detailRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(detailRecorder, authedRequest(http.MethodGet, "/api/cd/deployment/deploy-route-test", nil, token))
	if detailRecorder.Code != http.StatusOK {
		t.Fatalf("expected deployment detail status 200, got %d: %s", detailRecorder.Code, detailRecorder.Body.String())
	}
	var detail apiv1.DeploymentResp
	if err := json.NewDecoder(detailRecorder.Body).Decode(&detail); err != nil {
		t.Fatal(err)
	}
	if detail.ApplicationId == nil || *detail.ApplicationId != deploymentRouteApplicationId || detail.ApplicationName != "FileBrowser" || detail.CommandText != "docker compose -f docker-compose.yml up -d --remove-orphans --pull missing" {
		t.Fatalf("unexpected deployment detail: %+v", &detail)
	}

	logDir := filepath.Join(server.appCfg.DataRoot(), "cd", "filebrowser", "deployments")
	if err := os.MkdirAll(logDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(logDir, "deploy-route-test.log"), []byte("hello\nworld\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	logsRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(logsRecorder, authedRequest(http.MethodGet, "/api/cd/deployment/deploy-route-test/logs?offset=6", nil, token))
	if logsRecorder.Code != http.StatusOK {
		t.Fatalf("expected deployment logs status 200, got %d: %s", logsRecorder.Code, logsRecorder.Body.String())
	}
	var logsResp struct {
		Logs       string `json:"logs"`
		Offset     int    `json:"offset"`
		IsComplete bool   `json:"is_complete"`
		Status     string `json:"status"`
	}
	if err := json.NewDecoder(logsRecorder.Body).Decode(&logsResp); err != nil {
		t.Fatal(err)
	}
	if logsResp.Logs != "world\n" || logsResp.Offset != 12 || logsResp.IsComplete || logsResp.Status != "running" {
		t.Fatalf("unexpected deployment logs response: %+v", logsResp)
	}

	cancelRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(cancelRecorder, authedRequest(http.MethodPost, "/api/cd/deployment/deploy-route-test/cancel", bytes.NewBufferString(`{}`), token))
	if cancelRecorder.Code != http.StatusOK {
		t.Fatalf("expected deployment cancel status 200, got %d: %s", cancelRecorder.Code, cancelRecorder.Body.String())
	}
	var canceled apiv1.DeploymentResp
	if err := json.NewDecoder(cancelRecorder.Body).Decode(&canceled); err != nil {
		t.Fatal(err)
	}
	if canceled.Status != "canceled" || canceled.FinishedAt == nil || canceled.ErrorMessage == nil || *canceled.ErrorMessage != "Cancelled by user" {
		t.Fatalf("unexpected canceled deployment: %+v", &canceled)
	}
}

func TestDeploymentRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	routes := []struct {
		method string
		path   string
	}{
		{http.MethodGet, "/api/cd/deployment?project_id=" + deploymentRouteProjectId},
		{http.MethodGet, "/api/cd/deployment/deploy-route-test"},
		{http.MethodGet, "/api/cd/deployment/deploy-route-test/logs"},
		{http.MethodGet, "/api/cd/deployment/deploy-route-test/container-logs"},
		{http.MethodPost, "/api/cd/deployment/deploy-route-test/cancel"},
	}
	for _, route := range routes {
		recorder := httptest.NewRecorder()
		server.Handler().ServeHTTP(recorder, httptest.NewRequest(route.method, route.path, nil))
		if recorder.Code != http.StatusUnauthorized {
			t.Fatalf("expected %s %s status 401, got %d: %s", route.method, route.path, recorder.Code, recorder.Body.String())
		}
	}
}

func insertDeploymentRouteData(t *testing.T, database interface {
	Exec(query string, args ...any) (sql.Result, error)
}, deploymentId string, deploymentStatus string) {
	t.Helper()
	if _, err := database.Exec(`INSERT INTO deployment (id, project_id, application_id, application_name, operation_type, trigger_type, command_text, status, started_at, is_rollback) VALUES (?, ?, ?, 'FileBrowser', 'deploy', 'manual', 'docker compose -f docker-compose.yml up -d --remove-orphans --pull missing', ?, '2024-03-16T00:00:00Z', 0)`, deploymentId, deploymentRouteProjectId, deploymentRouteApplicationId, deploymentStatus); err != nil {
		t.Fatal(err)
	}
}

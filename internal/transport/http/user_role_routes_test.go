package transporthttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	authhandler "backend/internal/transport/http/handler/auth"
	rolehandler "backend/internal/transport/http/handler/role"
	userhandler "backend/internal/transport/http/handler/user"
	transportresponse "backend/internal/transport/http/response"
)

func testToken(t *testing.T, server Server) string {
	t.Helper()
	loginBody := bytes.NewBufferString(`{"username":"admin","password":"admin","csrf_token":"csrf","turnstile_token":"turnstile"}`)
	loginRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody))
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login status 200, got %d: %s", loginRecorder.Code, loginRecorder.Body.String())
	}
	var token authhandler.TokenResp
	if err := json.NewDecoder(loginRecorder.Body).Decode(&token); err != nil {
		t.Fatal(err)
	}
	return token.AccessToken
}

func authedRequest(method string, path string, body *bytes.Buffer, token string) *http.Request {
	var request *http.Request
	if body == nil {
		request = httptest.NewRequest(method, path, nil)
	} else {
		request = httptest.NewRequest(method, path, body)
		request.Header.Set("Content-Type", "application/json")
	}
	request.Header.Set("Authorization", "Bearer "+token)
	return request
}

func TestUserRoutes(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/user?page=1&per_page=10", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected user list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var users transportresponse.PaginatedResp[userhandler.UserListResp]
	if err := json.NewDecoder(listRecorder.Body).Decode(&users); err != nil {
		t.Fatal(err)
	}
	if users.Total == 0 || users.Items == nil || users.Items[0].RoleItems == nil {
		t.Fatalf("unexpected users response: %+v", users)
	}

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/user", bytes.NewBufferString(`{"username":"operator","password":"secret1","email":"operator@example.test"}`), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected user create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created userhandler.UserResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Username != "operator" || created.Status != "enabled" {
		t.Fatalf("unexpected created user: %+v", created)
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/user/"+created.Id, bytes.NewBufferString(`{"username":"operator2","status":"disabled"}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected user update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated userhandler.UserResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Username != "operator2" || updated.Status != "disabled" {
		t.Fatalf("unexpected updated user: %+v", updated)
	}

	enableRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(enableRecorder, authedRequest(http.MethodPost, "/api/user/"+created.Id+"/enable", nil, token))
	if enableRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected user enable status 204, got %d: %s", enableRecorder.Code, enableRecorder.Body.String())
	}

	roleRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(roleRecorder, authedRequest(http.MethodPut, "/api/user/"+created.Id+"/role", bytes.NewBufferString(`{"role_ids":["01KRXJXVPC6MQ75SZPWYZJSSAB"]}`), token))
	if roleRecorder.Code != http.StatusOK {
		t.Fatalf("expected user role update status 200, got %d: %s", roleRecorder.Code, roleRecorder.Body.String())
	}
	var withRole userhandler.UserResp
	if err := json.NewDecoder(roleRecorder.Body).Decode(&withRole); err != nil {
		t.Fatal(err)
	}
	if len(withRole.RoleItems) != 1 || withRole.RoleItems[0].Code != "admin" || len(withRole.RoleItems[0].PermissionCodes) == 0 {
		t.Fatalf("unexpected user roles: %+v", withRole)
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/user/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected user delete status 204, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestUserRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/user", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected user list status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestRoleRoutes(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	permissionsRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(permissionsRecorder, authedRequest(http.MethodGet, "/api/role/permission", nil, token))
	if permissionsRecorder.Code != http.StatusOK {
		t.Fatalf("expected permission list status 200, got %d: %s", permissionsRecorder.Code, permissionsRecorder.Body.String())
	}
	var permissions []rolehandler.PermissionResp
	if err := json.NewDecoder(permissionsRecorder.Body).Decode(&permissions); err != nil {
		t.Fatal(err)
	}
	if len(permissions) == 0 {
		t.Fatalf("expected seeded permissions")
	}

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/role", bytes.NewBufferString(`{"code":"auditor","name":"Auditor","description":"Audit users","permission_codes":["user:read"]}`), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected role create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created rolehandler.RoleResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Code != "auditor" || len(created.PermissionCodes) != 1 {
		t.Fatalf("unexpected created role: %+v", created)
	}

	listRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(listRecorder, authedRequest(http.MethodGet, "/api/role?page=1&per_page=10", nil, token))
	if listRecorder.Code != http.StatusOK {
		t.Fatalf("expected role list status 200, got %d: %s", listRecorder.Code, listRecorder.Body.String())
	}
	var roles transportresponse.PaginatedResp[rolehandler.RoleResp]
	if err := json.NewDecoder(listRecorder.Body).Decode(&roles); err != nil {
		t.Fatal(err)
	}
	if roles.Total < 2 || roles.Items == nil {
		t.Fatalf("unexpected roles response: %+v", roles)
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/role/"+created.Id, bytes.NewBufferString(`{"code":"auditor","name":"Auditor Updated","description":null,"permission_codes":["user:read","role:read"]}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected role update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated rolehandler.RoleResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Auditor Updated" || len(updated.PermissionCodes) != 2 {
		t.Fatalf("unexpected updated role: %+v", updated)
	}

	deleteRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deleteRecorder, authedRequest(http.MethodDelete, "/api/role/"+created.Id, nil, token))
	if deleteRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected role delete status 204, got %d: %s", deleteRecorder.Code, deleteRecorder.Body.String())
	}
}

func TestRoleCreateRejectsMissingPermission(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPost, "/api/role", bytes.NewBufferString(`{"code":"bad","name":"Bad","permission_codes":["missing:permission"]}`), token))
	if recorder.Code != http.StatusNotFound {
		t.Fatalf("expected role create status 404, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

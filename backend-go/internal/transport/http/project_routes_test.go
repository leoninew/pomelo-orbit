package transporthttp

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	userhandler "backend/internal/transport/http/handler/user"
)

func TestProjectRoutes(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	createRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(createRecorder, authedRequest(http.MethodPost, "/api/project", bytes.NewBufferString(`{"name":"Second Project","code":"second"}`), token))
	if createRecorder.Code != http.StatusCreated {
		t.Fatalf("expected project create status 201, got %d: %s", createRecorder.Code, createRecorder.Body.String())
	}
	var created projectResp
	if err := json.NewDecoder(createRecorder.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Code != "second" || !created.IsActive {
		t.Fatalf("unexpected created project: %+v", created)
	}

	getRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(getRecorder, authedRequest(http.MethodGet, "/api/project/"+created.Id, nil, token))
	if getRecorder.Code != http.StatusOK {
		t.Fatalf("expected project get status 200, got %d: %s", getRecorder.Code, getRecorder.Body.String())
	}

	updateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(updateRecorder, authedRequest(http.MethodPut, "/api/project/"+created.Id, bytes.NewBufferString(`{"name":"Second Project Updated","code":"second-updated"}`), token))
	if updateRecorder.Code != http.StatusOK {
		t.Fatalf("expected project update status 200, got %d: %s", updateRecorder.Code, updateRecorder.Body.String())
	}
	var updated projectResp
	if err := json.NewDecoder(updateRecorder.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Second Project Updated" || updated.Code != "second-updated" {
		t.Fatalf("unexpected updated project: %+v", updated)
	}

	memberRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(memberRecorder, authedRequest(http.MethodGet, "/api/project/"+created.Id+"/member", nil, token))
	if memberRecorder.Code != http.StatusOK {
		t.Fatalf("expected project member list status 200, got %d: %s", memberRecorder.Code, memberRecorder.Body.String())
	}
	var members []projectMemberResp
	if err := json.NewDecoder(memberRecorder.Body).Decode(&members); err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].Username != "admin" {
		t.Fatalf("unexpected project members: %+v", members)
	}

	userRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(userRecorder, authedRequest(http.MethodPost, "/api/user", bytes.NewBufferString(`{"username":"member","password":"secret1"}`), token))
	if userRecorder.Code != http.StatusCreated {
		t.Fatalf("expected user create status 201, got %d: %s", userRecorder.Code, userRecorder.Body.String())
	}
	var user userhandler.UserResp
	if err := json.NewDecoder(userRecorder.Body).Decode(&user); err != nil {
		t.Fatal(err)
	}

	addMemberRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(addMemberRecorder, authedRequest(http.MethodPost, "/api/project/"+created.Id+"/member", bytes.NewBufferString(`{"user_id":"`+user.Id+`"}`), token))
	if addMemberRecorder.Code != http.StatusOK {
		t.Fatalf("expected add member status 200, got %d: %s", addMemberRecorder.Code, addMemberRecorder.Body.String())
	}
	members = []projectMemberResp{}
	if err := json.NewDecoder(addMemberRecorder.Body).Decode(&members); err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("unexpected members after add: %+v", members)
	}

	removeMemberRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(removeMemberRecorder, authedRequest(http.MethodDelete, "/api/project/"+created.Id+"/member/"+user.Id, nil, token))
	if removeMemberRecorder.Code != http.StatusOK {
		t.Fatalf("expected remove member status 200, got %d: %s", removeMemberRecorder.Code, removeMemberRecorder.Body.String())
	}
	members = []projectMemberResp{}
	if err := json.NewDecoder(removeMemberRecorder.Body).Decode(&members); err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Fatalf("unexpected members after remove: %+v", members)
	}

	deprecateRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(deprecateRecorder, authedRequest(http.MethodPost, "/api/project/"+created.Id+"/deprecate", nil, token))
	if deprecateRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected deprecate status 204, got %d: %s", deprecateRecorder.Code, deprecateRecorder.Body.String())
	}
}

func TestProjectRoutesRequireAuth(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/project", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected project list status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

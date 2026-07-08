package transporthttp

import (
	"bytes"
	"encoding/json"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestAuthPasswordAndLoginHistoryRoutes(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	passwordRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(passwordRecorder, authedRequest(http.MethodPut, "/api/auth/password", bytes.NewBufferString(`{"old_password":"admin","new_password":"newpass1"}`), token))
	if passwordRecorder.Code != http.StatusNoContent {
		t.Fatalf("expected password change status 204, got %d: %s", passwordRecorder.Code, passwordRecorder.Body.String())
	}

	loginBody := bytes.NewBufferString(`{"username":"admin","password":"newpass1","csrf_token":"csrf","turnstile_token":"turnstile"}`)
	loginRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(loginRecorder, httptest.NewRequest(http.MethodPost, "/api/auth/login", loginBody))
	if loginRecorder.Code != http.StatusOK {
		t.Fatalf("expected login with new password status 200, got %d: %s", loginRecorder.Code, loginRecorder.Body.String())
	}

	historyRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(historyRecorder, authedRequest(http.MethodGet, "/api/auth/login-history?page=1&per_page=10", nil, token))
	if historyRecorder.Code != http.StatusOK {
		t.Fatalf("expected login history status 200, got %d: %s", historyRecorder.Code, historyRecorder.Body.String())
	}
	var history pomeloorbit.LoginHistoryPaginatedResp
	if err := json.NewDecoder(historyRecorder.Body).Decode(&history); err != nil {
		t.Fatal(err)
	}
	if history.Items == nil || history.Total == 0 {
		t.Fatalf("unexpected login history response: %+v", &history)
	}
}

func TestAuthPasswordRejectsWrongOldPassword(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()
	token := testToken(t, server)

	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, authedRequest(http.MethodPut, "/api/auth/password", bytes.NewBufferString(`{"old_password":"wrong","new_password":"newpass1"}`), token))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected password change status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestAuthGoogleRoutesReturnConfiguredError(t *testing.T) {
	server, database := newTestServer(t)
	defer func() { _ = database.Close() }()

	redirectRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(redirectRecorder, httptest.NewRequest(http.MethodGet, "/api/auth/google", nil))
	if redirectRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected google status 503, got %d: %s", redirectRecorder.Code, redirectRecorder.Body.String())
	}

	callbackRecorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(callbackRecorder, httptest.NewRequest(http.MethodPost, "/api/auth/google/callback", bytes.NewBufferString(`{"code":"code"}`)))
	if callbackRecorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected google callback status 503, got %d: %s", callbackRecorder.Code, callbackRecorder.Body.String())
	}
}

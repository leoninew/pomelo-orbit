package authz

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"gitee.com/leoninew/pomelo-orbit/internal/repository/model"
	authsvc "gitee.com/leoninew/pomelo-orbit/internal/service/auth"
)

const testSecret = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

type fakeAuthStore struct {
	user model.User
	err  error
}

func (s fakeAuthStore) UserById(ctx context.Context, id string) (model.User, error) {
	if s.err != nil {
		return model.User{}, s.err
	}
	return s.user, nil
}

func (s fakeAuthStore) UserPermissions(ctx context.Context, userId string) ([]string, error) {
	return nil, nil
}

func TestCurrentUserReturnsServiceUnavailableForUserLoadError(t *testing.T) {
	token := signedToken(t)
	authenticator := New(slog.Default(), fakeAuthStore{err: errors.New("database is locked")}, authsvc.NewTokenService(testSecret))

	recorder := httptest.NewRecorder()
	authenticator.CurrentUser(recorder, authedRequest(token))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCurrentUserReturnsUnauthorizedForMissingUser(t *testing.T) {
	token := signedToken(t)
	authenticator := New(slog.Default(), fakeAuthStore{err: sql.ErrNoRows}, authsvc.NewTokenService(testSecret))

	recorder := httptest.NewRecorder()
	authenticator.CurrentUser(recorder, authedRequest(token))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCurrentUserReturnsUnauthorizedForDisabledUser(t *testing.T) {
	token := signedToken(t)
	authenticator := New(slog.Default(), fakeAuthStore{user: model.User{Id: "user-1", Status: "disabled"}}, authsvc.NewTokenService(testSecret))

	recorder := httptest.NewRecorder()
	authenticator.CurrentUser(recorder, authedRequest(token))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCurrentUserReturnsUnauthorizedForInvalidToken(t *testing.T) {
	authenticator := New(slog.Default(), fakeAuthStore{user: model.User{Id: "user-1", Status: "enabled"}}, authsvc.NewTokenService(testSecret))

	recorder := httptest.NewRecorder()
	authenticator.CurrentUser(recorder, authedRequest("invalid-token"))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func signedToken(t *testing.T) string {
	t.Helper()
	token, err := authsvc.NewTokenService(testSecret).Sign("user-1", "admin")
	if err != nil {
		t.Fatal(err)
	}
	return token
}

func authedRequest(token string) *http.Request {
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set("Authorization", "Bearer "+token)
	return request
}

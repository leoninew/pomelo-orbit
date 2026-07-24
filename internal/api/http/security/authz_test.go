package security

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
	jwt "gitee.com/leoninew/PomeloOrbit-go/internal/auth/jwt"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
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

func (s fakeAuthStore) UserRoles(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s fakeAuthStore) UserPermissions(context.Context, string) ([]string, error) {
	return nil, nil
}

func (s fakeAuthStore) UserByUsername(context.Context, string) (model.User, error) {
	panic("unexpected call")
}

func (s fakeAuthStore) UserByEmail(context.Context, string) (model.User, error) {
	panic("unexpected call")
}

func (s fakeAuthStore) CreateUser(context.Context, model.User) error {
	panic("unexpected call")
}

func (s fakeAuthStore) UpdateUser(context.Context, model.User) error {
	panic("unexpected call")
}

func (s fakeAuthStore) SetUserStatus(context.Context, string, string) error {
	panic("unexpected call")
}

func (s fakeAuthStore) DeleteUser(context.Context, string) error {
	panic("unexpected call")
}

func (s fakeAuthStore) MarkUserLoggedIn(context.Context, string) error {
	panic("unexpected call")
}

func (s fakeAuthStore) ListUsers(context.Context, int, int, string) (repository.Page[model.User], error) {
	panic("unexpected call")
}

func (s fakeAuthStore) SaveLoginHistory(context.Context, model.LoginHistory) error {
	panic("unexpected call")
}

func (s fakeAuthStore) ListLoginHistory(context.Context, int, int, string) (repository.Page[model.LoginHistory], error) {
	panic("unexpected call")
}

func (s fakeAuthStore) UserRolesByUserIds(context.Context, []string) (map[string][]model.Role, error) {
	panic("unexpected call")
}

func (s fakeAuthStore) UserRoleDetails(context.Context, string) ([]model.Role, error) {
	panic("unexpected call")
}

func (s fakeAuthStore) SetUserRoles(context.Context, string, []string) error {
	panic("unexpected call")
}

func TestCurrentUserReturnsServiceUnavailableForUserLoadError(t *testing.T) {
	token := signedToken(t)
	authenticator := testAuthenticator(fakeAuthStore{err: errors.New("database is locked")})

	recorder := httptest.NewRecorder()
	authenticator.CurrentUser(testContext(recorder, authedRequest(token)))

	if recorder.Code != http.StatusServiceUnavailable {
		t.Fatalf("expected status 503, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCurrentUserReturnsUnauthorizedForMissingUser(t *testing.T) {
	token := signedToken(t)
	authenticator := testAuthenticator(fakeAuthStore{err: repository.ErrNotFound})

	recorder := httptest.NewRecorder()
	authenticator.CurrentUser(testContext(recorder, authedRequest(token)))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCurrentUserReturnsUnauthorizedForDisabledUser(t *testing.T) {
	token := signedToken(t)
	authenticator := testAuthenticator(fakeAuthStore{user: model.User{Id: "user-1", Status: "disabled"}})

	recorder := httptest.NewRecorder()
	authenticator.CurrentUser(testContext(recorder, authedRequest(token)))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func TestCurrentUserReturnsUnauthorizedForInvalidToken(t *testing.T) {
	authenticator := testAuthenticator(fakeAuthStore{user: model.User{Id: "user-1", Status: "enabled"}})

	recorder := httptest.NewRecorder()
	authenticator.CurrentUser(testContext(recorder, authedRequest("invalid-token")))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status 401, got %d: %s", recorder.Code, recorder.Body.String())
	}
}

func testAuthenticator(store fakeAuthStore) Authenticator {
	return New(slog.Default(), authsvc.New(store, store, jwt.NewTokenService(testSecret), slog.Default()))
}

func testContext(recorder *httptest.ResponseRecorder, request *http.Request) *gin.Context {
	gin.SetMode(gin.TestMode)
	c, _ := gin.CreateTestContext(recorder)
	c.Request = request
	return c
}

func signedToken(t *testing.T) string {
	t.Helper()
	token, err := jwt.NewTokenService(testSecret).Sign("user-1", "admin")
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

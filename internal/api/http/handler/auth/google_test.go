package authhandler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/middleware"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	authport "github.com/leoninew/pomelo-orbit/internal/application/auth/port"
	authsvc "github.com/leoninew/pomelo-orbit/internal/application/auth/usecase"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type googleHTTPFacts struct {
	repository.UserStore
	repository.AuthStore
}

func (googleHTTPFacts) UserByOAuthAccount(context.Context, string, string) (model.User, error) {
	return model.User{Id: "user-id", Username: "user", Status: "enabled"}, nil
}
func (googleHTTPFacts) MarkUserLoggedIn(context.Context, string) error             { return nil }
func (googleHTTPFacts) SaveLoginHistory(context.Context, model.LoginHistory) error { return nil }
func (googleHTTPFacts) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}
func (googleHTTPFacts) AuthorizationURL(state, _, _ string) string {
	return "https://accounts.google.com/authorize?state=" + state
}
func (googleHTTPFacts) Exchange(context.Context, string, string, string) (authport.GoogleIdentity, error) {
	return authport.GoogleIdentity{Subject: "google-user", Email: "user@example.test", EmailVerified: true}, nil
}

func googleHTTPRouter(enabled bool, redirectUri string) *gin.Engine {
	gin.SetMode(gin.TestMode)
	facts := googleHTTPFacts{}
	const secret = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	auth := authsvc.New(facts, facts, jwt.NewTokenService(secret), logger, secret)
	var provider authport.GoogleProvider
	if enabled {
		provider = facts
	}
	google := authsvc.NewGoogle(auth, provider, facts)
	handler := New(logger, config.TurnstileConfig{}, config.GoogleConfig{RedirectUri: redirectUri}, auth, google, security.Authenticator{}, nil)
	router := gin.New()
	router.Use(middleware.RequestId())
	router.GET("/api/auth/google", handler.GoogleOAuth)
	router.POST("/api/auth/google/callback", handler.GoogleCallback)
	return router
}

func TestGoogleHTTPAuthorizationAndCallback(t *testing.T) {
	for _, callback := range []string{"http://localhost:9020/google/callback", "https://orbit.example/google/callback"} {
		t.Run(callback, func(t *testing.T) {
			router := googleHTTPRouter(true, callback)
			start := httptest.NewRecorder()
			router.ServeHTTP(start, httptest.NewRequest(http.MethodGet, "/api/auth/google?redirect=%2Fprojects", nil))
			if start.Code != http.StatusFound {
				t.Fatalf("authorization status=%d body=%s", start.Code, start.Body)
			}
			location, err := url.Parse(start.Header().Get("Location"))
			if err != nil {
				t.Fatal(err)
			}
			cookies := start.Result().Cookies()
			if len(cookies) != 1 || !cookies[0].HttpOnly || cookies[0].SameSite != http.SameSiteLaxMode || cookies[0].Path != "/api/auth/google/callback" || cookies[0].Secure != strings.HasPrefix(callback, "https://") {
				t.Fatal("Google session cookie is not correctly scoped")
			}
			body, err := json.Marshal(map[string]string{"code": "google-code", "state": location.Query().Get("state")})
			if err != nil {
				t.Fatal(err)
			}
			request := httptest.NewRequest(http.MethodPost, "/api/auth/google/callback", strings.NewReader(string(body)))
			request.AddCookie(cookies[0])
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			var result struct {
				AccessToken string `json:"access_token"`
				TokenType   string `json:"token_type"`
				Redirect    string `json:"redirect"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if response.Code != http.StatusOK || result.AccessToken == "" || result.TokenType != "bearer" || result.Redirect != "/projects" || response.Header().Get("Cache-Control") != "no-store" {
				t.Fatalf("callback status=%d body=%s", response.Code, response.Body)
			}
			if cleared := response.Result().Cookies(); len(cleared) != 1 || cleared[0].MaxAge != -1 {
				t.Fatal("callback did not clear the login session")
			}
		})
	}
}

func TestGoogleHTTPErrorsUseContract(t *testing.T) {
	for _, test := range []struct {
		name, method, path, body string
		enabled                  bool
		status                   int
		code                     string
	}{
		{"disabled", http.MethodGet, "/api/auth/google", "", false, http.StatusServiceUnavailable, "service_unavailable"},
		{"missing session", http.MethodPost, "/api/auth/google/callback", `{"code":"code","state":"state"}`, true, http.StatusBadRequest, "validation_failed"},
		{"malformed JSON", http.MethodPost, "/api/auth/google/callback", `{`, true, http.StatusBadRequest, "validation_failed"},
	} {
		t.Run(test.name, func(t *testing.T) {
			router := googleHTTPRouter(test.enabled, "http://localhost:9020/google/callback")
			request := httptest.NewRequest(test.method, test.path, strings.NewReader(test.body))
			request.Header.Set("X-Request-ID", "google-login-request")
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			var result struct {
				Code      string `json:"code"`
				Error     string `json:"error"`
				RequestId string `json:"requestId"`
			}
			if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
				t.Fatal(err)
			}
			if response.Code != test.status || result.Code != test.code || result.Error == "" || result.RequestId != "google-login-request" {
				t.Fatalf("status=%d body=%s", response.Code, response.Body)
			}
		})
	}
}

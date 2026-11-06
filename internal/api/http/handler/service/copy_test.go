package servicehandler

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	authsvc "github.com/leoninew/pomelo-orbit/internal/application/auth/usecase"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type copyAuthStore struct{ repository.UserStore }

func (copyAuthStore) UserById(context.Context, string) (model.User, error) {
	return model.User{Id: "user", Status: "enabled"}, nil
}
func (copyAuthStore) UserRoles(context.Context, string) ([]string, error)       { return nil, nil }
func (copyAuthStore) UserPermissions(context.Context, string) ([]string, error) { return nil, nil }

type copyUsecaseFake struct {
	err                       error
	called                    bool
	input                     servicedto.ServiceCopyInput
	userId, source, serviceId string
}

func (f *copyUsecaseFake) Copy(_ context.Context, userId, source, serviceId string, input servicedto.ServiceCopyInput) (model.Service, error) {
	f.called, f.userId, f.source, f.serviceId, f.input = true, userId, source, serviceId, input
	return model.Service{Id: "copied-service", Code: "private-config-not-in-response"}, f.err
}

func TestCopyServiceHTTPContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	tokens := jwt.NewTokenService("AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	token, err := tokens.Sign("user", "user")
	if err != nil {
		t.Fatal(err)
	}
	authenticator := security.New(logger, authsvc.New(copyAuthStore{}, nil, tokens, logger, ""))
	for _, test := range []struct {
		name, body string
		all        bool
		err        error
		status     int
		noAuth     bool
	}{
		{name: "default latest", body: `{"target_project_id":"target","deployment_directory":"/custom/db","environment_target_revision":2}`, status: http.StatusCreated},
		{name: "all versions", body: `{"target_project_id":"target","copy_all_versions":true,"deployment_directory":"/custom/db","environment_target_revision":2}`, all: true, status: http.StatusCreated},
		{name: "conflict", body: `{"target_project_id":"target","deployment_directory":"/custom/db","environment_target_revision":2}`, err: apperror.New(apperror.KindConflict, "Already exists"), status: http.StatusConflict},
		{name: "invalid JSON", body: `{`, status: http.StatusBadRequest},
		{name: "unauthenticated", body: `{}`, noAuth: true, status: http.StatusUnauthorized},
	} {
		t.Run(test.name, func(t *testing.T) {
			usecase := &copyUsecaseFake{err: test.err}
			router := gin.New()
			router.POST("/api/service/:service_id/copy", NewCopy(usecase, authenticator).CopyService)
			request := httptest.NewRequest(http.MethodPost, "/api/service/source-service/copy?project_id=source", strings.NewReader(test.body))
			if !test.noAuth {
				request.Header.Set("Authorization", "Bearer "+token)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)
			if response.Code != test.status {
				t.Fatalf("status = %d, want %d", response.Code, test.status)
			}
			if test.status == http.StatusCreated || test.status == http.StatusConflict {
				if !usecase.called || usecase.userId != "user" || usecase.source != "source" || usecase.input.TargetProjectId != "target" || usecase.serviceId != "source-service" || usecase.input.CopyAllVersions != test.all || usecase.input.DeploymentDirectory != "/custom/db" || usecase.input.EnvironmentTargetRevision != 2 {
					t.Fatal("copy input was not passed correctly")
				}
			} else if usecase.called {
				t.Fatal("invalid request reached usecase")
			}
			if test.status == http.StatusCreated {
				var payload map[string]string
				if err := json.Unmarshal(response.Body.Bytes(), &payload); err != nil {
					t.Fatal(err)
				}
				if len(payload) != 1 || payload["id"] != "copied-service" {
					t.Fatal("response must contain only the copied ID")
				}
			}
		})
	}
}

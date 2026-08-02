package response

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/protobuf/encoding/protojson"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/requestid"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	authv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/auth"
)

func TestProtoJSONPreservesProtoJSONContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/", func(c *gin.Context) {
		ProtoJSON(c, http.StatusCreated, &authv1.TokenResp{AccessToken: "token"})
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	if got := recorder.Header().Get("Content-Type"); got != "application/json; charset=utf-8" {
		t.Fatalf("expected JSON content type, got %q", got)
	}
	response := &authv1.TokenResp{}
	if err := protojson.Unmarshal(recorder.Body.Bytes(), response); err != nil {
		t.Fatalf("decode proto json response: %v", err)
	}
	if response.GetAccessToken() != "token" {
		t.Fatalf("expected access token %q, got %q", "token", response.GetAccessToken())
	}
	if response.GetTokenType() != "" {
		t.Fatalf("expected empty token type, got %q", response.GetTokenType())
	}
}

func TestWriteStatusErrorWritesContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/", func(c *gin.Context) {
		c.Set(requestid.ContextKey, "request-1")
		WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	var response ErrorResp
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error json: %v", err)
	}
	if got, want := response, (ErrorResp{Code: "validation_failed", Error: "Invalid JSON body", RequestId: "request-1"}); got != want {
		t.Fatalf("unexpected error response: got %+v, want %+v", got, want)
	}
}

func TestWriteErrorSetsBearerChallengeForUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/", func(c *gin.Context) {
		WriteError(c, apperror.New(apperror.KindUnauthorized, ""))
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("expected status %d, got %d", http.StatusUnauthorized, recorder.Code)
	}
	if got := recorder.Header().Get("WWW-Authenticate"); got != "Bearer" {
		t.Fatalf("expected bearer authentication challenge, got %q", got)
	}
}

func TestWriteErrorClassifiesErrorsWithoutLeakingCause(t *testing.T) {
	cases := []struct {
		name       string
		err        error
		statusCode int
		code       string
	}{
		{name: "validation", err: apperror.New(apperror.KindValidation, "Invalid JSON body"), statusCode: http.StatusBadRequest, code: "validation_failed"},
		{name: "unauthorized", err: apperror.New(apperror.KindUnauthorized, ""), statusCode: http.StatusUnauthorized, code: "unauthorized"},
		{name: "forbidden", err: apperror.New(apperror.KindForbidden, ""), statusCode: http.StatusForbidden, code: "forbidden"},
		{name: "not found", err: apperror.New(apperror.KindNotFound, ""), statusCode: http.StatusNotFound, code: "not_found"},
		{name: "method not allowed", err: apperror.New(apperror.KindMethodNotAllowed, ""), statusCode: http.StatusMethodNotAllowed, code: "method_not_allowed"},
		{name: "conflict", err: apperror.New(apperror.KindConflict, ""), statusCode: http.StatusConflict, code: "conflict"},
		{name: "unavailable", err: apperror.New(apperror.KindUnavailable, ""), statusCode: http.StatusServiceUnavailable, code: "service_unavailable"},
		{name: "internal", err: apperror.Wrap(apperror.KindInternal, "database password", assertError("token=secret")), statusCode: http.StatusInternalServerError, code: "internal_error"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			engine := gin.New()
			engine.GET("/", func(c *gin.Context) {
				c.Set(requestid.ContextKey, "request-1")
				WriteError(c, tc.err)
			})

			recorder := httptest.NewRecorder()
			engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

			if recorder.Code != tc.statusCode {
				t.Fatalf("expected status %d, got %d", tc.statusCode, recorder.Code)
			}
			var response ErrorResp
			if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
				t.Fatalf("decode error json: %v", err)
			}
			if response.Code != tc.code || response.RequestId != "request-1" {
				t.Fatalf("unexpected error response: %+v", response)
			}
			if tc.statusCode == http.StatusInternalServerError && (response.Error != "Internal server error." || containsAny(response.Error, "database", "secret", "token")) {
				t.Fatalf("internal error leaked cause: %+v", response)
			}
		})
	}
}

type assertError string

func (e assertError) Error() string {
	return string(e)
}

func containsAny(value string, candidates ...string) bool {
	for _, candidate := range candidates {
		if strings.Contains(value, candidate) {
			return true
		}
	}
	return false
}

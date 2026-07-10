package response

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
)

func TestProtoJSONPreservesProtoJSONContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/", func(c *gin.Context) {
		ProtoJSON(c, http.StatusCreated, &pomeloorbit.TokenResp{AccessToken: "token"})
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d", http.StatusCreated, recorder.Code)
	}
	if got, want := recorder.Body.String(), `{"access_token":"token", "token_type":""}`; got != want {
		t.Fatalf("unexpected proto json: got %s, want %s", got, want)
	}
}

func TestErrorPreservesDetailEnvelope(t *testing.T) {
	gin.SetMode(gin.TestMode)
	engine := gin.New()
	engine.GET("/", func(c *gin.Context) {
		Error(c, http.StatusBadRequest, "Invalid JSON body")
	})

	recorder := httptest.NewRecorder()
	engine.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/", nil))

	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, recorder.Code)
	}
	if got, want := recorder.Body.String(), `{"detail":"Invalid JSON body"}`; got != want {
		t.Fatalf("unexpected error json: got %s, want %s", got, want)
	}
}

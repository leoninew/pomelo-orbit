package transport

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	repositoryv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/repository"
)

func TestDecodeJSONUsesProtoJSON(t *testing.T) {
	context := testContext(`{"name":"repo","repository_url":"https://example.test/repo.git"}`)
	var request repositoryv1.RepositoryCreateReq

	if err := DecodeJSON(context, &request); err != nil {
		t.Fatalf("DecodeJSON() error = %v", err)
	}
	if request.GetName() != "repo" || request.GetRepositoryUrl() != "https://example.test/repo.git" {
		t.Fatalf("unexpected decoded request: name=%q repository_url=%q", request.GetName(), request.GetRepositoryUrl())
	}
}

func TestDecodeJSONRejectsUnknownProtoFields(t *testing.T) {
	context := testContext(`{"name":"repo","unknown_field":"value"}`)
	var request repositoryv1.RepositoryCreateReq

	if err := DecodeJSON(context, &request); err == nil {
		t.Fatal("DecodeJSON() error = nil, want unknown field error")
	}
}

func testContext(body string) *gin.Context {
	gin.SetMode(gin.TestMode)
	context, _ := gin.CreateTestContext(httptest.NewRecorder())
	context.Request = httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	return context
}

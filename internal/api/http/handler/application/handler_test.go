package applicationhandler

import (
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"testing"

	transportmiddleware "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/middleware"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/requestid"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"github.com/gin-gonic/gin"
)

func TestWriteErrorMapsRuntimeCredentialReadFailureToSafeContract(t *testing.T) {
	gin.SetMode(gin.TestMode)
	handler := Handler{logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	router := gin.New()
	router.Use(transportmiddleware.RequestId())
	router.GET("/", func(c *gin.Context) {
		handler.writeError(c, apperror.Wrap(apperror.KindInternal, "Failed to read runtime_env credential", errors.New("invalid ciphertext")))
	})

	requestId := "01J1VY6M3R92K1WSPJ4AK84NQZ"
	request := httptest.NewRequest(http.MethodGet, "/", nil)
	request.Header.Set(requestid.HeaderName, requestId)
	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, request)

	if recorder.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, recorder.Code)
	}
	var response transportresponse.ErrorResp
	if err := json.Unmarshal(recorder.Body.Bytes(), &response); err != nil {
		t.Fatalf("decode error response: %v", err)
	}
	if got, want := response, (transportresponse.ErrorResp{Code: "internal_error", Error: "Internal server error.", RequestId: requestId}); got != want {
		t.Fatalf("unexpected error response: got %+v, want %+v", got, want)
	}
	if got := recorder.Header().Get(requestid.HeaderName); got != requestId {
		t.Fatalf("expected request id header %q, got %q", requestId, got)
	}
}

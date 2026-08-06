package delivery

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	deliverymcpclient "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/mcp/delivery"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestAuthenticatedStreamableHTTPHandlerBindsCurrentRequestUser(t *testing.T) {
	var actorUserId string
	handler := NewAuthenticatedStreamableHTTPHandler(func(_ context.Context, authorization string) (*mcp.Server, error) {
		if authorization != "Bearer current-token" {
			return nil, apperror.New(apperror.KindUnauthorized, "Invalid token")
		}
		actorUserId = "current-user"
		return NewServer(Dependencies{ActorUserId: actorUserId})
	})
	httpServer := httptest.NewServer(handler)
	defer httpServer.Close()

	session, err := deliverymcpclient.NewFactory(httpServer.URL).Connect(context.Background(), "Bearer current-token")
	if err != nil {
		t.Fatalf("ConnectStreamableHTTP() error = %v", err)
	}
	defer func() { _ = session.Close() }()
	if _, err := session.ListTools(context.Background()); err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if actorUserId != "current-user" {
		t.Fatalf("MCP actor user Id = %q, want current-user", actorUserId)
	}
}

func TestAuthenticatedStreamableHTTPHandlerRejectsRequestWithoutCredentials(t *testing.T) {
	handler := NewAuthenticatedStreamableHTTPHandler(func(_ context.Context, _ string) (*mcp.Server, error) {
		return nil, apperror.New(apperror.KindUnauthorized, "Not authenticated")
	})

	response := httptest.NewRecorder()
	handler.ServeHTTP(response, httptest.NewRequest(http.MethodGet, "/mcp", nil))
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want %d", response.Code, http.StatusUnauthorized)
	}
}

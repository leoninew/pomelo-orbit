package deliverymcp

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFactoryForwardsAuthorizationAndListsTools(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
	var authorization string
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(request *http.Request) *mcp.Server {
		authorization = request.Header.Get("Authorization")
		return server
	}, nil))
	defer httpServer.Close()

	client, err := NewFactory(httpServer.URL).Connect(context.Background(), "Bearer actor")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = client.Close() }()
	if _, err := client.ListTools(context.Background()); err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if authorization != "Bearer actor" {
		t.Fatalf("Authorization = %q, want Bearer actor", authorization)
	}
}

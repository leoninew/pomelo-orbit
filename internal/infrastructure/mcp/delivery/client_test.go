package deliverymcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestFactoryCreatesInMemoryActorBoundSession(t *testing.T) {
	client, err := NewFactory(func(actorUserId string) (*mcp.Server, error) {
		server := mcp.NewServer(&mcp.Implementation{Name: "test", Version: "1"}, nil)
		mcp.AddTool(server, &mcp.Tool{Name: "whoami"}, func(_ context.Context, _ *mcp.CallToolRequest, _ struct{}) (*mcp.CallToolResult, map[string]any, error) {
			return nil, map[string]any{"actor_user_id": actorUserId}, nil
		})
		return server, nil
	}).Connect(context.Background(), "actor-1")
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	defer func() { _ = client.Close() }()

	tools, err := client.ListTools(context.Background())
	if err != nil || len(tools) != 1 || tools[0].Name != "whoami" {
		t.Fatalf("ListTools() = %#v, %v", tools, err)
	}
	result, isError, err := client.CallTool(context.Background(), "whoami", json.RawMessage(`{}`))
	if err != nil || isError {
		t.Fatalf("CallTool() = %s, isError=%t, err=%v", result, isError, err)
	}
	if string(result) != `{"actor_user_id":"actor-1"}` {
		t.Fatalf("CallTool() result = %s", result)
	}
}

package delivery

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestWriteToolDoesNotExposeSuccessBeforeCommit(t *testing.T) {
	server := mcp.NewServer(&mcp.Implementation{Name: "transaction-test", Version: "1"}, nil)
	addWriteTool(server, failingToolCommit{}, "write", "write", func(context.Context, struct{}) (map[string]any, error) {
		return map[string]any{"saved": "uncommitted-result"}, nil
	})
	result, err := connectInMemory(t, server).CallTool(context.Background(), &mcp.CallToolParams{Name: "write"})
	if err != nil || !result.IsError {
		t.Fatalf("expected commit failure, result=%+v err=%v", result, err)
	}
	for _, content := range result.Content {
		if text, ok := content.(*mcp.TextContent); ok && strings.Contains(text.Text, "uncommitted-result") {
			t.Fatal("commit failure exposed uncommitted success")
		}
	}
}

type failingToolCommit struct{}

func (failingToolCommit) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	if err := fn(ctx); err != nil {
		return err
	}
	return errors.New("commit failed")
}

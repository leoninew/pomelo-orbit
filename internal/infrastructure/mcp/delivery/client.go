package deliverymcp

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/application/dialogue/port"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const implementationVersion = "0.1.0"

type ServerForActor func(string, string) (*mcp.Server, error)

// Factory creates a short-lived in-memory MCP session for one Dialogue turn.
// The Core is constructed by bootstrap so this infrastructure adapter never
// imports the delivery API package.
type Factory struct {
	serverForActor ServerForActor
}

func NewFactory(serverForActor ServerForActor) Factory {
	return Factory{serverForActor: serverForActor}
}

func (f Factory) Connect(ctx context.Context, actorUserId string, projectId string) (port.MCPClient, error) {
	actorUserId = strings.TrimSpace(actorUserId)
	if actorUserId == "" {
		return nil, errors.New("delivery MCP actor user ID is required")
	}
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return nil, errors.New("delivery MCP project ID is required")
	}
	if f.serverForActor == nil {
		return nil, errors.New("delivery MCP server factory is required")
	}
	server, err := f.serverForActor(actorUserId, projectId)
	if err != nil {
		return nil, err
	}
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(ctx, serverTransport, nil)
	if err != nil {
		return nil, err
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "pomelo-orbit-mcp", Version: implementationVersion}, nil)
	session, err := client.Connect(ctx, clientTransport, nil)
	if err != nil {
		return nil, errors.Join(err, serverSession.Close())
	}
	return sessionClient{session: session, serverSession: serverSession}, nil
}

type sessionClient struct {
	session       *mcp.ClientSession
	serverSession *mcp.ServerSession
}

func (c sessionClient) ListTools(ctx context.Context) ([]port.ToolDefinition, error) {
	result, err := c.session.ListTools(ctx, nil)
	if err != nil {
		return nil, err
	}
	tools := make([]port.ToolDefinition, 0, len(result.Tools))
	for _, tool := range result.Tools {
		schema, err := json.Marshal(tool.InputSchema)
		if err != nil {
			return nil, fmt.Errorf("marshal MCP tool schema %q: %w", tool.Name, err)
		}
		tools = append(tools, port.ToolDefinition{Name: tool.Name, Description: tool.Description, InputSchema: schema})
	}
	return tools, nil
}

func (c sessionClient) CallTool(ctx context.Context, name string, arguments json.RawMessage) (json.RawMessage, bool, error) {
	var values map[string]any
	if err := json.Unmarshal(arguments, &values); err != nil {
		return nil, false, fmt.Errorf("decode MCP tool arguments for %q: %w", name, err)
	}
	result, err := c.session.CallTool(ctx, &mcp.CallToolParams{Name: name, Arguments: values})
	if err != nil {
		return nil, false, err
	}
	if result.StructuredContent != nil {
		content, err := json.Marshal(result.StructuredContent)
		return content, result.IsError, err
	}
	content, err := json.Marshal(result.Content)
	return content, result.IsError, err
}

func (c sessionClient) Close() error {
	return errors.Join(c.session.Close(), c.serverSession.Close())
}

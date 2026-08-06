package deliverymcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/application/dialogue/port"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const implementationVersion = "0.1.0"

type Factory struct {
	endpoint string
}

func NewFactory(endpoint string) Factory {
	return Factory{endpoint: strings.TrimSpace(endpoint)}
}

func (f Factory) Connect(ctx context.Context, authorization string) (port.MCPClient, error) {
	if f.endpoint == "" {
		return nil, fmt.Errorf("delivery MCP endpoint is required")
	}
	client := mcp.NewClient(&mcp.Implementation{Name: "pomelo-orbit-http", Version: implementationVersion}, nil)
	session, err := client.Connect(ctx, &mcp.StreamableClientTransport{Endpoint: f.endpoint, HTTPClient: authorizationHTTPClient(authorization)}, nil)
	if err != nil {
		return nil, err
	}
	return sessionClient{session: session}, nil
}

type sessionClient struct {
	session *mcp.ClientSession
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
	return c.session.Close()
}

func authorizationHTTPClient(authorization string) *http.Client {
	if strings.TrimSpace(authorization) == "" {
		return nil
	}
	return &http.Client{Transport: authorizationTransport{authorization: authorization}}
}

type authorizationTransport struct{ authorization string }

func (t authorizationTransport) RoundTrip(request *http.Request) (*http.Response, error) {
	request = request.Clone(request.Context())
	request.Header.Set("Authorization", t.authorization)
	return http.DefaultTransport.RoundTrip(request)
}

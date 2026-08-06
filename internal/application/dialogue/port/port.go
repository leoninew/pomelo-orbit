package port

import (
	"context"
	"encoding/json"
)

type ToolDefinition struct {
	Name        string
	Description string
	InputSchema json.RawMessage
}

type ToolCall struct {
	Id        string
	Name      string
	Arguments json.RawMessage
}

type Message struct {
	Role       string
	Content    string
	ToolCallId string
	ToolCalls  []ToolCall
}

type CompletionRequest struct {
	Messages []Message
	Tools    []ToolDefinition
}

type CompletionResponse struct {
	Content   string
	ToolCalls []ToolCall
}

type LLMClient interface {
	IsConfigured() bool
	Complete(context.Context, CompletionRequest) (CompletionResponse, error)
}

type MCPClient interface {
	ListTools(context.Context) ([]ToolDefinition, error)
	CallTool(context.Context, string, json.RawMessage) (json.RawMessage, bool, error)
	Close() error
}

type MCPClientFactory interface {
	Connect(context.Context, string) (MCPClient, error)
}

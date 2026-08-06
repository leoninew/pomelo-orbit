package dialogueusecase

import (
	"context"
	"encoding/json"
	"testing"

	dialoguedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/dialogue/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/application/dialogue/port"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
)

func TestCompleteTurnRunsMCPToolCallsUntilAssistantReply(t *testing.T) {
	mcp := &fakeMCPClient{tools: []port.ToolDefinition{{Name: "orbit_list_applications", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
	factory := &fakeMCPFactory{client: mcp}
	llm := &fakeLLM{configured: true, responses: []port.CompletionResponse{
		{ToolCalls: []port.ToolCall{{Id: "call-1", Name: "orbit_list_applications", Arguments: json.RawMessage(`{"project_id":"project-1"}`)}}},
		{Content: "项目中有两个应用。"},
	}}
	service := New(llm, factory)

	result, err := service.CompleteTurn(context.Background(), "Bearer current-user", dialoguedto.TurnInput{
		ProjectId: "project-1",
		Messages:  []dialoguedto.Message{{Role: "user", Content: "列出应用"}},
	})
	if err != nil {
		t.Fatalf("CompleteTurn() error = %v", err)
	}
	if result.Message != "项目中有两个应用。" {
		t.Fatalf("Message = %q", result.Message)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "orbit_list_applications" {
		t.Fatalf("ToolCalls = %#v", result.ToolCalls)
	}
	if factory.authorization != "Bearer current-user" {
		t.Fatalf("authorization = %q", factory.authorization)
	}
	if len(mcp.calls) != 1 || mcp.calls[0].name != "orbit_list_applications" {
		t.Fatalf("MCP calls = %#v", mcp.calls)
	}
	if len(llm.requests) != 2 || len(llm.requests[1].Messages) != 4 {
		t.Fatalf("LLM requests = %#v", llm.requests)
	}
}

func TestCompleteTurnWithProgressReportsToolLifecycle(t *testing.T) {
	mcp := &fakeMCPClient{tools: []port.ToolDefinition{{Name: "orbit_list_applications", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
	llm := &fakeLLM{configured: true, responses: []port.CompletionResponse{
		{ToolCalls: []port.ToolCall{{Id: "call-1", Name: "orbit_list_applications", Arguments: json.RawMessage(`{"project_id":"project-1"}`)}}},
		{Content: "项目中有两个应用。"},
	}}
	service := New(llm, &fakeMCPFactory{client: mcp})
	var events []dialoguedto.StreamEvent

	_, err := service.CompleteTurnWithProgress(context.Background(), "Bearer current-user", dialoguedto.TurnInput{
		ProjectId: "project-1",
		Messages:  []dialoguedto.Message{{Role: "user", Content: "列出应用"}},
	}, func(event dialoguedto.StreamEvent) {
		events = append(events, event)
	})
	if err != nil {
		t.Fatalf("CompleteTurnWithProgress() error = %v", err)
	}
	if len(events) != 3 {
		t.Fatalf("progress events = %#v", events)
	}
	if events[0].Type != dialoguedto.StreamEventReady {
		t.Fatalf("first event = %#v", events[0])
	}
	if events[1].Type != dialoguedto.StreamEventToolCallStarted || events[1].ToolCall == nil || events[1].ToolCall.Name != "orbit_list_applications" {
		t.Fatalf("tool started event = %#v", events[1])
	}
	if events[2].Type != dialoguedto.StreamEventToolCallCompleted || events[2].ToolCall == nil || events[2].ToolCall.ResultJSON == "" {
		t.Fatalf("tool completed event = %#v", events[2])
	}
}

func TestCompleteTurnRequiresUserAsLastMessage(t *testing.T) {
	service := New(&fakeLLM{}, &fakeMCPFactory{client: &fakeMCPClient{}})
	_, err := service.CompleteTurn(context.Background(), "Bearer current-user", dialoguedto.TurnInput{
		ProjectId: "project-1",
		Messages:  []dialoguedto.Message{{Role: "assistant", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("CompleteTurn() error = nil, want validation error")
	}
}

func TestCompleteTurnReturnsConfigurationPromptBeforeMCPConnect(t *testing.T) {
	factory := &fakeMCPFactory{client: &fakeMCPClient{}}
	service := New(&fakeLLM{}, factory)

	_, err := service.CompleteTurn(context.Background(), "Bearer current-user", dialoguedto.TurnInput{
		ProjectId: "project-1",
		Messages:  []dialoguedto.Message{{Role: "user", Content: "列出应用"}},
	})
	if err == nil {
		t.Fatal("CompleteTurn() error = nil, want unavailable error")
	}
	appErr, ok := apperror.As(err)
	if !ok || appErr.Kind != apperror.KindUnavailable || appErr.Code != "deployment_dialogue_not_configured" {
		t.Fatalf("error = %#v", err)
	}
	if factory.authorization != "" {
		t.Fatalf("MCP Connect() was called with authorization %q", factory.authorization)
	}
}

type fakeLLM struct {
	configured bool
	requests   []port.CompletionRequest
	responses  []port.CompletionResponse
}

func (f *fakeLLM) IsConfigured() bool { return f.configured }

func (f *fakeLLM) Complete(_ context.Context, request port.CompletionRequest) (port.CompletionResponse, error) {
	f.requests = append(f.requests, request)
	if len(f.responses) == 0 {
		return port.CompletionResponse{}, nil
	}
	response := f.responses[0]
	f.responses = f.responses[1:]
	return response, nil
}

type fakeMCPFactory struct {
	authorization string
	client        port.MCPClient
}

func (f *fakeMCPFactory) Connect(_ context.Context, authorization string) (port.MCPClient, error) {
	f.authorization = authorization
	return f.client, nil
}

type fakeMCPClient struct {
	tools []port.ToolDefinition
	calls []fakeMCPCall
}

type fakeMCPCall struct {
	name string
}

func (f *fakeMCPClient) ListTools(context.Context) ([]port.ToolDefinition, error) {
	return f.tools, nil
}

func (f *fakeMCPClient) CallTool(_ context.Context, name string, _ json.RawMessage) (json.RawMessage, bool, error) {
	f.calls = append(f.calls, fakeMCPCall{name: name})
	return json.RawMessage(`{"applications":[]}`), false, nil
}

func (*fakeMCPClient) Close() error { return nil }

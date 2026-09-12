package dialogueusecase

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	dialoguedto "github.com/leoninew/pomelo-orbit/internal/application/dialogue/dto"
	"github.com/leoninew/pomelo-orbit/internal/application/dialogue/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestCompleteTurnRunsMCPToolCallsUntilAssistantReply(t *testing.T) {
	mcp := &fakeMCPClient{tools: []port.ToolDefinition{{Name: "orbit_list_applications", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
	factory := &fakeMCPFactory{client: mcp}
	store := newFakeDialogueStore()
	llm := &fakeLLM{configured: true, responses: []port.CompletionResponse{
		{ToolCalls: []port.ToolCall{{Id: "call-1", Name: "orbit_list_applications", Arguments: json.RawMessage(`{"project_id":"project-1"}`)}}},
		{Content: "项目中有两个应用。"},
	}}
	service := New(fakeDialogueProject{}, store, fakeDialogueTransaction{}, 32, llm, factory)

	result, err := service.CompleteTurn(context.Background(), "user-1", dialoguedto.TurnInput{
		ProjectId: "project-1",
		Messages:  []dialoguedto.Message{{Role: "user", Content: "列出应用"}},
	})
	if err != nil {
		t.Fatalf("CompleteTurn() error = %v", err)
	}
	if result.Message != "项目中有两个应用。" {
		t.Fatalf("Message = %q", result.Message)
	}
	if result.Conversation.Id == "" || result.Conversation.Title != "列出应用" {
		t.Fatalf("Conversation = %#v", result.Conversation)
	}
	if len(result.ToolCalls) != 1 || result.ToolCalls[0].Name != "orbit_list_applications" {
		t.Fatalf("ToolCalls = %#v", result.ToolCalls)
	}
	if factory.actorUserId != "user-1" {
		t.Fatalf("actor user ID = %q", factory.actorUserId)
	}
	if len(mcp.calls) != 1 || mcp.calls[0].name != "orbit_list_applications" {
		t.Fatalf("MCP calls = %#v", mcp.calls)
	}
	if len(llm.requests) != 2 || len(llm.requests[1].Messages) != 4 {
		t.Fatalf("LLM requests = %#v", llm.requests)
	}
	persisted := store.messages[result.Conversation.Id]
	if len(persisted) != 2 || persisted[0].Role != "user" || persisted[0].Content != "列出应用" || persisted[1].Role != "assistant" || persisted[1].Content != result.Message {
		t.Fatalf("persisted messages = %#v", persisted)
	}
}

func TestCompleteTurnDoesNotPersistWithoutFinalAssistantReply(t *testing.T) {
	store := newFakeDialogueStore()
	mcp := &fakeMCPClient{tools: []port.ToolDefinition{{Name: "orbit_list_applications"}}}
	service := New(
		fakeDialogueProject{},
		store,
		fakeDialogueTransaction{},
		32,
		&fakeLLM{configured: true, responses: []port.CompletionResponse{
			{ToolCalls: []port.ToolCall{{Id: "call-1", Name: "orbit_list_applications", Arguments: json.RawMessage(`{}`)}}},
		}},
		&fakeMCPFactory{client: mcp},
	)

	result, err := service.CompleteTurn(context.Background(), "user-1", dialoguedto.TurnInput{
		ProjectId: "project-1",
		Messages:  []dialoguedto.Message{{Role: "user", Content: "列出应用"}},
	})
	if err != nil {
		t.Fatalf("CompleteTurn() error = %v", err)
	}
	if result.Conversation.Id != "" {
		t.Fatalf("Conversation = %#v, want no persisted conversation", result.Conversation)
	}
	if len(store.conversations) != 0 || len(store.messages) != 0 {
		t.Fatalf("store = %#v, want no persistence", store)
	}
}

func TestCompleteTurnCreatesConversationWithClientConversationID(t *testing.T) {
	store := newFakeDialogueStore()
	service := New(
		fakeDialogueProject{},
		store,
		fakeDialogueTransaction{},
		32,
		&fakeLLM{configured: true, responses: []port.CompletionResponse{{Content: "回答"}}},
		&fakeMCPFactory{client: &fakeMCPClient{}},
	)

	result, err := service.CompleteTurn(context.Background(), "user-1", dialoguedto.TurnInput{
		ProjectId:      "project-1",
		ConversationId: "client-created-conversation",
		Messages:       []dialoguedto.Message{{Role: "user", Content: "问题"}},
	})
	if err != nil {
		t.Fatalf("CompleteTurn() error = %v", err)
	}
	if result.Conversation.Id != "client-created-conversation" {
		t.Fatalf("Conversation = %#v", result.Conversation)
	}
	if _, ok := store.conversations[result.Conversation.Id]; !ok {
		t.Fatalf("store = %#v", store.conversations)
	}
}

func TestCompleteTurnAppendsToExistingConversationAfterFinalAssistantReply(t *testing.T) {
	store := newFakeDialogueStore()
	conversation := model.DeploymentDialogueConversation{
		Id: "conversation-1", ProjectId: "project-1", CreatedByUserId: "user-1", Title: "旧问题",
		CreatedAt: time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC), UpdatedAt: time.Date(2026, 8, 15, 10, 0, 0, 0, time.UTC),
	}
	store.conversations[conversation.Id] = conversation
	store.messages[conversation.Id] = []model.DeploymentDialogueMessage{
		{Id: "message-1", ConversationId: conversation.Id, Role: "user", Content: "旧问题", CreatedAt: conversation.CreatedAt},
		{Id: "message-2", ConversationId: conversation.Id, Role: "assistant", Content: "旧回答", CreatedAt: conversation.CreatedAt.Add(time.Nanosecond)},
	}
	service := New(
		fakeDialogueProject{},
		store,
		fakeDialogueTransaction{},
		32,
		&fakeLLM{configured: true, responses: []port.CompletionResponse{{Content: "新回答"}}},
		&fakeMCPFactory{client: &fakeMCPClient{}},
	)

	result, err := service.CompleteTurn(context.Background(), "user-1", dialoguedto.TurnInput{
		ProjectId:      "project-1",
		ConversationId: conversation.Id,
		Messages: []dialoguedto.Message{
			{Role: "user", Content: "旧问题"},
			{Role: "assistant", Content: "旧回答"},
			{Role: "user", Content: "新问题"},
		},
	})
	if err != nil {
		t.Fatalf("CompleteTurn() error = %v", err)
	}
	if result.Conversation.Id != conversation.Id || len(store.conversations) != 1 {
		t.Fatalf("conversations = %#v", store.conversations)
	}
	persisted := store.messages[conversation.Id]
	if len(persisted) != 4 || persisted[2].Content != "新问题" || persisted[3].Content != "新回答" {
		t.Fatalf("persisted messages = %#v", persisted)
	}
}

func TestConversationHistoryCanBeReadAndDeleted(t *testing.T) {
	store := newFakeDialogueStore()
	conversation := model.DeploymentDialogueConversation{Id: "conversation-1", ProjectId: "project-1", Title: "问题"}
	store.conversations[conversation.Id] = conversation
	store.messages[conversation.Id] = []model.DeploymentDialogueMessage{{Id: "message-1", ConversationId: conversation.Id, Role: "user", Content: "问题"}}
	service := New(fakeDialogueProject{}, store, fakeDialogueTransaction{}, 32, &fakeLLM{}, &fakeMCPFactory{})

	items, err := service.ListConversations(context.Background(), "user-1", "project-1")
	if err != nil || len(items) != 1 || items[0].Id != conversation.Id {
		t.Fatalf("ListConversations() = %#v, %v", items, err)
	}
	detail, err := service.Conversation(context.Background(), "user-1", conversation.Id)
	if err != nil || len(detail.Messages) != 1 || detail.Messages[0].Content != "问题" {
		t.Fatalf("Conversation() = %#v, %v", detail, err)
	}
	if err := service.DeleteConversation(context.Background(), "user-1", conversation.Id); err != nil {
		t.Fatalf("DeleteConversation() error = %v", err)
	}
	if len(store.conversations) != 0 || len(store.messages) != 0 {
		t.Fatalf("store = %#v, want empty", store)
	}
}

func TestCompleteTurnWithProgressReportsToolLifecycle(t *testing.T) {
	mcp := &fakeMCPClient{tools: []port.ToolDefinition{{Name: "orbit_list_applications", InputSchema: json.RawMessage(`{"type":"object"}`)}}}
	llm := &fakeLLM{configured: true, responses: []port.CompletionResponse{
		{ToolCalls: []port.ToolCall{{Id: "call-1", Name: "orbit_list_applications", Arguments: json.RawMessage(`{"project_id":"project-1"}`)}}},
		{Content: "项目中有两个应用。"},
	}}
	service := newDialogueService(llm, &fakeMCPFactory{client: mcp})
	var events []dialoguedto.StreamEvent

	_, err := service.CompleteTurnWithProgress(context.Background(), "user-1", dialoguedto.TurnInput{
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
	service := newDialogueService(&fakeLLM{}, &fakeMCPFactory{client: &fakeMCPClient{}})
	_, err := service.CompleteTurn(context.Background(), "user-1", dialoguedto.TurnInput{
		ProjectId: "project-1",
		Messages:  []dialoguedto.Message{{Role: "assistant", Content: "hello"}},
	})
	if err == nil {
		t.Fatal("CompleteTurn() error = nil, want validation error")
	}
}

func TestCompleteTurnReturnsConfigurationPromptBeforeMCPConnect(t *testing.T) {
	factory := &fakeMCPFactory{client: &fakeMCPClient{}}
	service := newDialogueService(&fakeLLM{}, factory)

	_, err := service.CompleteTurn(context.Background(), "user-1", dialoguedto.TurnInput{
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
	if factory.actorUserId != "" {
		t.Fatalf("MCP Connect() was called with actor user ID %q", factory.actorUserId)
	}
}

func TestCompleteTurnRequiresVersionReadBeforeDeploy(t *testing.T) {
	mcp := &fakeMCPClient{tools: []port.ToolDefinition{{Name: "orbit_update_version_component_mounts"}, {Name: "orbit_get_version"}, {Name: "orbit_deploy"}}}
	llm := &fakeLLM{configured: true, responses: []port.CompletionResponse{
		{ToolCalls: []port.ToolCall{{Id: "write", Name: "orbit_update_version_component_mounts", Arguments: json.RawMessage(`{"version_id":"version-1","component_id":"component-1","mounts":[]}`)}}},
		{ToolCalls: []port.ToolCall{{Id: "early-deploy", Name: "orbit_deploy", Arguments: json.RawMessage(`{"service_id":"service-1"}`)}}},
		{ToolCalls: []port.ToolCall{{Id: "read", Name: "orbit_get_version", Arguments: json.RawMessage(`{"version_id":"version-1"}`)}}},
		{ToolCalls: []port.ToolCall{{Id: "deploy", Name: "orbit_deploy", Arguments: json.RawMessage(`{"service_id":"service-1"}`)}}},
		{Content: "已完成。"},
	}}

	result, err := newDialogueService(llm, &fakeMCPFactory{client: mcp}).CompleteTurn(context.Background(), "user-1", dialoguedto.TurnInput{ProjectId: "project-1", Messages: []dialoguedto.Message{{Role: "user", Content: "更新并部署"}}})
	if err != nil {
		t.Fatalf("CompleteTurn() error = %v", err)
	}
	if len(result.ToolCalls) != 4 || !result.ToolCalls[1].IsError {
		t.Fatalf("ToolCalls = %#v", result.ToolCalls)
	}
	if !strings.Contains(result.ToolCalls[1].ResultJSON, "orbit_get_version") {
		t.Fatalf("guard result = %s", result.ToolCalls[1].ResultJSON)
	}
	if len(mcp.calls) != 3 || mcp.calls[0].name != "orbit_update_version_component_mounts" || mcp.calls[1].name != "orbit_get_version" || mcp.calls[2].name != "orbit_deploy" {
		t.Fatalf("MCP calls = %#v", mcp.calls)
	}
}

func TestCompleteTurnRejectsDuplicateDeploymentForService(t *testing.T) {
	mcp := &fakeMCPClient{tools: []port.ToolDefinition{{Name: "orbit_deploy"}}}
	llm := &fakeLLM{configured: true, responses: []port.CompletionResponse{
		{ToolCalls: []port.ToolCall{{Id: "first", Name: "orbit_deploy", Arguments: json.RawMessage(`{"service_id":"service-1"}`)}}},
		{ToolCalls: []port.ToolCall{{Id: "second", Name: "orbit_deploy", Arguments: json.RawMessage(`{"service_id":"service-1"}`)}}},
		{Content: "已提交一次部署。"},
	}}

	result, err := newDialogueService(llm, &fakeMCPFactory{client: mcp}).CompleteTurn(context.Background(), "user-1", dialoguedto.TurnInput{ProjectId: "project-1", Messages: []dialoguedto.Message{{Role: "user", Content: "部署"}}})
	if err != nil {
		t.Fatalf("CompleteTurn() error = %v", err)
	}
	if len(result.ToolCalls) != 2 || !result.ToolCalls[1].IsError {
		t.Fatalf("ToolCalls = %#v", result.ToolCalls)
	}
	if len(mcp.calls) != 1 || mcp.calls[0].name != "orbit_deploy" {
		t.Fatalf("MCP calls = %#v", mcp.calls)
	}
}

func TestCompleteTurnStopsAfterConfiguredToolCallRounds(t *testing.T) {
	mcp := &fakeMCPClient{tools: []port.ToolDefinition{{Name: "orbit_list_applications"}}}
	service := New(
		fakeDialogueProject{},
		newFakeDialogueStore(),
		fakeDialogueTransaction{},
		1,
		&fakeLLM{configured: true, responses: []port.CompletionResponse{
			{ToolCalls: []port.ToolCall{{Id: "call-1", Name: "orbit_list_applications", Arguments: json.RawMessage(`{}`)}}},
		}},
		&fakeMCPFactory{client: mcp},
	)

	result, err := service.CompleteTurn(context.Background(), "user-1", dialoguedto.TurnInput{ProjectId: "project-1", Messages: []dialoguedto.Message{{Role: "user", Content: "列出应用"}}})
	if err != nil {
		t.Fatalf("CompleteTurn() error = %v", err)
	}
	if !strings.Contains(result.Message, "exceeded 1 tool-call rounds") {
		t.Fatalf("Message = %q", result.Message)
	}
	if len(mcp.calls) != 1 {
		t.Fatalf("MCP calls = %#v", mcp.calls)
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
	actorUserId string
	client      port.MCPClient
}

func (f *fakeMCPFactory) Connect(_ context.Context, actorUserId string, _ string) (port.MCPClient, error) {
	f.actorUserId = actorUserId
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

func newDialogueService(llm port.LLMClient, factory port.MCPClientFactory) Service {
	return New(fakeDialogueProject{}, newFakeDialogueStore(), fakeDialogueTransaction{}, 32, llm, factory)
}

type fakeDialogueProject struct{}

func (fakeDialogueProject) Project(_ context.Context, id string) (model.Project, error) {
	return model.Project{Id: id}, nil
}

func (fakeDialogueProject) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type fakeDialogueStore struct {
	conversations map[string]model.DeploymentDialogueConversation
	messages      map[string][]model.DeploymentDialogueMessage
}

func newFakeDialogueStore() *fakeDialogueStore {
	return &fakeDialogueStore{
		conversations: make(map[string]model.DeploymentDialogueConversation),
		messages:      make(map[string][]model.DeploymentDialogueMessage),
	}
}

func (s *fakeDialogueStore) ListDeploymentDialogueConversations(_ context.Context, projectId string) ([]model.DeploymentDialogueConversation, error) {
	items := make([]model.DeploymentDialogueConversation, 0)
	for _, conversation := range s.conversations {
		if conversation.ProjectId == projectId {
			items = append(items, conversation)
		}
	}
	return items, nil
}

func (s *fakeDialogueStore) DeploymentDialogueConversation(_ context.Context, id string) (model.DeploymentDialogueConversation, error) {
	conversation, ok := s.conversations[id]
	if !ok {
		return model.DeploymentDialogueConversation{}, repository.ErrNotFound
	}
	return conversation, nil
}

func (s *fakeDialogueStore) ListDeploymentDialogueMessages(_ context.Context, conversationId string) ([]model.DeploymentDialogueMessage, error) {
	return append([]model.DeploymentDialogueMessage(nil), s.messages[conversationId]...), nil
}

func (s *fakeDialogueStore) CreateDeploymentDialogueConversation(_ context.Context, conversation model.DeploymentDialogueConversation) error {
	s.conversations[conversation.Id] = conversation
	return nil
}

func (s *fakeDialogueStore) CreateDeploymentDialogueMessage(_ context.Context, message model.DeploymentDialogueMessage) error {
	s.messages[message.ConversationId] = append(s.messages[message.ConversationId], message)
	return nil
}

func (s *fakeDialogueStore) TouchDeploymentDialogueConversation(_ context.Context, id string, updatedAt time.Time) error {
	conversation, ok := s.conversations[id]
	if !ok {
		return repository.ErrNotFound
	}
	conversation.UpdatedAt = updatedAt
	s.conversations[id] = conversation
	return nil
}

func (s *fakeDialogueStore) DeleteDeploymentDialogueConversation(_ context.Context, id string) error {
	delete(s.conversations, id)
	delete(s.messages, id)
	return nil
}

type fakeDialogueTransaction struct{}

func (fakeDialogueTransaction) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

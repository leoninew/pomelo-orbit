package dto

import "time"

type Message struct {
	Role    string
	Content string
}

type TurnInput struct {
	ProjectId      string
	ConversationId string
	Messages       []Message
}

type ToolCall struct {
	Name          string
	ArgumentsJSON string
	ResultJSON    string
	IsError       bool
}

type TurnResult struct {
	Message      string
	ToolCalls    []ToolCall
	Conversation Conversation
}

type Conversation struct {
	Id        string
	ProjectId string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type ConversationDetail struct {
	Conversation Conversation
	Messages     []Message
}

const (
	StreamEventReady             = "ready"
	StreamEventToolCallStarted   = "tool_call_started"
	StreamEventToolCallCompleted = "tool_call_completed"
)

// StreamEvent reports a non-terminal step while a dialogue turn is running.
type StreamEvent struct {
	Type     string
	ToolCall *ToolCall
}

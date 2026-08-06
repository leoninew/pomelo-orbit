package dto

type Message struct {
	Role    string
	Content string
}

type TurnInput struct {
	ProjectId string
	Messages  []Message
}

type ToolCall struct {
	Name          string
	ArgumentsJSON string
	ResultJSON    string
	IsError       bool
}

type TurnResult struct {
	Message   string
	ToolCalls []ToolCall
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

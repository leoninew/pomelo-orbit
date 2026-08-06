package openai

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/application/dialogue/port"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

type Client struct {
	baseUrl string
	apiKey  string
	model   string
	http    *http.Client
}

func New(cfg config.LLMConfig) Client {
	timeout := cfg.Timeout
	if timeout <= 0 {
		timeout = config.DefaultLLMTimeout
	}
	return Client{
		baseUrl: strings.TrimRight(strings.TrimSpace(cfg.BaseUrl), "/"),
		apiKey:  strings.TrimSpace(cfg.ApiKey),
		model:   strings.TrimSpace(cfg.Model),
		http:    &http.Client{Timeout: timeout},
	}
}

func (c Client) IsConfigured() bool {
	return c.baseUrl != "" && c.apiKey != "" && c.model != ""
}

func (c Client) Complete(ctx context.Context, request port.CompletionRequest) (port.CompletionResponse, error) {
	if !c.IsConfigured() {
		return port.CompletionResponse{}, apperror.New(apperror.KindUnavailable, "Deployment dialogue is not configured")
	}
	payload, err := json.Marshal(completionRequest{Model: c.model, Messages: openAIMessages(request.Messages), Tools: openAITools(request.Tools), ToolChoice: "auto"})
	if err != nil {
		return port.CompletionResponse{}, apperror.Wrap(apperror.KindInternal, "encode deployment dialogue request", err)
	}
	httpRequest, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseUrl+"/chat/completions", bytes.NewReader(payload))
	if err != nil {
		return port.CompletionResponse{}, apperror.Wrap(apperror.KindInternal, "create deployment dialogue request", err)
	}
	httpRequest.Header.Set("Authorization", "Bearer "+c.apiKey)
	httpRequest.Header.Set("Content-Type", "application/json")
	response, err := c.http.Do(httpRequest)
	if err != nil {
		return port.CompletionResponse{}, apperror.Wrap(apperror.KindUnavailable, "Deployment dialogue provider is unavailable", err)
	}
	defer func() { _ = response.Body.Close() }()
	if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
		_, _ = io.Copy(io.Discard, response.Body)
		return port.CompletionResponse{}, apperror.New(apperror.KindUnavailable, "Deployment dialogue provider rejected the request")
	}
	var decoded completionResponse
	if err := json.NewDecoder(response.Body).Decode(&decoded); err != nil {
		return port.CompletionResponse{}, apperror.Wrap(apperror.KindUnavailable, "decode deployment dialogue provider response", err)
	}
	if len(decoded.Choices) == 0 {
		return port.CompletionResponse{}, apperror.New(apperror.KindUnavailable, "Deployment dialogue provider returned no choices")
	}
	message := decoded.Choices[0].Message
	result := port.CompletionResponse{ToolCalls: make([]port.ToolCall, 0, len(message.ToolCalls))}
	if message.Content != nil {
		result.Content = *message.Content
	}
	for _, call := range message.ToolCalls {
		arguments := json.RawMessage(call.Function.Arguments)
		if !json.Valid(arguments) {
			return port.CompletionResponse{}, apperror.New(apperror.KindUnavailable, fmt.Sprintf("Deployment dialogue provider returned invalid arguments for %s", call.Function.Name))
		}
		result.ToolCalls = append(result.ToolCalls, port.ToolCall{Id: call.Id, Name: call.Function.Name, Arguments: arguments})
	}
	return result, nil
}

type completionRequest struct {
	Model      string          `json:"model"`
	Messages   []openAIMessage `json:"messages"`
	Tools      []openAITool    `json:"tools"`
	ToolChoice string          `json:"tool_choice"`
}

type openAIMessage struct {
	Role       string           `json:"role"`
	Content    string           `json:"content,omitempty"`
	ToolCallId string           `json:"tool_call_id,omitempty"`
	ToolCalls  []openAIToolCall `json:"tool_calls,omitempty"`
}

type openAITool struct {
	Type     string         `json:"type"`
	Function openAIFunction `json:"function"`
}

type openAIFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

type openAIToolCall struct {
	Id       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type completionResponse struct {
	Choices []struct {
		Message struct {
			Content   *string          `json:"content"`
			ToolCalls []openAIToolCall `json:"tool_calls"`
		} `json:"message"`
	} `json:"choices"`
}

func openAIMessages(messages []port.Message) []openAIMessage {
	result := make([]openAIMessage, 0, len(messages))
	for _, message := range messages {
		item := openAIMessage{Role: message.Role, Content: message.Content, ToolCallId: message.ToolCallId}
		for _, call := range message.ToolCalls {
			toolCall := openAIToolCall{Id: call.Id, Type: "function"}
			toolCall.Function.Name = call.Name
			toolCall.Function.Arguments = string(call.Arguments)
			item.ToolCalls = append(item.ToolCalls, toolCall)
		}
		result = append(result, item)
	}
	return result
}

func openAITools(tools []port.ToolDefinition) []openAITool {
	result := make([]openAITool, 0, len(tools))
	for _, tool := range tools {
		result = append(result, openAITool{Type: "function", Function: openAIFunction{Name: tool.Name, Description: tool.Description, Parameters: tool.InputSchema}})
	}
	return result
}

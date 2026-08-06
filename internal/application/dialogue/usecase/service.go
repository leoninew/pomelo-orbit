package dialogueusecase

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	dialoguedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/dialogue/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/application/dialogue/port"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
)

const maxToolCallRounds = 12

type Service interface {
	CompleteTurn(context.Context, string, dialoguedto.TurnInput) (dialoguedto.TurnResult, error)
	CompleteTurnWithProgress(context.Context, string, dialoguedto.TurnInput, func(dialoguedto.StreamEvent)) (dialoguedto.TurnResult, error)
}

type service struct {
	llm        port.LLMClient
	mcpFactory port.MCPClientFactory
}

func New(llm port.LLMClient, mcpFactory port.MCPClientFactory) Service {
	return service{llm: llm, mcpFactory: mcpFactory}
}

func (s service) CompleteTurn(ctx context.Context, authorization string, input dialoguedto.TurnInput) (dialoguedto.TurnResult, error) {
	return s.completeTurn(ctx, authorization, input, nil)
}

func (s service) CompleteTurnWithProgress(ctx context.Context, authorization string, input dialoguedto.TurnInput, progress func(dialoguedto.StreamEvent)) (dialoguedto.TurnResult, error) {
	return s.completeTurn(ctx, authorization, input, progress)
}

func (s service) completeTurn(ctx context.Context, authorization string, input dialoguedto.TurnInput, progress func(dialoguedto.StreamEvent)) (dialoguedto.TurnResult, error) {
	if strings.TrimSpace(input.ProjectId) == "" {
		return dialoguedto.TurnResult{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if len(input.Messages) == 0 {
		return dialoguedto.TurnResult{}, apperror.New(apperror.KindValidation, "at least one dialogue message is required")
	}
	messages, err := dialogueMessages(input)
	if err != nil {
		return dialoguedto.TurnResult{}, err
	}
	if s.llm == nil || s.mcpFactory == nil {
		return dialoguedto.TurnResult{}, apperror.New(apperror.KindUnavailable, "Deployment dialogue is not configured")
	}
	if !s.llm.IsConfigured() {
		return dialoguedto.TurnResult{}, apperror.NewWithCode(
			apperror.KindUnavailable,
			"deployment_dialogue_not_configured",
			"Deployment dialogue is not configured",
		)
	}

	mcpClient, err := s.mcpFactory.Connect(ctx, authorization)
	if err != nil {
		return dialoguedto.TurnResult{}, apperror.Wrap(apperror.KindUnavailable, "Deployment dialogue MCP is unavailable", err)
	}
	defer func() { _ = mcpClient.Close() }()

	tools, err := mcpClient.ListTools(ctx)
	if err != nil {
		return dialoguedto.TurnResult{}, apperror.Wrap(apperror.KindUnavailable, "Deployment dialogue MCP is unavailable", err)
	}
	notifyProgress(progress, dialoguedto.StreamEvent{Type: dialoguedto.StreamEventReady})
	result := dialoguedto.TurnResult{}
	for range maxToolCallRounds {
		completion, err := s.llm.Complete(ctx, port.CompletionRequest{Messages: messages, Tools: tools})
		if err != nil {
			return dialoguedto.TurnResult{}, err
		}
		if len(completion.ToolCalls) == 0 {
			message := strings.TrimSpace(completion.Content)
			if message == "" {
				return dialoguedto.TurnResult{}, apperror.New(apperror.KindUnavailable, "Deployment dialogue returned an empty response")
			}
			result.Message = message
			return result, nil
		}

		messages = append(messages, port.Message{Role: "assistant", Content: completion.Content, ToolCalls: completion.ToolCalls})
		for _, call := range completion.ToolCalls {
			arguments := call.Arguments
			if len(arguments) == 0 {
				arguments = json.RawMessage(`{}`)
			}
			notifyProgress(progress, dialoguedto.StreamEvent{
				Type: dialoguedto.StreamEventToolCallStarted,
				ToolCall: &dialoguedto.ToolCall{
					Name:          call.Name,
					ArgumentsJSON: string(arguments),
				},
			})
			toolResult, isError, callErr := mcpClient.CallTool(ctx, call.Name, arguments)
			if callErr != nil {
				toolResult, _ = json.Marshal(map[string]string{"error": callErr.Error()})
				isError = true
			}
			toolCall := dialoguedto.ToolCall{
				Name:          call.Name,
				ArgumentsJSON: string(arguments),
				ResultJSON:    string(toolResult),
				IsError:       isError,
			}
			result.ToolCalls = append(result.ToolCalls, toolCall)
			notifyProgress(progress, dialoguedto.StreamEvent{
				Type:     dialoguedto.StreamEventToolCallCompleted,
				ToolCall: &toolCall,
			})
			messages = append(messages, port.Message{Role: "tool", Content: string(toolResult), ToolCallId: call.Id})
		}
	}

	return dialoguedto.TurnResult{}, apperror.New(apperror.KindUnavailable, fmt.Sprintf("Deployment dialogue exceeded %d tool-call rounds", maxToolCallRounds))
}

func notifyProgress(progress func(dialoguedto.StreamEvent), event dialoguedto.StreamEvent) {
	if progress != nil {
		progress(event)
	}
}

func dialogueMessages(input dialoguedto.TurnInput) ([]port.Message, error) {
	messages := []port.Message{{
		Role:    "system",
		Content: "You are the Pomelo Orbit continuous deployment assistant. Use the supplied MCP tools as the authoritative source for deployment state and as the only way to change applications, versions, services, gateways, deployments, and managed runtime state. Inspect relevant state before proposing or making changes. Explain completed operations with concrete identifiers. The active project_id is " + input.ProjectId + ".",
	}}
	for _, message := range input.Messages {
		role := strings.TrimSpace(message.Role)
		content := strings.TrimSpace(message.Content)
		if (role != "user" && role != "assistant") || content == "" {
			return nil, apperror.New(apperror.KindValidation, "messages must contain non-empty user or assistant entries")
		}
		messages = append(messages, port.Message{Role: role, Content: content})
	}
	if messages[len(messages)-1].Role != "user" {
		return nil, apperror.New(apperror.KindValidation, "the last dialogue message must be from the user")
	}
	return messages, nil
}

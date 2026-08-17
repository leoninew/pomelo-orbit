package dialogueusecase

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	dialoguedto "github.com/leoninew/pomelo-orbit/internal/application/dialogue/dto"
	"github.com/leoninew/pomelo-orbit/internal/application/dialogue/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

const maxToolCallRounds = 12

type Service interface {
	ListConversations(context.Context, string, string) ([]dialoguedto.Conversation, error)
	Conversation(context.Context, string, string) (dialoguedto.ConversationDetail, error)
	CompleteTurn(context.Context, string, string, dialoguedto.TurnInput) (dialoguedto.TurnResult, error)
	CompleteTurnWithProgress(context.Context, string, string, dialoguedto.TurnInput, func(dialoguedto.StreamEvent)) (dialoguedto.TurnResult, error)
	DeleteConversation(context.Context, string, string) error
}

type service struct {
	llm         port.LLMClient
	mcpFactory  port.MCPClientFactory
	project     repository.ProjectReader
	dialogue    repository.DeploymentDialogueStore
	transaction port.TransactionRunner
}

func New(project repository.ProjectReader, dialogue repository.DeploymentDialogueStore, transaction port.TransactionRunner, llm port.LLMClient, mcpFactory port.MCPClientFactory) Service {
	return service{project: project, dialogue: dialogue, transaction: transaction, llm: llm, mcpFactory: mcpFactory}
}

func (s service) ListConversations(ctx context.Context, userId, projectId string) ([]dialoguedto.Conversation, error) {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return nil, err
	}
	items, err := s.dialogue.ListDeploymentDialogueConversations(ctx, projectId)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list deployment dialogue conversations", err)
	}
	result := make([]dialoguedto.Conversation, 0, len(items))
	for _, item := range items {
		result = append(result, dialogueConversation(item))
	}
	return result, nil
}

func (s service) Conversation(ctx context.Context, userId, conversationId string) (dialoguedto.ConversationDetail, error) {
	conversation, err := s.loadConversationForUser(ctx, userId, conversationId)
	if err != nil {
		return dialoguedto.ConversationDetail{}, err
	}
	messages, err := s.dialogue.ListDeploymentDialogueMessages(ctx, conversation.Id)
	if err != nil {
		return dialoguedto.ConversationDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment dialogue messages", err)
	}
	result := dialoguedto.ConversationDetail{Conversation: dialogueConversation(conversation), Messages: make([]dialoguedto.Message, 0, len(messages))}
	for _, message := range messages {
		result.Messages = append(result.Messages, dialoguedto.Message{Role: message.Role, Content: message.Content})
	}
	return result, nil
}

func (s service) CompleteTurn(ctx context.Context, userId, authorization string, input dialoguedto.TurnInput) (dialoguedto.TurnResult, error) {
	return s.completeTurn(ctx, userId, authorization, input, nil)
}

func (s service) CompleteTurnWithProgress(ctx context.Context, userId, authorization string, input dialoguedto.TurnInput, progress func(dialoguedto.StreamEvent)) (dialoguedto.TurnResult, error) {
	return s.completeTurn(ctx, userId, authorization, input, progress)
}

func (s service) DeleteConversation(ctx context.Context, userId, conversationId string) error {
	conversation, err := s.loadConversationForUser(ctx, userId, conversationId)
	if err != nil {
		return err
	}
	if err := s.dialogue.DeleteDeploymentDialogueConversation(ctx, conversation.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete deployment dialogue conversation", err)
	}
	return nil
}

func (s service) completeTurn(ctx context.Context, userId, authorization string, input dialoguedto.TurnInput, progress func(dialoguedto.StreamEvent)) (dialoguedto.TurnResult, error) {
	input.ProjectId = strings.TrimSpace(input.ProjectId)
	input.ConversationId = strings.TrimSpace(input.ConversationId)
	if input.ProjectId == "" {
		return dialoguedto.TurnResult{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, input.ProjectId, userId); err != nil {
		return dialoguedto.TurnResult{}, err
	}
	if len(input.Messages) == 0 {
		return dialoguedto.TurnResult{}, apperror.New(apperror.KindValidation, "at least one dialogue message is required")
	}
	var conversation model.DeploymentDialogueConversation
	if input.ConversationId != "" {
		loaded, err := s.dialogue.DeploymentDialogueConversation(ctx, input.ConversationId)
		switch {
		case err == nil:
			if err := s.ensureProjectMembership(ctx, loaded.ProjectId, userId); err != nil {
				return dialoguedto.TurnResult{}, err
			}
			if loaded.ProjectId != input.ProjectId {
				return dialoguedto.TurnResult{}, apperror.New(apperror.KindNotFound, "Deployment dialogue conversation not found")
			}
			conversation = loaded
		case errors.Is(err, repository.ErrNotFound):
			// A client-created ID lets the stream client continue the new conversation
			// without adding persistence data to the established SSE complete event.
			conversation.Id = input.ConversationId
		default:
			return dialoguedto.TurnResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment dialogue conversation", err)
		}
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
	execution := newTurnExecution()
	for range maxToolCallRounds {
		completion, err := s.llm.Complete(ctx, port.CompletionRequest{Messages: messages, Tools: tools})
		if err != nil {
			return partialTurnResult(result, err)
		}
		if len(completion.ToolCalls) == 0 {
			if deploymentIds := execution.unwaitedDeploymentIds(); len(deploymentIds) > 0 {
				messages = append(messages, port.Message{Role: "user", Content: "You started Deployment " + strings.Join(deploymentIds, ", ") + ". Call orbit_wait_deployment for each before giving the final answer, then report each terminal status."})
				continue
			}
			message := strings.TrimSpace(completion.Content)
			if message == "" {
				return partialTurnResult(result, apperror.New(apperror.KindUnavailable, "Deployment dialogue returned an empty response"))
			}
			result.Message = message
			conversation, err = s.persistTurn(ctx, userId, input.ProjectId, conversation, input.Messages[len(input.Messages)-1].Content, message)
			if err != nil {
				return dialoguedto.TurnResult{}, err
			}
			result.Conversation = dialogueConversation(conversation)
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
			toolResult, isError, callErr := guardedToolCall(ctx, mcpClient, execution, call.Name, arguments)
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
			execution.record(call.Name, arguments, toolResult, isError)
			notifyProgress(progress, dialoguedto.StreamEvent{
				Type:     dialoguedto.StreamEventToolCallCompleted,
				ToolCall: &toolCall,
			})
			messages = append(messages, port.Message{Role: "tool", Content: string(toolResult), ToolCallId: call.Id})
		}
	}

	return partialTurnResult(result, apperror.New(apperror.KindUnavailable, fmt.Sprintf("Deployment dialogue exceeded %d tool-call rounds", maxToolCallRounds)))
}

func (s service) persistTurn(ctx context.Context, userId, projectId string, conversation model.DeploymentDialogueConversation, userContent, assistantContent string) (model.DeploymentDialogueConversation, error) {
	now := time.Now().UTC()
	isNewConversation := conversation.ProjectId == ""
	userContent = strings.TrimSpace(userContent)
	if userContent == "" {
		return model.DeploymentDialogueConversation{}, apperror.New(apperror.KindValidation, "the last dialogue message must be from the user")
	}
	if isNewConversation {
		if conversation.Id == "" {
			conversation.Id = idutil.NewId()
		}
		conversation = model.DeploymentDialogueConversation{
			Id: conversation.Id, ProjectId: projectId, CreatedByUserId: userId,
			Title: conversationTitle(userContent), CreatedAt: now, UpdatedAt: now.Add(time.Nanosecond),
		}
	}
	userMessage := model.DeploymentDialogueMessage{Id: idutil.NewId(), ConversationId: conversation.Id, Role: "user", Content: userContent, CreatedAt: now}
	assistantMessage := model.DeploymentDialogueMessage{Id: idutil.NewId(), ConversationId: conversation.Id, Role: "assistant", Content: assistantContent, CreatedAt: now.Add(time.Nanosecond)}
	if err := s.transaction.RunInTransaction(ctx, func(txCtx context.Context) error {
		if isNewConversation {
			if err := s.dialogue.CreateDeploymentDialogueConversation(txCtx, conversation); err != nil {
				return err
			}
		}
		if err := s.dialogue.CreateDeploymentDialogueMessage(txCtx, userMessage); err != nil {
			return err
		}
		if err := s.dialogue.CreateDeploymentDialogueMessage(txCtx, assistantMessage); err != nil {
			return err
		}
		if err := s.dialogue.TouchDeploymentDialogueConversation(txCtx, conversation.Id, assistantMessage.CreatedAt); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return model.DeploymentDialogueConversation{}, apperror.Wrap(apperror.KindInternal, "Failed to save deployment dialogue turn", err)
	}
	conversation.UpdatedAt = assistantMessage.CreatedAt
	return conversation, nil
}

func (s service) loadConversationForUser(ctx context.Context, userId, conversationId string) (model.DeploymentDialogueConversation, error) {
	conversation, err := s.dialogue.DeploymentDialogueConversation(ctx, strings.TrimSpace(conversationId))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.DeploymentDialogueConversation{}, apperror.New(apperror.KindNotFound, "Deployment dialogue conversation not found")
		}
		return model.DeploymentDialogueConversation{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment dialogue conversation", err)
	}
	if err := s.ensureProjectMembership(ctx, conversation.ProjectId, userId); err != nil {
		return model.DeploymentDialogueConversation{}, err
	}
	return conversation, nil
}

func (s service) ensureProjectMembership(ctx context.Context, projectId, userId string) error {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return apperror.New(apperror.KindValidation, "project_id is required")
	}
	if _, err := s.project.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.project.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func dialogueConversation(conversation model.DeploymentDialogueConversation) dialoguedto.Conversation {
	return dialoguedto.Conversation{Id: conversation.Id, ProjectId: conversation.ProjectId, Title: conversation.Title, CreatedAt: conversation.CreatedAt, UpdatedAt: conversation.UpdatedAt}
}

func conversationTitle(content string) string {
	title := []rune(strings.TrimSpace(content))
	if len(title) > 80 {
		title = title[:80]
	}
	return string(title)
}

func notifyProgress(progress func(dialoguedto.StreamEvent), event dialoguedto.StreamEvent) {
	if progress != nil {
		progress(event)
	}
}

func dialogueMessages(input dialoguedto.TurnInput) ([]port.Message, error) {
	messages := []port.Message{{
		Role:    "system",
		Content: "You are the Pomelo Orbit continuous deployment assistant. Use the supplied MCP tools as the authoritative source for deployment state and as the only way to change applications, versions, services, gateways, deployments, and managed runtime state. Treat the user's configuration as target state: inspect the relevant resource first, compare concrete fields, make the minimal necessary write when it differs, then read it back before deployment. Component collection writes replace the entire collection. To deploy a different Version, first update the Service binding with orbit_update_service_basic, preserving its instance key. orbit_deploy always uses the Service's saved Version. After a Version Component write call orbit_get_version for that version. After a Service write or creation call orbit_preview_service for that service before orbit_deploy. Create no more than one orbit_deploy for a Service in this user turn, and call orbit_wait_deployment for every Deployment you create before the final answer. Explain completed operations, terminal deployment status, and concrete identifiers. The active project_id is " + input.ProjectId + ".",
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

func guardedToolCall(ctx context.Context, client port.MCPClient, execution *turnExecution, name string, arguments json.RawMessage) (json.RawMessage, bool, error) {
	if err := execution.before(name, arguments); err != nil {
		return nil, true, err
	}
	return client.CallTool(ctx, name, arguments)
}

func partialTurnResult(result dialoguedto.TurnResult, cause error) (dialoguedto.TurnResult, error) {
	if len(result.ToolCalls) == 0 {
		return dialoguedto.TurnResult{}, cause
	}
	result.Message = "The deployment dialogue stopped before a final model response: " + cause.Error() + ". Completed tool calls remain available above; inspect their resource and deployment identifiers before continuing."
	return result, nil
}

package dialoguehandler

import (
	"fmt"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	"github.com/leoninew/pomelo-orbit/internal/api/http/codec"
	"github.com/leoninew/pomelo-orbit/internal/api/http/requestid"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	dialoguedto "github.com/leoninew/pomelo-orbit/internal/application/dialogue/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	dialoguev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/dialogue"
)

func (h Handler) CompleteTurn(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "dialogue:write")
	if !ok {
		return
	}
	var req dialoguev1.DeploymentDialogueTurnReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	result, err := h.service.CompleteTurn(c.Request.Context(), current.User.Id, c.GetHeader("Authorization"), dialogueTurnInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, dialogueTurnResponse(result))
}

func (h Handler) StreamTurn(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "dialogue:write")
	if !ok {
		return
	}
	var req dialoguev1.DeploymentDialogueTurnReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	streamStarted := false
	startStream := func() {
		if streamStarted {
			return
		}
		streamStarted = true
		c.Header("Cache-Control", "no-cache")
		c.Header("Connection", "keep-alive")
		c.Header("X-Accel-Buffering", "no")
		c.Header("Content-Type", "text/event-stream; charset=utf-8")
		c.Status(http.StatusOK)
		c.Writer.Flush()
	}
	emit := func(event *dialoguev1.DeploymentDialogueStreamEvent) {
		startStream()
		if err := writeStreamEvent(c, event); err != nil {
			h.logger.Error("write deployment dialogue stream event failed", "request_id", requestid.FromGinContext(c), "error", err)
		}
	}

	result, err := h.service.CompleteTurnWithProgress(c.Request.Context(), current.User.Id, c.GetHeader("Authorization"), dialogueTurnInput(&req), func(event dialoguedto.StreamEvent) {
		emit(dialogueStreamEvent(event))
	})
	if err != nil {
		if !streamStarted {
			transportresponse.WriteError(c, err)
			return
		}
		classification := apperror.Classify(err)
		h.logger.Error("deployment dialogue stream failed", "request_id", requestid.FromGinContext(c), "code", classification.Code, "error", err)
		emit(&dialoguev1.DeploymentDialogueStreamEvent{
			Type:      "error",
			Message:   classification.Message,
			Code:      classification.Code,
			RequestId: requestid.FromGinContext(c),
		})
		return
	}

	emit(&dialoguev1.DeploymentDialogueStreamEvent{Type: "complete", Message: result.Message})
}

func (h Handler) ListConversations(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "dialogue:read")
	if !ok {
		return
	}
	items, err := h.service.ListConversations(c.Request.Context(), current.User.Id, c.Request.URL.Query().Get("project_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	result := make([]dialoguev1.DeploymentDialogueConversation, 0, len(items))
	for _, item := range items {
		result = append(result, *dialogueConversationResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &dialoguev1.DeploymentDialogueConversationListResp{Items: transportresponse.Ptrs(result)})
}

func (h Handler) Conversation(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "dialogue:read")
	if !ok {
		return
	}
	detail, err := h.service.Conversation(c.Request.Context(), current.User.Id, c.Param("conversation_id"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	messages := make([]dialoguev1.DeploymentDialogueMessage, 0, len(detail.Messages))
	for _, message := range detail.Messages {
		messages = append(messages, dialogueMessageResponse(message))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &dialoguev1.DeploymentDialogueConversationDetailResp{Conversation: dialogueConversationResponse(detail.Conversation), Messages: transportresponse.Ptrs(messages)})
}

func (h Handler) DeleteConversation(c *gin.Context) {
	current, ok := h.authenticator.RequirePermission(c, "dialogue:write")
	if !ok {
		return
	}
	if err := h.service.DeleteConversation(c.Request.Context(), current.User.Id, c.Param("conversation_id")); err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func writeStreamEvent(c *gin.Context, event *dialoguev1.DeploymentDialogueStreamEvent) error {
	encoded, err := codec.MarshalProtoJSON(event)
	if err != nil {
		return err
	}
	if _, err := fmt.Fprintf(c.Writer, "data: %s\n\n", encoded); err != nil {
		return err
	}
	c.Writer.Flush()
	return nil
}

func dialogueStreamEvent(event dialoguedto.StreamEvent) *dialoguev1.DeploymentDialogueStreamEvent {
	result := &dialoguev1.DeploymentDialogueStreamEvent{Type: event.Type}
	if event.ToolCall != nil {
		result.ToolCall = &dialoguev1.DeploymentDialogueToolCall{
			Name:          event.ToolCall.Name,
			ArgumentsJson: event.ToolCall.ArgumentsJSON,
			ResultJson:    event.ToolCall.ResultJSON,
			IsError:       event.ToolCall.IsError,
		}
	}
	return result
}

func dialogueTurnInput(req *dialoguev1.DeploymentDialogueTurnReq) dialoguedto.TurnInput {
	if req == nil {
		return dialoguedto.TurnInput{}
	}
	result := dialoguedto.TurnInput{ProjectId: req.ProjectId, ConversationId: req.GetConversationId(), Messages: make([]dialoguedto.Message, 0, len(req.Messages))}
	for _, message := range req.Messages {
		if message != nil {
			result.Messages = append(result.Messages, dialoguedto.Message{Role: message.Role, Content: message.Content})
		}
	}
	return result
}

func dialogueTurnResponse(result dialoguedto.TurnResult) *dialoguev1.DeploymentDialogueTurnResp {
	toolCalls := make([]*dialoguev1.DeploymentDialogueToolCall, 0, len(result.ToolCalls))
	for _, call := range result.ToolCalls {
		toolCalls = append(toolCalls, &dialoguev1.DeploymentDialogueToolCall{Name: call.Name, ArgumentsJson: call.ArgumentsJSON, ResultJson: call.ResultJSON, IsError: call.IsError})
	}
	return &dialoguev1.DeploymentDialogueTurnResp{Message: result.Message, ToolCalls: toolCalls}
}

func dialogueConversationResponse(conversation dialoguedto.Conversation) *dialoguev1.DeploymentDialogueConversation {
	if conversation.Id == "" {
		return nil
	}
	return &dialoguev1.DeploymentDialogueConversation{
		Id:        conversation.Id,
		ProjectId: conversation.ProjectId,
		Title:     conversation.Title,
		CreatedAt: transportresponse.FormatTime(conversation.CreatedAt),
		UpdatedAt: transportresponse.FormatTime(conversation.UpdatedAt),
	}
}

func dialogueMessageResponse(message dialoguedto.Message) dialoguev1.DeploymentDialogueMessage {
	return dialoguev1.DeploymentDialogueMessage{Role: message.Role, Content: message.Content}
}

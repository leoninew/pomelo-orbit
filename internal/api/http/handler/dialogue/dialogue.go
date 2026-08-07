package dialoguehandler

import (
	"fmt"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/requestid"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	dialoguedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/dialogue/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	dialoguev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/dialogue"
	"github.com/gin-gonic/gin"
)

func (h Handler) CompleteTurn(c *gin.Context) {
	if _, ok := h.authenticator.CurrentUser(c); !ok {
		return
	}
	var req dialoguev1.DeploymentDialogueTurnReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	result, err := h.service.CompleteTurn(c.Request.Context(), c.GetHeader("Authorization"), dialogueTurnInput(&req))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, dialogueTurnResponse(result))
}

func (h Handler) StreamTurn(c *gin.Context) {
	if _, ok := h.authenticator.CurrentUser(c); !ok {
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

	result, err := h.service.CompleteTurnWithProgress(c.Request.Context(), c.GetHeader("Authorization"), dialogueTurnInput(&req), func(event dialoguedto.StreamEvent) {
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
	result := dialoguedto.TurnInput{ProjectId: req.ProjectId, Messages: make([]dialoguedto.Message, 0, len(req.Messages))}
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

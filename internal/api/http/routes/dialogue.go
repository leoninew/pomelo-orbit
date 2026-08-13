package routes

import (
	"github.com/gin-gonic/gin"

	dialoguehandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/dialogue"
)

func (r Router) registerDialogue(engine *gin.Engine) {
	handler := dialoguehandler.New(r.logger, r.deps.DialogueService, r.deps.Authenticator)
	engine.POST("/api/deployment-dialogue/turn", handler.CompleteTurn)
	engine.POST("/api/deployment-dialogue/turn/stream", handler.StreamTurn)
}

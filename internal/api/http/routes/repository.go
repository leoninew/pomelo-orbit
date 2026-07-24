package routes

import (
	"github.com/gin-gonic/gin"

	repositoryhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/repository"
)

func (r Router) registerRepository(engine *gin.Engine) {
	handler := repositoryhandler.New(r.logger, r.deps.RepositoryService, r.deps.Authenticator)

	engine.GET("/api/repository", handler.ListRepositories)
	engine.POST("/api/repository", handler.CreateRepository)
	engine.GET("/api/repository/:repository_id", handler.GetRepository)
	engine.PUT("/api/repository/:repository_id", handler.UpdateRepository)
	engine.DELETE("/api/repository/:repository_id", handler.DeleteRepository)
	engine.GET("/api/repository/:repository_id/webhook", handler.ListRepositoryWebhooks)
	engine.POST("/api/repository/:repository_id/webhook", handler.CreateRepositoryWebhook)
	engine.PUT("/api/repository/:repository_id/webhook/:webhook_id", handler.UpdateRepositoryWebhook)
	engine.DELETE("/api/repository/:repository_id/webhook/:webhook_id", handler.DeleteRepositoryWebhook)
	engine.GET("/api/webhook/:webhook_id", handler.GetRepositoryWebhook)
	engine.POST("/api/webhook/:webhook_id", handler.ReceiveRepositoryWebhook)
}

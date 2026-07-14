package routes

import (
	"github.com/gin-gonic/gin"

	cihandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/ci"
)

func (r Router) registerRepository(engine *gin.Engine) {
	handler := cihandler.New(r.logger, r.deps.CIService, r.deps.Authenticator)

	engine.GET("/api/ci/repository", handler.ListRepositories)
	engine.POST("/api/ci/repository", handler.CreateRepository)
	engine.GET("/api/ci/repository/:repository_id", handler.GetRepository)
	engine.PUT("/api/ci/repository/:repository_id", handler.UpdateRepository)
	engine.DELETE("/api/ci/repository/:repository_id", handler.DeleteRepository)
	engine.GET("/api/ci/repository/:repository_id/webhook", handler.ListRepositoryWebhooks)
	engine.POST("/api/ci/repository/:repository_id/webhook", handler.CreateRepositoryWebhook)
	engine.PUT("/api/ci/repository/:repository_id/webhook/:webhook_id", handler.UpdateRepositoryWebhook)
	engine.DELETE("/api/ci/repository/:repository_id/webhook/:webhook_id", handler.DeleteRepositoryWebhook)
	engine.GET("/api/ci/webhook/:webhook_id", handler.GetRepositoryWebhook)
	engine.POST("/api/ci/webhook/:webhook_id", handler.ReceiveRepositoryWebhook)
}

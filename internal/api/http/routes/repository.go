package routes

import (
	"github.com/gin-gonic/gin"

	repositoryhandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/repository"
)

func (r Router) registerRepository(engine *gin.Engine) {
	handler := repositoryhandler.New(r.logger, r.deps.RepositoryService, r.deps.Authenticator)

	engine.GET("/api/repository", handler.ListRepositories)
	engine.POST("/api/repository", handler.CreateRepository)
	engine.GET("/api/repository/:repository_id", handler.GetRepository)
	engine.PUT("/api/repository/:repository_id", handler.UpdateRepository)
	engine.DELETE("/api/repository/:repository_id", handler.DeleteRepository)
}

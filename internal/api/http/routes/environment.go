package routes

import (
	"github.com/gin-gonic/gin"

	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
)

func (r Router) registerEnvironment(engine *gin.Engine) {
	handler := cdhandler.New(r.logger, r.deps.CDService, r.deps.Authenticator)

	engine.GET("/api/cd/environment", handler.ListEnvironments)
	engine.POST("/api/cd/environment", handler.CreateEnvironment)
	engine.GET("/api/cd/environment/:env_id", handler.GetEnvironment)
	engine.PUT("/api/cd/environment/:env_id", handler.UpdateEnvironment)
	engine.DELETE("/api/cd/environment/:env_id", handler.DeleteEnvironment)
}

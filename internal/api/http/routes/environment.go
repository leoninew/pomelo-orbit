package routes

import (
	"github.com/gin-gonic/gin"

	environmenthandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/environment"
)

func (r Router) registerEnvironment(engine *gin.Engine) {
	handler := environmenthandler.New(r.logger, r.deps.EnvironmentService, r.deps.Authenticator)

	engine.GET("/api/environment", handler.ListEnvironments)
	engine.POST("/api/environment", handler.CreateEnvironment)
	engine.GET("/api/environment/:env_id", handler.GetEnvironment)
	engine.PUT("/api/environment/:env_id", handler.UpdateEnvironment)
	engine.DELETE("/api/environment/:env_id", handler.DeleteEnvironment)
}

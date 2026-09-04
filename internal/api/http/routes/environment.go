package routes

import (
	"github.com/gin-gonic/gin"

	environmenthandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/environment"
)

func (r Router) registerEnvironment(engine *gin.Engine) {
	handler := environmenthandler.New(r.logger, r.deps.EnvironmentService, r.deps.Authenticator)
	engine.GET("/api/project/:project_id/environment", handler.GetProjectEnvironment)
	engine.PUT("/api/project/:project_id/environment", handler.UpdateProjectEnvironment)
	engine.POST("/api/project/:project_id/environment/probe", handler.ProbeProjectEnvironment)
}

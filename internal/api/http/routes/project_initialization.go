package routes

import (
	"github.com/gin-gonic/gin"

	projectinitializationhandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/project_initialization"
)

func (r Router) registerProjectInitialization(engine *gin.Engine) {
	handler := projectinitializationhandler.New(r.logger, r.deps.ProjectInitializationService, r.deps.Authenticator)
	engine.GET("/api/project-initialization", handler.GetStatus)
	engine.POST("/api/project-initialization/environment/test", handler.TestEnvironment)
	engine.POST("/api/project-initialization/environment", handler.SaveEnvironment)
	engine.POST("/api/project-initialization/environment/windows-command", handler.PrepareWindowsEnvironment)
	engine.POST("/api/project-initialization/bootstrap", handler.BootstrapEnvironment)
	engine.POST("/api/project-initialization/probe", handler.ProbeEnvironment)
	engine.POST("/api/project-initialization/gateway", handler.CreateGateway)
}

package routes

import (
	"github.com/gin-gonic/gin"

	projectinitializationhandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/project_initialization"
)

func (r Router) registerProjectInitialization(engine *gin.Engine) {
	handler := projectinitializationhandler.New(r.logger, r.deps.ProjectInitializationService, r.deps.Authenticator)
	engine.GET("/api/project/:project_id/initialization", handler.GetStatus)
	engine.POST("/api/project/:project_id/initialization/environment/test", handler.TestEnvironment)
	engine.POST("/api/project/:project_id/initialization/environment", handler.SaveEnvironment)
	engine.GET("/api/project/:project_id/initialization/environment/deployment-key", handler.GetDeploymentPublicKey)
	engine.POST("/api/project/:project_id/initialization/bootstrap", handler.BootstrapEnvironment)
	engine.POST("/api/project/:project_id/initialization/probe", handler.ProbeEnvironment)
	engine.POST("/api/project/:project_id/initialization/gateway", handler.CreateGateway)
}

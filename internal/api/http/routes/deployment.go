package routes

import (
	"github.com/gin-gonic/gin"

	deploymenthandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/deployment"
)

func (r Router) registerDeployment(engine *gin.Engine) {
	handler := deploymenthandler.New(r.logger, r.deps.DeploymentService, r.deps.Authenticator)

	engine.GET("/api/deployment", handler.ListDeployments)
	engine.GET("/api/deployment/:deployment_id", handler.GetDeployment)
	engine.GET("/api/deployment/:deployment_id/logs", handler.GetDeploymentLogs)
	engine.GET("/api/deployment/:deployment_id/container-logs", handler.GetDeploymentContainerLogs)
	engine.POST("/api/deployment/:deployment_id/cancel", handler.CancelDeployment)
}

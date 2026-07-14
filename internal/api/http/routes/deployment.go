package routes

import (
	"github.com/gin-gonic/gin"

	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
)

func (r Router) registerDeployment(engine *gin.Engine) {
	handler := cdhandler.New(r.logger, r.deps.CDService, r.deps.Authenticator)

	engine.GET("/api/cd/deployment", handler.ListDeployments)
	engine.GET("/api/cd/deployment/:deployment_id", handler.GetDeployment)
	engine.GET("/api/cd/deployment/:deployment_id/logs", handler.GetDeploymentLogs)
	engine.GET("/api/cd/deployment/:deployment_id/container-logs", handler.GetDeploymentContainerLogs)
	engine.POST("/api/cd/deployment/:deployment_id/cancel", handler.CancelDeployment)
}

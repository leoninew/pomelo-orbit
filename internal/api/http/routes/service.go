package routes

import (
	"github.com/gin-gonic/gin"

	deploymenthandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/deployment"
	servicehandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/service"
)

func (r Router) registerService(engine *gin.Engine) {
	handler := servicehandler.New(r.logger, r.deps.ServiceService, r.deps.Authenticator)
	deploymentHandler := deploymenthandler.New(r.logger, r.deps.DeploymentService, r.deps.Authenticator)

	engine.GET("/api/service", handler.ListServices)
	engine.POST("/api/service", handler.CreateService)
	engine.GET("/api/service/:service_id", handler.GetService)
	engine.DELETE("/api/service/:service_id", handler.DeleteService)
	engine.PUT("/api/service/:service_id/basic", handler.UpdateServiceBasic)
	engine.PUT("/api/service/:service_id/config", handler.UpdateServiceConfiguration)
	engine.POST("/api/service/:service_id/preview", deploymentHandler.PreviewService)
	engine.POST("/api/service/:service_id/deploy", deploymentHandler.DeployService)
}

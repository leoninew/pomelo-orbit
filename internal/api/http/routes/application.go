package routes

import (
	"github.com/gin-gonic/gin"

	applicationhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/application"
	deploymenthandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/deployment"
	servicehandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/service"
)

func (r Router) registerApplication(engine *gin.Engine) {
	handler := applicationhandler.New(r.logger, r.deps.ApplicationService, r.deps.ServiceService, r.deps.Authenticator)
	deploymentHandler := deploymenthandler.New(r.logger, r.deps.DeploymentService, r.deps.Authenticator)
	serviceHandler := servicehandler.New(r.logger, r.deps.ServiceService, r.deps.Authenticator)

	engine.GET("/api/application", handler.ListApplications)
	engine.POST("/api/application", handler.CreateApplication)
	engine.POST("/api/application/import", handler.ImportApplication)
	engine.GET("/api/application/:app_id", handler.GetApplication)
	engine.PUT("/api/application/:app_id", handler.UpdateApplication)
	engine.DELETE("/api/application/:app_id", deploymentHandler.DeleteApplication)
	engine.POST("/api/application/:app_id/deploy", deploymentHandler.DeployApplication)
	engine.GET("/api/application/:app_id/export", handler.ExportApplication)
	engine.POST("/api/application/:app_id/stop", deploymentHandler.StopApplication)
	engine.POST("/api/application/:app_id/restart", deploymentHandler.RestartApplication)
	engine.GET("/api/application/:app_id/status", deploymentHandler.GetApplicationStatus)
	engine.GET("/api/application/:app_id/logs", deploymentHandler.GetApplicationLogs)
	engine.GET("/api/application/:app_id/service", serviceHandler.ListApplicationServices)
	engine.GET("/api/application/:app_id/version", handler.ListVersions)
	engine.POST("/api/application/:app_id/version", handler.CreateVersion)
	engine.GET("/api/version/:version_id", handler.GetVersion)
	engine.PUT("/api/version/:version_id", handler.UpdateVersion)
	engine.DELETE("/api/version/:version_id", handler.DeleteVersion)
	engine.POST("/api/version/:version_id/publish", handler.PublishVersion)
	engine.POST("/api/version/:version_id/fork", handler.ForkVersion)
	engine.POST("/api/version/:version_id/preview", deploymentHandler.PreviewVersion)
}

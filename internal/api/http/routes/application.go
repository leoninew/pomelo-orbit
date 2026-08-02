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
	engine.GET("/api/application/:app_id/export", handler.ExportApplication)
	engine.POST("/api/application/:app_id/stop", deploymentHandler.StopApplication)
	engine.POST("/api/application/:app_id/restart", deploymentHandler.RestartApplication)
	engine.GET("/api/application/:app_id/status", deploymentHandler.GetApplicationStatus)
	engine.GET("/api/application/:app_id/logs", deploymentHandler.GetApplicationLogs)
	engine.GET("/api/application/:app_id/service", serviceHandler.ListApplicationServices)
	engine.GET("/api/application/:app_id/version", handler.ListVersions)
	engine.POST("/api/application/:app_id/version", handler.CreateVersion)
	engine.GET("/api/version/:version_id", handler.GetVersion)
	engine.POST("/api/version/:version_id/preview", deploymentHandler.PreviewVersion)
	engine.PUT("/api/version/:version_id", handler.UpdateVersion)
	engine.DELETE("/api/version/:version_id", handler.DeleteVersion)
	engine.POST("/api/version/:version_id/component", handler.CreateVersionComponent)
	engine.GET("/api/version/:version_id/component/:component_id", handler.GetVersionComponent)
	engine.PUT("/api/version/:version_id/component/:component_id/basic", handler.UpdateVersionComponentBasic)
	engine.PUT("/api/version/:version_id/component/:component_id/runtime", handler.UpdateVersionComponentRuntime)
	engine.PUT("/api/version/:version_id/component/:component_id/endpoints", handler.UpdateVersionComponentEndpoints)
	engine.PUT("/api/version/:version_id/component/:component_id/env", handler.UpdateVersionComponentEnv)
	engine.PUT("/api/version/:version_id/component/:component_id/mounts", handler.UpdateVersionComponentMounts)
	engine.PUT("/api/version/:version_id/component/:component_id/dependencies", handler.UpdateVersionComponentDependencies)
	engine.PUT("/api/version/:version_id/component/:component_id/devices", handler.UpdateVersionComponentDevices)
	engine.PUT("/api/version/:version_id/component/:component_id/advanced", handler.UpdateVersionComponentAdvanced)
	engine.DELETE("/api/version/:version_id/component/:component_id", handler.DeleteVersionComponent)
	engine.POST("/api/version/:version_id/publish", handler.PublishVersion)
	engine.POST("/api/version/:version_id/unpublish", handler.UnpublishVersion)
	engine.POST("/api/version/:version_id/fork", handler.ForkVersion)
}

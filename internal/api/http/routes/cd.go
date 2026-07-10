package routes

import (
	"github.com/gin-gonic/gin"

	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
)

func (r Router) registerCD(engine *gin.Engine) {
	handler := cdhandler.New(r.logger, r.deps.CDService, r.deps.Authenticator)

	engine.GET("/api/cd/application", handler.ListApplications)
	engine.POST("/api/cd/application", handler.CreateApplication)
	engine.GET("/api/cd/application/:app_id", handler.GetApplication)
	engine.PUT("/api/cd/application/:app_id", handler.UpdateApplication)
	engine.DELETE("/api/cd/application/:app_id", handler.DeleteApplication)
	engine.POST("/api/cd/application/:app_id/deploy", handler.DeployApplication)

	engine.GET("/api/cd/deployment", handler.ListDeployments)
	engine.GET("/api/cd/deployment/:deployment_id", handler.GetDeployment)
	engine.GET("/api/cd/deployment/:deployment_id/logs", handler.GetDeploymentLogs)
	engine.GET("/api/cd/deployment/:deployment_id/container-logs", handler.GetDeploymentContainerLogs)
	engine.POST("/api/cd/deployment/:deployment_id/cancel", handler.CancelDeployment)

	engine.POST("/api/cd/application/import", handler.ImportApplication)
	engine.GET("/api/cd/application/:app_id/export", handler.ExportApplication)
	engine.POST("/api/cd/application/:app_id/compose-preview", handler.PreviewApplicationCompose)
	engine.GET("/api/cd/application/:app_id/files", handler.ListApplicationFiles)
	engine.POST("/api/cd/application/:app_id/file", handler.CreateApplicationFile)
	engine.GET("/api/cd/application/:app_id/file/:file_id", handler.ReadApplicationFile)
	engine.PUT("/api/cd/application/:app_id/file/:file_id", handler.UpdateApplicationFile)
	engine.DELETE("/api/cd/application/:app_id/file/:file_id", handler.DeleteApplicationFile)
	engine.POST("/api/cd/application/:app_id/stop", handler.StopApplication)
	engine.POST("/api/cd/application/:app_id/restart", handler.RestartApplication)
	engine.GET("/api/cd/application/:app_id/status", handler.GetApplicationStatus)
	engine.GET("/api/cd/application/:app_id/logs", handler.GetApplicationLogs)
	engine.GET("/api/cd/application/:app_id/route", handler.ListApplicationRoutes)
	engine.POST("/api/cd/application/:app_id/route", handler.CreateApplicationRoute)
	engine.PUT("/api/cd/application/:app_id/route/:route_id", handler.UpdateApplicationRoute)
	engine.DELETE("/api/cd/application/:app_id/route/:route_id", handler.DeleteApplicationRoute)
	engine.GET("/api/cd/application/:app_id/compose-service", handler.ListApplicationComposeServices)
	engine.GET("/api/cd/application/:app_id/service-config", handler.ListApplicationServiceConfigs)
	engine.PUT("/api/cd/application/:app_id/service-config/:service_name", handler.UpdateApplicationServiceConfig)

	engine.GET("/api/cd/route", handler.ListRoutes)
	engine.POST("/api/cd/route", handler.CreateRoute)
	engine.POST("/api/cd/route/sync", handler.SyncRoutes)
	engine.GET("/api/cd/route/:route_id", handler.GetRoute)
	engine.PUT("/api/cd/route/:route_id", handler.UpdateRoute)
	engine.DELETE("/api/cd/route/:route_id", handler.DeleteRoute)
	engine.POST("/api/cd/route/:route_id/enable", handler.EnableRoute)
	engine.POST("/api/cd/route/:route_id/disable", handler.DisableRoute)
	engine.POST("/api/cd/route/:route_id/cert", handler.UploadRouteCert)
	engine.DELETE("/api/cd/route/:route_id/https", handler.DisableRouteHTTPS)
	engine.POST("/api/cd/route/:route_id/letsencrypt", handler.EnableRouteLetsEncrypt)
	engine.POST("/api/cd/route/:route_id/mkcert", handler.EnableRouteMkcert)

	engine.GET("/api/cd/traefik-route/config", handler.GetTraefikRouteConfig)
	engine.GET("/api/cd/traefik-route", handler.ListTraefikRoutes)
}

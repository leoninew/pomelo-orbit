package routes

import (
	"github.com/gin-gonic/gin"

	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
)

func (r Router) registerApplicationExtra(engine *gin.Engine) {
	handler := cdhandler.New(r.logger, r.deps.CDService, r.deps.Authenticator)

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
}

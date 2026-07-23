package routes

import (
	"github.com/gin-gonic/gin"

	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
)

func (r Router) registerApplicationExtra(engine *gin.Engine) {
	handler := cdhandler.New(r.logger, r.deps.CDService, r.deps.Authenticator)

	engine.POST("/api/cd/application/import", handler.ImportApplication)
	engine.GET("/api/cd/application/:app_id/export", handler.ExportApplication)
	engine.POST("/api/cd/application/:app_id/stop", handler.StopApplication)
	engine.POST("/api/cd/application/:app_id/restart", handler.RestartApplication)
	engine.GET("/api/cd/application/:app_id/status", handler.GetApplicationStatus)
	engine.GET("/api/cd/application/:app_id/logs", handler.GetApplicationLogs)
	engine.GET("/api/cd/application/:app_id/service", handler.ListApplicationServices)
	engine.GET("/api/cd/application/:app_id/version", handler.ListVersions)
	engine.POST("/api/cd/application/:app_id/version", handler.CreateVersion)
	engine.GET("/api/cd/version/:version_id", handler.GetVersion)
	engine.PUT("/api/cd/version/:version_id", handler.UpdateVersion)
	engine.DELETE("/api/cd/version/:version_id", handler.DeleteVersion)
	engine.POST("/api/cd/version/:version_id/publish", handler.PublishVersion)
	engine.POST("/api/cd/version/:version_id/fork", handler.ForkVersion)
	engine.POST("/api/cd/version/:version_id/preview", handler.PreviewVersion)
}

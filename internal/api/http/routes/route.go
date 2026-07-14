package routes

import (
	"github.com/gin-gonic/gin"

	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
)

func (r Router) registerRoute(engine *gin.Engine) {
	handler := cdhandler.New(r.logger, r.deps.CDService, r.deps.Authenticator)

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
}

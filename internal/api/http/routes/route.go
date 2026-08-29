package routes

import (
	"github.com/gin-gonic/gin"

	routehandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/route"
)

func (r Router) registerRoute(engine *gin.Engine) {
	handler := routehandler.New(r.logger, r.deps.RouteService, r.deps.Authenticator)

	engine.GET("/api/route", handler.ListRoutes)
	engine.POST("/api/route", handler.CreateRoute)
	engine.POST("/api/route/sync/preview", handler.PreviewRouteSync)
	engine.POST("/api/route/sync/confirm", handler.ConfirmRouteSync)
	engine.GET("/api/route/:route_id", handler.GetRoute)
	engine.PUT("/api/route/:route_id", handler.UpdateRoute)
	engine.DELETE("/api/route/:route_id", handler.DeleteRoute)
	engine.POST("/api/route/:route_id/enable", handler.EnableRoute)
	engine.POST("/api/route/:route_id/disable", handler.DisableRoute)
	engine.POST("/api/route/:route_id/cert", handler.UploadRouteCert)
	engine.DELETE("/api/route/:route_id/https", handler.DisableRouteHTTPS)
	engine.POST("/api/route/:route_id/letsencrypt", handler.EnableRouteLetsEncrypt)
	engine.POST("/api/route/:route_id/mkcert", handler.EnableRouteMkcert)

	engine.GET("/api/route/traefik/config", handler.GetTraefikRouteConfig)
	engine.GET("/api/route/traefik", handler.ListTraefikRoutes)
}

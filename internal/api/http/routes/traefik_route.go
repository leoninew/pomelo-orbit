package routes

import (
	"github.com/gin-gonic/gin"

	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
)

func (r Router) registerTraefikRoute(engine *gin.Engine) {
	handler := cdhandler.New(r.logger, r.deps.CDService, r.deps.Authenticator)

	engine.GET("/api/cd/traefik-route/config", handler.GetTraefikRouteConfig)
	engine.GET("/api/cd/traefik-route", handler.ListTraefikRoutes)
}

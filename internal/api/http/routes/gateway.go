package routes

import (
	"github.com/gin-gonic/gin"

	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
)

func (r Router) registerGateway(engine *gin.Engine) {
	handler := cdhandler.New(r.logger, r.deps.CDService, r.deps.Authenticator)

	engine.GET("/api/cd/gateway", handler.ListGateways)
	engine.POST("/api/cd/gateway", handler.CreateGateway)
	engine.GET("/api/cd/gateway/:gateway_id", handler.GetGateway)
	engine.PUT("/api/cd/gateway/:gateway_id", handler.UpdateGateway)
	engine.DELETE("/api/cd/gateway/:gateway_id", handler.DeleteGateway)
}

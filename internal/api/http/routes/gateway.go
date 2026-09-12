package routes

import (
	"github.com/gin-gonic/gin"

	gatewayhandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/gateway"
)

func (r Router) registerGateway(engine *gin.Engine) {
	handler := gatewayhandler.New(r.logger, r.deps.GatewayService, r.deps.Authenticator)

	engine.GET("/api/gateway", handler.ListGateways)
	engine.GET("/api/gateway/:gateway_id", handler.GetGateway)
	engine.PUT("/api/gateway/:gateway_id", handler.UpdateGateway)
	engine.DELETE("/api/gateway/:gateway_id", handler.DeleteGateway)
}

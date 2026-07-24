package routes

import (
	"github.com/gin-gonic/gin"

	servicehandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/service"
)

func (r Router) registerService(engine *gin.Engine) {
	handler := servicehandler.New(r.logger, r.deps.ServiceService, r.deps.Authenticator)

	engine.GET("/api/service", handler.ListServices)
	engine.GET("/api/service/:service_id", handler.GetService)
}

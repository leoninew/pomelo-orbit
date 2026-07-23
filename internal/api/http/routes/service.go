package routes

import (
	"github.com/gin-gonic/gin"

	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
)

func (r Router) registerService(engine *gin.Engine) {
	handler := cdhandler.New(r.logger, r.deps.CDService, r.deps.Authenticator)

	engine.GET("/api/cd/service", handler.ListServices)
	engine.GET("/api/cd/service/:service_id", handler.GetService)
}

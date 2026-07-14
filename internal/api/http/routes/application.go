package routes

import (
	"github.com/gin-gonic/gin"

	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
)

func (r Router) registerApplication(engine *gin.Engine) {
	handler := cdhandler.New(r.logger, r.deps.CDService, r.deps.Authenticator)

	engine.GET("/api/cd/application", handler.ListApplications)
	engine.POST("/api/cd/application", handler.CreateApplication)
	engine.GET("/api/cd/application/:app_id", handler.GetApplication)
	engine.PUT("/api/cd/application/:app_id", handler.UpdateApplication)
	engine.DELETE("/api/cd/application/:app_id", handler.DeleteApplication)
	engine.POST("/api/cd/application/:app_id/deploy", handler.DeployApplication)
}

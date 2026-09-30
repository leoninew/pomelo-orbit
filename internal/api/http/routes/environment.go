package routes

import (
	"github.com/gin-gonic/gin"

	environmenthandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/environment"
)

func (r Router) registerEnvironment(engine *gin.Engine) {
	handler := environmenthandler.New(r.logger, r.deps.EnvironmentService, r.deps.Authenticator).
		WithTerminal(r.deps.EnvironmentTerminalService, r.deps.EnvironmentTerminalRunner, r.cfg.Server.CorsAllowedOrigins)
	engine.GET("/api/environment", handler.GetProjectEnvironment)
	engine.PUT("/api/environment", handler.UpdateProjectEnvironment)
	engine.POST("/api/environment/ssh-command", handler.PrepareProjectEnvironmentSSHCommand)
	engine.POST("/api/environment/probe", handler.ProbeProjectEnvironment)
	engine.POST("/api/environment/terminal/ticket", handler.IssueTerminalTicket)
	engine.GET("/api/environment/terminal", handler.ConnectTerminal)
}

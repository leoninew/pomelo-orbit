package routes

import (
	"github.com/gin-gonic/gin"
	clihandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/cli"
)

func (r Router) registerCLI(engine *gin.Engine) {
	h := clihandler.New(r.deps.Authenticator, r.deps.EnvironmentAccessService, r.deps.EnvironmentCommandService, r.deps.EnvironmentTerminalService, r.deps.EnvironmentTerminalRunner, r.logger)
	engine.GET("/api/cli/project", h.List)
	engine.GET("/api/cli/environment/terminal", h.Terminal)
	engine.POST("/api/cli/environment/exec", h.Exec)
}

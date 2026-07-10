package routes

import (
	"github.com/gin-gonic/gin"

	settingshandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/settings"
)

func (r Router) registerSettings(engine *gin.Engine) {
	handler := settingshandler.New(r.logger, r.deps.SettingsService, r.deps.Authenticator)
	engine.GET("/api/settings/config", handler.GetConfig)
	engine.PUT("/api/settings/config", handler.UpdateConfig)
	engine.DELETE("/api/settings/config", handler.ResetConfig)
}

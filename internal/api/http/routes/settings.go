package routes

import (
	"github.com/gin-gonic/gin"

	settingshandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/settings"
)

func (r Router) registerSettings(engine *gin.Engine) {
	handler := settingshandler.New(r.logger, r.deps.SettingsService, r.deps.Authenticator)
	engine.GET("/api/setting/config", handler.GetConfig)
	engine.PUT("/api/setting/config", handler.UpdateConfig)
	engine.DELETE("/api/setting/config", handler.ResetConfig)
}

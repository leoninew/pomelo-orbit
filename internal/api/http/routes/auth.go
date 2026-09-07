package routes

import (
	"github.com/gin-gonic/gin"

	authhandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/auth"
)

func (r Router) registerAuth(engine *gin.Engine) {
	handler := authhandler.New(r.logger, r.cfg.Turnstile, r.deps.AuthService, r.deps.Authenticator, r.deps.TurnstileVerifier)
	engine.GET("/api/auth/csrf-token", handler.GetCSRFToken)
	engine.GET("/api/auth/turnstile-config", handler.GetTurnstileConfig)
	engine.POST("/api/auth/login", handler.Login)
	engine.POST("/api/auth/logout", handler.Logout)

	engine.GET("/api/auth/me", handler.GetMe)
	engine.PUT("/api/auth/password", handler.ChangePassword)
	engine.GET("/api/auth/login-history", handler.ListLoginHistory)
	engine.GET("/api/auth/mcp-access-token", handler.ListMCPAccessTokens)
	engine.POST("/api/auth/mcp-access-token", handler.CreateMCPAccessToken)
	engine.DELETE("/api/auth/mcp-access-token/:token_id", handler.RevokeMCPAccessToken)
	engine.GET("/api/auth/google", handler.GoogleOAuth)
	engine.POST("/api/auth/google/callback", handler.GoogleCallback)
}

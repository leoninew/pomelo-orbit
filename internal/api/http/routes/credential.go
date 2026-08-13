package routes

import (
	"github.com/gin-gonic/gin"

	credentialhandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/credential"
)

func (r Router) registerCredential(engine *gin.Engine) {
	handler := credentialhandler.New(r.logger, r.deps.CredentialService, r.deps.Authenticator)

	engine.GET("/api/credential", handler.ListCredentials)
	engine.POST("/api/credential", handler.CreateCredential)
	engine.POST("/api/credential/import", handler.ImportCredential)
	engine.GET("/api/credential/:credential_id", handler.GetCredential)
	engine.PUT("/api/credential/:credential_id", handler.UpdateCredential)
	engine.DELETE("/api/credential/:credential_id", handler.DeleteCredential)
	engine.GET("/api/credential/:credential_id/export", handler.ExportCredential)
}

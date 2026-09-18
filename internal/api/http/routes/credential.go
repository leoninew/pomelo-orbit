package routes

import (
	"github.com/gin-gonic/gin"

	credentialhandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/credential"
)

func (r Router) registerCredential(engine *gin.Engine) {
	handler := credentialhandler.New(r.logger, r.deps.CredentialService, r.deps.Authenticator)

	engine.GET("/api/repository-credential", handler.ListCredentials)
	engine.POST("/api/repository-credential", handler.CreateCredential)
	engine.POST("/api/repository-credential/import", handler.ImportCredential)
	engine.GET("/api/repository-credential/:credential_id", handler.GetCredential)
	engine.PUT("/api/repository-credential/:credential_id", handler.UpdateCredential)
	engine.DELETE("/api/repository-credential/:credential_id", handler.DeleteCredential)
	engine.GET("/api/repository-credential/:credential_id/export", handler.ExportCredential)
}

package routes

import (
	"github.com/gin-gonic/gin"

	cihandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/ci"
)

func (r Router) registerCredential(engine *gin.Engine) {
	handler := cihandler.New(r.logger, r.deps.CIService, r.deps.Authenticator)

	engine.GET("/api/ci/credential", handler.ListCredentials)
	engine.POST("/api/ci/credential", handler.CreateCredential)
	engine.POST("/api/ci/credential/import", handler.ImportCredential)
	engine.GET("/api/ci/credential/:credential_id", handler.GetCredential)
	engine.PUT("/api/ci/credential/:credential_id", handler.UpdateCredential)
	engine.DELETE("/api/ci/credential/:credential_id", handler.DeleteCredential)
	engine.GET("/api/ci/credential/:credential_id/export", handler.ExportCredential)
}

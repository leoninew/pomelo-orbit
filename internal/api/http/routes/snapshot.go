package routes

import (
	"github.com/gin-gonic/gin"

	cihandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/ci"
)

func (r Router) registerSnapshot(engine *gin.Engine) {
	handler := cihandler.New(r.logger, r.deps.CIService, r.deps.Authenticator)

	engine.GET("/api/ci/snapshot/:snapshot_id", handler.GetPipelineSnapshot)
}

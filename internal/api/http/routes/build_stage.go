package routes

import (
	"github.com/gin-gonic/gin"

	cihandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/ci"
)

func (r Router) registerBuildStage(engine *gin.Engine) {
	handler := cihandler.New(r.logger, r.deps.CIService, r.deps.Authenticator)

	engine.GET("/api/ci/build-stage", handler.ListBuildStages)
	engine.POST("/api/ci/build-stage", handler.CreateBuildStage)
	engine.GET("/api/ci/build-stage/:stage_id", handler.GetBuildStage)
	engine.PUT("/api/ci/build-stage/:stage_id", handler.UpdateBuildStage)
	engine.DELETE("/api/ci/build-stage/:stage_id", handler.DeleteBuildStage)
	engine.POST("/api/ci/build-stage/:stage_id/duplicate", handler.DuplicateBuildStage)
}

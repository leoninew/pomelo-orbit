package routes

import (
	"github.com/gin-gonic/gin"

	cihandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/ci"
)

func (r Router) registerPipelineRun(engine *gin.Engine) {
	handler := cihandler.New(r.logger, r.deps.CIService, r.deps.Authenticator)

	engine.GET("/api/ci/repository/:repository_id/run", handler.ListRepositoryRuns)
	engine.POST("/api/ci/repository/:repository_id/trigger", handler.TriggerRepository)
	engine.GET("/api/ci/run", handler.ListPipelineRuns)
	engine.GET("/api/ci/run/:run_id", handler.GetPipelineRun)
	engine.GET("/api/ci/run/:run_id/artifacts", handler.ListPipelineRunArtifacts)
	engine.GET("/api/ci/run/:run_id/stages/:stage_run_id/log", handler.GetPipelineStageLog)
	engine.POST("/api/ci/run/:run_id/cancel", handler.CancelPipelineRun)
	engine.POST("/api/ci/run/:run_id/retry", handler.RetryPipelineRun)
}

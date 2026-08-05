package routes

import (
	"github.com/gin-gonic/gin"

	pipelinerunhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/pipeline_run"
)

func (r Router) registerPipelineRun(engine *gin.Engine) {
	handler := pipelinerunhandler.New(r.logger, r.deps.PipelineRunService, r.deps.Authenticator)

	engine.GET("/api/repository/:repository_id/pipeline-run", handler.ListRepositoryRuns)
	engine.POST("/api/repository/:repository_id/trigger", handler.TriggerRepository)
	engine.GET("/api/pipeline-run", handler.ListPipelineRuns)
	engine.GET("/api/pipeline-run/:run_id", handler.GetPipelineRun)
	engine.GET("/api/pipeline-run/:run_id/artifact", handler.ListPipelineRunArtifacts)
	engine.GET("/api/pipeline-run/:run_id/stage/:stage_run_id/log", handler.GetPipelineStageLog)
	engine.POST("/api/pipeline-run/:run_id/cancel", handler.CancelPipelineRun)
	engine.POST("/api/pipeline-run/:run_id/retry", handler.RetryPipelineRun)
	engine.GET("/api/pipeline-run/artifact", handler.ListArtifacts)
	engine.GET("/api/pipeline-run/artifact/:artifact_id", handler.GetArtifact)
}

package routes

import (
	"github.com/gin-gonic/gin"

	pipelinerunhandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/pipeline_run"
)

func (r Router) registerPipelineRun(engine *gin.Engine) {
	handler := pipelinerunhandler.New(r.logger, r.deps.PipelineRunService, r.deps.Authenticator)

	engine.GET("/api/pipeline-run", handler.ListPipelineRuns)
	engine.POST("/api/pipeline/:pipeline_id/trigger", handler.TriggerPipeline)
	engine.POST("/api/pipeline/:pipeline_id/variable-preview", handler.PreviewPipelineRunVariables)
	engine.GET("/api/pipeline-run/:run_id", handler.GetPipelineRun)
	engine.DELETE("/api/pipeline-run/:run_id", handler.DeletePipelineRun)
	engine.GET("/api/pipeline-run/:run_id/artifact", handler.ListPipelineRunArtifacts)
	engine.GET("/api/pipeline-run/:run_id/stage/:stage_run_id/log", handler.GetPipelineStageLog)
	engine.POST("/api/pipeline-run/:run_id/cancel", handler.CancelPipelineRun)
	engine.POST("/api/pipeline-run/:run_id/retry", handler.RetryPipelineRun)
	engine.GET("/api/pipeline-run/artifact", handler.ListArtifacts)
	engine.GET("/api/pipeline-run/artifact/:artifact_id", handler.GetArtifact)
}

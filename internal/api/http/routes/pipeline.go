package routes

import (
	"github.com/gin-gonic/gin"

	pipelinehandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/pipeline"
)

func (r Router) registerPipeline(engine *gin.Engine) {
	handler := pipelinehandler.New(r.logger, r.deps.PipelineService, r.deps.Authenticator)

	engine.GET("/api/pipeline", handler.ListPipelines)
	engine.POST("/api/pipeline", handler.CreatePipeline)
	engine.GET("/api/pipeline/:pipeline_id", handler.GetPipeline)
	engine.PUT("/api/pipeline/:pipeline_id", handler.UpdatePipeline)
	engine.DELETE("/api/pipeline/:pipeline_id", handler.DeletePipeline)
	engine.POST("/api/pipeline/:pipeline_id/instantiate", handler.InstantiatePipeline)
	engine.POST("/api/pipeline/:pipeline_id/stage", handler.CreatePipelineStage)
	engine.PUT("/api/pipeline/:pipeline_id/stage/:stage_id", handler.UpdatePipelineStage)
	engine.DELETE("/api/pipeline/:pipeline_id/stage/:stage_id", handler.DeletePipelineStage)
	engine.GET("/api/pipeline/snapshot/:snapshot_id", handler.GetPipelineSnapshot)
}

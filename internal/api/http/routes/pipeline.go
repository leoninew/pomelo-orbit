package routes

import (
	"github.com/gin-gonic/gin"

	pipelinehandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/pipeline"
)

func (r Router) registerPipeline(engine *gin.Engine) {
	handler := pipelinehandler.New(r.logger, r.deps.PipelineService, r.deps.Authenticator)

	engine.GET("/api/pipeline/template", handler.ListPipelineTemplates)
	engine.POST("/api/pipeline/template", handler.CreatePipelineTemplate)
	engine.POST("/api/pipeline/template/resolve-variables", handler.ResolvePipelineTemplateVariables)
	engine.GET("/api/pipeline/template/:template_id", handler.GetPipelineTemplate)
	engine.PUT("/api/pipeline/template/:template_id", handler.UpdatePipelineTemplate)
	engine.DELETE("/api/pipeline/template/:template_id", handler.DeletePipelineTemplate)
	engine.POST("/api/pipeline/template/:template_id/duplicate", handler.DuplicatePipelineTemplate)

	engine.GET("/api/pipeline/stage", handler.ListPipelineStages)
	engine.POST("/api/pipeline/stage", handler.CreatePipelineStage)
	engine.GET("/api/pipeline/stage/:stage_id", handler.GetPipelineStage)
	engine.PUT("/api/pipeline/stage/:stage_id", handler.UpdatePipelineStage)
	engine.DELETE("/api/pipeline/stage/:stage_id", handler.DeletePipelineStage)
	engine.POST("/api/pipeline/stage/:stage_id/duplicate", handler.DuplicatePipelineStage)

	engine.GET("/api/pipeline/snapshot/:snapshot_id", handler.GetPipelineSnapshot)
}

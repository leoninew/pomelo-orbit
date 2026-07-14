package routes

import (
	"github.com/gin-gonic/gin"

	cihandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/ci"
)

func (r Router) registerTemplate(engine *gin.Engine) {
	handler := cihandler.New(r.logger, r.deps.CIService, r.deps.Authenticator)

	engine.GET("/api/ci/template", handler.ListPipelineTemplates)
	engine.POST("/api/ci/template", handler.CreatePipelineTemplate)
	engine.POST("/api/ci/template/resolve-variables", handler.ResolvePipelineTemplateVariables)
	engine.GET("/api/ci/template/:template_id", handler.GetPipelineTemplate)
	engine.PUT("/api/ci/template/:template_id", handler.UpdatePipelineTemplate)
	engine.DELETE("/api/ci/template/:template_id", handler.DeletePipelineTemplate)
	engine.POST("/api/ci/template/:template_id/duplicate", handler.DuplicatePipelineTemplate)
}

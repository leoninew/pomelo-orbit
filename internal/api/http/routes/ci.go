package routes

import (
	"github.com/gin-gonic/gin"

	cihandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/ci"
)

func (r Router) registerCI(engine *gin.Engine) {
	handler := cihandler.New(r.logger, r.deps.CIService, r.deps.Authenticator)

	engine.GET("/api/ci/repository", handler.ListRepositories)
	engine.POST("/api/ci/repository", handler.CreateRepository)
	engine.GET("/api/ci/repository/:repository_id", handler.GetRepository)
	engine.PUT("/api/ci/repository/:repository_id", handler.UpdateRepository)
	engine.DELETE("/api/ci/repository/:repository_id", handler.DeleteRepository)
	engine.GET("/api/ci/repository/:repository_id/webhook", handler.ListRepositoryWebhooks)
	engine.POST("/api/ci/repository/:repository_id/webhook", handler.CreateRepositoryWebhook)
	engine.PUT("/api/ci/repository/:repository_id/webhook/:webhook_id", handler.UpdateRepositoryWebhook)
	engine.DELETE("/api/ci/repository/:repository_id/webhook/:webhook_id", handler.DeleteRepositoryWebhook)
	engine.GET("/api/ci/webhook/:webhook_id", handler.GetRepositoryWebhook)
	engine.POST("/api/ci/webhook/:webhook_id", handler.ReceiveRepositoryWebhook)

	engine.GET("/api/ci/template", handler.ListPipelineTemplates)
	engine.POST("/api/ci/template", handler.CreatePipelineTemplate)
	engine.POST("/api/ci/template/resolve-variables", handler.ResolvePipelineTemplateVariables)
	engine.GET("/api/ci/template/:template_id", handler.GetPipelineTemplate)
	engine.PUT("/api/ci/template/:template_id", handler.UpdatePipelineTemplate)
	engine.DELETE("/api/ci/template/:template_id", handler.DeletePipelineTemplate)
	engine.POST("/api/ci/template/:template_id/duplicate", handler.DuplicatePipelineTemplate)

	engine.GET("/api/ci/build-stage", handler.ListBuildStages)
	engine.POST("/api/ci/build-stage", handler.CreateBuildStage)
	engine.GET("/api/ci/build-stage/:stage_id", handler.GetBuildStage)
	engine.PUT("/api/ci/build-stage/:stage_id", handler.UpdateBuildStage)
	engine.DELETE("/api/ci/build-stage/:stage_id", handler.DeleteBuildStage)
	engine.POST("/api/ci/build-stage/:stage_id/duplicate", handler.DuplicateBuildStage)

	engine.GET("/api/ci/repository/:repository_id/run", handler.ListRepositoryRuns)
	engine.POST("/api/ci/repository/:repository_id/trigger", handler.TriggerRepository)
	engine.GET("/api/ci/run", handler.ListPipelineRuns)
	engine.GET("/api/ci/run/:run_id", handler.GetPipelineRun)
	engine.GET("/api/ci/run/:run_id/artifacts", handler.ListPipelineRunArtifacts)
	engine.GET("/api/ci/run/:run_id/stages/:stage_run_id/log", handler.GetPipelineStageLog)
	engine.POST("/api/ci/run/:run_id/cancel", handler.CancelPipelineRun)
	engine.POST("/api/ci/run/:run_id/retry", handler.RetryPipelineRun)

	engine.GET("/api/ci/snapshot/:snapshot_id", handler.GetPipelineSnapshot)
	engine.GET("/api/ci/artifact", handler.ListArtifacts)

	engine.GET("/api/ci/credential", handler.ListCredentials)
	engine.POST("/api/ci/credential", handler.CreateCredential)
	engine.POST("/api/ci/credential/import", handler.ImportCredential)
	engine.GET("/api/ci/credential/:credential_id", handler.GetCredential)
	engine.PUT("/api/ci/credential/:credential_id", handler.UpdateCredential)
	engine.DELETE("/api/ci/credential/:credential_id", handler.DeleteCredential)
	engine.GET("/api/ci/credential/:credential_id/export", handler.ExportCredential)
}

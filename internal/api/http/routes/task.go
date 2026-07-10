package routes

import (
	"github.com/gin-gonic/gin"

	taskhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/task"
)

func (r Router) registerTask(engine *gin.Engine) {
	handler := taskhandler.New(r.logger, r.deps.TaskService)
	engine.POST("/api/background/task", handler.CreateTask)
	engine.POST("/api/background/ci/pipeline-run/:run_id/execute", handler.EnqueueCIPipelineRun)
	engine.POST("/api/background/cd/application/:app_id/deploy/:deployment_id", handler.EnqueueCDApplicationDeploy)
	engine.POST("/api/background/cd/application/:app_id/restart/:deployment_id", handler.EnqueueCDApplicationRestart)
	engine.POST("/api/background/cd/application/:app_id/stop/:deployment_id", handler.EnqueueCDApplicationStop)
	engine.GET("/api/background/task/:id", handler.GetTask)
}

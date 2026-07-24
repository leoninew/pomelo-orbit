package routes

import (
	"github.com/gin-gonic/gin"

	taskhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/task"
)

func (r Router) registerTask(engine *gin.Engine) {
	handler := taskhandler.New(r.logger, r.deps.TaskService)
	engine.POST("/api/background/task", handler.CreateTask)
	engine.POST("/api/background/pipeline-run/:run_id/execute", handler.EnqueuePipelineRun)
	engine.POST("/api/background/application/:app_id/deployment/:deployment_id", handler.EnqueueDeployment)
	engine.POST("/api/background/application/:app_id/deployment/:deployment_id/restart", handler.EnqueueDeploymentRestart)
	engine.POST("/api/background/application/:app_id/deployment/:deployment_id/stop", handler.EnqueueDeploymentStop)
	engine.GET("/api/background/task/:id", handler.GetTask)
}

package taskhandler

import (
	"log/slog"
	"net/http"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	deploymentv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/deployment"
	pipelinerunv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/pipeline_run"
	taskv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/task"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
)

type Handler struct {
	logger  *slog.Logger
	service tasksvc.Service
}

func New(logger *slog.Logger, service tasksvc.Service) Handler {
	return Handler{logger: logger, service: service}
}

func (h Handler) CreateTask(c *gin.Context) {
	var req taskv1.CreateTaskReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	item, err := h.service.Create(c.Request.Context(), taskCreateInput(&req))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	resp := taskResponse(item)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) EnqueuePipelineRun(c *gin.Context) {
	var req pipelinerunv1.PipelineRunExecuteTaskReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	runId := strings.TrimSpace(c.Param("run_id"))
	if runId == "" {
		transportresponse.Error(c, http.StatusBadRequest, "run_id is required")
		return
	}
	h.enqueueTypedTask(c, status.TaskTypePipelineRunExecute, pipelineRunExecutePayload(runId))
}

func (h Handler) EnqueueDeployment(c *gin.Context) {
	var req deploymentv1.ApplicationDeployTaskReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	appId := strings.TrimSpace(c.Param("app_id"))
	deploymentId := strings.TrimSpace(c.Param("deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.Error(c, http.StatusBadRequest, "app_id and deployment_id are required")
		return
	}
	h.enqueueTypedTask(c, status.TaskTypeDeploymentDeploy, applicationDeploymentPayload(appId, deploymentId))
}

func (h Handler) EnqueueDeploymentRestart(c *gin.Context) {
	var req deploymentv1.ApplicationRestartTaskReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	appId := strings.TrimSpace(c.Param("app_id"))
	deploymentId := strings.TrimSpace(c.Param("deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.Error(c, http.StatusBadRequest, "app_id and deployment_id are required")
		return
	}
	h.enqueueTypedTask(c, status.TaskTypeDeploymentRestart, applicationDeploymentPayload(appId, deploymentId))
}

func (h Handler) EnqueueDeploymentStop(c *gin.Context) {
	var req deploymentv1.ApplicationStopTaskReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	appId := strings.TrimSpace(c.Param("app_id"))
	deploymentId := strings.TrimSpace(c.Param("deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.Error(c, http.StatusBadRequest, "app_id and deployment_id are required")
		return
	}
	h.enqueueTypedTask(c, status.TaskTypeDeploymentStop, applicationDeploymentPayload(appId, deploymentId))
}

func (h Handler) enqueueTypedTask(c *gin.Context, taskType string, payload any) {
	item, err := h.service.EnqueueTyped(c.Request.Context(), taskType, payload)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	resp := taskResponse(item)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetTask(c *gin.Context) {
	item, err := h.service.FindById(c.Request.Context(), c.Param("id"))
	if err != nil {
		transportresponse.Error(c, http.StatusNotFound, "Task not found")
		return
	}
	resp := taskResponse(item)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) writeServiceError(c *gin.Context, err error) {
	if apperror.IsKind(err, apperror.KindValidation) {
		transportresponse.Error(c, http.StatusBadRequest, err.Error())
		return
	}
	h.logger.Error("task service failed", "error", err)
	transportresponse.Error(c, http.StatusInternalServerError, "Failed to process task")
}

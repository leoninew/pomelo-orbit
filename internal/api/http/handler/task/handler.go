package taskhandler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	"google.golang.org/protobuf/types/known/structpb"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
)

type router interface {
	GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
}

type Handler struct {
	logger  *slog.Logger
	service tasksvc.Service
}

func New(logger *slog.Logger, service tasksvc.Service) Handler {
	return Handler{logger: logger, service: service}
}

func (h Handler) Register(r router) {
	r.POST("/api/background/task", h.createTask)
	r.POST("/api/background/ci/pipeline-run/:run_id/execute", h.enqueueCIPipelineRun)
	r.POST("/api/background/cd/application/:app_id/deploy/:deployment_id", h.enqueueCDApplicationDeploy)
	r.POST("/api/background/cd/application/:app_id/restart/:deployment_id", h.enqueueCDApplicationRestart)
	r.POST("/api/background/cd/application/:app_id/stop/:deployment_id", h.enqueueCDApplicationStop)
	r.GET("/api/background/task/:id", h.getTask)
}

func (h Handler) createTask(c *gin.Context) {
	var req pomeloorbit.CreateTaskReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}

	item, err := h.service.Create(c.Request.Context(), tasksvc.CreateInput{Id: req.Id, TaskType: req.TaskType, Payload: rawPayload(req.Payload), PayloadJSON: req.PayloadJson, MaxAttempts: int(req.MaxAttempts)})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	resp := taskResponse(item)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) enqueueCIPipelineRun(c *gin.Context) {
	var req pomeloorbit.PipelineRunExecuteTaskReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	runId := strings.TrimSpace(c.Param("run_id"))
	if runId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "run_id is required"})
		return
	}
	h.enqueueTypedTask(c, status.TaskTypeCIPipelineRunExecute, map[string]string{"pipeline_run_id": runId})
}

func (h Handler) enqueueCDApplicationDeploy(c *gin.Context) {
	var req pomeloorbit.ApplicationDeployTaskReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	appId := strings.TrimSpace(c.Param("app_id"))
	deploymentId := strings.TrimSpace(c.Param("deployment_id"))
	if appId == "" || deploymentId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "app_id and deployment_id are required"})
		return
	}
	h.enqueueTypedTask(c, status.TaskTypeCDApplicationDeploy, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (h Handler) enqueueCDApplicationRestart(c *gin.Context) {
	var req pomeloorbit.ApplicationRestartTaskReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	appId := strings.TrimSpace(c.Param("app_id"))
	deploymentId := strings.TrimSpace(c.Param("deployment_id"))
	if appId == "" || deploymentId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "app_id and deployment_id are required"})
		return
	}
	h.enqueueTypedTask(c, status.TaskTypeCDApplicationRestart, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (h Handler) enqueueCDApplicationStop(c *gin.Context) {
	var req pomeloorbit.ApplicationStopTaskReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	appId := strings.TrimSpace(c.Param("app_id"))
	deploymentId := strings.TrimSpace(c.Param("deployment_id"))
	if appId == "" || deploymentId == "" {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "app_id and deployment_id are required"})
		return
	}
	h.enqueueTypedTask(c, status.TaskTypeCDApplicationStop, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (h Handler) enqueueTypedTask(c *gin.Context, taskType string, payload any) {
	item, err := h.service.EnqueueTyped(c.Request.Context(), taskType, payload)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	resp := taskResponse(item)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getTask(c *gin.Context) {
	item, err := h.service.FindById(c.Request.Context(), c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"detail": "Task not found"})
		return
	}
	resp := taskResponse(item)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func taskResponse(item *taskrepo.Task) pomeloorbit.TaskResp {
	return pomeloorbit.TaskResp{Id: item.Id, TaskType: item.TaskType, PayloadJson: item.PayloadJSON, Status: item.Status, Attempts: int32(item.Attempts), MaxAttempts: int32(item.MaxAttempts), LockedBy: item.LockedBy, LockedAt: transportresponse.FormatOptionalTime(item.LockedAt), StartedAt: transportresponse.FormatOptionalTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), ErrorMessage: item.ErrorMessage, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func rawPayload(payload *structpb.Value) json.RawMessage {
	if payload == nil {
		return nil
	}
	data, err := transportcodec.MarshalProtoJSON(payload)
	if err != nil {
		return nil
	}
	return data
}

func (h Handler) writeServiceError(c *gin.Context, err error) {
	if apperror.IsKind(err, apperror.KindValidation) {
		c.JSON(http.StatusBadRequest, gin.H{"detail": err.Error()})
		return
	}
	h.logger.Error("task service failed", "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to process task"})
}

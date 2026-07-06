package taskhandler

import (
	"encoding/json"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"

	"gitee.com/leoninew/PomeloOrbit-go/internal/apperror"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/task"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/service/task"
	"gitee.com/leoninew/PomeloOrbit-go/internal/status"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/response"
)

type router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
}

type Handler struct {
	logger  *slog.Logger
	service tasksvc.Service
}

func New(logger *slog.Logger, service tasksvc.Service) Handler {
	return Handler{logger: logger, service: service}
}

func (h Handler) Register(r router) {
	r.Post("/api/background/task", h.createTask)
	r.Post("/api/background/ci/pipeline-run/{run_id}/execute", h.enqueueCIPipelineRun)
	r.Post("/api/background/cd/application/{app_id}/deploy/{deployment_id}", h.enqueueCDApplicationDeploy)
	r.Post("/api/background/cd/application/{app_id}/restart/{deployment_id}", h.enqueueCDApplicationRestart)
	r.Post("/api/background/cd/application/{app_id}/stop/{deployment_id}", h.enqueueCDApplicationStop)
	r.Get("/api/background/task/{id}", h.getTask)
}

func (h Handler) createTask(w http.ResponseWriter, r *http.Request) {
	var req pomeloorbit.CreateTaskReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	item, err := h.service.Create(r.Context(), tasksvc.CreateInput{Id: req.Id, TaskType: req.TaskType, Payload: rawPayload(req.Payload), PayloadJSON: req.PayloadJson, MaxAttempts: int(req.MaxAttempts)})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	resp := taskResponse(item)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) enqueueCIPipelineRun(w http.ResponseWriter, r *http.Request) {
	var req pomeloorbit.PipelineRunExecuteTaskReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	runId := strings.TrimSpace(chi.URLParam(r, "run_id"))
	if runId == "" {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "run_id is required")
		return
	}
	h.enqueueTypedTask(w, r, status.TaskTypeCIPipelineRunExecute, map[string]string{"pipeline_run_id": runId})
}

func (h Handler) enqueueCDApplicationDeploy(w http.ResponseWriter, r *http.Request) {
	var req pomeloorbit.ApplicationDeployTaskReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	appId := strings.TrimSpace(chi.URLParam(r, "app_id"))
	deploymentId := strings.TrimSpace(chi.URLParam(r, "deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "app_id and deployment_id are required")
		return
	}
	h.enqueueTypedTask(w, r, status.TaskTypeCDApplicationDeploy, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (h Handler) enqueueCDApplicationRestart(w http.ResponseWriter, r *http.Request) {
	var req pomeloorbit.ApplicationRestartTaskReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	appId := strings.TrimSpace(chi.URLParam(r, "app_id"))
	deploymentId := strings.TrimSpace(chi.URLParam(r, "deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "app_id and deployment_id are required")
		return
	}
	h.enqueueTypedTask(w, r, status.TaskTypeCDApplicationRestart, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (h Handler) enqueueCDApplicationStop(w http.ResponseWriter, r *http.Request) {
	var req pomeloorbit.ApplicationStopTaskReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	appId := strings.TrimSpace(chi.URLParam(r, "app_id"))
	deploymentId := strings.TrimSpace(chi.URLParam(r, "deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "app_id and deployment_id are required")
		return
	}
	h.enqueueTypedTask(w, r, status.TaskTypeCDApplicationStop, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (h Handler) enqueueTypedTask(w http.ResponseWriter, r *http.Request, taskType string, payload any) {
	item, err := h.service.EnqueueTyped(r.Context(), taskType, payload)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	resp := taskResponse(item)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) getTask(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.FindById(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		transportresponse.Error(h.logger, w, http.StatusNotFound, "Task not found")
		return
	}
	resp := taskResponse(item)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func taskResponse(item *taskrepo.Task) pomeloorbit.TaskResp {
	return pomeloorbit.TaskResp{Id: item.Id, TaskType: item.TaskType, PayloadJson: item.PayloadJSON, Status: item.Status, Attempts: int32(item.Attempts), MaxAttempts: int32(item.MaxAttempts), LockedBy: item.LockedBy, LockedAt: transportresponse.FormatOptionalTime(item.LockedAt), StartedAt: transportresponse.FormatOptionalTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), ErrorMessage: item.ErrorMessage, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func rawPayload(payload *structpb.Value) json.RawMessage {
	if payload == nil {
		return nil
	}
	data, err := protojson.MarshalOptions{UseProtoNames: true}.Marshal(payload)
	if err != nil {
		return nil
	}
	return data
}

func (h Handler) writeServiceError(w http.ResponseWriter, err error) {
	if apperror.IsKind(err, apperror.KindValidation) {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, err.Error())
		return
	}
	h.logger.Error("task service failed", "error", err)
	transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to process task")
}

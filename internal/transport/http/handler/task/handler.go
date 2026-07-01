package taskhandler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"backend/internal/apperror"
	taskrepo "backend/internal/repository/task"
	tasksvc "backend/internal/service/task"
	"backend/internal/status"
	transportresponse "backend/internal/transport/http/response"
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
	var req CreateTaskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}

	item, err := h.service.Create(r.Context(), tasksvc.CreateInput{Id: req.Id, TaskType: req.TaskType, Payload: req.Payload, PayloadJSON: req.PayloadJSON, MaxAttempts: req.MaxAttempts})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusCreated, taskResponse(item))
}

func (h Handler) enqueueCIPipelineRun(w http.ResponseWriter, r *http.Request) {
	runId := strings.TrimSpace(chi.URLParam(r, "run_id"))
	if runId == "" {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "run_id is required")
		return
	}
	h.enqueueTypedTask(w, r, status.TaskTypeCIPipelineRunExecute, map[string]string{"pipeline_run_id": runId})
}

func (h Handler) enqueueCDApplicationDeploy(w http.ResponseWriter, r *http.Request) {
	appId := strings.TrimSpace(chi.URLParam(r, "app_id"))
	deploymentId := strings.TrimSpace(chi.URLParam(r, "deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "app_id and deployment_id are required")
		return
	}
	h.enqueueTypedTask(w, r, status.TaskTypeCDApplicationDeploy, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (h Handler) enqueueCDApplicationRestart(w http.ResponseWriter, r *http.Request) {
	appId := strings.TrimSpace(chi.URLParam(r, "app_id"))
	deploymentId := strings.TrimSpace(chi.URLParam(r, "deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "app_id and deployment_id are required")
		return
	}
	h.enqueueTypedTask(w, r, status.TaskTypeCDApplicationRestart, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (h Handler) enqueueCDApplicationStop(w http.ResponseWriter, r *http.Request) {
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
	transportresponse.JSON(h.logger, w, http.StatusCreated, taskResponse(item))
}

func (h Handler) getTask(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.FindById(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		transportresponse.Error(h.logger, w, http.StatusNotFound, "Task not found")
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, taskResponse(item))
}

func taskResponse(item *taskrepo.Task) TaskResp {
	return TaskResp{Id: item.Id, TaskType: item.TaskType, PayloadJSON: item.PayloadJSON, Status: item.Status, Attempts: item.Attempts, MaxAttempts: item.MaxAttempts, LockedBy: item.LockedBy, LockedAt: transportresponse.FormatOptionalTime(item.LockedAt), StartedAt: transportresponse.FormatOptionalTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), ErrorMessage: item.ErrorMessage, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func (h Handler) writeServiceError(w http.ResponseWriter, err error) {
	if apperror.IsKind(err, apperror.KindValidation) {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, err.Error())
		return
	}
	h.logger.Error("task service failed", "error", err)
	transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to process task")
}

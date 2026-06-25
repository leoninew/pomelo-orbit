package taskhandler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"backend/internal/apperror"
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

type createTaskReq struct {
	Id          string          `json:"id"`
	TaskType    string          `json:"task_type"`
	Payload     json.RawMessage `json:"payload"`
	PayloadJSON string          `json:"payload_json"`
	MaxAttempts int             `json:"max_attempts"`
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
	var req createTaskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}

	item, err := h.service.Create(r.Context(), tasksvc.CreateInput{Id: req.Id, TaskType: req.TaskType, Payload: req.Payload, PayloadJSON: req.PayloadJSON, MaxAttempts: req.MaxAttempts})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusCreated, item)
}

func (h Handler) enqueueCIPipelineRun(w http.ResponseWriter, r *http.Request) {
	runId := strings.TrimSpace(chi.URLParam(r, "run_id"))
	if runId == "" {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "run_id is required"})
		return
	}
	h.enqueueTypedTask(w, r, status.TaskTypeCIPipelineRunExecute, map[string]string{"pipeline_run_id": runId})
}

func (h Handler) enqueueCDApplicationDeploy(w http.ResponseWriter, r *http.Request) {
	appId := strings.TrimSpace(chi.URLParam(r, "app_id"))
	deploymentId := strings.TrimSpace(chi.URLParam(r, "deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "app_id and deployment_id are required"})
		return
	}
	h.enqueueTypedTask(w, r, status.TaskTypeCDApplicationDeploy, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (h Handler) enqueueCDApplicationRestart(w http.ResponseWriter, r *http.Request) {
	appId := strings.TrimSpace(chi.URLParam(r, "app_id"))
	deploymentId := strings.TrimSpace(chi.URLParam(r, "deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "app_id and deployment_id are required"})
		return
	}
	h.enqueueTypedTask(w, r, status.TaskTypeCDApplicationRestart, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (h Handler) enqueueCDApplicationStop(w http.ResponseWriter, r *http.Request) {
	appId := strings.TrimSpace(chi.URLParam(r, "app_id"))
	deploymentId := strings.TrimSpace(chi.URLParam(r, "deployment_id"))
	if appId == "" || deploymentId == "" {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "app_id and deployment_id are required"})
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
	transportresponse.JSON(w, http.StatusCreated, item)
}

func (h Handler) getTask(w http.ResponseWriter, r *http.Request) {
	item, err := h.service.FindById(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		transportresponse.JSON(w, http.StatusNotFound, map[string]string{"detail": "Task not found"})
		return
	}
	transportresponse.JSON(w, http.StatusOK, item)
}

func (h Handler) writeServiceError(w http.ResponseWriter, err error) {
	if apperror.IsKind(err, apperror.KindValidation) {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": err.Error()})
		return
	}
	h.logger.Error("task service failed", "error", err)
	transportresponse.JSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to process task"})
}

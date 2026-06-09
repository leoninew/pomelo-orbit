package httpserver

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"backend/internal/config"
	"backend/internal/orbit"
	"backend/internal/status"
	"backend/internal/task"
)

type chiRouter interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
}

type Server struct {
	appCfg             config.Config
	cfg                config.ServerConfig
	logger             *slog.Logger
	store              orbit.Store
	tasks              task.Repository
	defaultMaxAttempts int
}

type createTaskReq struct {
	Id          string          `json:"id"`
	TaskType    string          `json:"task_type"`
	Payload     json.RawMessage `json:"payload"`
	PayloadJSON string          `json:"payload_json"`
	MaxAttempts int             `json:"max_attempts"`
}

func New(cfg config.Config, logger *slog.Logger, store orbit.Store, tasks task.Repository, defaultMaxAttempts int) Server {
	return Server{appCfg: cfg, cfg: cfg.Server, logger: logger, store: store, tasks: tasks, defaultMaxAttempts: defaultMaxAttempts}
}

func (s Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(middleware.Recoverer)
	r.Use(s.logRequest)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	s.registerAuthRoutes(r)
	s.registerProjectRoutes(r)
	s.registerDashboardRoutes(r)
	r.Post("/api/background/task", s.createTask)
	r.Post("/api/background/ci/pipeline-run/{run_id}/execute", s.enqueueCIPipelineRun)
	r.Post("/api/background/cd/application/{app_id}/deploy/{deployment_id}", s.enqueueCDApplicationDeploy)
	r.Post("/api/background/cd/application/{app_id}/restart/{deployment_id}", s.enqueueCDApplicationRestart)
	r.Get("/api/background/task/{id}", s.getTask)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
	})
	return r
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
}

func (s Server) createTask(w http.ResponseWriter, r *http.Request) {
	var req createTaskReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}

	req.TaskType = strings.TrimSpace(req.TaskType)
	if req.TaskType == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "task_type is required"})
		return
	}
	payloadJSON := strings.TrimSpace(req.PayloadJSON)
	if payloadJSON == "" && len(req.Payload) > 0 {
		payloadJSON = string(req.Payload)
	}
	if payloadJSON == "" {
		payloadJSON = "{}"
	}
	if !json.Valid([]byte(payloadJSON)) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "payload must be valid JSON"})
		return
	}

	id := strings.TrimSpace(req.Id)
	if id == "" {
		generated, err := newTaskId()
		if err != nil {
			s.logger.Error("generate task id failed", "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to generate task id"})
			return
		}
		id = generated
	}
	maxAttempts := req.MaxAttempts
	if maxAttempts == 0 {
		maxAttempts = s.defaultMaxAttempts
	}
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	if maxAttempts < 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "max_attempts must be at least 1"})
		return
	}

	if err := s.tasks.Enqueue(r.Context(), id, req.TaskType, payloadJSON, maxAttempts); err != nil {
		s.logger.Error("enqueue task failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to enqueue task"})
		return
	}
	item, err := s.tasks.FindById(r.Context(), id)
	if err != nil {
		s.logger.Error("read enqueued task failed", "task_id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to read task"})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s Server) enqueueCIPipelineRun(w http.ResponseWriter, r *http.Request) {
	runId := strings.TrimSpace(chi.URLParam(r, "run_id"))
	if runId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "run_id is required"})
		return
	}
	s.enqueueTypedTask(w, r, status.TaskTypeCIPipelineRunExecute, map[string]string{"pipeline_run_id": runId})
}

func (s Server) enqueueCDApplicationDeploy(w http.ResponseWriter, r *http.Request) {
	appId := strings.TrimSpace(chi.URLParam(r, "app_id"))
	deploymentId := strings.TrimSpace(chi.URLParam(r, "deployment_id"))
	if appId == "" || deploymentId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "app_id and deployment_id are required"})
		return
	}
	s.enqueueTypedTask(w, r, status.TaskTypeCDApplicationDeploy, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (s Server) enqueueCDApplicationRestart(w http.ResponseWriter, r *http.Request) {
	appId := strings.TrimSpace(chi.URLParam(r, "app_id"))
	deploymentId := strings.TrimSpace(chi.URLParam(r, "deployment_id"))
	if appId == "" || deploymentId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "app_id and deployment_id are required"})
		return
	}
	s.enqueueTypedTask(w, r, status.TaskTypeCDApplicationRestart, map[string]string{"application_id": appId, "deployment_id": deploymentId})
}

func (s Server) enqueueTypedTask(w http.ResponseWriter, r *http.Request, taskType string, payload any) {
	id, err := newTaskId()
	if err != nil {
		s.logger.Error("generate task id failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to generate task id"})
		return
	}
	payloadJSON, err := json.Marshal(payload)
	if err != nil {
		s.logger.Error("marshal task payload failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to marshal task payload"})
		return
	}
	maxAttempts := s.defaultMaxAttempts
	if maxAttempts == 0 {
		maxAttempts = 3
	}
	if err := s.tasks.Enqueue(r.Context(), id, taskType, string(payloadJSON), maxAttempts); err != nil {
		s.logger.Error("enqueue task failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to enqueue task"})
		return
	}
	item, err := s.tasks.FindById(r.Context(), id)
	if err != nil {
		s.logger.Error("read enqueued task failed", "task_id", id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to read task"})
		return
	}
	writeJSON(w, http.StatusCreated, item)
}

func (s Server) getTask(w http.ResponseWriter, r *http.Request) {
	item, err := s.tasks.FindById(r.Context(), chi.URLParam(r, "id"))
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Task not found"})
		return
	}
	writeJSON(w, http.StatusOK, item)
}

func (s Server) logRequest(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startedAt := time.Now()
		next.ServeHTTP(w, r)
		s.logger.Info("request completed", "method", r.Method, "path", r.URL.Path, "duration_ms", time.Since(startedAt).Milliseconds())
	})
}

func newTaskId() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return "task_" + hex.EncodeToString(b[:]), nil
}

func urlParam(r *http.Request, key string) string {
	return strings.TrimSpace(chi.URLParam(r, key))
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	if err := json.NewEncoder(w).Encode(value); err != nil && !errors.Is(err, http.ErrHandlerTimeout) {
		slog.Error("write response failed", "error", err)
	}
}

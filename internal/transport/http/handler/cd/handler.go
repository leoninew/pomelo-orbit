package cdhandler

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/apperror"
	"backend/internal/repository"
	"backend/internal/repository/model"
	cdsvc "backend/internal/service/cd"
	"backend/internal/transport/http/handler/authz"
	transportresponse "backend/internal/transport/http/response"
)

type router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
}

type Handler struct {
	logger        *slog.Logger
	service       cdsvc.Service
	authenticator authz.Authenticator
}

type ApplicationResp struct {
	Id              string  `json:"id"`
	ProjectId       *string `json:"project_id"`
	Name            string  `json:"name"`
	Code            string  `json:"code"`
	ImagePullPolicy string  `json:"image_pull_policy"`
	Status          string  `json:"status"`
	RouteManaged    bool    `json:"route_managed"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type ApplicationCreateReq struct {
	Name            string `json:"name"`
	Code            string `json:"code"`
	ImagePullPolicy string `json:"image_pull_policy"`
	RouteManaged    bool   `json:"route_managed"`
}

type ApplicationUpdateReq struct {
	Name            *string `json:"name"`
	Code            *string `json:"code"`
	ImagePullPolicy *string `json:"image_pull_policy"`
	RouteManaged    *bool   `json:"route_managed"`
}

type DeploymentResp struct {
	Id                       string  `json:"id"`
	ProjectId                *string `json:"project_id"`
	ApplicationId            *string `json:"application_id"`
	ApplicationName          string  `json:"application_name"`
	OperationType            string  `json:"operation_type"`
	TriggerType              string  `json:"trigger_type"`
	Status                   string  `json:"status"`
	StartedAt                string  `json:"started_at"`
	FinishedAt               *string `json:"finished_at"`
	DurationMs               *int    `json:"duration_ms"`
	LogText                  *string `json:"log_text"`
	ErrorMessage             *string `json:"error_message"`
	IsRollback               bool    `json:"is_rollback"`
	RollbackFromDeploymentId *string `json:"rollback_from_deployment_id"`
}

func New(logger *slog.Logger, service cdsvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) RegisterApplicationRoutes(r router) {
	r.Get("/api/cd/application", h.listApplications)
	r.Post("/api/cd/application", h.createApplication)
	r.Get("/api/cd/application/{app_id}", h.getApplication)
	r.Put("/api/cd/application/{app_id}", h.updateApplication)
	r.Delete("/api/cd/application/{app_id}", h.deleteApplication)
	r.Post("/api/cd/application/{app_id}/deploy", h.deployApplication)
}

func (h Handler) RegisterDeploymentRoutes(r router) {
	r.Get("/api/cd/deployment", h.listDeployments)
	r.Get("/api/cd/deployment/{deployment_id}", h.getDeployment)
	r.Get("/api/cd/deployment/{deployment_id}/logs", h.getDeploymentLogs)
	r.Get("/api/cd/deployment/{deployment_id}/stream-log", h.streamDeploymentLog)
	r.Post("/api/cd/deployment/{deployment_id}/cancel", h.cancelDeployment)
}

func (h Handler) listApplications(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListApplications(r.Context(), current.Id, transportresponse.QueryProjectId(r.URL.Query().Get("project_id")), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, transportresponse.NewPaginatedResp(mapPage(items, applicationResponse)))
}

func (h Handler) createApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req ApplicationCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(h.logger, w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	app, err := h.service.CreateApplication(r.Context(), current.Id, cdsvc.ApplicationCreateInput{ProjectId: r.URL.Query().Get("project_id"), Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, RouteManaged: req.RouteManaged})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusCreated, applicationResponse(app))
}

func (h Handler) getApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	app, err := h.service.ApplicationForUser(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, applicationResponse(app))
}

func (h Handler) updateApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req ApplicationUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(h.logger, w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	app, err := h.service.UpdateApplication(r.Context(), current.Id, chi.URLParam(r, "app_id"), cdsvc.ApplicationUpdateInput{Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, RouteManaged: req.RouteManaged})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, applicationResponse(app))
}

func (h Handler) deleteApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteApplication(r.Context(), current.Id, chi.URLParam(r, "app_id"), cdsvc.ApplicationDeleteInput{RemoveDir: r.URL.Query().Get("remove_dir") == "true"}); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) deployApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	deploymentId, err := h.service.DeployApplication(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, map[string]string{"deployment_id": deploymentId})
}

func (h Handler) listDeployments(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListDeployments(r.Context(), current.Id, cdsvc.DeploymentListInput{ProjectId: r.URL.Query().Get("project_id"), ApplicationId: r.URL.Query().Get("application_id"), Status: r.URL.Query().Get("status"), Search: r.URL.Query().Get("search"), DateFrom: r.URL.Query().Get("date_from"), DateTo: r.URL.Query().Get("date_to"), Page: page, PerPage: perPage})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, transportresponse.NewPaginatedResp(mapPage(items, deploymentResponse)))
}

func (h Handler) getDeployment(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	deployment, err := h.service.DeploymentForUser(r.Context(), current.Id, chi.URLParam(r, "deployment_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, deploymentResponse(deployment))
}

func (h Handler) getDeploymentLogs(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	log, err := h.service.DeploymentLog(r.Context(), current.Id, chi.URLParam(r, "deployment_id"), transportresponse.QueryInt(r.URL.Query().Get("offset"), 0))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, map[string]any{"logs": log.Logs, "offset": log.Offset, "is_complete": log.IsComplete, "status": log.Status})
}

func (h Handler) streamDeploymentLog(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		transportresponse.JSON(h.logger, w, http.StatusInternalServerError, map[string]string{"detail": "Streaming is not supported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")
	log, err := h.service.DeploymentLog(r.Context(), current.Id, chi.URLParam(r, "deployment_id"), 0)
	if err != nil {
		h.writeError(w, err)
		return
	}
	if log.Logs != "" {
		data, err := json.Marshal(map[string]any{"logs": log.Logs, "offset": log.Offset})
		if err != nil {
			h.logger.Error("marshal deployment log event failed", "deployment_id", chi.URLParam(r, "deployment_id"), "error", err)
			return
		}
		_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
		flusher.Flush()
	}
	if log.IsComplete {
		_, _ = w.Write([]byte("event: complete\ndata: {}\n\n"))
		flusher.Flush()
	}
}

func (h Handler) cancelDeployment(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	deployment, err := h.service.CancelDeployment(r.Context(), current.Id, chi.URLParam(r, "deployment_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, deploymentResponse(deployment))
}

func (h Handler) writeError(w http.ResponseWriter, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("cd request failed", "error", err)
	}
	transportresponse.JSON(h.logger, w, apperror.StatusCode(err), map[string]string{"detail": err.Error()})
}

func applicationResponse(item model.Application) ApplicationResp {
	return ApplicationResponse(item)
}

func deploymentResponse(item model.Deployment) DeploymentResp {
	return DeploymentResponse(item)
}

func ApplicationResponse(item model.Application) ApplicationResp {
	return ApplicationResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, ImagePullPolicy: item.ImagePullPolicy, Status: item.Status, RouteManaged: item.RouteManaged, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func DeploymentResponse(item model.Deployment) DeploymentResp {
	return DeploymentResp{Id: item.Id, ProjectId: item.ProjectId, ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, OperationType: item.OperationType, TriggerType: item.TriggerType, Status: item.Status, StartedAt: transportresponse.FormatTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), DurationMs: item.DurationMs, LogText: item.LogText, ErrorMessage: item.ErrorMessage, IsRollback: item.IsRollback, RollbackFromDeploymentId: item.RollbackFromDeploymentId}
}

func mapPage[T any, U any](page repository.Page[T], convert func(T) U) repository.Page[U] {
	items := make([]U, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, convert(item))
	}
	return repository.Page[U]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}
}

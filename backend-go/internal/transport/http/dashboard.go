package transporthttp

import (
	"database/sql"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"backend/internal/repository"
)

var applicationCreateCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)
var applicationUpdateCodePattern = regexp.MustCompile(`^[a-z0-9-]+$`)

type applicationResp struct {
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

type applicationCreateReq struct {
	Name            string `json:"name"`
	Code            string `json:"code"`
	ImagePullPolicy string `json:"image_pull_policy"`
}

type applicationUpdateReq struct {
	Name            *string `json:"name"`
	Code            *string `json:"code"`
	ImagePullPolicy *string `json:"image_pull_policy"`
	RouteManaged    *bool   `json:"route_managed"`
}

type configFileReq struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type configFileResp struct {
	Id        string `json:"id"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

type applicationRouteReq struct {
	ServiceName string `json:"service_name"`
	Domain      string `json:"domain"`
	Port        int    `json:"port"`
}

type applicationRouteResp struct {
	Id          string `json:"id"`
	ServiceName string `json:"service_name"`
	Domain      string `json:"domain"`
	Port        int    `json:"port"`
	CreatedAt   string `json:"created_at"`
	UpdatedAt   string `json:"updated_at"`
}

type applicationServiceConfigUpdateReq struct {
	Image *string `json:"image"`
}

type applicationServiceConfigResp struct {
	ServiceName   string  `json:"service_name"`
	DefaultDomain string  `json:"default_domain"`
	DefaultPort   int     `json:"default_port"`
	BaseImage     *string `json:"base_image"`
	Image         *string `json:"image"`
	ConfigId      *string `json:"config_id"`
	CreatedAt     *string `json:"created_at"`
	UpdatedAt     *string `json:"updated_at"`
}

type composeServiceResp struct {
	ServiceName   string `json:"service_name"`
	DefaultDomain string `json:"default_domain"`
	DefaultPort   int    `json:"default_port"`
}

type applicationExportResp struct {
	Version         string                              `json:"version"`
	Name            string                              `json:"name"`
	Code            string                              `json:"code"`
	ImagePullPolicy string                              `json:"image_pull_policy"`
	RouteManaged    bool                                `json:"route_managed"`
	ConfigFiles     []configFileReq                     `json:"config_files"`
	ServiceConfigs  []applicationServiceConfigImportReq `json:"service_configs"`
	Routes          []applicationRouteReq               `json:"routes"`
}

type applicationImportReq struct {
	Version         string                              `json:"version"`
	Name            string                              `json:"name"`
	Code            string                              `json:"code"`
	ImagePullPolicy string                              `json:"image_pull_policy"`
	RouteManaged    bool                                `json:"route_managed"`
	ConfigFiles     []configFileReq                     `json:"config_files"`
	ServiceConfigs  []applicationServiceConfigImportReq `json:"service_configs"`
	Routes          []applicationRouteReq               `json:"routes"`
}

type applicationServiceConfigImportReq struct {
	ServiceName string  `json:"service_name"`
	Image       *string `json:"image"`
	Environment *string `json:"environment"`
	Volumes     *string `json:"volumes"`
}

type deploymentResp struct {
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

func (s Server) registerDashboardRoutes(r chiRouter) {
	r.Get("/api/ci/credential", s.listCredentials)
	r.Post("/api/ci/credential", s.createCredential)
	r.Post("/api/ci/credential/import", s.importCredential)
	r.Get("/api/ci/credential/{credential_id}", s.getCredential)
	r.Put("/api/ci/credential/{credential_id}", s.updateCredential)
	r.Delete("/api/ci/credential/{credential_id}", s.deleteCredential)
	r.Get("/api/ci/credential/{credential_id}/export", s.exportCredential)
	r.Get("/api/cd/route", s.listRoutes)
	r.Post("/api/cd/route", s.createRoute)
	r.Post("/api/cd/route/sync", s.syncRoutes)
	r.Get("/api/cd/route/{route_id}", s.getRoute)
	r.Put("/api/cd/route/{route_id}", s.updateRoute)
	r.Delete("/api/cd/route/{route_id}", s.deleteRoute)
	r.Post("/api/cd/route/{route_id}/enable", s.enableRoute)
	r.Post("/api/cd/route/{route_id}/disable", s.disableRoute)
	r.Post("/api/cd/route/{route_id}/cert", s.uploadRouteCert)
	r.Delete("/api/cd/route/{route_id}/https", s.disableRouteHTTPS)
	r.Post("/api/cd/route/{route_id}/letsencrypt", s.enableRouteLetsEncrypt)
	r.Post("/api/cd/route/{route_id}/mkcert", s.enableRouteMkcert)
	r.Get("/api/cd/traefik-route/config", s.getTraefikRouteConfig)
	r.Get("/api/cd/traefik-route", s.listTraefikRoutes)
	r.Get("/api/cd/application", s.listApplications)
	r.Post("/api/cd/application", s.createApplication)
	r.Post("/api/cd/application/import", s.importApplication)
	r.Get("/api/cd/application/{app_id}", s.getApplication)
	r.Get("/api/cd/application/{app_id}/export", s.exportApplication)
	r.Put("/api/cd/application/{app_id}", s.updateApplication)
	r.Delete("/api/cd/application/{app_id}", s.deleteApplication)
	r.Post("/api/cd/application/{app_id}/compose-preview", s.previewApplicationCompose)
	r.Get("/api/cd/application/{app_id}/files", s.listApplicationFiles)
	r.Post("/api/cd/application/{app_id}/file", s.createApplicationFile)
	r.Get("/api/cd/application/{app_id}/file/{file_id}", s.readApplicationFile)
	r.Put("/api/cd/application/{app_id}/file/{file_id}", s.updateApplicationFile)
	r.Delete("/api/cd/application/{app_id}/file/{file_id}", s.deleteApplicationFile)
	r.Post("/api/cd/application/{app_id}/deploy", s.deployApplication)
	r.Post("/api/cd/application/{app_id}/stop", s.stopApplication)
	r.Post("/api/cd/application/{app_id}/restart", s.restartApplication)
	r.Get("/api/cd/application/{app_id}/status", s.getApplicationStatus)
	r.Get("/api/cd/application/{app_id}/logs", s.getApplicationLogs)
	r.Get("/api/cd/application/{app_id}/route", s.listApplicationRoutes)
	r.Post("/api/cd/application/{app_id}/route", s.createApplicationRoute)
	r.Put("/api/cd/application/{app_id}/route/{route_id}", s.updateApplicationRoute)
	r.Delete("/api/cd/application/{app_id}/route/{route_id}", s.deleteApplicationRoute)
	r.Get("/api/cd/application/{app_id}/compose-service", s.listApplicationComposeServices)
	r.Get("/api/cd/application/{app_id}/service-config", s.listApplicationServiceConfigs)
	r.Put("/api/cd/application/{app_id}/service-config/{service_name}", s.updateApplicationServiceConfig)
	r.Get("/api/cd/deployment", s.listDeployments)
	r.Get("/api/cd/deployment/{deployment_id}", s.getDeployment)
	r.Get("/api/cd/deployment/{deployment_id}/logs", s.getDeploymentLogs)
	r.Get("/api/cd/deployment/{deployment_id}/stream-log", s.streamDeploymentLog)
	r.Post("/api/cd/deployment/{deployment_id}/cancel", s.cancelDeployment)
}

func (s Server) listApplications(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := queryProjectId(r.URL.Query().Get("project_id"))
	if projectId != nil {
		if !s.ensureProjectMembership(w, r, *projectId, current.Id) {
			return
		}
	}
	page, perPage := pageParams(r)
	items, err := s.store.ListApplications(r.Context(), projectId, page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list applications failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list applications"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, applicationResponse)))
}

func (s Server) createApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}
	var req applicationCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeApplicationCreateReq(w, &req) || !s.ensureApplicationNameAvailable(w, r, req.Name) {
		return
	}
	app := repository.Application{Id: repository.NewId(), ProjectId: &projectId, Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, Status: repository.ApplicationStatusUndeployed}
	if err := s.store.CreateApplication(r.Context(), app); err != nil {
		s.logger.Error("create application failed", "application_code", app.Code, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create application"})
		return
	}
	created, err := s.store.Application(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("load created application failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application"})
		return
	}
	writeJSON(w, http.StatusCreated, applicationResponse(created))
}

func (s Server) getApplication(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, applicationResponse(app))
}

func (s Server) updateApplication(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	var req applicationUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" || len(name) > 100 {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid application fields"})
			return
		}
		app.Name = name
	}
	if req.Code != nil {
		code := strings.TrimSpace(*req.Code)
		if code == "" || len(code) > 100 || !applicationUpdateCodePattern.MatchString(code) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid application fields"})
			return
		}
		app.Code = code
	}
	if req.ImagePullPolicy != nil {
		policy := strings.TrimSpace(*req.ImagePullPolicy)
		if !validImagePullPolicy(policy) {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid application fields"})
			return
		}
		app.ImagePullPolicy = policy
	}
	if req.RouteManaged != nil {
		app.RouteManaged = *req.RouteManaged
	}
	if err := s.store.UpdateApplication(r.Context(), app); err != nil {
		s.logger.Error("update application failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update application"})
		return
	}
	updated, err := s.store.Application(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("load updated application failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application"})
		return
	}
	writeJSON(w, http.StatusOK, applicationResponse(updated))
}

func (s Server) deleteApplication(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	if app.Status == repository.ApplicationStatusDeploying {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "应用正在部署中, 请稍后再试"})
		return
	}
	if app.Status == repository.ApplicationStatusDeployed {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "应用正在运行中, 请先停止后再删除"})
		return
	}
	if r.URL.Query().Get("remove_dir") == "true" {
		if err := os.RemoveAll(filepath.Join(s.appCfg.DataRoot(), "cd", app.Code)); err != nil {
			s.logger.Error("remove application directory failed", "application_id", app.Id, "application_code", app.Code, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to remove application directory"})
			return
		}
	}
	if err := s.store.DeleteApplication(r.Context(), app.Id); err != nil {
		s.logger.Error("delete application failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete application"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) listDeployments(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}
	dateFrom, ok := parseOptionalRunTime(w, r.URL.Query().Get("date_from"), "date_from")
	if !ok {
		return
	}
	dateTo, ok := parseOptionalRunTime(w, r.URL.Query().Get("date_to"), "date_to")
	if !ok {
		return
	}
	page, perPage := pageParams(r)
	items, err := s.store.ListDeployments(r.Context(), projectId, r.URL.Query().Get("application_id"), r.URL.Query().Get("status"), r.URL.Query().Get("search"), dateFrom, dateTo, page, perPage)
	if err != nil {
		s.logger.Error("list deployments failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list deployments"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, deploymentResponse)))
}

func (s Server) getDeployment(w http.ResponseWriter, r *http.Request) {
	deployment, ok := s.loadDeploymentForCurrentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, deploymentResponse(deployment))
}

func (s Server) getDeploymentLogs(w http.ResponseWriter, r *http.Request) {
	deployment, ok := s.loadDeploymentForCurrentUser(w, r)
	if !ok {
		return
	}
	offset := queryInt(r.URL.Query().Get("offset"), 0)
	if offset < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "offset must be greater than or equal to 0"})
		return
	}
	logs, newOffset, ok := s.readDeploymentLog(w, r, deployment, offset)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"logs": logs, "offset": newOffset, "is_complete": deploymentStatusComplete(deployment.Status), "status": deployment.Status})
}

func (s Server) streamDeploymentLog(w http.ResponseWriter, r *http.Request) {
	deployment, ok := s.loadDeploymentForCurrentUser(w, r)
	if !ok {
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Streaming is not supported"})
		return
	}
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("X-Accel-Buffering", "no")

	offset := 0
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()
	for {
		logs, newOffset, ok := s.readDeploymentLog(w, r, deployment, offset)
		if !ok {
			return
		}
		if logs != "" {
			data, err := json.Marshal(map[string]any{"logs": logs, "offset": newOffset})
			if err != nil {
				s.logger.Error("marshal deployment log event failed", "deployment_id", deployment.Id, "error", err)
				return
			}
			_, _ = w.Write([]byte("data: " + string(data) + "\n\n"))
			flusher.Flush()
			offset = newOffset
		}
		if deploymentStatusComplete(deployment.Status) {
			_, _ = w.Write([]byte("event: complete\ndata: {}\n\n"))
			flusher.Flush()
			return
		}
		select {
		case <-r.Context().Done():
			return
		case <-ticker.C:
			updated, err := s.store.Deployment(r.Context(), deployment.Id)
			if err != nil {
				s.logger.Error("load streaming deployment failed", "deployment_id", deployment.Id, "error", err)
				return
			}
			deployment = updated
		}
	}
}

func (s Server) cancelDeployment(w http.ResponseWriter, r *http.Request) {
	deployment, ok := s.loadDeploymentForCurrentUser(w, r)
	if !ok {
		return
	}
	if deployment.Status != repository.WorkStatusWaitingToRun && deployment.Status != repository.WorkStatusRunning {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Cannot cancel deployment with status " + deployment.Status})
		return
	}
	if err := s.store.CancelDeployment(r.Context(), deployment.Id); err != nil {
		s.logger.Error("cancel deployment failed", "deployment_id", deployment.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to cancel deployment"})
		return
	}
	updated, err := s.store.Deployment(r.Context(), deployment.Id)
	if err != nil {
		s.logger.Error("load canceled deployment failed", "deployment_id", deployment.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load deployment"})
		return
	}
	writeJSON(w, http.StatusOK, deploymentResponse(updated))
}

func pageParams(r *http.Request) (int, int) {
	return queryInt(r.URL.Query().Get("page"), 1), queryInt(r.URL.Query().Get("per_page"), 10)
}

func mapPage[T any, U any](page repository.Page[T], convert func(T) U) repository.Page[U] {
	items := make([]U, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, convert(item))
	}
	return repository.Page[U]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}
}

func (s Server) loadDeploymentForCurrentUser(w http.ResponseWriter, r *http.Request) (repository.Deployment, bool) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return repository.Deployment{}, false
	}
	deploymentId := urlParam(r, "deployment_id")
	deployment, err := s.store.Deployment(r.Context(), deploymentId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Deployment " + deploymentId + " not found"})
			return repository.Deployment{}, false
		}
		s.logger.Error("load deployment failed", "deployment_id", deploymentId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load deployment"})
		return repository.Deployment{}, false
	}
	if deployment.ProjectId != nil {
		if !s.ensureProjectMembership(w, r, *deployment.ProjectId, current.Id) {
			return repository.Deployment{}, false
		}
		return deployment, true
	}
	if deployment.ApplicationId != nil {
		app, err := s.store.Application(r.Context(), *deployment.ApplicationId)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Application " + *deployment.ApplicationId + " not found"})
				return repository.Deployment{}, false
			}
			s.logger.Error("load deployment application failed", "deployment_id", deployment.Id, "application_id", *deployment.ApplicationId, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application"})
			return repository.Deployment{}, false
		}
		if app.ProjectId != nil && !s.ensureProjectMembership(w, r, *app.ProjectId, current.Id) {
			return repository.Deployment{}, false
		}
	}
	return deployment, true
}

func (s Server) loadApplicationForCurrentUser(w http.ResponseWriter, r *http.Request) (repository.Application, bool) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return repository.Application{}, false
	}
	applicationId := urlParam(r, "app_id")
	app, err := s.store.Application(r.Context(), applicationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Application " + applicationId + " not found"})
			return repository.Application{}, false
		}
		s.logger.Error("load application failed", "application_id", applicationId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application"})
		return repository.Application{}, false
	}
	if app.ProjectId != nil && !s.ensureProjectMembership(w, r, *app.ProjectId, current.Id) {
		return repository.Application{}, false
	}
	return app, true
}

func (s Server) loadRepositoryForCurrentUser(w http.ResponseWriter, r *http.Request) (repository.Repository, bool) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return repository.Repository{}, false
	}
	repositoryId := urlParam(r, "repository_id")
	repo, err := s.store.Repository(r.Context(), repositoryId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Repository " + repositoryId + " not found"})
			return repository.Repository{}, false
		}
		s.logger.Error("load repository failed", "repository_id", repositoryId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load repository"})
		return repository.Repository{}, false
	}
	if repo.ProjectId != nil && !s.ensureProjectMembership(w, r, *repo.ProjectId, current.Id) {
		return repository.Repository{}, false
	}
	return repo, true
}

func (s Server) ensureProjectMembership(w http.ResponseWriter, r *http.Request, projectId string, userId string) bool {
	if _, err := s.store.Project(r.Context(), projectId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Project " + projectId + " not found"})
			return false
		}
		s.logger.Error("load project failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load project"})
		return false
	}
	member, err := s.store.IsProjectMember(r.Context(), projectId, userId)
	if err != nil {
		s.logger.Error("check project member failed", "project_id", projectId, "user_id", userId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check project member"})
		return false
	}
	if !member {
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": "Permission denied"})
		return false
	}
	return true
}

func (s Server) ensureApplicationNameAvailable(w http.ResponseWriter, r *http.Request, name string) bool {
	existing, err := s.store.ApplicationByName(r.Context(), name)
	if err == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Application '" + existing.Name + "' already exists"})
		return false
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("check application name failed", "application_name", name, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check application name"})
		return false
	}
	return true
}

func parseOptionalRunTime(w http.ResponseWriter, value string, name string) (*time.Time, bool) {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil, true
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": name + " must be ISO 8601"})
		return nil, false
	}
	return &parsed, true
}

func (s Server) readDeploymentLog(w http.ResponseWriter, r *http.Request, deployment repository.Deployment, offset int) (string, int, bool) {
	if deployment.ApplicationId == nil || strings.TrimSpace(*deployment.ApplicationId) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Deployment " + deployment.Id + " has no associated application"})
		return "", offset, false
	}
	app, err := s.store.Application(r.Context(), *deployment.ApplicationId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Application " + *deployment.ApplicationId + " not found"})
			return "", offset, false
		}
		s.logger.Error("load deployment application failed", "deployment_id", deployment.Id, "application_id", *deployment.ApplicationId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application"})
		return "", offset, false
	}
	logPath := filepath.Join(s.appCfg.DataRoot(), "cd", app.Code, "deployments", deployment.Id+".log")
	file, err := os.Open(logPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return "", offset, true
		}
		s.logger.Error("open deployment log failed", "deployment_id", deployment.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to read deployment log"})
		return "", offset, false
	}
	defer func() { _ = file.Close() }()
	if _, err := file.Seek(int64(offset), io.SeekStart); err != nil {
		s.logger.Error("seek deployment log failed", "deployment_id", deployment.Id, "offset", offset, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to read deployment log"})
		return "", offset, false
	}
	content, err := io.ReadAll(file)
	if err != nil {
		s.logger.Error("read deployment log failed", "deployment_id", deployment.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to read deployment log"})
		return "", offset, false
	}
	return string(content), offset + len(content), true
}

func deploymentStatusComplete(status string) bool {
	return status == repository.WorkStatusRanToCompletion || status == repository.WorkStatusFaulted || status == repository.WorkStatusCanceled
}

func normalizeApplicationCreateReq(w http.ResponseWriter, req *applicationCreateReq) bool {
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.ImagePullPolicy = strings.TrimSpace(req.ImagePullPolicy)
	if req.Name == "" || len(req.Name) > 100 || req.Code == "" || len(req.Code) > 100 || !applicationCreateCodePattern.MatchString(req.Code) || !validImagePullPolicy(req.ImagePullPolicy) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid application fields"})
		return false
	}
	return true
}

func validImagePullPolicy(value string) bool {
	return value == "always" || value == "missing" || value == "never"
}

func applicationResponse(item repository.Application) applicationResp {
	return applicationResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, ImagePullPolicy: item.ImagePullPolicy, Status: item.Status, RouteManaged: item.RouteManaged, CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt)}
}

func deploymentResponse(item repository.Deployment) deploymentResp {
	return deploymentResp{Id: item.Id, ProjectId: item.ProjectId, ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, OperationType: item.OperationType, TriggerType: item.TriggerType, Status: item.Status, StartedAt: formatTime(item.StartedAt), FinishedAt: formatOptionalTime(item.FinishedAt), DurationMs: item.DurationMs, LogText: item.LogText, ErrorMessage: item.ErrorMessage, IsRollback: item.IsRollback, RollbackFromDeploymentId: item.RollbackFromDeploymentId}
}

package transporthttp

import (
	"database/sql"
	"errors"
	"net/http"
	"regexp"

	"backend/internal/repository"
	cdhandler "backend/internal/transport/http/handler/cd"
)

var applicationCreateCodePattern = regexp.MustCompile(`^[a-z][a-z0-9-]*$`)

type applicationResp = cdhandler.ApplicationResp

type deploymentResp = cdhandler.DeploymentResp

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

func (s Server) registerDashboardRoutes(r chiRouter) {
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
	r.Post("/api/cd/application/import", s.importApplication)
	r.Get("/api/cd/application/{app_id}/export", s.exportApplication)
	r.Post("/api/cd/application/{app_id}/compose-preview", s.previewApplicationCompose)
	r.Get("/api/cd/application/{app_id}/files", s.listApplicationFiles)
	r.Post("/api/cd/application/{app_id}/file", s.createApplicationFile)
	r.Get("/api/cd/application/{app_id}/file/{file_id}", s.readApplicationFile)
	r.Put("/api/cd/application/{app_id}/file/{file_id}", s.updateApplicationFile)
	r.Delete("/api/cd/application/{app_id}/file/{file_id}", s.deleteApplicationFile)
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

func validImagePullPolicy(value string) bool {
	return value == "always" || value == "missing" || value == "never"
}

func applicationResponse(item repository.Application) cdhandler.ApplicationResp {
	return cdhandler.ApplicationResponse(item)
}

func deploymentResponse(item repository.Deployment) cdhandler.DeploymentResp {
	return cdhandler.DeploymentResponse(item)
}

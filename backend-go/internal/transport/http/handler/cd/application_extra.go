package cdhandler

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/repository"
	cdsvc "backend/internal/service/cd"
	transportresponse "backend/internal/transport/http/response"
)

type configFileReq struct {
	Path    string `json:"path"`
	Content string `json:"content"`
}

type ConfigFileResp struct {
	Id        string `json:"id"`
	Path      string `json:"path"`
	CreatedAt string `json:"created_at"`
}

type applicationRouteReq struct {
	ServiceName string `json:"service_name"`
	Domain      string `json:"domain"`
	Port        int    `json:"port"`
}

type ApplicationRouteResp struct {
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

type ComposeServiceResp struct {
	ServiceName   string `json:"service_name"`
	DefaultDomain string `json:"default_domain"`
	DefaultPort   int    `json:"default_port"`
}

type ApplicationExportResp struct {
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

func (h Handler) RegisterApplicationExtraRoutes(r router) {
	r.Post("/api/cd/application/import", h.importApplication)
	r.Get("/api/cd/application/{app_id}/export", h.exportApplication)
	r.Post("/api/cd/application/{app_id}/compose-preview", h.previewApplicationCompose)
	r.Get("/api/cd/application/{app_id}/files", h.listApplicationFiles)
	r.Post("/api/cd/application/{app_id}/file", h.createApplicationFile)
	r.Get("/api/cd/application/{app_id}/file/{file_id}", h.readApplicationFile)
	r.Put("/api/cd/application/{app_id}/file/{file_id}", h.updateApplicationFile)
	r.Delete("/api/cd/application/{app_id}/file/{file_id}", h.deleteApplicationFile)
	r.Post("/api/cd/application/{app_id}/stop", h.stopApplication)
	r.Post("/api/cd/application/{app_id}/restart", h.restartApplication)
	r.Get("/api/cd/application/{app_id}/status", h.getApplicationStatus)
	r.Get("/api/cd/application/{app_id}/logs", h.getApplicationLogs)
	r.Get("/api/cd/application/{app_id}/route", h.listApplicationRoutes)
	r.Post("/api/cd/application/{app_id}/route", h.createApplicationRoute)
	r.Put("/api/cd/application/{app_id}/route/{route_id}", h.updateApplicationRoute)
	r.Delete("/api/cd/application/{app_id}/route/{route_id}", h.deleteApplicationRoute)
	r.Get("/api/cd/application/{app_id}/compose-service", h.listApplicationComposeServices)
	r.Get("/api/cd/application/{app_id}/service-config", h.listApplicationServiceConfigs)
	r.Put("/api/cd/application/{app_id}/service-config/{service_name}", h.updateApplicationServiceConfig)
}

func (h Handler) importApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req applicationImportReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	files := make([]cdsvc.ConfigFileInput, 0, len(req.ConfigFiles))
	for _, file := range req.ConfigFiles {
		files = append(files, cdsvc.ConfigFileInput{Path: file.Path, Content: file.Content})
	}
	serviceConfigs := make([]cdsvc.ApplicationServiceConfigImportInput, 0, len(req.ServiceConfigs))
	for _, item := range req.ServiceConfigs {
		serviceConfigs = append(serviceConfigs, cdsvc.ApplicationServiceConfigImportInput{ServiceName: item.ServiceName, Image: item.Image, Environment: item.Environment, Volumes: item.Volumes})
	}
	routes := make([]cdsvc.ApplicationRouteInput, 0, len(req.Routes))
	for _, item := range req.Routes {
		routes = append(routes, cdsvc.ApplicationRouteInput{ServiceName: item.ServiceName, Domain: item.Domain, Port: item.Port})
	}
	app, err := h.service.ImportApplication(r.Context(), current.Id, cdsvc.ApplicationImportInput{ProjectId: r.URL.Query().Get("project_id"), Version: req.Version, Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, RouteManaged: req.RouteManaged, ConfigFiles: files, ServiceConfigs: serviceConfigs, ApplicationRoutes: routes})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusCreated, applicationResponse(app))
}

func (h Handler) exportApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	exported, err := h.service.ExportApplication(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := ApplicationExportResp{Version: "1.0", Name: exported.Application.Name, Code: exported.Application.Code, ImagePullPolicy: exported.Application.ImagePullPolicy, RouteManaged: exported.Application.RouteManaged, ConfigFiles: []configFileReq{}, ServiceConfigs: []applicationServiceConfigImportReq{}, Routes: []applicationRouteReq{}}
	for _, file := range exported.ConfigFiles {
		resp.ConfigFiles = append(resp.ConfigFiles, configFileReq{Path: file.Path, Content: file.Content})
	}
	for _, config := range exported.ServiceConfigs {
		resp.ServiceConfigs = append(resp.ServiceConfigs, applicationServiceConfigImportReq{ServiceName: config.ServiceName, Image: config.Image, Environment: config.Environment, Volumes: config.Volumes})
	}
	for _, route := range exported.Routes {
		resp.Routes = append(resp.Routes, applicationRouteReq{ServiceName: route.ServiceName, Domain: route.Domain, Port: route.Port})
	}
	transportresponse.JSON(w, http.StatusOK, resp)
}

func (h Handler) listApplicationFiles(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	files, err := h.service.ListApplicationFiles(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := make([]ConfigFileResp, 0, len(files))
	for _, file := range files {
		resp = append(resp, configFileResponse(file))
	}
	transportresponse.JSON(w, http.StatusOK, resp)
}

func (h Handler) createApplicationFile(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req configFileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	file, err := h.service.CreateApplicationFile(r.Context(), current.Id, chi.URLParam(r, "app_id"), cdsvc.ConfigFileInput{Path: req.Path, Content: req.Content})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, configFileResponse(file))
}

func (h Handler) readApplicationFile(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	file, err := h.service.ApplicationFileForUser(r.Context(), current.Id, chi.URLParam(r, "app_id"), chi.URLParam(r, "file_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, map[string]string{"content": file.Content, "path": file.Path})
}

func (h Handler) updateApplicationFile(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req configFileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	file, err := h.service.UpdateApplicationFile(r.Context(), current.Id, chi.URLParam(r, "app_id"), chi.URLParam(r, "file_id"), cdsvc.ConfigFileInput{Path: req.Path, Content: req.Content})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, configFileResponse(file))
}

func (h Handler) deleteApplicationFile(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteApplicationFile(r.Context(), current.Id, chi.URLParam(r, "app_id"), chi.URLParam(r, "file_id")); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) stopApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	removeVolumes := false
	var body struct {
		RemoveVolumes bool `json:"remove_volumes"`
	}
	if r.Body != nil {
		_ = json.NewDecoder(r.Body).Decode(&body)
		removeVolumes = body.RemoveVolumes
	}
	deploymentId, err := h.service.StopApplication(r.Context(), current.Id, chi.URLParam(r, "app_id"), removeVolumes)
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, map[string]string{"deployment_id": deploymentId})
}

func (h Handler) restartApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	deploymentId, err := h.service.RestartApplication(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, map[string]string{"deployment_id": deploymentId})
}

func (h Handler) getApplicationStatus(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	status, err := h.service.ApplicationStatus(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		transportresponse.JSON(w, http.StatusInternalServerError, map[string]string{"status": status})
		return
	}
	transportresponse.JSON(w, http.StatusOK, map[string]string{"status": status})
}

func (h Handler) getApplicationLogs(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	logs, err := h.service.ApplicationLogs(r.Context(), current.Id, chi.URLParam(r, "app_id"), transportresponse.QueryInt(r.URL.Query().Get("tail"), 100))
	if err != nil {
		transportresponse.JSON(w, http.StatusInternalServerError, map[string]string{"logs": logs})
		return
	}
	transportresponse.JSON(w, http.StatusOK, map[string]string{"logs": logs})
}

func (h Handler) previewApplicationCompose(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	compose, err := h.service.ApplicationComposePreview(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, map[string]string{"compose_yaml": compose})
}

func (h Handler) listApplicationRoutes(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	routes, err := h.service.ListApplicationRoutes(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := make([]ApplicationRouteResp, 0, len(routes))
	for _, route := range routes {
		resp = append(resp, applicationRouteResponse(route))
	}
	transportresponse.JSON(w, http.StatusOK, resp)
}

func (h Handler) createApplicationRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req applicationRouteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	route, err := h.service.CreateApplicationRoute(r.Context(), current.Id, chi.URLParam(r, "app_id"), cdsvc.ApplicationRouteInput{ServiceName: req.ServiceName, Domain: req.Domain, Port: req.Port})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusCreated, applicationRouteResponse(route))
}

func (h Handler) updateApplicationRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req applicationRouteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	route, err := h.service.UpdateApplicationRoute(r.Context(), current.Id, chi.URLParam(r, "app_id"), chi.URLParam(r, "route_id"), cdsvc.ApplicationRouteInput{ServiceName: req.ServiceName, Domain: req.Domain, Port: req.Port})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, applicationRouteResponse(route))
}

func (h Handler) deleteApplicationRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteApplicationRoute(r.Context(), current.Id, chi.URLParam(r, "app_id"), chi.URLParam(r, "route_id")); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) listApplicationComposeServices(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	views, err := h.service.ApplicationComposeServices(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := make([]ComposeServiceResp, 0, len(views))
	for _, view := range views {
		resp = append(resp, ComposeServiceResp{ServiceName: view.ServiceName, DefaultDomain: view.DefaultDomain, DefaultPort: view.DefaultPort})
	}
	transportresponse.JSON(w, http.StatusOK, resp)
}

func (h Handler) listApplicationServiceConfigs(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	views, err := h.service.ApplicationServiceConfigs(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, views)
}

func (h Handler) updateApplicationServiceConfig(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req applicationServiceConfigUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	view, err := h.service.UpdateApplicationServiceConfig(r.Context(), current.Id, chi.URLParam(r, "app_id"), chi.URLParam(r, "service_name"), req.Image)
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(w, http.StatusOK, view)
}

func configFileResponse(file repository.ApplicationConfigFile) ConfigFileResp {
	return ConfigFileResp{Id: file.Id, Path: file.Path, CreatedAt: transportresponse.FormatTime(file.CreatedAt)}
}

func applicationRouteResponse(route repository.ApplicationRoute) ApplicationRouteResp {
	return ApplicationRouteResp{Id: route.Id, ServiceName: route.ServiceName, Domain: route.Domain, Port: route.Port, CreatedAt: transportresponse.FormatTime(route.CreatedAt), UpdatedAt: transportresponse.FormatTime(route.UpdatedAt)}
}

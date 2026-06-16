package transporthttp

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	"backend/internal/repository"
	"backend/internal/status"
	"backend/internal/templatex"
	"gopkg.in/yaml.v3"
)

func (s Server) ensureApplicationCodeAvailable(w http.ResponseWriter, r *http.Request, code string) bool {
	existing, err := s.store.ApplicationByCode(r.Context(), code)
	if err == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Application code '" + existing.Code + "' already exists"})
		return false
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("check application code failed", "application_code", code, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check application code"})
		return false
	}
	return true
}

func (s Server) importApplication(w http.ResponseWriter, r *http.Request) {
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
	var req applicationImportReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeApplicationImportReq(w, &req) || !s.ensureApplicationNameAvailable(w, r, req.Name) || !s.ensureApplicationCodeAvailable(w, r, req.Code) {
		return
	}
	app := repository.Application{Id: repository.NewId(), ProjectId: &projectId, Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, RouteManaged: req.RouteManaged, Status: repository.ApplicationStatusUndeployed}
	files := make([]repository.ApplicationConfigFile, 0, len(req.ConfigFiles))
	for _, file := range req.ConfigFiles {
		files = append(files, repository.ApplicationConfigFile{Id: repository.NewId(), ApplicationId: app.Id, Path: file.Path, Content: file.Content})
	}
	serviceConfigs := make([]repository.ApplicationServiceConfig, 0, len(req.ServiceConfigs))
	for _, item := range req.ServiceConfigs {
		image := normalizeOptionalText(item.Image)
		environment := normalizeOptionalText(item.Environment)
		volumes := normalizeOptionalText(item.Volumes)
		if image == nil && environment == nil && volumes == nil {
			continue
		}
		serviceConfigs = append(serviceConfigs, repository.ApplicationServiceConfig{Id: repository.NewId(), ApplicationId: app.Id, ServiceName: item.ServiceName, Image: image, Environment: environment, Volumes: volumes})
	}
	routes := make([]repository.ApplicationRoute, 0, len(req.Routes))
	for _, item := range req.Routes {
		routes = append(routes, repository.ApplicationRoute{Id: repository.NewId(), ApplicationId: app.Id, ServiceName: item.ServiceName, Domain: item.Domain, Port: item.Port})
	}
	if err := s.store.CreateApplicationBundle(r.Context(), app, files, serviceConfigs, routes); err != nil {
		s.logger.Error("import application failed", "application_code", app.Code, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to import application"})
		return
	}
	created, err := s.store.Application(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("load imported application failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application"})
		return
	}
	writeJSON(w, http.StatusCreated, applicationResponse(created))
}

func (s Server) exportApplication(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	files, err := s.store.ConfigFiles(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("load application config files failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application config files"})
		return
	}
	serviceConfigs, err := s.store.ServiceConfigs(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("load application service configs failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application service configs"})
		return
	}
	routes, err := s.store.Routes(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("load application routes failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application routes"})
		return
	}
	resp := applicationExportResp{Version: "1.0", Name: app.Name, Code: app.Code, ImagePullPolicy: app.ImagePullPolicy, RouteManaged: app.RouteManaged, ConfigFiles: []configFileReq{}, ServiceConfigs: []applicationServiceConfigImportReq{}, Routes: []applicationRouteReq{}}
	for _, file := range files {
		resp.ConfigFiles = append(resp.ConfigFiles, configFileReq{Path: file.Path, Content: file.Content})
	}
	for _, config := range serviceConfigs {
		resp.ServiceConfigs = append(resp.ServiceConfigs, applicationServiceConfigImportReq{ServiceName: config.ServiceName, Image: config.Image, Environment: config.Environment, Volumes: config.Volumes})
	}
	for _, route := range routes {
		resp.Routes = append(resp.Routes, applicationRouteReq{ServiceName: route.ServiceName, Domain: route.Domain, Port: route.Port})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) listApplicationFiles(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	files, err := s.store.ConfigFiles(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("list application files failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list application files"})
		return
	}
	resp := make([]configFileResp, 0, len(files))
	for _, file := range files {
		resp = append(resp, configFileResponse(file))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) createApplicationFile(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	var req configFileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeConfigFileReq(w, &req) {
		return
	}
	file := repository.ApplicationConfigFile{Id: repository.NewId(), ApplicationId: app.Id, Path: req.Path, Content: req.Content}
	if err := s.store.CreateConfigFile(r.Context(), file); err != nil {
		s.logger.Error("create application file failed", "application_id", app.Id, "path", file.Path, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create application file"})
		return
	}
	created, err := s.store.ConfigFile(r.Context(), file.Id)
	if err != nil {
		s.logger.Error("load created application file failed", "file_id", file.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application file"})
		return
	}
	writeJSON(w, http.StatusOK, configFileResponse(created))
}

func (s Server) readApplicationFile(w http.ResponseWriter, r *http.Request) {
	file, ok := s.loadApplicationFileForCurrentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"content": file.Content, "path": file.Path})
}

func (s Server) updateApplicationFile(w http.ResponseWriter, r *http.Request) {
	file, ok := s.loadApplicationFileForCurrentUser(w, r)
	if !ok {
		return
	}
	var req configFileReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeConfigFileReq(w, &req) {
		return
	}
	file.Path = req.Path
	file.Content = req.Content
	if err := s.store.UpdateConfigFile(r.Context(), file); err != nil {
		s.logger.Error("update application file failed", "file_id", file.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update application file"})
		return
	}
	updated, err := s.store.ConfigFile(r.Context(), file.Id)
	if err != nil {
		s.logger.Error("load updated application file failed", "file_id", file.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application file"})
		return
	}
	writeJSON(w, http.StatusOK, configFileResponse(updated))
}

func (s Server) deleteApplicationFile(w http.ResponseWriter, r *http.Request) {
	file, ok := s.loadApplicationFileForCurrentUser(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteConfigFile(r.Context(), file.Id); err != nil {
		s.logger.Error("delete application file failed", "file_id", file.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete application file"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) deployApplication(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	if !s.ensureApplicationComposeFile(w, r, app.Id) {
		return
	}
	deployment := s.newApplicationDeployment(app, "deploy")
	if err := s.store.CreateDeployment(r.Context(), deployment); err != nil {
		s.logger.Error("create deploy record failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create deployment"})
		return
	}
	if _, err := s.taskService.EnqueueTyped(r.Context(), status.TaskTypeCDApplicationDeploy, map[string]string{"application_id": app.Id, "deployment_id": deployment.Id}); err != nil {
		s.logger.Error("enqueue application deploy failed", "application_id", app.Id, "deployment_id", deployment.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to enqueue deployment"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deployment_id": deployment.Id})
}

func (s Server) stopApplication(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	if app.Status != repository.ApplicationStatusDeployed {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "应用未在运行中, 无法停止"})
		return
	}
	deployment := s.newApplicationDeployment(app, "stop")
	if err := s.store.CreateDeployment(r.Context(), deployment); err != nil {
		s.logger.Error("create stop record failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create deployment"})
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
	cmd := []string{"docker", "compose", "-f", "docker-compose.yml", "down"}
	if removeVolumes {
		cmd = append(cmd, "-v")
	}
	appDir := filepath.Join(s.appCfg.DataRoot(), "cd", app.Code)
	if output, err := runApplicationCommand(r, appDir, cmd...); err != nil {
		_ = s.store.CompleteDeployment(r.Context(), deployment.Id, repository.WorkStatusFaulted, outputOrError(output, err))
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to stop application"})
		return
	}
	if err := s.store.MarkApplicationStatus(r.Context(), app.Id, repository.ApplicationStatusUndeployed); err != nil {
		s.logger.Error("mark application stopped failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update application status"})
		return
	}
	if err := s.store.CompleteDeployment(r.Context(), deployment.Id, repository.WorkStatusRanToCompletion, ""); err != nil {
		s.logger.Error("complete stop deployment failed", "deployment_id", deployment.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to complete deployment"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deployment_id": deployment.Id})
}

func (s Server) restartApplication(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	if app.Status != repository.ApplicationStatusDeployed {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "应用未在运行中, 无法重启"})
		return
	}
	if !s.ensureApplicationComposeFile(w, r, app.Id) {
		return
	}
	deployment := s.newApplicationDeployment(app, "restart")
	if err := s.store.CreateDeployment(r.Context(), deployment); err != nil {
		s.logger.Error("create restart record failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create deployment"})
		return
	}
	if _, err := s.taskService.EnqueueTyped(r.Context(), status.TaskTypeCDApplicationRestart, map[string]string{"application_id": app.Id, "deployment_id": deployment.Id}); err != nil {
		s.logger.Error("enqueue application restart failed", "application_id", app.Id, "deployment_id", deployment.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to enqueue deployment"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"deployment_id": deployment.Id})
}

func (s Server) getApplicationStatus(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	output, err := runApplicationCommand(r, filepath.Join(s.appCfg.DataRoot(), "cd", app.Code), "docker", "compose", "-f", "docker-compose.yml", "ps", "--format", "json")
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"status": outputOrError(output, err)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": output})
}

func (s Server) getApplicationLogs(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	tail := queryInt(r.URL.Query().Get("tail"), 100)
	if tail < 1 || tail > 1000 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "tail must be between 1 and 1000"})
		return
	}
	output, err := runApplicationCommand(r, filepath.Join(s.appCfg.DataRoot(), "cd", app.Code), "docker", "compose", "-f", "docker-compose.yml", "logs", "--tail", strconv.Itoa(tail))
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"logs": outputOrError(output, err)})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"logs": output})
}

func (s Server) previewApplicationCompose(w http.ResponseWriter, r *http.Request) {
	app, compose, ok := s.loadApplicationCompose(w, r)
	if !ok {
		return
	}
	text, ok := s.renderApplicationCompose(w, r, app, compose)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"compose_yaml": text})
}

func (s Server) listApplicationRoutes(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	routes, err := s.store.Routes(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("list application routes failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list application routes"})
		return
	}
	resp := make([]applicationRouteResp, 0, len(routes))
	for _, route := range routes {
		resp = append(resp, applicationRouteResponse(route))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) createApplicationRoute(w http.ResponseWriter, r *http.Request) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return
	}
	if !app.RouteManaged {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "应用未启用路由托管"})
		return
	}
	var req applicationRouteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeApplicationRouteReq(w, &req) {
		return
	}
	route := repository.ApplicationRoute{Id: repository.NewId(), ApplicationId: app.Id, ServiceName: req.ServiceName, Domain: req.Domain, Port: req.Port}
	if err := s.store.CreateApplicationRoute(r.Context(), route); err != nil {
		s.logger.Error("create application route failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create application route"})
		return
	}
	created, err := s.store.ApplicationRoute(r.Context(), route.Id)
	if err != nil {
		s.logger.Error("load created application route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application route"})
		return
	}
	writeJSON(w, http.StatusCreated, applicationRouteResponse(created))
}

func (s Server) updateApplicationRoute(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadApplicationRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	var req applicationRouteReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeApplicationRouteReq(w, &req) {
		return
	}
	route.ServiceName = req.ServiceName
	route.Domain = req.Domain
	route.Port = req.Port
	if err := s.store.UpdateApplicationRoute(r.Context(), route); err != nil {
		s.logger.Error("update application route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update application route"})
		return
	}
	updated, err := s.store.ApplicationRoute(r.Context(), route.Id)
	if err != nil {
		s.logger.Error("load updated application route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application route"})
		return
	}
	writeJSON(w, http.StatusOK, applicationRouteResponse(updated))
}

func (s Server) deleteApplicationRoute(w http.ResponseWriter, r *http.Request) {
	route, ok := s.loadApplicationRouteForCurrentUser(w, r)
	if !ok {
		return
	}
	if err := s.store.DeleteApplicationRoute(r.Context(), route.Id); err != nil {
		s.logger.Error("delete application route failed", "route_id", route.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete application route"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) listApplicationComposeServices(w http.ResponseWriter, r *http.Request) {
	views, ok := s.applicationServiceConfigViews(w, r)
	if !ok {
		return
	}
	resp := make([]composeServiceResp, 0, len(views))
	for _, view := range views {
		resp = append(resp, composeServiceResp{ServiceName: view.ServiceName, DefaultDomain: view.DefaultDomain, DefaultPort: view.DefaultPort})
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) listApplicationServiceConfigs(w http.ResponseWriter, r *http.Request) {
	views, ok := s.applicationServiceConfigViews(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, views)
}

func (s Server) updateApplicationServiceConfig(w http.ResponseWriter, r *http.Request) {
	app, compose, ok := s.loadApplicationCompose(w, r)
	if !ok {
		return
	}
	services, ok := s.composeServices(w, r, app, compose)
	if !ok {
		return
	}
	serviceName := urlParam(r, "service_name")
	raw, exists := services[serviceName]
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Service " + serviceName + " not found"})
		return
	}
	var req applicationServiceConfigUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	image := normalizeOptionalText(req.Image)
	if image == nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Image cannot be empty"})
		return
	}
	config, err := s.store.ApplicationServiceConfig(r.Context(), app.Id, serviceName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			config = repository.ApplicationServiceConfig{Id: repository.NewId(), ApplicationId: app.Id, ServiceName: serviceName}
		} else {
			s.logger.Error("load application service config failed", "application_id", app.Id, "service_name", serviceName, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application service config"})
			return
		}
	}
	config.Image = image
	if err := s.store.UpsertApplicationServiceConfig(r.Context(), config); err != nil {
		s.logger.Error("update application service config failed", "application_id", app.Id, "service_name", serviceName, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update application service config"})
		return
	}
	updated, err := s.store.ApplicationServiceConfig(r.Context(), app.Id, serviceName)
	if err != nil {
		s.logger.Error("load updated application service config failed", "application_id", app.Id, "service_name", serviceName, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application service config"})
		return
	}
	writeJSON(w, http.StatusOK, s.applicationServiceConfigView(app, serviceName, raw, &updated))
}

func configFileResponse(file repository.ApplicationConfigFile) configFileResp {
	return configFileResp{Id: file.Id, Path: file.Path, CreatedAt: formatTime(file.CreatedAt)}
}

func applicationRouteResponse(route repository.ApplicationRoute) applicationRouteResp {
	return applicationRouteResp{Id: route.Id, ServiceName: route.ServiceName, Domain: route.Domain, Port: route.Port, CreatedAt: formatTime(route.CreatedAt), UpdatedAt: formatTime(route.UpdatedAt)}
}

func (s Server) newApplicationDeployment(app repository.Application, operationType string) repository.Deployment {
	return repository.Deployment{Id: repository.NewId(), ProjectId: app.ProjectId, ApplicationId: &app.Id, ApplicationName: app.Name, OperationType: operationType, TriggerType: "manual", Status: repository.WorkStatusWaitingToRun, IsRollback: false}
}

func (s Server) loadApplicationFileForCurrentUser(w http.ResponseWriter, r *http.Request) (repository.ApplicationConfigFile, bool) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return repository.ApplicationConfigFile{}, false
	}
	fileId := urlParam(r, "file_id")
	file, err := s.store.ConfigFile(r.Context(), fileId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Config file " + fileId + " not found"})
			return repository.ApplicationConfigFile{}, false
		}
		s.logger.Error("load application file failed", "file_id", fileId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application file"})
		return repository.ApplicationConfigFile{}, false
	}
	if file.ApplicationId != app.Id {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Config file " + fileId + " not found"})
		return repository.ApplicationConfigFile{}, false
	}
	return file, true
}

func (s Server) loadApplicationRouteForCurrentUser(w http.ResponseWriter, r *http.Request) (repository.ApplicationRoute, bool) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return repository.ApplicationRoute{}, false
	}
	routeId := urlParam(r, "route_id")
	route, err := s.store.ApplicationRoute(r.Context(), routeId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Route " + routeId + " not found"})
			return repository.ApplicationRoute{}, false
		}
		s.logger.Error("load application route failed", "route_id", routeId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application route"})
		return repository.ApplicationRoute{}, false
	}
	if route.ApplicationId != app.Id {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Route " + routeId + " not found"})
		return repository.ApplicationRoute{}, false
	}
	return route, true
}

func (s Server) ensureApplicationComposeFile(w http.ResponseWriter, r *http.Request, applicationId string) bool {
	files, err := s.store.ConfigFiles(r.Context(), applicationId)
	if err != nil {
		s.logger.Error("load application config files failed", "application_id", applicationId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application config files"})
		return false
	}
	for _, file := range files {
		if file.Path == "docker-compose.yml" || file.Path == "docker-compose.yml.jinja" {
			return true
		}
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "No docker-compose file found for application " + applicationId})
	return false
}

func (s Server) loadApplicationCompose(w http.ResponseWriter, r *http.Request) (repository.Application, repository.ApplicationConfigFile, bool) {
	app, ok := s.loadApplicationForCurrentUser(w, r)
	if !ok {
		return repository.Application{}, repository.ApplicationConfigFile{}, false
	}
	files, err := s.store.ConfigFiles(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("load application config files failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application config files"})
		return repository.Application{}, repository.ApplicationConfigFile{}, false
	}
	for _, file := range files {
		if file.Path == "docker-compose.yml" || file.Path == "docker-compose.yml.jinja" {
			return app, file, true
		}
	}
	writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "No docker-compose file found for this application"})
	return repository.Application{}, repository.ApplicationConfigFile{}, false
}

func (s Server) renderApplicationCompose(w http.ResponseWriter, r *http.Request, app repository.Application, compose repository.ApplicationConfigFile) (string, bool) {
	content := compose.Content
	var err error
	if strings.HasSuffix(compose.Path, ".jinja") {
		content, err = renderApplicationTemplate(content, app.Code, s)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": err.Error()})
			return "", false
		}
	}
	serviceConfigs, err := s.store.ServiceConfigs(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("load application service configs failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application service configs"})
		return "", false
	}
	content = applyApplicationServiceConfigs(content, serviceConfigs)
	if app.RouteManaged {
		routes, err := s.store.Routes(r.Context(), app.Id)
		if err != nil {
			s.logger.Error("load application routes failed", "application_id", app.Id, "error", err)
			writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application routes"})
			return "", false
		}
		content, err = injectApplicationRouteLabels(content, routes, s.appCfg.Cert.LetsEncrypt.Enabled)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": err.Error()})
			return "", false
		}
	}
	return content, true
}

func (s Server) composeServices(w http.ResponseWriter, r *http.Request, app repository.Application, compose repository.ApplicationConfigFile) (map[string]any, bool) {
	content := compose.Content
	if strings.HasSuffix(compose.Path, ".jinja") {
		rendered, err := renderApplicationTemplate(content, app.Code, s)
		if err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": err.Error()})
			return nil, false
		}
		content = rendered
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(content), &data); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "docker-compose 解析失败: " + err.Error()})
		return nil, false
	}
	services, ok := data["services"].(map[string]any)
	if !ok {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "docker-compose services 节点无效"})
		return nil, false
	}
	return services, true
}

func (s Server) applicationServiceConfigViews(w http.ResponseWriter, r *http.Request) ([]applicationServiceConfigResp, bool) {
	app, compose, ok := s.loadApplicationCompose(w, r)
	if !ok {
		if app.Id == "" {
			return nil, false
		}
		return []applicationServiceConfigResp{}, true
	}
	services, ok := s.composeServices(w, r, app, compose)
	if !ok {
		return nil, false
	}
	configs, err := s.store.ServiceConfigs(r.Context(), app.Id)
	if err != nil {
		s.logger.Error("load application service configs failed", "application_id", app.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load application service configs"})
		return nil, false
	}
	byName := map[string]repository.ApplicationServiceConfig{}
	for _, config := range configs {
		byName[config.ServiceName] = config
	}
	resp := make([]applicationServiceConfigResp, 0, len(services))
	for name, raw := range services {
		var config *repository.ApplicationServiceConfig
		if value, ok := byName[name]; ok {
			config = &value
		}
		resp = append(resp, s.applicationServiceConfigView(app, name, raw, config))
	}
	return resp, true
}

func (s Server) applicationServiceConfigView(app repository.Application, serviceName string, raw any, config *repository.ApplicationServiceConfig) applicationServiceConfigResp {
	service, _ := raw.(map[string]any)
	baseImage := optionalStringFromAny(service["image"])
	var image *string
	var configId *string
	var createdAt *string
	var updatedAt *string
	if config != nil {
		image = normalizeOptionalText(config.Image)
		configId = &config.Id
		created := formatTime(config.CreatedAt)
		updated := formatTime(config.UpdatedAt)
		createdAt = &created
		updatedAt = &updated
	}
	return applicationServiceConfigResp{ServiceName: serviceName, DefaultDomain: app.Code + "." + s.appCfg.Traefik.DomainSuffix, DefaultPort: extractDefaultPort(service), BaseImage: baseImage, Image: image, ConfigId: configId, CreatedAt: createdAt, UpdatedAt: updatedAt}
}

func renderApplicationTemplate(content string, appCode string, s Server) (string, error) {
	context := map[string]any{
		"app": map[string]any{
			"code":             appCode,
			"physical_dir":     s.appCfg.DataRoot(),
			"physical_app_dir": filepath.Join(s.appCfg.DataRoot(), "cd", appCode),
		},
		"config": map[string]any{"domain_suffix": s.appCfg.Traefik.DomainSuffix},
		"cert": map[string]any{"letsencrypt": map[string]any{
			"enabled":      s.appCfg.Cert.LetsEncrypt.Enabled,
			"email":        s.appCfg.Cert.LetsEncrypt.Email,
			"challenge":    s.appCfg.Cert.LetsEncrypt.Challenge,
			"dns_provider": s.appCfg.Cert.LetsEncrypt.DNSProvider,
		}},
	}
	return templatex.Render(content, context)
}

func applyApplicationServiceConfigs(compose string, configs []repository.ApplicationServiceConfig) string {
	if len(configs) == 0 {
		return compose
	}
	var data map[string]any
	if err := yaml.Unmarshal([]byte(compose), &data); err != nil {
		return compose
	}
	services, ok := data["services"].(map[string]any)
	if !ok {
		return compose
	}
	for _, config := range configs {
		service, ok := services[config.ServiceName].(map[string]any)
		if !ok || config.Image == nil || strings.TrimSpace(*config.Image) == "" {
			continue
		}
		service["image"] = strings.TrimSpace(*config.Image)
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return compose
	}
	return string(out)
}

func injectApplicationRouteLabels(compose string, routes []repository.ApplicationRoute, letsEncrypt bool) (string, error) {
	var data map[string]any
	if err := yaml.Unmarshal([]byte(compose), &data); err != nil {
		return "", err
	}
	services, ok := data["services"].(map[string]any)
	if !ok {
		return "", errors.New("docker-compose services must be a mapping")
	}
	for _, raw := range services {
		if service, ok := raw.(map[string]any); ok {
			delete(service, "labels")
		}
	}
	for _, route := range routes {
		service, ok := services[route.ServiceName].(map[string]any)
		if !ok {
			return "", errors.New("service " + route.ServiceName + " not found in docker-compose.yml")
		}
		labels := []string{
			"traefik.enable=true",
			"traefik.http.routers." + route.ServiceName + ".rule=Host(`" + route.Domain + "`)",
			"traefik.http.services." + route.ServiceName + ".loadbalancer.server.port=" + strconv.Itoa(route.Port),
		}
		if letsEncrypt {
			labels = append(labels, "traefik.http.routers."+route.ServiceName+".entrypoints=websecure", "traefik.http.routers."+route.ServiceName+".tls=true", "traefik.http.routers."+route.ServiceName+".tls.certresolver=letsencrypt")
		} else {
			labels = append(labels, "traefik.http.routers."+route.ServiceName+".entrypoints=web")
		}
		service["labels"] = labels
	}
	out, err := yaml.Marshal(data)
	if err != nil {
		return "", err
	}
	return string(out), nil
}

func runApplicationCommand(r *http.Request, cwd string, args ...string) (string, error) {
	if len(args) == 0 {
		return "", errors.New("command is required")
	}
	cmd := exec.CommandContext(r.Context(), args[0], args[1:]...)
	cmd.Dir = cwd
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	err := cmd.Run()
	return output.String(), err
}

func outputOrError(output string, err error) string {
	if strings.TrimSpace(output) != "" {
		return output
	}
	return err.Error()
}

func normalizeApplicationImportReq(w http.ResponseWriter, req *applicationImportReq) bool {
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	if req.ImagePullPolicy == "" {
		req.ImagePullPolicy = "missing"
	}
	req.ImagePullPolicy = strings.TrimSpace(req.ImagePullPolicy)
	if req.Name == "" || len(req.Name) > 100 || req.Code == "" || len(req.Code) > 100 || !applicationCreateCodePattern.MatchString(req.Code) || !validImagePullPolicy(req.ImagePullPolicy) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid application fields"})
		return false
	}
	for i := range req.ConfigFiles {
		if !normalizeConfigFileReq(w, &req.ConfigFiles[i]) {
			return false
		}
	}
	for i := range req.ServiceConfigs {
		req.ServiceConfigs[i].ServiceName = strings.TrimSpace(req.ServiceConfigs[i].ServiceName)
		if req.ServiceConfigs[i].ServiceName == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid application service config fields"})
			return false
		}
	}
	for i := range req.Routes {
		if !normalizeApplicationRouteReq(w, &req.Routes[i]) {
			return false
		}
	}
	return true
}

func normalizeConfigFileReq(w http.ResponseWriter, req *configFileReq) bool {
	req.Path = strings.TrimSpace(req.Path)
	if req.Path == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid config file fields"})
		return false
	}
	return true
}

func normalizeApplicationRouteReq(w http.ResponseWriter, req *applicationRouteReq) bool {
	req.ServiceName = strings.TrimSpace(req.ServiceName)
	req.Domain = strings.TrimSpace(req.Domain)
	if req.ServiceName == "" || req.Domain == "" || req.Port < 1 || req.Port > 65535 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid application route fields"})
		return false
	}
	return true
}

func normalizeOptionalText(value *string) *string {
	if value == nil {
		return nil
	}
	text := strings.TrimSpace(*value)
	if text == "" {
		return nil
	}
	return &text
}

func optionalStringFromAny(value any) *string {
	if value == nil {
		return nil
	}
	text := strings.TrimSpace(toString(value))
	if text == "" {
		return nil
	}
	return &text
}

func toString(value any) string {
	switch typed := value.(type) {
	case string:
		return typed
	case int:
		return strconv.Itoa(typed)
	case int64:
		return strconv.FormatInt(typed, 10)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	default:
		return ""
	}
}

func extractDefaultPort(service map[string]any) int {
	ports, ok := service["ports"].([]any)
	if !ok || len(ports) == 0 {
		return 80
	}
	first := toString(ports[0])
	parts := strings.Split(strings.Split(first, "/")[0], ":")
	port, err := strconv.Atoi(parts[len(parts)-1])
	if err != nil {
		return 80
	}
	return port
}

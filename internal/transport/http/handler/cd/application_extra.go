package cdhandler

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/repository/model"
	cdsvc "backend/internal/service/cd"
	transportresponse "backend/internal/transport/http/response"
)

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
	var req ApplicationImportReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
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
		routes = append(routes, cdsvc.ApplicationRouteInput{ServiceName: item.ServiceName, Domain: item.Domain, Port: int(item.Port)})
	}
	app, err := h.service.ImportApplication(r.Context(), current.Id, cdsvc.ApplicationImportInput{ProjectId: r.URL.Query().Get("project_id"), Version: req.Version, Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, RouteManaged: req.RouteManaged, ConfigFiles: files, ServiceConfigs: serviceConfigs, ApplicationRoutes: routes})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := applicationResponse(app)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
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
	resp := ApplicationExportResp{Version: "1.0", Name: exported.Application.Name, Code: exported.Application.Code, ImagePullPolicy: exported.Application.ImagePullPolicy, RouteManaged: exported.Application.RouteManaged, ConfigFiles: []*ApplicationExportConfigFileResp{}, ServiceConfigs: []*ApplicationServiceConfigExportResp{}, Routes: []*ApplicationExportRouteResp{}}
	for _, file := range exported.ConfigFiles {
		resp.ConfigFiles = append(resp.ConfigFiles, &ApplicationExportConfigFileResp{Path: file.Path, Content: file.Content})
	}
	for _, config := range exported.ServiceConfigs {
		resp.ServiceConfigs = append(resp.ServiceConfigs, &ApplicationServiceConfigExportResp{ServiceName: config.ServiceName, Image: config.Image, Environment: config.Environment, Volumes: config.Volumes})
	}
	for _, route := range exported.Routes {
		resp.Routes = append(resp.Routes, &ApplicationExportRouteResp{ServiceName: route.ServiceName, Domain: route.Domain, Port: int32(route.Port)})
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
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
	transportresponse.JSON(h.logger, w, http.StatusOK, &ConfigFileListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) createApplicationFile(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req ConfigFileReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	file, err := h.service.CreateApplicationFile(r.Context(), current.Id, chi.URLParam(r, "app_id"), cdsvc.ConfigFileInput{Path: req.Path, Content: req.Content})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := configFileResponse(file)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
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
	transportresponse.JSON(h.logger, w, http.StatusOK, &ApplicationFileContentResp{Content: file.Content, Path: file.Path})
}

func (h Handler) updateApplicationFile(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req ConfigFileReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	file, err := h.service.UpdateApplicationFile(r.Context(), current.Id, chi.URLParam(r, "app_id"), chi.URLParam(r, "file_id"), cdsvc.ConfigFileInput{Path: req.Path, Content: req.Content})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := configFileResponse(file)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
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
	var req ApplicationStopReq
	if r.Body != nil {
		_ = transportresponse.DecodeJSON(r.Body, &req)
	}
	deploymentId, err := h.service.StopApplication(r.Context(), current.Id, chi.URLParam(r, "app_id"), req.RemoveVolumes)
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &DeploymentActionResp{DeploymentId: deploymentId})
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
	transportresponse.JSON(h.logger, w, http.StatusOK, &DeploymentActionResp{DeploymentId: deploymentId})
}

func (h Handler) getApplicationStatus(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	status, err := h.service.ApplicationStatus(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		transportresponse.JSON(h.logger, w, http.StatusInternalServerError, &ApplicationStatusResp{Status: status})
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &ApplicationStatusResp{Status: status})
}

func (h Handler) getApplicationLogs(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	logs, err := h.service.ApplicationLogs(r.Context(), current.Id, chi.URLParam(r, "app_id"), transportresponse.QueryInt(r.URL.Query().Get("tail"), 100))
	if err != nil {
		transportresponse.JSON(h.logger, w, http.StatusInternalServerError, &ApplicationLogsResp{Logs: logs})
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &ApplicationLogsResp{Logs: logs})
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
	transportresponse.JSON(h.logger, w, http.StatusOK, &ApplicationComposePreviewResp{ComposeYaml: compose})
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
	transportresponse.JSON(h.logger, w, http.StatusOK, &ApplicationRouteListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) createApplicationRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req ApplicationRouteReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.CreateApplicationRoute(r.Context(), current.Id, chi.URLParam(r, "app_id"), cdsvc.ApplicationRouteInput{ServiceName: req.ServiceName, Domain: req.Domain, Port: int(req.Port)})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := applicationRouteResponse(route)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) updateApplicationRoute(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req ApplicationRouteReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.UpdateApplicationRoute(r.Context(), current.Id, chi.URLParam(r, "app_id"), chi.URLParam(r, "route_id"), cdsvc.ApplicationRouteInput{ServiceName: req.ServiceName, Domain: req.Domain, Port: int(req.Port)})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := applicationRouteResponse(route)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
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
		resp = append(resp, ComposeServiceResp{ServiceName: view.ServiceName, DefaultDomain: view.DefaultDomain, DefaultPort: int32(view.DefaultPort)})
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &ComposeServiceListResp{Items: transportresponse.Ptrs(resp)})
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
	transportresponse.JSON(h.logger, w, http.StatusOK, &ApplicationServiceConfigListResp{Items: transportresponse.Ptrs(applicationServiceConfigResponses(views))})
}

func (h Handler) updateApplicationServiceConfig(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req ApplicationServiceConfigUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.UpdateApplicationServiceConfig(r.Context(), current.Id, chi.URLParam(r, "app_id"), chi.URLParam(r, "service_name"), req.Image)
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := applicationServiceConfigResponse(view)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func configFileResponse(file model.ApplicationConfigFile) ConfigFileResp {
	return ConfigFileResp{Id: file.Id, Path: file.Path, CreatedAt: transportresponse.FormatTime(file.CreatedAt)}
}

func applicationRouteResponse(route model.ApplicationRoute) ApplicationRouteResp {
	return ApplicationRouteResp{Id: route.Id, ServiceName: route.ServiceName, Domain: route.Domain, Port: int32(route.Port), CreatedAt: transportresponse.FormatTime(route.CreatedAt), UpdatedAt: transportresponse.FormatTime(route.UpdatedAt)}
}

func applicationServiceConfigResponses(views []cdsvc.ApplicationServiceConfigView) []ApplicationServiceConfigResp {
	resp := make([]ApplicationServiceConfigResp, 0, len(views))
	for _, view := range views {
		resp = append(resp, applicationServiceConfigResponse(view))
	}
	return resp
}

func applicationServiceConfigResponse(view cdsvc.ApplicationServiceConfigView) ApplicationServiceConfigResp {
	return ApplicationServiceConfigResp{ServiceName: view.ServiceName, DefaultDomain: view.DefaultDomain, DefaultPort: int32(view.DefaultPort), BaseImage: view.BaseImage, Image: view.Image, ConfigId: view.ConfigId, CreatedAt: view.CreatedAt, UpdatedAt: view.UpdatedAt}
}

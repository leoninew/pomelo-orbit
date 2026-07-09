package cdhandler

import (
	"net/http"

	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (h Handler) RegisterApplicationExtraRoutes(r router) {
	r.POST("/api/cd/application/import", h.importApplication)
	r.GET("/api/cd/application/:app_id/export", h.exportApplication)
	r.POST("/api/cd/application/:app_id/compose-preview", h.previewApplicationCompose)
	r.GET("/api/cd/application/:app_id/files", h.listApplicationFiles)
	r.POST("/api/cd/application/:app_id/file", h.createApplicationFile)
	r.GET("/api/cd/application/:app_id/file/:file_id", h.readApplicationFile)
	r.PUT("/api/cd/application/:app_id/file/:file_id", h.updateApplicationFile)
	r.DELETE("/api/cd/application/:app_id/file/:file_id", h.deleteApplicationFile)
	r.POST("/api/cd/application/:app_id/stop", h.stopApplication)
	r.POST("/api/cd/application/:app_id/restart", h.restartApplication)
	r.GET("/api/cd/application/:app_id/status", h.getApplicationStatus)
	r.GET("/api/cd/application/:app_id/logs", h.getApplicationLogs)
	r.GET("/api/cd/application/:app_id/route", h.listApplicationRoutes)
	r.POST("/api/cd/application/:app_id/route", h.createApplicationRoute)
	r.PUT("/api/cd/application/:app_id/route/:route_id", h.updateApplicationRoute)
	r.DELETE("/api/cd/application/:app_id/route/:route_id", h.deleteApplicationRoute)
	r.GET("/api/cd/application/:app_id/compose-service", h.listApplicationComposeServices)
	r.GET("/api/cd/application/:app_id/service-config", h.listApplicationServiceConfigs)
	r.PUT("/api/cd/application/:app_id/service-config/:service_name", h.updateApplicationServiceConfig)
}

func (h Handler) importApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationImportReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	files := make([]cddto.ConfigFileInput, 0, len(req.ConfigFiles))
	for _, file := range req.ConfigFiles {
		files = append(files, cddto.ConfigFileInput{Path: file.Path, Content: file.Content})
	}
	serviceConfigs := make([]cddto.ApplicationServiceConfigImportInput, 0, len(req.ServiceConfigs))
	for _, item := range req.ServiceConfigs {
		serviceConfigs = append(serviceConfigs, cddto.ApplicationServiceConfigImportInput{ServiceName: item.ServiceName, Image: item.Image, Environment: item.Environment, Volumes: item.Volumes})
	}
	routes := make([]cddto.ApplicationRouteInput, 0, len(req.Routes))
	for _, item := range req.Routes {
		routes = append(routes, cddto.ApplicationRouteInput{ServiceName: item.ServiceName, Domain: item.Domain, Port: int(item.Port)})
	}
	app, err := h.service.ImportApplication(c.Request.Context(), current.Id, cddto.ApplicationImportInput{ProjectId: c.Request.URL.Query().Get("project_id"), Version: req.Version, Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, RouteManaged: req.RouteManaged, ConfigFiles: files, ServiceConfigs: serviceConfigs, ApplicationRoutes: routes})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponse(app)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) exportApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	exported, err := h.service.ExportApplication(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := pomeloorbit.ApplicationExportResp{Version: "1.0", Name: exported.Application.Name, Code: exported.Application.Code, ImagePullPolicy: exported.Application.ImagePullPolicy, RouteManaged: exported.Application.RouteManaged, ConfigFiles: []*pomeloorbit.ApplicationExportConfigFileResp{}, ServiceConfigs: []*pomeloorbit.ApplicationServiceConfigExportResp{}, Routes: []*pomeloorbit.ApplicationExportRouteResp{}}
	for _, file := range exported.ConfigFiles {
		resp.ConfigFiles = append(resp.ConfigFiles, &pomeloorbit.ApplicationExportConfigFileResp{Path: file.Path, Content: file.Content})
	}
	for _, config := range exported.ServiceConfigs {
		resp.ServiceConfigs = append(resp.ServiceConfigs, &pomeloorbit.ApplicationServiceConfigExportResp{ServiceName: config.ServiceName, Image: config.Image, Environment: config.Environment, Volumes: config.Volumes})
	}
	for _, route := range exported.Routes {
		resp.Routes = append(resp.Routes, &pomeloorbit.ApplicationExportRouteResp{ServiceName: route.ServiceName, Domain: route.Domain, Port: int32(route.Port)})
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) listApplicationFiles(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	files, err := h.service.ListApplicationFiles(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.ConfigFileResp, 0, len(files))
	for _, file := range files {
		resp = append(resp, configFileResponse(file))
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ConfigFileListResp{Items: transportresponse.Ptrs(resp)}})
}

func (h Handler) createApplicationFile(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ConfigFileReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	file, err := h.service.CreateApplicationFile(c.Request.Context(), current.Id, c.Param("app_id"), cddto.ConfigFileInput{Path: req.Path, Content: req.Content})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := configFileResponse(file)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) readApplicationFile(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	file, err := h.service.ApplicationFileForUser(c.Request.Context(), current.Id, c.Param("app_id"), c.Param("file_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ApplicationFileContentResp{Content: file.Content, Path: file.Path}})
}

func (h Handler) updateApplicationFile(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ConfigFileReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	file, err := h.service.UpdateApplicationFile(c.Request.Context(), current.Id, c.Param("app_id"), c.Param("file_id"), cddto.ConfigFileInput{Path: req.Path, Content: req.Content})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := configFileResponse(file)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deleteApplicationFile(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteApplicationFile(c.Request.Context(), current.Id, c.Param("app_id"), c.Param("file_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) stopApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationStopReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	deploymentId, err := h.service.StopApplication(c.Request.Context(), current.Id, c.Param("app_id"), req.RemoveVolumes)
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.DeploymentActionResp{DeploymentId: deploymentId}})
}

func (h Handler) restartApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationRestartReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	deploymentId, err := h.service.RestartApplication(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.DeploymentActionResp{DeploymentId: deploymentId}})
}

func (h Handler) getApplicationStatus(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	status, err := h.service.ApplicationStatus(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		c.Render(http.StatusInternalServerError, transportcodec.ProtoJSON{Message: &pomeloorbit.ApplicationStatusResp{Status: status}})
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ApplicationStatusResp{Status: status}})
}

func (h Handler) getApplicationLogs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	logs, err := h.service.ApplicationLogs(c.Request.Context(), current.Id, c.Param("app_id"), transportresponse.QueryInt(c.Request.URL.Query().Get("tail"), 100))
	if err != nil {
		c.Render(http.StatusInternalServerError, transportcodec.ProtoJSON{Message: &pomeloorbit.ApplicationLogsResp{Logs: logs}})
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ApplicationLogsResp{Logs: logs}})
}

func (h Handler) previewApplicationCompose(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationComposePreviewReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	compose, err := h.service.ApplicationComposePreview(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ApplicationComposePreviewResp{ComposeYaml: compose}})
}

func (h Handler) listApplicationRoutes(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	routes, err := h.service.ListApplicationRoutes(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.ApplicationRouteResp, 0, len(routes))
	for _, route := range routes {
		resp = append(resp, applicationRouteResponse(route))
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ApplicationRouteListResp{Items: transportresponse.Ptrs(resp)}})
}

func (h Handler) createApplicationRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationRouteReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	route, err := h.service.CreateApplicationRoute(c.Request.Context(), current.Id, c.Param("app_id"), cddto.ApplicationRouteInput{ServiceName: req.ServiceName, Domain: req.Domain, Port: int(req.Port)})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationRouteResponse(route)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) updateApplicationRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationRouteReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	route, err := h.service.UpdateApplicationRoute(c.Request.Context(), current.Id, c.Param("app_id"), c.Param("route_id"), cddto.ApplicationRouteInput{ServiceName: req.ServiceName, Domain: req.Domain, Port: int(req.Port)})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationRouteResponse(route)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deleteApplicationRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteApplicationRoute(c.Request.Context(), current.Id, c.Param("app_id"), c.Param("route_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) listApplicationComposeServices(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	views, err := h.service.ApplicationComposeServices(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.ComposeServiceResp, 0, len(views))
	for _, view := range views {
		resp = append(resp, pomeloorbit.ComposeServiceResp{ServiceName: view.ServiceName, DefaultDomain: view.DefaultDomain, DefaultPort: int32(view.DefaultPort)})
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ComposeServiceListResp{Items: transportresponse.Ptrs(resp)}})
}

func (h Handler) listApplicationServiceConfigs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	views, err := h.service.ApplicationServiceConfigs(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ApplicationServiceConfigListResp{Items: transportresponse.Ptrs(applicationServiceConfigResponses(views))}})
}

func (h Handler) updateApplicationServiceConfig(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationServiceConfigUpdateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	view, err := h.service.UpdateApplicationServiceConfig(c.Request.Context(), current.Id, c.Param("app_id"), c.Param("service_name"), req.Image)
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationServiceConfigResponse(view)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func configFileResponse(file model.ApplicationConfigFile) pomeloorbit.ConfigFileResp {
	return pomeloorbit.ConfigFileResp{Id: file.Id, Path: file.Path, CreatedAt: transportresponse.FormatTime(file.CreatedAt)}
}

func applicationRouteResponse(route model.ApplicationRoute) pomeloorbit.ApplicationRouteResp {
	return pomeloorbit.ApplicationRouteResp{Id: route.Id, ServiceName: route.ServiceName, Domain: route.Domain, Port: int32(route.Port), CreatedAt: transportresponse.FormatTime(route.CreatedAt), UpdatedAt: transportresponse.FormatTime(route.UpdatedAt)}
}

func applicationServiceConfigResponses(views []cddto.ApplicationServiceConfigView) []pomeloorbit.ApplicationServiceConfigResp {
	resp := make([]pomeloorbit.ApplicationServiceConfigResp, 0, len(views))
	for _, view := range views {
		resp = append(resp, applicationServiceConfigResponse(view))
	}
	return resp
}

func applicationServiceConfigResponse(view cddto.ApplicationServiceConfigView) pomeloorbit.ApplicationServiceConfigResp {
	return pomeloorbit.ApplicationServiceConfigResp{ServiceName: view.ServiceName, DefaultDomain: view.DefaultDomain, DefaultPort: int32(view.DefaultPort), BaseImage: view.BaseImage, Image: view.Image, ConfigId: view.ConfigId, CreatedAt: view.CreatedAt, UpdatedAt: view.UpdatedAt}
}

package cdhandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (h Handler) ImportApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationImportReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
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
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) ExportApplication(c *gin.Context) {
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
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) ListApplicationFiles(c *gin.Context) {
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
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ConfigFileListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) CreateApplicationFile(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ConfigFileReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	file, err := h.service.CreateApplicationFile(c.Request.Context(), current.Id, c.Param("app_id"), cddto.ConfigFileInput{Path: req.Path, Content: req.Content})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := configFileResponse(file)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) ReadApplicationFile(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	file, err := h.service.ApplicationFileForUser(c.Request.Context(), current.Id, c.Param("app_id"), c.Param("file_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ApplicationFileContentResp{Content: file.Content, Path: file.Path})
}

func (h Handler) UpdateApplicationFile(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ConfigFileReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	file, err := h.service.UpdateApplicationFile(c.Request.Context(), current.Id, c.Param("app_id"), c.Param("file_id"), cddto.ConfigFileInput{Path: req.Path, Content: req.Content})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := configFileResponse(file)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteApplicationFile(c *gin.Context) {
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

func (h Handler) StopApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationStopReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deploymentId, err := h.service.StopApplication(c.Request.Context(), current.Id, c.Param("app_id"), req.RemoveVolumes)
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.DeploymentActionResp{DeploymentId: deploymentId})
}

func (h Handler) RestartApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationRestartReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deploymentId, err := h.service.RestartApplication(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.DeploymentActionResp{DeploymentId: deploymentId})
}

func (h Handler) GetApplicationStatus(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	status, err := h.service.ApplicationStatus(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		transportresponse.ProtoJSON(c, http.StatusInternalServerError, &pomeloorbit.ApplicationStatusResp{Status: status})
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ApplicationStatusResp{Status: status})
}

func (h Handler) GetApplicationLogs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	logs, err := h.service.ApplicationLogs(c.Request.Context(), current.Id, c.Param("app_id"), binding.QueryInt(c.Request.URL.Query().Get("tail"), 100))
	if err != nil {
		transportresponse.ProtoJSON(c, http.StatusInternalServerError, &pomeloorbit.ApplicationLogsResp{Logs: logs})
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ApplicationLogsResp{Logs: logs})
}

func (h Handler) PreviewApplicationCompose(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationComposePreviewReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	compose, err := h.service.ApplicationComposePreview(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ApplicationComposePreviewResp{ComposeYaml: compose})
}

func (h Handler) ListApplicationRoutes(c *gin.Context) {
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
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ApplicationRouteListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) CreateApplicationRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationRouteReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.CreateApplicationRoute(c.Request.Context(), current.Id, c.Param("app_id"), cddto.ApplicationRouteInput{ServiceName: req.ServiceName, Domain: req.Domain, Port: int(req.Port)})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationRouteResponse(route)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) UpdateApplicationRoute(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationRouteReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	route, err := h.service.UpdateApplicationRoute(c.Request.Context(), current.Id, c.Param("app_id"), c.Param("route_id"), cddto.ApplicationRouteInput{ServiceName: req.ServiceName, Domain: req.Domain, Port: int(req.Port)})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationRouteResponse(route)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteApplicationRoute(c *gin.Context) {
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

func (h Handler) ListApplicationComposeServices(c *gin.Context) {
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
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ComposeServiceListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) ListApplicationServiceConfigs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	views, err := h.service.ApplicationServiceConfigs(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ApplicationServiceConfigListResp{Items: transportresponse.Ptrs(applicationServiceConfigResponses(views))})
}

func (h Handler) UpdateApplicationServiceConfig(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationServiceConfigUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.UpdateApplicationServiceConfig(c.Request.Context(), current.Id, c.Param("app_id"), c.Param("service_name"), req.Image)
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationServiceConfigResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
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

package cdhandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
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
	app, err := h.service.ImportApplication(c.Request.Context(), current.Id, applicationImportInput(c.Request.URL.Query().Get("project_id"), &req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	services, _ := h.service.ListServicesByApplication(c.Request.Context(), current.Id, app.Id)
	resp := applicationResponse(app, services)
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
	transportresponse.ProtoJSON(c, http.StatusOK, applicationExportResponse(exported))
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
	deploymentId, err := h.service.StopApplication(c.Request.Context(), current.Id, c.Param("app_id"), applicationServiceTargetFromStop(&req))
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
	deploymentId, err := h.service.RestartApplication(c.Request.Context(), current.Id, c.Param("app_id"), applicationServiceTargetFromRestart(&req))
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
	status, err := h.service.ApplicationStatus(c.Request.Context(), current.Id, c.Param("app_id"), applicationServiceTargetFromQuery(c))
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
	logs, err := h.service.ApplicationLogs(c.Request.Context(), current.Id, c.Param("app_id"), binding.QueryInt(c.Request.URL.Query().Get("tail"), 100), applicationServiceTargetFromQuery(c))
	if err != nil {
		transportresponse.ProtoJSON(c, http.StatusInternalServerError, &pomeloorbit.ApplicationLogsResp{Logs: logs})
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ApplicationLogsResp{Logs: logs})
}

func (h Handler) ListApplicationServices(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	services, err := h.service.ListServicesByApplication(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := serviceResponses(services)
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ServiceListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) ListVersions(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	views, err := h.service.ListVersions(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := versionResponses(views)
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.VersionListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) CreateVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.VersionCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if req.ApplicationId == "" {
		req.ApplicationId = c.Param("app_id")
	}
	view, err := h.service.CreateVersion(c.Request.Context(), current.Id, versionCreateInput(&req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.VersionForUser(c.Request.Context(), current.Id, c.Param("version_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.VersionUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.UpdateVersion(c.Request.Context(), current.Id, c.Param("version_id"), versionUpdateInput(&req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) PublishVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	view, err := h.service.PublishVersion(c.Request.Context(), current.Id, c.Param("version_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) ForkVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.VersionForkReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	view, err := h.service.ForkVersion(c.Request.Context(), current.Id, c.Param("version_id"), req.Label)
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := versionResponse(view)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) PreviewVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.VersionPreviewReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	instanceKey := ""
	if req.InstanceKey != nil {
		instanceKey = *req.InstanceKey
	}
	compose, err := h.service.PreviewVersion(c.Request.Context(), current.Id, c.Param("version_id"), req.EnvironmentId, instanceKey)
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.VersionPreviewResp{ComposeYaml: compose})
}

func applicationServiceTargetFromStop(req *pomeloorbit.ApplicationStopReq) cddto.ApplicationServiceTargetInput {
	return applicationServiceTargetInput(
		derefString(req.EnvironmentId),
		derefString(req.InstanceKey),
		derefString(req.ServiceId),
		req.RemoveVolumes,
	)
}

func applicationServiceTargetFromRestart(req *pomeloorbit.ApplicationRestartReq) cddto.ApplicationServiceTargetInput {
	return applicationServiceTargetInput(
		derefString(req.EnvironmentId),
		derefString(req.InstanceKey),
		derefString(req.ServiceId),
		false,
	)
}

func applicationServiceTargetFromQuery(c *gin.Context) cddto.ApplicationServiceTargetInput {
	return applicationServiceTargetInput(
		c.Request.URL.Query().Get("environment_id"),
		c.Request.URL.Query().Get("instance_key"),
		c.Request.URL.Query().Get("service_id"),
		false,
	)
}

func derefString(v *string) string {
	if v == nil {
		return ""
	}
	return *v
}

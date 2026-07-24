package deploymenthandler

import (
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	applicationv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/application"
	deploymentv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/deployment"
	"github.com/gin-gonic/gin"
)

func (h Handler) ListDeployments(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListDeployments(c.Request.Context(), current.Id, deploymentListInput(c.Request.URL.Query().Get("project_id"), c.Request.URL.Query().Get("application_id"), c.Request.URL.Query().Get("status"), c.Request.URL.Query().Get("search"), c.Request.URL.Query().Get("date_from"), c.Request.URL.Query().Get("date_to"), page, perPage))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := deploymentResponses(items.Items)
	transportresponse.ProtoJSON(c, http.StatusOK, &deploymentv1.DeploymentPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) GetDeployment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	deployment, err := h.service.DeploymentForUser(c.Request.Context(), current.Id, c.Param("deployment_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := deploymentResponse(deployment)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) GetDeploymentLogs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	log, err := h.service.DeploymentLog(c.Request.Context(), current.Id, c.Param("deployment_id"), binding.QueryInt(c.Request.URL.Query().Get("offset"), 0))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := deploymentLogsResponse(log)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) GetDeploymentContainerLogs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	log, err := h.service.DeploymentContainerLog(c.Request.Context(), current.Id, c.Param("deployment_id"), binding.QueryInt(c.Request.URL.Query().Get("tail"), 200))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := deploymentContainerLogsResponse(log)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) CancelDeployment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req deploymentv1.DeploymentCancelReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deployment, err := h.service.CancelDeployment(c.Request.Context(), current.Id, c.Param("deployment_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := deploymentResponse(deployment)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeployApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.ApplicationDeployReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deploymentID, err := h.service.DeployApplication(c.Request.Context(), current.Id, c.Param("app_id"), deploymentdto.DeployInput{
		VersionId: req.VersionId, EnvironmentId: req.EnvironmentId, InstanceKey: req.InstanceKey,
		ForceRecreate: req.ForceRecreate, RuntimeConfig: req.RuntimeConfig,
	})
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &applicationv1.DeploymentActionResp{DeploymentId: deploymentID})
}

func (h Handler) StopApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.ApplicationStopReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deploymentID, err := h.service.StopApplication(c.Request.Context(), current.Id, c.Param("app_id"), deploymentTarget(derefString(req.EnvironmentId), derefString(req.InstanceKey), derefString(req.ServiceId), req.RemoveVolumes))
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &applicationv1.DeploymentActionResp{DeploymentId: deploymentID})
}

func (h Handler) RestartApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.ApplicationRestartReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deploymentID, err := h.service.RestartApplication(c.Request.Context(), current.Id, c.Param("app_id"), deploymentTarget(derefString(req.EnvironmentId), derefString(req.InstanceKey), derefString(req.ServiceId), false))
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &applicationv1.DeploymentActionResp{DeploymentId: deploymentID})
}

func (h Handler) GetApplicationStatus(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	value, err := h.service.ApplicationStatus(c.Request.Context(), current.Id, c.Param("app_id"), deploymentTargetFromQuery(c))
	if err != nil {
		transportresponse.ProtoJSON(c, http.StatusInternalServerError, &applicationv1.ApplicationStatusResp{Status: value})
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &applicationv1.ApplicationStatusResp{Status: value})
}

func (h Handler) GetApplicationLogs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	value, err := h.service.ApplicationLogs(c.Request.Context(), current.Id, c.Param("app_id"), binding.QueryInt(c.Request.URL.Query().Get("tail"), 100), deploymentTargetFromQuery(c), c.Request.URL.Query().Get("component"))
	if err != nil {
		transportresponse.ProtoJSON(c, http.StatusInternalServerError, &applicationv1.ApplicationLogsResp{Logs: value})
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &applicationv1.ApplicationLogsResp{Logs: value})
}

func (h Handler) PreviewVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionPreviewReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	compose, err := h.service.PreviewVersion(c.Request.Context(), current.Id, c.Param("version_id"), req.EnvironmentId, derefString(req.InstanceKey))
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &applicationv1.VersionPreviewResp{ComposeYaml: compose})
}

func (h Handler) DeleteApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteApplication(c.Request.Context(), current.Id, c.Param("app_id"), c.Request.URL.Query().Get("remove_dir") == "true"); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func deploymentTarget(environmentID string, instanceKey string, serviceID string, removeVolumes bool) deploymentdto.ServiceTargetInput {
	return deploymentdto.ServiceTargetInput{EnvironmentId: environmentID, InstanceKey: instanceKey, ServiceId: serviceID, RemoveVolumes: removeVolumes}
}

func deploymentTargetFromQuery(c *gin.Context) deploymentdto.ServiceTargetInput {
	return deploymentTarget(c.Request.URL.Query().Get("environment_id"), c.Request.URL.Query().Get("instance_key"), c.Request.URL.Query().Get("service_id"), false)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

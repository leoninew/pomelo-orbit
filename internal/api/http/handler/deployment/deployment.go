package deploymenthandler

import (
	"net/http"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	applicationv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/application"
	deploymentv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/deployment"
	servicev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/service"
)

func (h Handler) ListDeployments(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transport.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transport.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListDeployments(c.Request.Context(), current.Id, deploymentListInput(c.Request.URL.Query().Get("project_id"), c.Request.URL.Query().Get("application_id"), c.Request.URL.Query().Get("status"), c.Request.URL.Query().Get("search"), c.Request.URL.Query().Get("date_from"), c.Request.URL.Query().Get("date_to"), page, perPage))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := deploymentResponses(items.Items)
	transport.WriteProtoJSON(c, http.StatusOK, &deploymentv1.DeploymentPaginatedResp{Items: transport.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transport.PageCount(items.Total, items.PerPage))})
}

func (h Handler) GetDeployment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	deployment, err := h.service.DeploymentForUser(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("deployment_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := deploymentResponse(deployment)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteDeployment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteDeployment(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("deployment_id")); err != nil {
		transport.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) GetDeploymentLogs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	log, err := h.service.DeploymentLog(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("deployment_id"), transport.QueryInt(c.Request.URL.Query().Get("offset"), 0))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := deploymentLogsResponse(log)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) GetDeploymentContainerLogs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	log, err := h.service.DeploymentContainerLog(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("deployment_id"), transport.QueryInt(c.Request.URL.Query().Get("tail"), 200))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := deploymentContainerLogsResponse(log)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) CancelDeployment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req deploymentv1.DeploymentCancelReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deployment, err := h.service.CancelDeployment(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("deployment_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := deploymentResponse(deployment)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeployService(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req servicev1.ServiceDeployReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	result, err := h.service.DeployService(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("service_id"), deploymentdto.DeployServiceInput{ForceRecreate: req.ForceRecreate, JoinTraefikNetwork: req.JoinTraefikNetwork})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &servicev1.ServiceDeployResp{DeploymentId: result.DeploymentId, Warnings: result.Warnings})
}

func (h Handler) StopApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.ApplicationStopReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deploymentId, err := h.service.StopApplication(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("app_id"), deploymentTarget(req.ServiceId, req.RemoveVolumes))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &applicationv1.DeploymentActionResp{DeploymentId: deploymentId})
}

func (h Handler) RestartApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.ApplicationRestartReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deploymentId, err := h.service.RestartApplication(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("app_id"), deploymentTarget(req.ServiceId, false))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &applicationv1.DeploymentActionResp{DeploymentId: deploymentId})
}

func (h Handler) GetApplicationStatus(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	containers, err := h.service.ApplicationStatus(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("app_id"), deploymentTargetFromQuery(c))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, applicationStatusResponse(containers))
}

func (h Handler) GetApplicationLogs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	value, err := h.service.ApplicationLogs(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("app_id"), transport.QueryInt(c.Request.URL.Query().Get("tail"), 100), deploymentTargetFromQuery(c), c.Request.URL.Query().Get("component"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &applicationv1.ApplicationLogsResp{Logs: value})
}

func (h Handler) PreviewService(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req servicev1.ServicePreviewReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	compose, err := h.service.PreviewService(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("service_id"), deploymentdto.PreviewComposeInput{JoinTraefikNetwork: req.JoinTraefikNetwork})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &servicev1.ServicePreviewResp{ComposeYaml: compose})
}

func (h Handler) PreviewVersion(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req applicationv1.VersionPreviewReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	compose, err := h.service.PreviewVersion(c.Request.Context(), current.Id, c.Query("project_id"), c.Param("version_id"), deploymentdto.PreviewComposeInput{JoinTraefikNetwork: req.JoinTraefikNetwork})
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &applicationv1.VersionPreviewResp{ComposeYaml: compose})
}

func deploymentTarget(serviceId string, removeVolumes bool) deploymentdto.ServiceTargetInput {
	return deploymentdto.ServiceTargetInput{ServiceId: serviceId, RemoveVolumes: removeVolumes}
}

func deploymentTargetFromQuery(c *gin.Context) deploymentdto.ServiceTargetInput {
	return deploymentTarget(c.Request.URL.Query().Get("service_id"), false)
}

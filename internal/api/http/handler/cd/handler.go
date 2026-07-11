package cdhandler

import (
	"log/slog"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cdsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
)

type Handler struct {
	logger        *slog.Logger
	service       cdsvc.Service
	authenticator authz.Authenticator
}

func New(logger *slog.Logger, service cdsvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) ListApplications(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListApplications(c.Request.Context(), current.Id, binding.QueryProjectId(c.Request.URL.Query().Get("project_id")), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponses(items.Items)
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.ApplicationPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreateApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	app, err := h.service.CreateApplication(c.Request.Context(), current.Id, applicationCreateInput(c.Request.URL.Query().Get("project_id"), &req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponse(app)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	app, err := h.service.ApplicationForUser(c.Request.Context(), current.Id, c.Param("app_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponse(app)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	app, err := h.service.UpdateApplication(c.Request.Context(), current.Id, c.Param("app_id"), applicationUpdateInput(&req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponse(app)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteApplication(c.Request.Context(), current.Id, c.Param("app_id"), applicationDeleteInput(c.Request.URL.Query().Get("remove_dir") == "true")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) DeployApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationDeployReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deploymentId, err := h.service.DeployApplication(c.Request.Context(), current.Id, c.Param("app_id"), applicationDeployInput(&req))
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.DeploymentActionResp{DeploymentId: deploymentId})
}

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
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.DeploymentPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
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
	var req pomeloorbit.DeploymentCancelReq
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

func (h Handler) writeError(c *gin.Context, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("cd request failed", "error", err)
	}
	transportresponse.Error(c, apperror.StatusCode(err), err.Error())
}

package cdhandler

import (
	"log/slog"
	"net/http"

	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cddto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	cdsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type router interface {
	GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
}

type Handler struct {
	logger        *slog.Logger
	service       cdsvc.Service
	authenticator authz.Authenticator
}

func New(logger *slog.Logger, service cdsvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) RegisterApplicationRoutes(r router) {
	r.GET("/api/cd/application", h.listApplications)
	r.POST("/api/cd/application", h.createApplication)
	r.GET("/api/cd/application/:app_id", h.getApplication)
	r.PUT("/api/cd/application/:app_id", h.updateApplication)
	r.DELETE("/api/cd/application/:app_id", h.deleteApplication)
	r.POST("/api/cd/application/:app_id/deploy", h.deployApplication)
}

func (h Handler) RegisterDeploymentRoutes(r router) {
	r.GET("/api/cd/deployment", h.listDeployments)
	r.GET("/api/cd/deployment/:deployment_id", h.getDeployment)
	r.GET("/api/cd/deployment/:deployment_id/logs", h.getDeploymentLogs)
	r.GET("/api/cd/deployment/:deployment_id/container-logs", h.getDeploymentContainerLogs)
	r.POST("/api/cd/deployment/:deployment_id/cancel", h.cancelDeployment)
}

func (h Handler) listApplications(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListApplications(c.Request.Context(), current.Id, transportresponse.QueryProjectId(c.Request.URL.Query().Get("project_id")), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := mapPage(items, applicationResponse)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ApplicationPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) createApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationCreateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	app, err := h.service.CreateApplication(c.Request.Context(), current.Id, cddto.ApplicationCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, RouteManaged: req.RouteManaged})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponse(app)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getApplication(c *gin.Context) {
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
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) updateApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationUpdateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	app, err := h.service.UpdateApplication(c.Request.Context(), current.Id, c.Param("app_id"), cddto.ApplicationUpdateInput{Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, RouteManaged: req.RouteManaged})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := applicationResponse(app)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deleteApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteApplication(c.Request.Context(), current.Id, c.Param("app_id"), cddto.ApplicationDeleteInput{RemoveDir: c.Request.URL.Query().Get("remove_dir") == "true"}); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) deployApplication(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationDeployReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	deploymentId, err := h.service.DeployApplication(c.Request.Context(), current.Id, c.Param("app_id"), cddto.ApplicationDeployInput{ForceRecreate: req.ForceRecreate})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.DeploymentActionResp{DeploymentId: deploymentId}})
}

func (h Handler) listDeployments(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListDeployments(c.Request.Context(), current.Id, cddto.DeploymentListInput{ProjectId: c.Request.URL.Query().Get("project_id"), ApplicationId: c.Request.URL.Query().Get("application_id"), Status: c.Request.URL.Query().Get("status"), Search: c.Request.URL.Query().Get("search"), DateFrom: c.Request.URL.Query().Get("date_from"), DateTo: c.Request.URL.Query().Get("date_to"), Page: page, PerPage: perPage})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := mapPage(items, deploymentResponse)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.DeploymentPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) getDeployment(c *gin.Context) {
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
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getDeploymentLogs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	log, err := h.service.DeploymentLog(c.Request.Context(), current.Id, c.Param("deployment_id"), transportresponse.QueryInt(c.Request.URL.Query().Get("offset"), 0))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.DeploymentLogsResp{Logs: log.Logs, Offset: int32(log.Offset), IsComplete: log.IsComplete, Status: log.Status}})
}

func (h Handler) getDeploymentContainerLogs(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	log, err := h.service.DeploymentContainerLog(c.Request.Context(), current.Id, c.Param("deployment_id"), transportresponse.QueryInt(c.Request.URL.Query().Get("tail"), 200))
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.DeploymentContainerLogsResp{Logs: log.Logs, Source: log.Source, IsRealtimeSupported: log.IsRealtimeSupported}})
}

func (h Handler) cancelDeployment(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.DeploymentCancelReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	deployment, err := h.service.CancelDeployment(c.Request.Context(), current.Id, c.Param("deployment_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := deploymentResponse(deployment)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) writeError(c *gin.Context, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("cd request failed", "error", err)
	}
	c.JSON(apperror.StatusCode(err), gin.H{"detail": err.Error()})
}

func applicationResponse(item model.Application) pomeloorbit.ApplicationResp {
	return ApplicationResponse(item)
}

func deploymentResponse(item model.Deployment) pomeloorbit.DeploymentResp {
	return DeploymentResponse(item)
}

func ApplicationResponse(item model.Application) pomeloorbit.ApplicationResp {
	return pomeloorbit.ApplicationResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, ImagePullPolicy: item.ImagePullPolicy, Status: item.Status, RouteManaged: item.RouteManaged, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func DeploymentResponse(item model.Deployment) pomeloorbit.DeploymentResp {
	return pomeloorbit.DeploymentResp{Id: item.Id, ProjectId: item.ProjectId, ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, OperationType: item.OperationType, TriggerType: item.TriggerType, CommandText: item.CommandText, Status: item.Status, StartedAt: transportresponse.FormatTime(item.StartedAt), FinishedAt: transportresponse.FormatOptionalTime(item.FinishedAt), DurationMs: transportresponse.OptionalInt32(item.DurationMs), LogText: item.LogText, ErrorMessage: item.ErrorMessage, IsRollback: item.IsRollback, RollbackFromDeploymentId: item.RollbackFromDeploymentId}
}

func mapPage[T any, U any](page repository.Page[T], convert func(T) U) repository.Page[U] {
	items := make([]U, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, convert(item))
	}
	return repository.Page[U]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}
}

package cdhandler

import (
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"gitee.com/leoninew/PomeloOrbit-go/internal/apperror"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/model"
	cdsvc "gitee.com/leoninew/PomeloOrbit-go/internal/service/cd"
	"gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http/response"
)

type router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
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
	r.Get("/api/cd/application", h.listApplications)
	r.Post("/api/cd/application", h.createApplication)
	r.Get("/api/cd/application/{app_id}", h.getApplication)
	r.Put("/api/cd/application/{app_id}", h.updateApplication)
	r.Delete("/api/cd/application/{app_id}", h.deleteApplication)
	r.Post("/api/cd/application/{app_id}/deploy", h.deployApplication)
}

func (h Handler) RegisterDeploymentRoutes(r router) {
	r.Get("/api/cd/deployment", h.listDeployments)
	r.Get("/api/cd/deployment/{deployment_id}", h.getDeployment)
	r.Get("/api/cd/deployment/{deployment_id}/logs", h.getDeploymentLogs)
	r.Get("/api/cd/deployment/{deployment_id}/container-logs", h.getDeploymentContainerLogs)
	r.Post("/api/cd/deployment/{deployment_id}/cancel", h.cancelDeployment)
}

func (h Handler) listApplications(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListApplications(r.Context(), current.Id, transportresponse.QueryProjectId(r.URL.Query().Get("project_id")), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := mapPage(items, applicationResponse)
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.ApplicationPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))})
}

func (h Handler) createApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationCreateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	app, err := h.service.CreateApplication(r.Context(), current.Id, cdsvc.ApplicationCreateInput{ProjectId: r.URL.Query().Get("project_id"), Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, RouteManaged: req.RouteManaged})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := applicationResponse(app)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) getApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	app, err := h.service.ApplicationForUser(r.Context(), current.Id, chi.URLParam(r, "app_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := applicationResponse(app)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) updateApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	app, err := h.service.UpdateApplication(r.Context(), current.Id, chi.URLParam(r, "app_id"), cdsvc.ApplicationUpdateInput{Name: req.Name, Code: req.Code, ImagePullPolicy: req.ImagePullPolicy, RouteManaged: req.RouteManaged})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := applicationResponse(app)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) deleteApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteApplication(r.Context(), current.Id, chi.URLParam(r, "app_id"), cdsvc.ApplicationDeleteInput{RemoveDir: r.URL.Query().Get("remove_dir") == "true"}); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) deployApplication(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.ApplicationDeployReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deploymentId, err := h.service.DeployApplication(r.Context(), current.Id, chi.URLParam(r, "app_id"), cdsvc.ApplicationDeployInput{ForceRecreate: req.ForceRecreate})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.DeploymentActionResp{DeploymentId: deploymentId})
}

func (h Handler) listDeployments(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListDeployments(r.Context(), current.Id, cdsvc.DeploymentListInput{ProjectId: r.URL.Query().Get("project_id"), ApplicationId: r.URL.Query().Get("application_id"), Status: r.URL.Query().Get("status"), Search: r.URL.Query().Get("search"), DateFrom: r.URL.Query().Get("date_from"), DateTo: r.URL.Query().Get("date_to"), Page: page, PerPage: perPage})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := mapPage(items, deploymentResponse)
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.DeploymentPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))})
}

func (h Handler) getDeployment(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	deployment, err := h.service.DeploymentForUser(r.Context(), current.Id, chi.URLParam(r, "deployment_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := deploymentResponse(deployment)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) getDeploymentLogs(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	log, err := h.service.DeploymentLog(r.Context(), current.Id, chi.URLParam(r, "deployment_id"), transportresponse.QueryInt(r.URL.Query().Get("offset"), 0))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.DeploymentLogsResp{Logs: log.Logs, Offset: int32(log.Offset), IsComplete: log.IsComplete, Status: log.Status})
}

func (h Handler) getDeploymentContainerLogs(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	log, err := h.service.DeploymentContainerLog(r.Context(), current.Id, chi.URLParam(r, "deployment_id"), transportresponse.QueryInt(r.URL.Query().Get("tail"), 200))
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &pomeloorbit.DeploymentContainerLogsResp{Logs: log.Logs, Source: log.Source, IsRealtimeSupported: log.IsRealtimeSupported})
}

func (h Handler) cancelDeployment(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req pomeloorbit.DeploymentCancelReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	deployment, err := h.service.CancelDeployment(r.Context(), current.Id, chi.URLParam(r, "deployment_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := deploymentResponse(deployment)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) writeError(w http.ResponseWriter, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("cd request failed", "error", err)
	}
	transportresponse.Error(h.logger, w, apperror.StatusCode(err), err.Error())
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

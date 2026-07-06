package cihandler

import (
	apiv1 "backend/internal/gen/orbit/api/v1"
	"io"
	"log/slog"
	"net/http"

	"github.com/go-chi/chi/v5"

	"backend/internal/apperror"
	"backend/internal/repository"
	"backend/internal/repository/model"
	cisvc "backend/internal/service/ci"
	"backend/internal/transport/http/handler/authz"
	transportresponse "backend/internal/transport/http/response"
)

type router interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
}

type Handler struct {
	logger        *slog.Logger
	service       cisvc.Service
	authenticator authz.Authenticator
}

func New(logger *slog.Logger, service cisvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) RegisterRepositoryRoutes(r router) {
	r.Get("/api/ci/repository", h.listRepositories)
	r.Post("/api/ci/repository", h.createRepository)
	r.Get("/api/ci/repository/{repository_id}", h.getRepository)
	r.Put("/api/ci/repository/{repository_id}", h.updateRepository)
	r.Delete("/api/ci/repository/{repository_id}", h.deleteRepository)
	r.Get("/api/ci/repository/{repository_id}/webhook", h.listRepositoryWebhooks)
	r.Post("/api/ci/repository/{repository_id}/webhook", h.createRepositoryWebhook)
	r.Put("/api/ci/repository/{repository_id}/webhook/{webhook_id}", h.updateRepositoryWebhook)
	r.Delete("/api/ci/repository/{repository_id}/webhook/{webhook_id}", h.deleteRepositoryWebhook)
	r.Get("/api/ci/webhook/{webhook_id}", h.getRepositoryWebhook)
	r.Post("/api/ci/webhook/{webhook_id}", h.receiveRepositoryWebhook)
}

func (h Handler) listRepositories(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(r.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(r.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListRepositories(r.Context(), current.Id, transportresponse.QueryProjectId(r.URL.Query().Get("project_id")), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := mapPage(items, repositoryListResponse)
	transportresponse.JSON(h.logger, w, http.StatusOK, &apiv1.RepositoryPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))})
}

func (h Handler) createRepository(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req apiv1.RepositoryCreateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.CreateRepository(r.Context(), current.Id, cisvc.RepositoryCreateInput{ProjectId: r.URL.Query().Get("project_id"), Name: req.Name, Code: req.Code, RepositoryURL: req.RepositoryUrl, GitCredentialId: req.GitCredentialId, VariableOverrides: variableDeclarationRequestMaps(req.VariableOverrides), DefaultBranch: req.DefaultBranch})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := repositoryDetailResponse(detail)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) getRepository(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	detail, err := h.service.RepositoryForUser(r.Context(), current.Id, chi.URLParam(r, "repository_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := repositoryDetailResponse(detail)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) updateRepository(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req apiv1.RepositoryUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var variableOverrides *[]map[string]any
	if req.VariableOverrides != nil {
		items := variableDeclarationRequestMaps(req.VariableOverrides.Items)
		variableOverrides = &items
	}
	detail, err := h.service.UpdateRepository(r.Context(), current.Id, chi.URLParam(r, "repository_id"), cisvc.RepositoryUpdateInput{Name: req.Name, RepositoryURL: req.RepositoryUrl, GitCredentialId: req.GitCredentialId, VariableOverrides: variableOverrides, DefaultBranch: req.DefaultBranch})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := repositoryDetailResponse(detail)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) deleteRepository(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteRepository(r.Context(), current.Id, chi.URLParam(r, "repository_id")); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) listRepositoryWebhooks(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListRepositoryWebhooks(r.Context(), current.Id, chi.URLParam(r, "repository_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := make([]apiv1.RepositoryWebhookResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, repositoryWebhookResponse(item))
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &apiv1.RepositoryWebhookListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) createRepositoryWebhook(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req apiv1.RepositoryWebhookCreateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	webhook, err := h.service.CreateRepositoryWebhook(r.Context(), current.Id, chi.URLParam(r, "repository_id"), cisvc.WebhookCreateInput{Name: req.Name, TemplateId: req.TemplateId, Secret: req.Secret, BranchFilter: req.BranchFilter})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := repositoryWebhookResponse(webhook)
	transportresponse.JSON(h.logger, w, http.StatusCreated, &resp)
}

func (h Handler) getRepositoryWebhook(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	webhook, err := h.service.RepositoryWebhookForUser(r.Context(), current.Id, chi.URLParam(r, "webhook_id"))
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := repositoryWebhookResponse(webhook)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) updateRepositoryWebhook(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req repositoryWebhookUpdateReq
	if err := transportresponse.DecodeJSON(r.Body, &req); err != nil {
		transportresponse.Error(h.logger, w, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	webhook, err := h.service.UpdateRepositoryWebhook(r.Context(), current.Id, chi.URLParam(r, "repository_id"), chi.URLParam(r, "webhook_id"), cisvc.WebhookUpdateInput{Name: req.Body.Name, TemplateId: req.Body.TemplateId, Secret: req.Body.Secret, BranchFilter: req.Body.BranchFilter, BranchSet: req.BranchSet, Enabled: req.Body.Enabled})
	if err != nil {
		h.writeError(w, err)
		return
	}
	resp := repositoryWebhookResponse(webhook)
	transportresponse.JSON(h.logger, w, http.StatusOK, &resp)
}

func (h Handler) deleteRepositoryWebhook(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	if err := h.service.DeleteRepositoryWebhook(r.Context(), current.Id, chi.URLParam(r, "repository_id"), chi.URLParam(r, "webhook_id")); err != nil {
		h.writeError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) receiveRepositoryWebhook(w http.ResponseWriter, r *http.Request) {
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		h.logger.Error("read webhook payload failed", "webhook_id", chi.URLParam(r, "webhook_id"), "error", err)
		transportresponse.Error(h.logger, w, http.StatusInternalServerError, "Failed to read webhook payload")
		return
	}
	result, err := h.service.ReceiveRepositoryWebhook(r.Context(), cisvc.WebhookReceiveInput{WebhookId: chi.URLParam(r, "webhook_id"), Headers: requestHeaders(r), Payload: payload})
	if err != nil {
		h.writeError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, &apiv1.RepositoryWebhookReceiveResp{Status: result.Status, Reason: result.Reason, RunId: result.RunId})
}

func (h Handler) writeError(w http.ResponseWriter, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("ci repository request failed", "error", err)
	}
	transportresponse.Error(h.logger, w, apperror.StatusCode(err), err.Error())
}

func repositoryListResponse(item model.Repository) apiv1.RepositoryResp {
	return apiv1.RepositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryUrl: item.RepositoryURL, HasCredential: item.GitCredentialId != nil, GitCredentialId: transportresponse.OptionalStringValue(item.GitCredentialId), DefaultBranch: item.DefaultBranch, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func repositoryDetailResponse(detail cisvc.RepositoryDetail) apiv1.RepositoryResp {
	item := detail.Repository
	return apiv1.RepositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryUrl: item.RepositoryURL, HasCredential: item.GitCredentialId != nil, GitCredentialId: transportresponse.OptionalStringValue(item.GitCredentialId), GitCredentialName: detail.GitCredentialName, VariableDeclarations: transportresponse.Ptrs(variableDeclarationResponses(detail.VariableDeclarations)), DefaultBranch: item.DefaultBranch, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func repositoryWebhookResponse(item model.RepositoryWebhook) apiv1.RepositoryWebhookResp {
	return apiv1.RepositoryWebhookResp{Id: item.Id, RepositoryId: item.RepositoryId, Name: item.Name, TemplateId: item.TemplateId, BranchFilter: item.BranchFilter, Secret: item.EncryptedSecret, Enabled: item.Enabled, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func requestHeaders(r *http.Request) map[string]string {
	return map[string]string{"X-Hub-Signature-256": r.Header.Get("X-Hub-Signature-256"), "X-Gitlab-Token": r.Header.Get("X-Gitlab-Token")}
}

func mapPage[T any, U any](page repository.Page[T], convert func(T) U) repository.Page[U] {
	items := make([]U, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, convert(item))
	}
	return repository.Page[U]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}
}

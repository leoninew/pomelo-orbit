package cihandler

import (
	"io"
	"log/slog"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type Handler struct {
	logger        *slog.Logger
	service       cisvc.Service
	authenticator authz.Authenticator
}

func New(logger *slog.Logger, service cisvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) ListRepositories(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListRepositories(c.Request.Context(), current.Id, binding.QueryProjectId(c.Request.URL.Query().Get("project_id")), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.RepositoryResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, repositoryListResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.RepositoryPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreateRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RepositoryCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.CreateRepository(c.Request.Context(), current.Id, cidto.RepositoryCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Code: req.Code, RepositoryURL: req.RepositoryUrl, GitCredentialId: req.GitCredentialId, VariableOverrides: variableDeclarationRequestMaps(req.VariableOverrides), DefaultBranch: req.DefaultBranch})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := repositoryDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	detail, err := h.service.RepositoryForUser(c.Request.Context(), current.Id, c.Param("repository_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := repositoryDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RepositoryUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var variableOverrides *[]map[string]any
	if req.VariableOverrides != nil {
		items := variableDeclarationRequestMaps(req.VariableOverrides.Items)
		variableOverrides = &items
	}
	detail, err := h.service.UpdateRepository(c.Request.Context(), current.Id, c.Param("repository_id"), cidto.RepositoryUpdateInput{Name: req.Name, RepositoryURL: req.RepositoryUrl, GitCredentialId: req.GitCredentialId, VariableOverrides: variableOverrides, DefaultBranch: req.DefaultBranch})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := repositoryDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteRepository(c.Request.Context(), current.Id, c.Param("repository_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) ListRepositoryWebhooks(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListRepositoryWebhooks(c.Request.Context(), current.Id, c.Param("repository_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := make([]pomeloorbit.RepositoryWebhookResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, repositoryWebhookResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.RepositoryWebhookListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) CreateRepositoryWebhook(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RepositoryWebhookCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	webhook, err := h.service.CreateRepositoryWebhook(c.Request.Context(), current.Id, c.Param("repository_id"), cidto.WebhookCreateInput{Name: req.Name, TemplateId: req.TemplateId, Secret: req.Secret, BranchFilter: req.BranchFilter})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := repositoryWebhookResponse(webhook)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetRepositoryWebhook(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	webhook, err := h.service.RepositoryWebhookForUser(c.Request.Context(), current.Id, c.Param("webhook_id"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := repositoryWebhookResponse(webhook)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateRepositoryWebhook(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req binding.RepositoryWebhookUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.Error(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	webhook, err := h.service.UpdateRepositoryWebhook(c.Request.Context(), current.Id, c.Param("repository_id"), c.Param("webhook_id"), cidto.WebhookUpdateInput{Name: req.Body.Name, TemplateId: req.Body.TemplateId, Secret: req.Body.Secret, BranchFilter: req.Body.BranchFilter, BranchSet: req.BranchSet, Enabled: req.Body.Enabled})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := repositoryWebhookResponse(webhook)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeleteRepositoryWebhook(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	if err := h.service.DeleteRepositoryWebhook(c.Request.Context(), current.Id, c.Param("repository_id"), c.Param("webhook_id")); err != nil {
		h.writeError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) ReceiveRepositoryWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("read webhook payload failed", "webhook_id", c.Param("webhook_id"), "error", err)
		transportresponse.Error(c, http.StatusInternalServerError, "Failed to read webhook payload")
		return
	}
	result, err := h.service.ReceiveRepositoryWebhook(c.Request.Context(), cidto.WebhookReceiveInput{WebhookId: c.Param("webhook_id"), Headers: requestHeaders(c), Payload: payload})
	if err != nil {
		h.writeError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.RepositoryWebhookReceiveResp{Status: result.Status, Reason: result.Reason, RunId: result.RunId})
}

func (h Handler) writeError(c *gin.Context, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("ci repository request failed", "error", err)
	}
	transportresponse.Error(c, apperror.StatusCode(err), err.Error())
}

func repositoryListResponse(item model.Repository) pomeloorbit.RepositoryResp {
	return pomeloorbit.RepositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryUrl: item.RepositoryURL, HasCredential: item.GitCredentialId != nil, GitCredentialId: transportresponse.OptionalStringValue(item.GitCredentialId), DefaultBranch: item.DefaultBranch, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func repositoryDetailResponse(detail cidto.RepositoryDetail) pomeloorbit.RepositoryResp {
	item := detail.Repository
	return pomeloorbit.RepositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryUrl: item.RepositoryURL, HasCredential: item.GitCredentialId != nil, GitCredentialId: transportresponse.OptionalStringValue(item.GitCredentialId), GitCredentialName: detail.GitCredentialName, VariableDeclarations: transportresponse.Ptrs(variableDeclarationResponses(detail.VariableDeclarations)), DefaultBranch: item.DefaultBranch, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func repositoryWebhookResponse(item model.RepositoryWebhook) pomeloorbit.RepositoryWebhookResp {
	return pomeloorbit.RepositoryWebhookResp{Id: item.Id, RepositoryId: item.RepositoryId, Name: item.Name, TemplateId: item.TemplateId, BranchFilter: item.BranchFilter, Secret: item.EncryptedSecret, Enabled: item.Enabled, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func requestHeaders(c *gin.Context) map[string]string {
	return map[string]string{"X-Hub-Signature-256": c.GetHeader("X-Hub-Signature-256"), "X-Gitlab-Token": c.GetHeader("X-Gitlab-Token")}
}

package repositoryhandler

import (
	"io"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	repositoryv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/repository"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	repositorydto "gitee.com/leoninew/PomeloOrbit-go/internal/application/repository/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (h Handler) ListRepositories(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := binding.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := binding.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListRepositories(c.Request.Context(), current.Id, binding.QueryProjectId(c.Request.URL.Query().Get("project_id")), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	resp := make([]repositoryv1.RepositoryResp, 0, len(items.Items))
	for _, item := range items.Items {
		resp = append(resp, repositoryListResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &repositoryv1.RepositoryPaginatedResp{Items: transportresponse.Ptrs(resp), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}

func (h Handler) CreateRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req repositoryv1.RepositoryCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.CreateRepository(c.Request.Context(), current.Id, repositorydto.RepositoryCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Code: req.Code, RepositoryType: req.RepositoryType, RepositoryUrl: req.RepositoryUrl, GitCredentialId: req.GitCredentialId, VariableOverrides: variableDeclarationRequestMaps(req.VariableOverrides), DefaultBranch: req.DefaultBranch})
	if err != nil {
		transportresponse.WriteError(c, err)
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
		transportresponse.WriteError(c, err)
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
	var req repositoryv1.RepositoryUpdateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var variableOverrides *[]map[string]any
	if req.VariableOverrides != nil {
		items := variableDeclarationRequestMaps(req.VariableOverrides.Items)
		variableOverrides = &items
	}
	detail, err := h.service.UpdateRepository(c.Request.Context(), current.Id, c.Param("repository_id"), repositorydto.RepositoryUpdateInput{Name: req.Name, RepositoryType: req.RepositoryType, RepositoryUrl: req.RepositoryUrl, GitCredentialId: req.GitCredentialId, VariableOverrides: variableOverrides, DefaultBranch: req.DefaultBranch})
	if err != nil {
		transportresponse.WriteError(c, err)
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
		transportresponse.WriteError(c, err)
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
		transportresponse.WriteError(c, err)
		return
	}
	resp := make([]repositoryv1.RepositoryWebhookResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, repositoryWebhookResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &repositoryv1.RepositoryWebhookListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) CreateRepositoryWebhook(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req repositoryv1.RepositoryWebhookCreateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	webhook, err := h.service.CreateRepositoryWebhook(c.Request.Context(), current.Id, c.Param("repository_id"), repositorydto.WebhookCreateInput{Name: req.Name, TemplateId: req.TemplateId, Secret: req.Secret, BranchFilter: req.BranchFilter})
	if err != nil {
		transportresponse.WriteError(c, err)
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
		transportresponse.WriteError(c, err)
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
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	webhook, err := h.service.UpdateRepositoryWebhook(c.Request.Context(), current.Id, c.Param("repository_id"), c.Param("webhook_id"), repositorydto.WebhookUpdateInput{Name: req.Body.Name, TemplateId: req.Body.TemplateId, Secret: req.Body.Secret, BranchFilter: req.Body.BranchFilter, BranchSet: req.BranchSet, Enabled: req.Body.Enabled})
	if err != nil {
		transportresponse.WriteError(c, err)
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
		transportresponse.WriteError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) ReceiveRepositoryWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("read webhook payload failed", "webhook_id", c.Param("webhook_id"), "error", err)
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	result, err := h.service.ReceiveRepositoryWebhook(c.Request.Context(), repositorydto.WebhookReceiveInput{WebhookId: c.Param("webhook_id"), Headers: requestHeaders(c), Payload: payload})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &repositoryv1.RepositoryWebhookReceiveResp{Status: result.Status, Reason: result.Reason, RunId: result.RunId})
}

func repositoryListResponse(item model.Repository) repositoryv1.RepositoryResp {
	return repositoryv1.RepositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryType: item.RepositoryType, RepositoryUrl: item.RepositoryUrl, HasCredential: item.GitCredentialId != nil, GitCredentialId: transportresponse.OptionalStringValue(item.GitCredentialId), DefaultBranch: item.DefaultBranch, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func repositoryDetailResponse(detail repositorydto.RepositoryDetail) repositoryv1.RepositoryResp {
	item := detail.Repository
	return repositoryv1.RepositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryType: item.RepositoryType, RepositoryUrl: item.RepositoryUrl, HasCredential: item.GitCredentialId != nil, GitCredentialId: transportresponse.OptionalStringValue(item.GitCredentialId), GitCredentialName: detail.GitCredentialName, VariableDeclarations: transportresponse.Ptrs(variableDeclarationResponses(detail.VariableDeclarations)), DefaultBranch: item.DefaultBranch, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func repositoryWebhookResponse(item model.RepositoryWebhook) repositoryv1.RepositoryWebhookResp {
	return repositoryv1.RepositoryWebhookResp{Id: item.Id, RepositoryId: item.RepositoryId, Name: item.Name, TemplateId: item.TemplateId, BranchFilter: item.BranchFilter, Secret: item.EncryptedSecret, Enabled: item.Enabled, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

func requestHeaders(c *gin.Context) map[string]string {
	return map[string]string{"X-Hub-Signature-256": c.GetHeader("X-Hub-Signature-256"), "X-Gitlab-Token": c.GetHeader("X-Gitlab-Token")}
}

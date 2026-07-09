package cihandler

import (
	"io"
	"log/slog"
	"net/http"

	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/usecase"
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
	service       cisvc.Service
	authenticator authz.Authenticator
}

func New(logger *slog.Logger, service cisvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) RegisterRepositoryRoutes(r router) {
	r.GET("/api/ci/repository", h.listRepositories)
	r.POST("/api/ci/repository", h.createRepository)
	r.GET("/api/ci/repository/:repository_id", h.getRepository)
	r.PUT("/api/ci/repository/:repository_id", h.updateRepository)
	r.DELETE("/api/ci/repository/:repository_id", h.deleteRepository)
	r.GET("/api/ci/repository/:repository_id/webhook", h.listRepositoryWebhooks)
	r.POST("/api/ci/repository/:repository_id/webhook", h.createRepositoryWebhook)
	r.PUT("/api/ci/repository/:repository_id/webhook/:webhook_id", h.updateRepositoryWebhook)
	r.DELETE("/api/ci/repository/:repository_id/webhook/:webhook_id", h.deleteRepositoryWebhook)
	r.GET("/api/ci/webhook/:webhook_id", h.getRepositoryWebhook)
	r.POST("/api/ci/webhook/:webhook_id", h.receiveRepositoryWebhook)
}

func (h Handler) listRepositories(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page := transportresponse.QueryInt(c.Request.URL.Query().Get("page"), 1)
	perPage := transportresponse.QueryInt(c.Request.URL.Query().Get("per_page"), 10)
	items, err := h.service.ListRepositories(c.Request.Context(), current.Id, transportresponse.QueryProjectId(c.Request.URL.Query().Get("project_id")), page, perPage, c.Request.URL.Query().Get("search"))
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := mapPage(items, repositoryListResponse)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.RepositoryPaginatedResp{Items: transportresponse.Ptrs(resp.Items), Total: int32(resp.Total), Page: int32(resp.Page), PerPage: int32(resp.PerPage), Pages: int32(transportresponse.PageCount(resp.Total, resp.PerPage))}})
}

func (h Handler) createRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RepositoryCreateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	detail, err := h.service.CreateRepository(c.Request.Context(), current.Id, cidto.RepositoryCreateInput{ProjectId: c.Request.URL.Query().Get("project_id"), Name: req.Name, Code: req.Code, RepositoryURL: req.RepositoryUrl, GitCredentialId: req.GitCredentialId, VariableOverrides: variableDeclarationRequestMaps(req.VariableOverrides), DefaultBranch: req.DefaultBranch})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := repositoryDetailResponse(detail)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getRepository(c *gin.Context) {
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
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) updateRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RepositoryUpdateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
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
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deleteRepository(c *gin.Context) {
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

func (h Handler) listRepositoryWebhooks(c *gin.Context) {
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
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.RepositoryWebhookListResp{Items: transportresponse.Ptrs(resp)}})
}

func (h Handler) createRepositoryWebhook(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.RepositoryWebhookCreateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	webhook, err := h.service.CreateRepositoryWebhook(c.Request.Context(), current.Id, c.Param("repository_id"), cidto.WebhookCreateInput{Name: req.Name, TemplateId: req.TemplateId, Secret: req.Secret, BranchFilter: req.BranchFilter})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := repositoryWebhookResponse(webhook)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getRepositoryWebhook(c *gin.Context) {
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
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) updateRepositoryWebhook(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req repositoryWebhookUpdateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	webhook, err := h.service.UpdateRepositoryWebhook(c.Request.Context(), current.Id, c.Param("repository_id"), c.Param("webhook_id"), cidto.WebhookUpdateInput{Name: req.Body.Name, TemplateId: req.Body.TemplateId, Secret: req.Body.Secret, BranchFilter: req.Body.BranchFilter, BranchSet: req.BranchSet, Enabled: req.Body.Enabled})
	if err != nil {
		h.writeError(c, err)
		return
	}
	resp := repositoryWebhookResponse(webhook)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deleteRepositoryWebhook(c *gin.Context) {
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

func (h Handler) receiveRepositoryWebhook(c *gin.Context) {
	payload, err := io.ReadAll(c.Request.Body)
	if err != nil {
		h.logger.Error("read webhook payload failed", "webhook_id", c.Param("webhook_id"), "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to read webhook payload"})
		return
	}
	result, err := h.service.ReceiveRepositoryWebhook(c.Request.Context(), cidto.WebhookReceiveInput{WebhookId: c.Param("webhook_id"), Headers: requestHeaders(c), Payload: payload})
	if err != nil {
		h.writeError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.RepositoryWebhookReceiveResp{Status: result.Status, Reason: result.Reason, RunId: result.RunId}})
}

func (h Handler) writeError(c *gin.Context, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("ci repository request failed", "error", err)
	}
	c.JSON(apperror.StatusCode(err), gin.H{"detail": err.Error()})
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

func mapPage[T any, U any](page repository.Page[T], convert func(T) U) repository.Page[U] {
	items := make([]U, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, convert(item))
	}
	return repository.Page[U]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}
}

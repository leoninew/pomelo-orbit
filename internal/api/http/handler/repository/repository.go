package repositoryhandler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"github.com/leoninew/pomelo-orbit/internal/api/http/binding"
	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"
	repositorydto "github.com/leoninew/pomelo-orbit/internal/application/repository/dto"
	repositoryv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/repository"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func (h Handler) ListRepositories(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	page, perPage := binding.QueryInt(c.Query("page"), 1), binding.QueryInt(c.Query("per_page"), 20)
	projectID := c.Query("project_id")
	var filter *string
	if projectID != "" {
		filter = &projectID
	}
	items, err := h.service.ListRepositories(c.Request.Context(), current.Id, filter, page, perPage, c.Query("search"))
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := make([]repositoryv1.RepositoryResp, 0, len(items.Items))
	for _, item := range items.Items {
		response = append(response, repositoryListResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &repositoryv1.RepositoryPaginatedResp{Items: transportresponse.Ptrs(response), Total: int32(items.Total), Page: int32(items.Page), PerPage: int32(items.PerPage), Pages: int32(transportresponse.PageCount(items.Total, items.PerPage))})
}
func (h Handler) CreateRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req repositoryv1.RepositoryCreateReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	detail, err := h.service.CreateRepository(c.Request.Context(), current.Id, repositorydto.RepositoryCreateInput{ProjectId: c.Query("project_id"), Name: req.Name, Code: req.Code, RepositoryType: req.RepositoryType, RepositoryUrl: req.RepositoryUrl, GitCredentialId: req.GitCredentialId, VariableOverrides: variableDeclarationRequestMaps(req.VariableOverrides), DefaultBranch: req.DefaultBranch})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := repositoryDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusCreated, &response)
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
	response := repositoryDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &response)
}
func (h Handler) UpdateRepository(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req repositoryv1.RepositoryUpdateReq
	if binding.DecodeJSON(c, &req) != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	var variables *[]map[string]any
	if req.VariableOverrides != nil {
		values := variableDeclarationRequestMaps(req.VariableOverrides.Items)
		variables = &values
	}
	detail, err := h.service.UpdateRepository(c.Request.Context(), current.Id, c.Param("repository_id"), repositorydto.RepositoryUpdateInput{Name: req.Name, RepositoryType: req.RepositoryType, RepositoryUrl: req.RepositoryUrl, GitCredentialId: req.GitCredentialId, VariableOverrides: variables, DefaultBranch: req.DefaultBranch})
	if err != nil {
		transportresponse.WriteError(c, err)
		return
	}
	response := repositoryDetailResponse(detail)
	transportresponse.ProtoJSON(c, http.StatusOK, &response)
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
func repositoryListResponse(item model.Repository) repositoryv1.RepositoryResp {
	return repositoryv1.RepositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryType: item.RepositoryType, RepositoryUrl: item.RepositoryUrl, HasCredential: item.GitCredentialId != nil, GitCredentialId: transportresponse.OptionalStringValue(item.GitCredentialId), DefaultBranch: item.DefaultBranch, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}
func repositoryDetailResponse(detail repositorydto.RepositoryDetail) repositoryv1.RepositoryResp {
	item := detail.Repository
	return repositoryv1.RepositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryType: item.RepositoryType, RepositoryUrl: item.RepositoryUrl, HasCredential: item.GitCredentialId != nil, GitCredentialId: transportresponse.OptionalStringValue(item.GitCredentialId), GitCredentialName: detail.GitCredentialName, VariableDeclarations: transportresponse.Ptrs(variableDeclarationResponses(detail.VariableDeclarations)), DefaultBranch: item.DefaultBranch, CreatedAt: transportresponse.FormatTime(item.CreatedAt), UpdatedAt: transportresponse.FormatTime(item.UpdatedAt)}
}

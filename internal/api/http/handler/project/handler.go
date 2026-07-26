package projecthandler

import (
	"log/slog"
	"net/http"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/binding"
	projectv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/project"
	"github.com/gin-gonic/gin"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	projectsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/project/usecase"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type Handler struct {
	logger        *slog.Logger
	service       projectsvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service projectsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) ListProjects(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListByMember(c.Request.Context(), current.Id)
	if err != nil {
		h.logger.Error("list projects failed", "user_id", current.Id, "error", err)
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	resp := make([]projectv1.ProjectResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, projectResponse(item))
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &projectv1.ProjectListResp{Items: transportresponse.Ptrs(resp)})
}

func (h Handler) CreateProject(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req projectv1.ProjectSaveReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	project, err := h.service.Create(c.Request.Context(), current.Id, projectSaveInput(&req))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	resp := projectResponse(project)
	transportresponse.ProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetProject(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	resp := projectResponse(project)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateProject(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	var req projectv1.ProjectSaveReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	updated, err := h.service.Update(c.Request.Context(), project, projectSaveInput(&req))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	resp := projectResponse(updated)
	transportresponse.ProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) DeprecateProject(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	project, ok := h.loadProjectForUser(c, current.Id)
	if !ok {
		return
	}
	var req projectv1.ProjectDeprecateReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.Deprecate(c.Request.Context(), project, current.Id); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) ListProjectMembers(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	h.writeProjectMembers(c, project.Id)
}

func (h Handler) AddProjectMember(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	var req projectv1.ProjectMemberReq
	if err := binding.DecodeJSON(c, &req); err != nil {
		transportresponse.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	members, err := h.service.AddMember(c.Request.Context(), project.Id, req.UserId)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &projectv1.ProjectMemberListResp{Items: transportresponse.Ptrs(projectMemberResponses(members))})
}

func (h Handler) RemoveProjectMember(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	members, err := h.service.RemoveMember(c.Request.Context(), project.Id, c.Param("user_id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &projectv1.ProjectMemberListResp{Items: transportresponse.Ptrs(projectMemberResponses(members))})
}

func (h Handler) writeProjectMembers(c *gin.Context, projectId string) {
	members, err := h.service.Members(c.Request.Context(), projectId)
	if err != nil {
		h.logger.Error("list project members failed", "project_id", projectId, "error", err)
		transportresponse.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	transportresponse.ProtoJSON(c, http.StatusOK, &projectv1.ProjectMemberListResp{Items: transportresponse.Ptrs(projectMemberResponses(members))})
}

func (h Handler) loadProjectForCurrentUser(c *gin.Context) (model.Project, bool) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return model.Project{}, false
	}
	return h.loadProjectForUser(c, current.Id)
}

func (h Handler) loadProjectForUser(c *gin.Context, userId string) (model.Project, bool) {
	projectId := strings.TrimSpace(c.Param("project_id"))
	project, err := h.service.LoadForUser(c.Request.Context(), projectId, userId)
	if err != nil {
		h.writeServiceError(c, err)
		return model.Project{}, false
	}
	return project, true
}

func (h Handler) writeServiceError(c *gin.Context, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("project request failed", "error", err)
	}
	transportresponse.WriteError(c, err)
}

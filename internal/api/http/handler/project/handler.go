package projecthandler

import (
	"log/slog"
	"net/http"
	"strings"

	"github.com/leoninew/pomelo-orbit/internal/api/http/transport"

	"github.com/gin-gonic/gin"
	projectv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/project"

	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	projectsvc "github.com/leoninew/pomelo-orbit/internal/application/project/usecase"
	handoversvc "github.com/leoninew/pomelo-orbit/internal/application/project_handover/usecase"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type Handler struct {
	logger        *slog.Logger
	service       projectsvc.Service
	handover      *handoversvc.Service
	authenticator security.Authenticator
}

func New(logger *slog.Logger, service projectsvc.Service, authenticator security.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) WithHandover(service handoversvc.Service) Handler {
	h.handover = &service
	return h
}

func (h Handler) ListProjects(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListByMember(c.Request.Context(), current.Id)
	if err != nil {
		h.logger.Error("list projects failed", "user_id", current.Id, "error", err)
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	resp := make([]projectv1.ProjectResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, projectResponse(item))
	}
	transport.WriteProtoJSON(c, http.StatusOK, &projectv1.ProjectListResp{Items: transport.Ptrs(resp)})
}

func (h Handler) CreateProject(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req projectv1.ProjectCreateReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	project, err := h.service.Create(c.Request.Context(), current.Id, projectCreateInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := projectResponse(project)
	transport.WriteProtoJSON(c, http.StatusCreated, &resp)
}

func (h Handler) GetProject(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	resp := projectResponse(project)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
}

func (h Handler) UpdateProject(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	var req projectv1.ProjectSaveReq
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	updated, err := h.service.Update(c.Request.Context(), project, projectSaveInput(&req))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	resp := projectResponse(updated)
	transport.WriteProtoJSON(c, http.StatusOK, &resp)
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
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	if err := h.service.Deprecate(c.Request.Context(), project, current.Id); err != nil {
		transport.WriteError(c, err)
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
	if err := transport.DecodeJSON(c, &req); err != nil {
		transport.WriteStatusError(c, http.StatusBadRequest, "Invalid JSON body")
		return
	}
	members, err := h.service.AddMember(c.Request.Context(), project.Id, req.UserId)
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &projectv1.ProjectMemberListResp{Items: transport.Ptrs(projectMemberResponses(members))})
}

func (h Handler) RemoveProjectMember(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	members, err := h.service.RemoveMember(c.Request.Context(), project.Id, c.Param("user_id"))
	if err != nil {
		transport.WriteError(c, err)
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &projectv1.ProjectMemberListResp{Items: transport.Ptrs(projectMemberResponses(members))})
}

func (h Handler) writeProjectMembers(c *gin.Context, projectId string) {
	members, err := h.service.Members(c.Request.Context(), projectId)
	if err != nil {
		h.logger.Error("list project members failed", "project_id", projectId, "error", err)
		transport.WriteError(c, apperror.Wrap(apperror.KindInternal, "", err))
		return
	}
	transport.WriteProtoJSON(c, http.StatusOK, &projectv1.ProjectMemberListResp{Items: transport.Ptrs(projectMemberResponses(members))})
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
		transport.WriteError(c, err)
		return model.Project{}, false
	}
	return project, true
}

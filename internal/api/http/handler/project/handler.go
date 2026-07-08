package projecthandler

import (
	"log/slog"
	"net/http"
	"strings"

	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"github.com/gin-gonic/gin"

	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"
	projectsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/project"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type router interface {
	GET(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	POST(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	PUT(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
	DELETE(relativePath string, handlers ...gin.HandlerFunc) gin.IRoutes
}

type Handler struct {
	logger        *slog.Logger
	service       projectsvc.Service
	authenticator authz.Authenticator
}

func New(logger *slog.Logger, service projectsvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) Register(r router) {
	r.GET("/api/project", h.listProjects)
	r.POST("/api/project", h.createProject)
	r.GET("/api/project/:project_id", h.getProject)
	r.PUT("/api/project/:project_id", h.updateProject)
	r.POST("/api/project/:project_id/deprecate", h.deprecateProject)
	r.GET("/api/project/:project_id/member", h.listProjectMembers)
	r.POST("/api/project/:project_id/member", h.addProjectMember)
	r.DELETE("/api/project/:project_id/member/:user_id", h.removeProjectMember)
}

func (h Handler) listProjects(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	items, err := h.service.ListByMember(c.Request.Context(), current.Id)
	if err != nil {
		h.logger.Error("list projects failed", "user_id", current.Id, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to list projects"})
		return
	}
	resp := make([]pomeloorbit.ProjectResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, ProjectResponse(item))
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ProjectListResp{Items: transportresponse.Ptrs(resp)}})
}

func (h Handler) createProject(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ProjectSaveReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	project, err := h.service.Create(c.Request.Context(), current.Id, projectsvc.SaveInput{Name: req.Name, Code: req.Code})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	resp := ProjectResponse(project)
	c.Render(http.StatusCreated, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) getProject(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	resp := ProjectResponse(project)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) updateProject(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ProjectSaveReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	updated, err := h.service.Update(c.Request.Context(), project, projectsvc.SaveInput{Name: req.Name, Code: req.Code})
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	resp := ProjectResponse(updated)
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &resp})
}

func (h Handler) deprecateProject(c *gin.Context) {
	current, ok := h.authenticator.CurrentUser(c)
	if !ok {
		return
	}
	project, ok := h.loadProjectForUser(c, current.Id)
	if !ok {
		return
	}
	var req pomeloorbit.ProjectDeprecateReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	if err := h.service.Deprecate(c.Request.Context(), project, current.Id); err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Status(http.StatusNoContent)
}

func (h Handler) listProjectMembers(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	h.writeProjectMembers(c, project.Id)
}

func (h Handler) addProjectMember(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	var req pomeloorbit.ProjectMemberReq
	if err := transportresponse.DecodeJSON(c, &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"detail": "Invalid JSON body"})
		return
	}
	members, err := h.service.AddMember(c.Request.Context(), project.Id, req.UserId)
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ProjectMemberListResp{Items: transportresponse.Ptrs(projectMemberResponses(members))}})
}

func (h Handler) removeProjectMember(c *gin.Context) {
	project, ok := h.loadProjectForCurrentUser(c)
	if !ok {
		return
	}
	members, err := h.service.RemoveMember(c.Request.Context(), project.Id, c.Param("user_id"))
	if err != nil {
		h.writeServiceError(c, err)
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ProjectMemberListResp{Items: transportresponse.Ptrs(projectMemberResponses(members))}})
}

func (h Handler) writeProjectMembers(c *gin.Context, projectId string) {
	members, err := h.service.Members(c.Request.Context(), projectId)
	if err != nil {
		h.logger.Error("list project members failed", "project_id", projectId, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"detail": "Failed to list project members"})
		return
	}
	c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.ProjectMemberListResp{Items: transportresponse.Ptrs(projectMemberResponses(members))}})
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
	c.JSON(apperror.StatusCode(err), gin.H{"detail": err.Error()})
}

func ProjectResponse(project model.Project) pomeloorbit.ProjectResp {
	return pomeloorbit.ProjectResp{
		Id:        project.Id,
		Name:      project.Name,
		Code:      project.Code,
		IsActive:  project.IsActive,
		CreatedAt: transportresponse.FormatTime(project.CreatedAt),
		UpdatedAt: transportresponse.FormatTime(project.UpdatedAt),
	}
}

func projectMemberResponses(users []model.User) []pomeloorbit.ProjectMemberResp {
	resp := make([]pomeloorbit.ProjectMemberResp, 0, len(users))
	for _, user := range users {
		resp = append(resp, ProjectMemberResponse(user))
	}
	return resp
}

func ProjectMemberResponse(user model.User) pomeloorbit.ProjectMemberResp {
	return pomeloorbit.ProjectMemberResp{Id: user.Id, Username: user.Username, Email: user.Email, Status: user.Status, AuthSource: user.AuthSource, LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt)}
}

package projecthandler

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"

	"backend/internal/apperror"
	"backend/internal/repository/model"
	projectsvc "backend/internal/service/project"
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
	service       projectsvc.Service
	authenticator authz.Authenticator
}

type ProjectResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type ProjectMemberResp struct {
	Id          string  `json:"id"`
	Username    string  `json:"username"`
	Email       *string `json:"email"`
	Status      string  `json:"status"`
	AuthSource  string  `json:"auth_source"`
	LastLoginAt *string `json:"last_login_at"`
}

type projectSaveReq struct {
	Name string `json:"name"`
	Code string `json:"code"`
}

type projectMemberReq struct {
	UserId string `json:"user_id"`
}

func New(logger *slog.Logger, service projectsvc.Service, authenticator authz.Authenticator) Handler {
	return Handler{logger: logger, service: service, authenticator: authenticator}
}

func (h Handler) Register(r router) {
	r.Get("/api/project", h.listProjects)
	r.Post("/api/project", h.createProject)
	r.Get("/api/project/{project_id}", h.getProject)
	r.Put("/api/project/{project_id}", h.updateProject)
	r.Post("/api/project/{project_id}/deprecate", h.deprecateProject)
	r.Get("/api/project/{project_id}/member", h.listProjectMembers)
	r.Post("/api/project/{project_id}/member", h.addProjectMember)
	r.Delete("/api/project/{project_id}/member/{user_id}", h.removeProjectMember)
}

func (h Handler) listProjects(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	items, err := h.service.ListByMember(r.Context(), current.Id)
	if err != nil {
		h.logger.Error("list projects failed", "user_id", current.Id, "error", err)
		transportresponse.JSON(h.logger, w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list projects"})
		return
	}
	resp := make([]ProjectResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, ProjectResponse(item))
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, resp)
}

func (h Handler) createProject(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	var req projectSaveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(h.logger, w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	project, err := h.service.Create(r.Context(), current.Id, projectsvc.SaveInput{Name: req.Name, Code: req.Code})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusCreated, ProjectResponse(project))
}

func (h Handler) getProject(w http.ResponseWriter, r *http.Request) {
	project, ok := h.loadProjectForCurrentUser(w, r)
	if !ok {
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, ProjectResponse(project))
}

func (h Handler) updateProject(w http.ResponseWriter, r *http.Request) {
	project, ok := h.loadProjectForCurrentUser(w, r)
	if !ok {
		return
	}
	var req projectSaveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(h.logger, w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	updated, err := h.service.Update(r.Context(), project, projectsvc.SaveInput{Name: req.Name, Code: req.Code})
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, ProjectResponse(updated))
}

func (h Handler) deprecateProject(w http.ResponseWriter, r *http.Request) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return
	}
	project, ok := h.loadProjectForUser(w, r, current.Id)
	if !ok {
		return
	}
	if err := h.service.Deprecate(r.Context(), project, current.Id); err != nil {
		h.writeServiceError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h Handler) listProjectMembers(w http.ResponseWriter, r *http.Request) {
	project, ok := h.loadProjectForCurrentUser(w, r)
	if !ok {
		return
	}
	h.writeProjectMembers(w, r, project.Id)
}

func (h Handler) addProjectMember(w http.ResponseWriter, r *http.Request) {
	project, ok := h.loadProjectForCurrentUser(w, r)
	if !ok {
		return
	}
	var req projectMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		transportresponse.JSON(h.logger, w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	members, err := h.service.AddMember(r.Context(), project.Id, req.UserId)
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, projectMemberResponses(members))
}

func (h Handler) removeProjectMember(w http.ResponseWriter, r *http.Request) {
	project, ok := h.loadProjectForCurrentUser(w, r)
	if !ok {
		return
	}
	members, err := h.service.RemoveMember(r.Context(), project.Id, chi.URLParam(r, "user_id"))
	if err != nil {
		h.writeServiceError(w, err)
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, projectMemberResponses(members))
}

func (h Handler) writeProjectMembers(w http.ResponseWriter, r *http.Request, projectId string) {
	members, err := h.service.Members(r.Context(), projectId)
	if err != nil {
		h.logger.Error("list project members failed", "project_id", projectId, "error", err)
		transportresponse.JSON(h.logger, w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list project members"})
		return
	}
	transportresponse.JSON(h.logger, w, http.StatusOK, projectMemberResponses(members))
}

func (h Handler) loadProjectForCurrentUser(w http.ResponseWriter, r *http.Request) (model.Project, bool) {
	current, ok := h.authenticator.CurrentUser(w, r)
	if !ok {
		return model.Project{}, false
	}
	return h.loadProjectForUser(w, r, current.Id)
}

func (h Handler) loadProjectForUser(w http.ResponseWriter, r *http.Request, userId string) (model.Project, bool) {
	projectId := strings.TrimSpace(chi.URLParam(r, "project_id"))
	project, err := h.service.LoadForUser(r.Context(), projectId, userId)
	if err != nil {
		h.writeServiceError(w, err)
		return model.Project{}, false
	}
	return project, true
}

func (h Handler) writeServiceError(w http.ResponseWriter, err error) {
	if apperror.StatusCode(err) == http.StatusInternalServerError {
		h.logger.Error("project request failed", "error", err)
	}
	transportresponse.JSON(h.logger, w, apperror.StatusCode(err), map[string]string{"detail": err.Error()})
}

func ProjectResponse(project model.Project) ProjectResp {
	return ProjectResp{
		Id:        project.Id,
		Name:      project.Name,
		Code:      project.Code,
		IsActive:  project.IsActive,
		CreatedAt: transportresponse.FormatTime(project.CreatedAt),
		UpdatedAt: transportresponse.FormatTime(project.UpdatedAt),
	}
}

func projectMemberResponses(users []model.User) []ProjectMemberResp {
	resp := make([]ProjectMemberResp, 0, len(users))
	for _, user := range users {
		resp = append(resp, ProjectMemberResponse(user))
	}
	return resp
}

func ProjectMemberResponse(user model.User) ProjectMemberResp {
	return ProjectMemberResp{Id: user.Id, Username: user.Username, Email: user.Email, Status: user.Status, AuthSource: user.AuthSource, LastLoginAt: transportresponse.FormatOptionalTime(user.LastLoginAt)}
}

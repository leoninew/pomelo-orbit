package httpserver

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"regexp"
	"strings"
	"time"

	"backend/internal/orbit"
)

var projectCodePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

type projectResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type projectMemberResp struct {
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

func (s Server) registerProjectRoutes(r chiRouter) {
	r.Get("/api/project", s.listProjects)
	r.Post("/api/project", s.createProject)
	r.Get("/api/project/{project_id}", s.getProject)
	r.Put("/api/project/{project_id}", s.updateProject)
	r.Post("/api/project/{project_id}/deprecate", s.deprecateProject)
	r.Get("/api/project/{project_id}/member", s.listProjectMembers)
	r.Post("/api/project/{project_id}/member", s.addProjectMember)
	r.Delete("/api/project/{project_id}/member/{user_id}", s.removeProjectMember)
}

func (s Server) listProjects(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	items, err := s.store.ListProjectsByMember(r.Context(), current.Id)
	if err != nil {
		s.logger.Error("list projects failed", "user_id", current.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list projects"})
		return
	}
	resp := make([]projectResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, projectResponse(item))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) createProject(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	var req projectSaveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeAndValidateProjectReq(w, &req.Name, &req.Code) {
		return
	}
	if !s.ensureProjectCodeAvailable(w, r, req.Code, "") {
		return
	}
	now := time.Now().UTC()
	project := orbit.Project{Id: orbit.NewId(), Name: req.Name, Code: req.Code, IsActive: true, CreatedAt: now, UpdatedAt: now}
	if err := s.store.CreateProject(r.Context(), project, current.Id); err != nil {
		s.logger.Error("create project failed", "project_code", project.Code, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create project"})
		return
	}
	writeJSON(w, http.StatusCreated, projectResponse(project))
}

func (s Server) getProject(w http.ResponseWriter, r *http.Request) {
	project, ok := s.loadProjectForCurrentUser(w, r)
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, projectResponse(project))
}

func (s Server) updateProject(w http.ResponseWriter, r *http.Request) {
	project, ok := s.loadProjectForCurrentUser(w, r)
	if !ok {
		return
	}
	var req projectSaveReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeAndValidateProjectReq(w, &req.Name, &req.Code) {
		return
	}
	if !s.ensureProjectCodeAvailable(w, r, req.Code, project.Id) {
		return
	}
	project.Name = req.Name
	project.Code = req.Code
	if err := s.store.UpdateProject(r.Context(), project); err != nil {
		s.logger.Error("update project failed", "project_id", project.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update project"})
		return
	}
	updated, err := s.store.Project(r.Context(), project.Id)
	if err != nil {
		s.logger.Error("load updated project failed", "project_id", project.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load project"})
		return
	}
	writeJSON(w, http.StatusOK, projectResponse(updated))
}

func (s Server) deprecateProject(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	project, ok := s.loadProjectForUser(w, r, current.Id)
	if !ok {
		return
	}
	activeProjects, err := s.store.ListActiveProjectsByMember(r.Context(), current.Id)
	if err != nil {
		s.logger.Error("list active projects failed", "user_id", current.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list active projects"})
		return
	}
	if len(activeProjects) <= 1 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Cannot deprecate the last active project"})
		return
	}
	repoCount, err := s.store.CountProjectRepositories(r.Context(), project.Id)
	if err != nil {
		s.logger.Error("count project repositories failed", "project_id", project.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to count project repositories"})
		return
	}
	if repoCount > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": fmt.Sprintf("Cannot deprecate project with %d repositories", repoCount)})
		return
	}
	appCount, err := s.store.CountProjectApplications(r.Context(), project.Id)
	if err != nil {
		s.logger.Error("count project applications failed", "project_id", project.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to count project applications"})
		return
	}
	if appCount > 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": fmt.Sprintf("Cannot deprecate project with %d applications", appCount)})
		return
	}
	if err := s.store.DeprecateProject(r.Context(), project.Id); err != nil {
		s.logger.Error("deprecate project failed", "project_id", project.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to deprecate project"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) listProjectMembers(w http.ResponseWriter, r *http.Request) {
	project, ok := s.loadProjectForCurrentUser(w, r)
	if !ok {
		return
	}
	s.writeProjectMembers(w, r, project.Id)
}

func (s Server) addProjectMember(w http.ResponseWriter, r *http.Request) {
	project, ok := s.loadProjectForCurrentUser(w, r)
	if !ok {
		return
	}
	var req projectMemberReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	req.UserId = strings.TrimSpace(req.UserId)
	if req.UserId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "user_id is required"})
		return
	}
	if _, err := s.store.UserById(r.Context(), req.UserId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "User " + req.UserId + " not found"})
			return
		}
		s.logger.Error("load member user failed", "user_id", req.UserId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load user"})
		return
	}
	if err := s.store.AddProjectMember(r.Context(), project.Id, req.UserId); err != nil {
		s.logger.Error("add project member failed", "project_id", project.Id, "user_id", req.UserId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to add project member"})
		return
	}
	s.writeProjectMembers(w, r, project.Id)
}

func (s Server) removeProjectMember(w http.ResponseWriter, r *http.Request) {
	project, ok := s.loadProjectForCurrentUser(w, r)
	if !ok {
		return
	}
	userId := urlParam(r, "user_id")
	if err := s.store.RemoveProjectMember(r.Context(), project.Id, userId); err != nil {
		s.logger.Error("remove project member failed", "project_id", project.Id, "user_id", userId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to remove project member"})
		return
	}
	s.writeProjectMembers(w, r, project.Id)
}

func (s Server) writeProjectMembers(w http.ResponseWriter, r *http.Request, projectId string) {
	members, err := s.store.ProjectMembers(r.Context(), projectId)
	if err != nil {
		s.logger.Error("list project members failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list project members"})
		return
	}
	resp := make([]projectMemberResp, 0, len(members))
	for _, member := range members {
		resp = append(resp, projectMemberResponse(member))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) loadProjectForCurrentUser(w http.ResponseWriter, r *http.Request) (orbit.Project, bool) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return orbit.Project{}, false
	}
	return s.loadProjectForUser(w, r, current.Id)
}

func (s Server) loadProjectForUser(w http.ResponseWriter, r *http.Request, userId string) (orbit.Project, bool) {
	projectId := urlParam(r, "project_id")
	project, err := s.store.Project(r.Context(), projectId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Project " + projectId + " not found"})
			return orbit.Project{}, false
		}
		s.logger.Error("load project failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load project"})
		return orbit.Project{}, false
	}
	member, err := s.store.IsProjectMember(r.Context(), project.Id, userId)
	if err != nil {
		s.logger.Error("check project member failed", "project_id", project.Id, "user_id", userId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check project member"})
		return orbit.Project{}, false
	}
	if !member {
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": "Permission denied"})
		return orbit.Project{}, false
	}
	return project, true
}

func (s Server) ensureProjectCodeAvailable(w http.ResponseWriter, r *http.Request, code string, currentProjectId string) bool {
	existing, err := s.store.ProjectByCode(r.Context(), code)
	if err == nil {
		if existing.Id != currentProjectId {
			writeJSON(w, http.StatusConflict, map[string]string{"detail": "Project code " + code + " already exists"})
			return false
		}
		return true
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("check project code failed", "project_code", code, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check project code"})
		return false
	}
	return true
}

func normalizeAndValidateProjectReq(w http.ResponseWriter, name *string, code *string) bool {
	*name = strings.TrimSpace(*name)
	*code = strings.TrimSpace(*code)
	if *name == "" || len(*name) > 100 || *code == "" || len(*code) > 100 || !projectCodePattern.MatchString(*code) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid project fields"})
		return false
	}
	return true
}

func projectResponse(project orbit.Project) projectResp {
	return projectResp{
		Id:        project.Id,
		Name:      project.Name,
		Code:      project.Code,
		IsActive:  project.IsActive,
		CreatedAt: formatTime(project.CreatedAt),
		UpdatedAt: formatTime(project.UpdatedAt),
	}
}

func projectMemberResponse(user orbit.User) projectMemberResp {
	return projectMemberResp{Id: user.Id, Username: user.Username, Email: user.Email, Status: user.Status, AuthSource: user.AuthSource, LastLoginAt: formatOptionalTime(user.LastLoginAt)}
}

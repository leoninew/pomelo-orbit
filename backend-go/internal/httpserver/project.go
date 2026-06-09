package httpserver

import (
	"net/http"

	"backend/internal/orbit"
)

type projectResp struct {
	Id        string `json:"id"`
	Name      string `json:"name"`
	Code      string `json:"code"`
	IsActive  bool   `json:"is_active"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

func (s Server) registerProjectRoutes(r chiRouter) {
	r.Get("/api/project", s.listProjects)
	r.Get("/api/project/{project_id}", s.getProject)
}

func (s Server) listProjects(w http.ResponseWriter, r *http.Request) {
	items, err := s.store.ListProjects(r.Context())
	if err != nil {
		s.logger.Error("list projects failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list projects"})
		return
	}
	resp := make([]projectResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, projectResponse(item))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) getProject(w http.ResponseWriter, r *http.Request) {
	projectId := urlParam(r, "project_id")
	item, err := s.store.Project(r.Context(), projectId)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Project not found"})
		return
	}
	writeJSON(w, http.StatusOK, projectResponse(item))
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

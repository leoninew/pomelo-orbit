package httpserver

import (
	"net/http"

	"backend/internal/orbit"
)

type repositoryResp struct {
	Id                string  `json:"id"`
	ProjectId         *string `json:"project_id"`
	Name              string  `json:"name"`
	Code              string  `json:"code"`
	RepositoryURL     string  `json:"repository_url"`
	GitCredentialId   *string `json:"git_credential_id"`
	VariableOverrides string  `json:"variable_overrides"`
	DefaultBranch     string  `json:"default_branch"`
	CreatedAt         string  `json:"created_at"`
	UpdatedAt         string  `json:"updated_at"`
}

type pipelineRunResp struct {
	Id                string  `json:"id"`
	ProjectId         *string `json:"project_id"`
	RepositoryId      string  `json:"repository_id"`
	RepositoryName    string  `json:"repository_name"`
	SnapshotId        string  `json:"snapshot_id"`
	TemplateId        string  `json:"template_id"`
	TemplateName      string  `json:"template_name"`
	TemplateVersion   int     `json:"template_version"`
	Trigger           string  `json:"trigger"`
	TriggerRef        string  `json:"trigger_ref"`
	VariablesSnapshot string  `json:"variables_snapshot"`
	Status            string  `json:"status"`
	RetryOf           *string `json:"retry_of"`
	StartedAt         *string `json:"started_at"`
	FinishedAt        *string `json:"finished_at"`
	ErrorMessage      *string `json:"error_message"`
	CreatedAt         string  `json:"created_at"`
}

type applicationResp struct {
	Id              string  `json:"id"`
	ProjectId       *string `json:"project_id"`
	Name            string  `json:"name"`
	Code            string  `json:"code"`
	ImagePullPolicy string  `json:"image_pull_policy"`
	Status          string  `json:"status"`
	RouteManaged    bool    `json:"route_managed"`
	CreatedAt       string  `json:"created_at"`
	UpdatedAt       string  `json:"updated_at"`
}

type deploymentResp struct {
	Id                       string  `json:"id"`
	ProjectId                *string `json:"project_id"`
	ApplicationId            *string `json:"application_id"`
	ApplicationName          string  `json:"application_name"`
	OperationType            string  `json:"operation_type"`
	TriggerType              string  `json:"trigger_type"`
	Status                   string  `json:"status"`
	StartedAt                string  `json:"started_at"`
	FinishedAt               *string `json:"finished_at"`
	DurationMs               *int    `json:"duration_ms"`
	LogText                  *string `json:"log_text"`
	ErrorMessage             *string `json:"error_message"`
	IsRollback               bool    `json:"is_rollback"`
	RollbackFromDeploymentId *string `json:"rollback_from_deployment_id"`
}

func (s Server) registerDashboardRoutes(r chiRouter) {
	r.Get("/api/ci/repository", s.listRepositories)
	r.Get("/api/ci/run", s.listPipelineRuns)
	r.Get("/api/cd/application", s.listApplications)
	r.Get("/api/cd/deployment", s.listDeployments)
}

func (s Server) listRepositories(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	items, err := s.store.ListRepositories(r.Context(), queryProjectId(r.URL.Query().Get("project_id")), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list repositories failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list repositories"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, repositoryResponse)))
}

func (s Server) listPipelineRuns(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	items, err := s.store.ListPipelineRuns(r.Context(), queryProjectId(r.URL.Query().Get("project_id")), page, perPage)
	if err != nil {
		s.logger.Error("list pipeline runs failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list pipeline runs"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, pipelineRunResponse)))
}

func (s Server) listApplications(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	items, err := s.store.ListApplications(r.Context(), queryProjectId(r.URL.Query().Get("project_id")), page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list applications failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list applications"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, applicationResponse)))
}

func (s Server) listDeployments(w http.ResponseWriter, r *http.Request) {
	page, perPage := pageParams(r)
	items, err := s.store.ListDeployments(r.Context(), queryProjectId(r.URL.Query().Get("project_id")), page, perPage)
	if err != nil {
		s.logger.Error("list deployments failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list deployments"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, deploymentResponse)))
}

func pageParams(r *http.Request) (int, int) {
	return queryInt(r.URL.Query().Get("page"), 1), queryInt(r.URL.Query().Get("per_page"), 10)
}

func mapPage[T any, U any](page orbit.Page[T], convert func(T) U) orbit.Page[U] {
	items := make([]U, 0, len(page.Items))
	for _, item := range page.Items {
		items = append(items, convert(item))
	}
	return orbit.Page[U]{Items: items, Total: page.Total, Page: page.Page, PerPage: page.PerPage}
}

func repositoryResponse(item orbit.Repository) repositoryResp {
	return repositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryURL: item.RepositoryURL, GitCredentialId: item.GitCredentialId, VariableOverrides: item.VariableOverrides, DefaultBranch: item.DefaultBranch, CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt)}
}

func pipelineRunResponse(item orbit.PipelineRun) pipelineRunResp {
	return pipelineRunResp{Id: item.Id, ProjectId: item.ProjectId, RepositoryId: item.RepositoryId, RepositoryName: item.RepositoryName, SnapshotId: item.SnapshotId, TemplateId: item.TemplateId, TemplateName: item.TemplateName, TemplateVersion: item.TemplateVersion, Trigger: item.Trigger, TriggerRef: item.TriggerRef, VariablesSnapshot: item.VariablesSnapshot, Status: item.Status, RetryOf: item.RetryOf, StartedAt: formatOptionalTime(item.StartedAt), FinishedAt: formatOptionalTime(item.FinishedAt), ErrorMessage: item.ErrorMessage, CreatedAt: formatTime(item.CreatedAt)}
}

func applicationResponse(item orbit.Application) applicationResp {
	return applicationResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, ImagePullPolicy: item.ImagePullPolicy, Status: item.Status, RouteManaged: item.RouteManaged, CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt)}
}

func deploymentResponse(item orbit.Deployment) deploymentResp {
	return deploymentResp{Id: item.Id, ProjectId: item.ProjectId, ApplicationId: item.ApplicationId, ApplicationName: item.ApplicationName, OperationType: item.OperationType, TriggerType: item.TriggerType, Status: item.Status, StartedAt: formatTime(item.StartedAt), FinishedAt: formatOptionalTime(item.FinishedAt), DurationMs: item.DurationMs, LogText: item.LogText, ErrorMessage: item.ErrorMessage, IsRollback: item.IsRollback, RollbackFromDeploymentId: item.RollbackFromDeploymentId}
}

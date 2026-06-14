package httpserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"regexp"
	"strings"

	"backend/internal/orbit"
	"backend/internal/status"
)

var repositoryCodePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

type repositoryResp struct {
	Id                   string           `json:"id"`
	ProjectId            *string          `json:"project_id,omitempty"`
	Name                 string           `json:"name"`
	Code                 string           `json:"code"`
	RepositoryURL        string           `json:"repository_url"`
	HasCredential        bool             `json:"has_credential"`
	GitCredentialId      *string          `json:"git_credential_id"`
	GitCredentialName    *string          `json:"git_credential_name,omitempty"`
	VariableDeclarations []map[string]any `json:"variable_declarations,omitempty"`
	DefaultBranch        string           `json:"default_branch"`
	CreatedAt            string           `json:"created_at"`
	UpdatedAt            string           `json:"updated_at"`
}

type repositoryCreateReq struct {
	Name              string           `json:"name"`
	Code              string           `json:"code"`
	RepositoryURL     string           `json:"repository_url"`
	GitCredentialId   *string          `json:"git_credential_id"`
	VariableOverrides []map[string]any `json:"variable_overrides"`
	DefaultBranch     string           `json:"default_branch"`
}

type repositoryUpdateReq struct {
	Name              *string           `json:"name"`
	RepositoryURL     *string           `json:"repository_url"`
	GitCredentialId   *string           `json:"git_credential_id"`
	VariableOverrides *[]map[string]any `json:"variable_overrides"`
	DefaultBranch     *string           `json:"default_branch"`
}

type pipelineRunTriggerReq struct {
	TemplateId string            `json:"template_id"`
	TriggerRef string            `json:"trigger_ref"`
	Variables  map[string]string `json:"variables"`
}

type repositoryWebhookResp struct {
	Id           string  `json:"id"`
	RepositoryId string  `json:"repository_id"`
	Name         string  `json:"name"`
	TemplateId   string  `json:"template_id"`
	BranchFilter *string `json:"branch_filter"`
	Enabled      bool    `json:"enabled"`
	CreatedAt    string  `json:"created_at"`
	UpdatedAt    string  `json:"updated_at"`
}

type repositoryWebhookCreateReq struct {
	Name         string  `json:"name"`
	TemplateId   string  `json:"template_id"`
	Secret       string  `json:"secret"`
	BranchFilter *string `json:"branch_filter"`
}

type repositoryWebhookUpdateReq struct {
	Name         *string `json:"name"`
	TemplateId   *string `json:"template_id"`
	Secret       *string `json:"secret"`
	BranchFilter *string `json:"branch_filter"`
	Enabled      *bool   `json:"enabled"`
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
	r.Post("/api/ci/repository", s.createRepository)
	r.Get("/api/ci/repository/{repository_id}", s.getRepository)
	r.Put("/api/ci/repository/{repository_id}", s.updateRepository)
	r.Delete("/api/ci/repository/{repository_id}", s.deleteRepository)
	r.Get("/api/ci/repository/{repository_id}/run", s.listRepositoryRuns)
	r.Get("/api/ci/repository/{repository_id}/webhook", s.listRepositoryWebhooks)
	r.Post("/api/ci/repository/{repository_id}/webhook", s.createRepositoryWebhook)
	r.Put("/api/ci/repository/{repository_id}/webhook/{webhook_id}", s.updateRepositoryWebhook)
	r.Delete("/api/ci/repository/{repository_id}/webhook/{webhook_id}", s.deleteRepositoryWebhook)
	r.Post("/api/ci/repository/{repository_id}/trigger", s.triggerRepository)
	r.Get("/api/ci/webhook/{webhook_id}", s.getRepositoryWebhook)
	r.Post("/api/ci/webhook/{webhook_id}", s.receiveRepositoryWebhook)
	r.Get("/api/ci/run", s.listPipelineRuns)
	r.Get("/api/cd/application", s.listApplications)
	r.Get("/api/cd/deployment", s.listDeployments)
}

func (s Server) listRepositories(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := queryProjectId(r.URL.Query().Get("project_id"))
	if projectId != nil {
		if !s.ensureProjectMembership(w, r, *projectId, current.Id) {
			return
		}
	}
	page, perPage := pageParams(r)
	items, err := s.store.ListRepositories(r.Context(), projectId, page, perPage, r.URL.Query().Get("search"))
	if err != nil {
		s.logger.Error("list repositories failed", "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list repositories"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, repositoryListResponse)))
}

func (s Server) createRepository(w http.ResponseWriter, r *http.Request) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return
	}
	projectId := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "project_id is required"})
		return
	}
	if !s.ensureProjectMembership(w, r, projectId, current.Id) {
		return
	}
	var req repositoryCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeRepositoryCreateReq(w, &req) {
		return
	}
	if !s.ensureRepositoryCodeAvailable(w, r, &projectId, req.Code) || !s.ensureRepositoryCredential(w, r, req.GitCredentialId) {
		return
	}
	overrides, ok := marshalVariableOverrides(w, req.VariableOverrides)
	if !ok {
		return
	}
	repo := orbit.Repository{Id: orbit.NewId(), ProjectId: &projectId, Name: req.Name, Code: req.Code, RepositoryURL: req.RepositoryURL, GitCredentialId: req.GitCredentialId, VariableOverrides: overrides, DefaultBranch: req.DefaultBranch}
	if err := s.store.CreateRepository(r.Context(), repo); err != nil {
		s.logger.Error("create repository failed", "repository_code", repo.Code, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create repository"})
		return
	}
	created, err := s.store.Repository(r.Context(), repo.Id)
	if err != nil {
		s.logger.Error("load created repository failed", "repository_id", repo.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load repository"})
		return
	}
	s.writeRepositoryDetail(w, r, created, http.StatusCreated)
}

func (s Server) getRepository(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.loadRepositoryForCurrentUser(w, r)
	if !ok {
		return
	}
	s.writeRepositoryDetail(w, r, repo, http.StatusOK)
}

func (s Server) updateRepository(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.loadRepositoryForCurrentUser(w, r)
	if !ok {
		return
	}
	var req repositoryUpdateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository fields"})
			return
		}
		repo.Name = name
	}
	if req.RepositoryURL != nil {
		url := strings.TrimSpace(*req.RepositoryURL)
		if url == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository fields"})
			return
		}
		repo.RepositoryURL = url
	}
	if req.GitCredentialId != nil {
		credentialId := strings.TrimSpace(*req.GitCredentialId)
		if credentialId == "" {
			repo.GitCredentialId = nil
		} else {
			repo.GitCredentialId = &credentialId
		}
		if !s.ensureRepositoryCredential(w, r, repo.GitCredentialId) {
			return
		}
	}
	if req.VariableOverrides != nil {
		overrides, ok := marshalVariableOverrides(w, *req.VariableOverrides)
		if !ok {
			return
		}
		repo.VariableOverrides = overrides
	}
	if req.DefaultBranch != nil {
		branch := strings.TrimSpace(*req.DefaultBranch)
		if branch == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository fields"})
			return
		}
		repo.DefaultBranch = branch
	}
	if err := s.store.UpdateRepository(r.Context(), repo); err != nil {
		s.logger.Error("update repository failed", "repository_id", repo.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update repository"})
		return
	}
	updated, err := s.store.Repository(r.Context(), repo.Id)
	if err != nil {
		s.logger.Error("load updated repository failed", "repository_id", repo.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load repository"})
		return
	}
	s.writeRepositoryDetail(w, r, updated, http.StatusOK)
}

func (s Server) deleteRepository(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.loadRepositoryForCurrentUser(w, r)
	if !ok {
		return
	}
	running, err := s.store.RepositoryHasRunningPipelines(r.Context(), repo.Id)
	if err != nil {
		s.logger.Error("check repository pipelines failed", "repository_id", repo.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check repository pipelines"})
		return
	}
	if running {
		writeJSON(w, http.StatusConflict, map[string]string{"detail": "Repository has running pipelines, cannot delete"})
		return
	}
	if err := s.store.DeleteRepository(r.Context(), repo.Id); err != nil {
		s.logger.Error("delete repository failed", "repository_id", repo.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete repository"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) listRepositoryRuns(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.loadRepositoryForCurrentUser(w, r)
	if !ok {
		return
	}
	page, perPage := pageParams(r)
	items, err := s.store.ListPipelineRunsByRepository(r.Context(), repo.Id, page, perPage)
	if err != nil {
		s.logger.Error("list repository runs failed", "repository_id", repo.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list repository runs"})
		return
	}
	writeJSON(w, http.StatusOK, newPaginatedResp(mapPage(items, pipelineRunResponse)))
}

func (s Server) listRepositoryWebhooks(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.loadRepositoryForCurrentUser(w, r)
	if !ok {
		return
	}
	items, err := s.store.ListRepositoryWebhooks(r.Context(), repo.Id)
	if err != nil {
		s.logger.Error("list repository webhooks failed", "repository_id", repo.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to list repository webhooks"})
		return
	}
	resp := make([]repositoryWebhookResp, 0, len(items))
	for _, item := range items {
		resp = append(resp, repositoryWebhookResponse(item))
	}
	writeJSON(w, http.StatusOK, resp)
}

func (s Server) createRepositoryWebhook(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.loadRepositoryForCurrentUser(w, r)
	if !ok {
		return
	}
	var req repositoryWebhookCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !normalizeRepositoryWebhookCreateReq(w, &req) || !s.ensurePipelineTemplate(w, r, req.TemplateId) {
		return
	}
	webhook := orbit.RepositoryWebhook{Id: orbit.NewId(), RepositoryId: repo.Id, Name: req.Name, TemplateId: req.TemplateId, BranchFilter: req.BranchFilter, EncryptedSecret: req.Secret, Enabled: true}
	if err := s.store.CreateRepositoryWebhook(r.Context(), webhook); err != nil {
		s.logger.Error("create repository webhook failed", "repository_id", repo.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create repository webhook"})
		return
	}
	created, err := s.store.RepositoryWebhook(r.Context(), webhook.Id)
	if err != nil {
		s.logger.Error("load created repository webhook failed", "webhook_id", webhook.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load repository webhook"})
		return
	}
	writeJSON(w, http.StatusCreated, repositoryWebhookResponse(created))
}

func (s Server) getRepositoryWebhook(w http.ResponseWriter, r *http.Request) {
	webhook, ok := s.loadRepositoryWebhookForCurrentUser(w, r, urlParam(r, "webhook_id"))
	if !ok {
		return
	}
	writeJSON(w, http.StatusOK, repositoryWebhookResponse(webhook))
}

func (s Server) updateRepositoryWebhook(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.loadRepositoryForCurrentUser(w, r)
	if !ok {
		return
	}
	webhook, ok := s.loadRepositoryWebhook(w, r, urlParam(r, "webhook_id"))
	if !ok {
		return
	}
	if webhook.RepositoryId != repo.Id {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Webhook " + webhook.Id + " not found"})
		return
	}
	var req map[string]json.RawMessage
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	if !applyRepositoryWebhookUpdateReq(w, r, s, &webhook, req) {
		return
	}
	if err := s.store.UpdateRepositoryWebhook(r.Context(), webhook); err != nil {
		s.logger.Error("update repository webhook failed", "webhook_id", webhook.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to update repository webhook"})
		return
	}
	updated, err := s.store.RepositoryWebhook(r.Context(), webhook.Id)
	if err != nil {
		s.logger.Error("load updated repository webhook failed", "webhook_id", webhook.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load repository webhook"})
		return
	}
	writeJSON(w, http.StatusOK, repositoryWebhookResponse(updated))
}

func (s Server) deleteRepositoryWebhook(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.loadRepositoryForCurrentUser(w, r)
	if !ok {
		return
	}
	webhook, ok := s.loadRepositoryWebhook(w, r, urlParam(r, "webhook_id"))
	if !ok {
		return
	}
	if webhook.RepositoryId != repo.Id {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Webhook " + webhook.Id + " not found"})
		return
	}
	if err := s.store.DeleteRepositoryWebhook(r.Context(), webhook.Id); err != nil {
		s.logger.Error("delete repository webhook failed", "webhook_id", webhook.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to delete repository webhook"})
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (s Server) receiveRepositoryWebhook(w http.ResponseWriter, r *http.Request) {
	webhook, ok := s.loadRepositoryWebhook(w, r, urlParam(r, "webhook_id"))
	if !ok {
		return
	}
	if !webhook.Enabled {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "webhook disabled"})
		return
	}
	payload, err := io.ReadAll(r.Body)
	if err != nil {
		s.logger.Error("read webhook payload failed", "webhook_id", webhook.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to read webhook payload"})
		return
	}
	var body map[string]any
	if err := json.Unmarshal(payload, &body); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON payload"})
		return
	}
	branch, commitSha, author, ok := s.parseWebhookEvent(w, r, webhook, payload, body)
	if !ok {
		return
	}
	if webhook.BranchFilter == nil {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "branch filtered"})
		return
	}
	if *webhook.BranchFilter != "*" && !matchBranchFilter(branch, *webhook.BranchFilter) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ignored", "reason": "branch filtered"})
		return
	}
	repo, err := s.store.Repository(r.Context(), webhook.RepositoryId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Repository " + webhook.RepositoryId + " not found"})
			return
		}
		s.logger.Error("load repository failed", "repository_id", webhook.RepositoryId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load repository"})
		return
	}
	template, err := s.store.PipelineTemplate(r.Context(), webhook.TemplateId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Pipeline template " + webhook.TemplateId + " not found"})
			return
		}
		s.logger.Error("load pipeline template failed", "template_id", webhook.TemplateId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline template"})
		return
	}
	snapshot, err := s.store.LatestPipelineSnapshot(r.Context(), webhook.TemplateId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Pipeline snapshot for template " + webhook.TemplateId + " not found"})
			return
		}
		s.logger.Error("load pipeline snapshot failed", "template_id", webhook.TemplateId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline snapshot"})
		return
	}
	variablesSnapshot, ok := marshalTriggerVariables(w, map[string]string{"commit_sha": commitSha, "author": author, "event_type": "push"})
	if !ok {
		return
	}
	triggerRef := branch
	if triggerRef == "" {
		triggerRef = commitSha
	}
	run := orbit.PipelineRun{Id: orbit.NewId(), ProjectId: repo.ProjectId, RepositoryId: repo.Id, RepositoryName: repo.Name, SnapshotId: snapshot.Id, TemplateId: template.Id, TemplateName: template.Name, TemplateVersion: snapshot.Version, Trigger: "webhook", TriggerRef: triggerRef, VariablesSnapshot: variablesSnapshot, Status: orbit.WorkStatusWaitingToRun}
	if err := s.store.CreatePipelineRun(r.Context(), run); err != nil {
		s.logger.Error("create webhook pipeline run failed", "webhook_id", webhook.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create pipeline run"})
		return
	}
	payloadJSON, _ := json.Marshal(map[string]any{"pipeline_run_id": run.Id, "variables": map[string]string{"commit_sha": commitSha, "author": author, "event_type": "push"}})
	if err := s.tasks.Enqueue(r.Context(), orbit.NewId(), status.TaskTypeCIPipelineRunExecute, string(payloadJSON), s.defaultMaxAttempts); err != nil {
		s.logger.Error("enqueue webhook pipeline run failed", "run_id", run.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to enqueue pipeline run"})
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "triggered", "run_id": run.Id})
}

func (s Server) triggerRepository(w http.ResponseWriter, r *http.Request) {
	repo, ok := s.loadRepositoryForCurrentUser(w, r)
	if !ok {
		return
	}
	var req pipelineRunTriggerReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid JSON body"})
		return
	}
	req.TemplateId = strings.TrimSpace(req.TemplateId)
	if req.TemplateId == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "template_id is required"})
		return
	}
	template, err := s.store.PipelineTemplate(r.Context(), req.TemplateId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Pipeline template " + req.TemplateId + " not found"})
			return
		}
		s.logger.Error("load pipeline template failed", "template_id", req.TemplateId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline template"})
		return
	}
	snapshot, err := s.store.LatestPipelineSnapshot(r.Context(), req.TemplateId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Pipeline snapshot for template " + req.TemplateId + " not found"})
			return
		}
		s.logger.Error("load pipeline snapshot failed", "template_id", req.TemplateId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline snapshot"})
		return
	}
	variablesSnapshot, ok := marshalTriggerVariables(w, req.Variables)
	if !ok {
		return
	}
	triggerRef := strings.TrimSpace(req.TriggerRef)
	if triggerRef == "" {
		triggerRef = repo.DefaultBranch
	}
	run := orbit.PipelineRun{Id: orbit.NewId(), ProjectId: repo.ProjectId, RepositoryId: repo.Id, RepositoryName: repo.Name, SnapshotId: snapshot.Id, TemplateId: template.Id, TemplateName: template.Name, TemplateVersion: snapshot.Version, Trigger: "manual", TriggerRef: triggerRef, VariablesSnapshot: variablesSnapshot, Status: orbit.WorkStatusWaitingToRun}
	if err := s.store.CreatePipelineRun(r.Context(), run); err != nil {
		s.logger.Error("create pipeline run failed", "repository_id", repo.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to create pipeline run"})
		return
	}
	payload, _ := json.Marshal(map[string]any{"pipeline_run_id": run.Id, "variables": req.Variables})
	if err := s.tasks.Enqueue(r.Context(), orbit.NewId(), status.TaskTypeCIPipelineRunExecute, string(payload), s.defaultMaxAttempts); err != nil {
		s.logger.Error("enqueue pipeline run failed", "run_id", run.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to enqueue pipeline run"})
		return
	}
	created, err := s.store.PipelineRun(r.Context(), run.Id)
	if err != nil {
		s.logger.Error("load created pipeline run failed", "run_id", run.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline run"})
		return
	}
	writeJSON(w, http.StatusCreated, pipelineRunResponse(created))
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

func repositoryListResponse(item orbit.Repository) repositoryResp {
	return repositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryURL: item.RepositoryURL, HasCredential: item.GitCredentialId != nil, GitCredentialId: item.GitCredentialId, DefaultBranch: item.DefaultBranch, CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt)}
}

func (s Server) writeRepositoryDetail(w http.ResponseWriter, r *http.Request, item orbit.Repository, status int) {
	credentialName, err := s.repositoryCredentialName(r, item.GitCredentialId)
	if err != nil {
		s.logger.Error("load repository credential failed", "repository_id", item.Id, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load repository credential"})
		return
	}
	variables, ok := repositoryVariables(w, item.VariableOverrides)
	if !ok {
		return
	}
	writeJSON(w, status, repositoryResp{Id: item.Id, ProjectId: item.ProjectId, Name: item.Name, Code: item.Code, RepositoryURL: item.RepositoryURL, HasCredential: item.GitCredentialId != nil, GitCredentialId: item.GitCredentialId, GitCredentialName: credentialName, VariableDeclarations: variables, DefaultBranch: item.DefaultBranch, CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt)})
}

func (s Server) repositoryCredentialName(r *http.Request, credentialId *string) (*string, error) {
	if credentialId == nil || strings.TrimSpace(*credentialId) == "" {
		return nil, nil
	}
	return s.store.CredentialName(r.Context(), *credentialId)
}

func repositoryWebhookResponse(item orbit.RepositoryWebhook) repositoryWebhookResp {
	return repositoryWebhookResp{Id: item.Id, RepositoryId: item.RepositoryId, Name: item.Name, TemplateId: item.TemplateId, BranchFilter: item.BranchFilter, Enabled: item.Enabled, CreatedAt: formatTime(item.CreatedAt), UpdatedAt: formatTime(item.UpdatedAt)}
}

func (s Server) loadRepositoryForCurrentUser(w http.ResponseWriter, r *http.Request) (orbit.Repository, bool) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return orbit.Repository{}, false
	}
	repositoryId := urlParam(r, "repository_id")
	repo, err := s.store.Repository(r.Context(), repositoryId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Repository " + repositoryId + " not found"})
			return orbit.Repository{}, false
		}
		s.logger.Error("load repository failed", "repository_id", repositoryId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load repository"})
		return orbit.Repository{}, false
	}
	if repo.ProjectId != nil && !s.ensureProjectMembership(w, r, *repo.ProjectId, current.Id) {
		return orbit.Repository{}, false
	}
	return repo, true
}

func (s Server) loadRepositoryWebhookForCurrentUser(w http.ResponseWriter, r *http.Request, webhookId string) (orbit.RepositoryWebhook, bool) {
	current, ok := s.currentUser(w, r)
	if !ok {
		return orbit.RepositoryWebhook{}, false
	}
	webhook, ok := s.loadRepositoryWebhook(w, r, webhookId)
	if !ok {
		return orbit.RepositoryWebhook{}, false
	}
	repo, err := s.store.Repository(r.Context(), webhook.RepositoryId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Repository " + webhook.RepositoryId + " not found"})
			return orbit.RepositoryWebhook{}, false
		}
		s.logger.Error("load repository failed", "repository_id", webhook.RepositoryId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load repository"})
		return orbit.RepositoryWebhook{}, false
	}
	if repo.ProjectId != nil && !s.ensureProjectMembership(w, r, *repo.ProjectId, current.Id) {
		return orbit.RepositoryWebhook{}, false
	}
	return webhook, true
}

func (s Server) loadRepositoryWebhook(w http.ResponseWriter, r *http.Request, webhookId string) (orbit.RepositoryWebhook, bool) {
	webhook, err := s.store.RepositoryWebhook(r.Context(), webhookId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Webhook " + webhookId + " not found"})
			return orbit.RepositoryWebhook{}, false
		}
		s.logger.Error("load repository webhook failed", "webhook_id", webhookId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load repository webhook"})
		return orbit.RepositoryWebhook{}, false
	}
	return webhook, true
}

func (s Server) ensureProjectMembership(w http.ResponseWriter, r *http.Request, projectId string, userId string) bool {
	if _, err := s.store.Project(r.Context(), projectId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Project " + projectId + " not found"})
			return false
		}
		s.logger.Error("load project failed", "project_id", projectId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load project"})
		return false
	}
	member, err := s.store.IsProjectMember(r.Context(), projectId, userId)
	if err != nil {
		s.logger.Error("check project member failed", "project_id", projectId, "user_id", userId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check project member"})
		return false
	}
	if !member {
		writeJSON(w, http.StatusForbidden, map[string]string{"detail": "Permission denied"})
		return false
	}
	return true
}

func (s Server) ensureRepositoryCodeAvailable(w http.ResponseWriter, r *http.Request, projectId *string, code string) bool {
	existing, err := s.store.RepositoryByCode(r.Context(), projectId, code)
	if err == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"detail": "Repository code '" + existing.Code + "' already exists"})
		return false
	}
	if !errors.Is(err, sql.ErrNoRows) {
		s.logger.Error("check repository code failed", "repository_code", code, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check repository code"})
		return false
	}
	return true
}

func (s Server) ensureRepositoryCredential(w http.ResponseWriter, r *http.Request, credentialId *string) bool {
	if credentialId == nil || strings.TrimSpace(*credentialId) == "" {
		return true
	}
	exists, err := s.store.CredentialExists(r.Context(), *credentialId)
	if err != nil {
		s.logger.Error("check credential failed", "credential_id", *credentialId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to check credential"})
		return false
	}
	if !exists {
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Credential " + *credentialId + " not found"})
		return false
	}
	return true
}

func (s Server) ensurePipelineTemplate(w http.ResponseWriter, r *http.Request, templateId string) bool {
	if _, err := s.store.PipelineTemplate(r.Context(), templateId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Pipeline template " + templateId + " not found"})
			return false
		}
		s.logger.Error("load pipeline template failed", "template_id", templateId, "error", err)
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Failed to load pipeline template"})
		return false
	}
	return true
}

func normalizeRepositoryCreateReq(w http.ResponseWriter, req *repositoryCreateReq) bool {
	req.Name = strings.TrimSpace(req.Name)
	req.Code = strings.TrimSpace(req.Code)
	req.RepositoryURL = strings.TrimSpace(req.RepositoryURL)
	req.DefaultBranch = strings.TrimSpace(req.DefaultBranch)
	if req.DefaultBranch == "" {
		req.DefaultBranch = "master"
	}
	if req.GitCredentialId != nil {
		credentialId := strings.TrimSpace(*req.GitCredentialId)
		if credentialId == "" {
			req.GitCredentialId = nil
		} else {
			req.GitCredentialId = &credentialId
		}
	}
	if req.Name == "" || req.Code == "" || !repositoryCodePattern.MatchString(req.Code) || req.RepositoryURL == "" || req.DefaultBranch == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository fields"})
		return false
	}
	return true
}

func normalizeRepositoryWebhookCreateReq(w http.ResponseWriter, req *repositoryWebhookCreateReq) bool {
	req.Name = strings.TrimSpace(req.Name)
	req.TemplateId = strings.TrimSpace(req.TemplateId)
	req.Secret = strings.TrimSpace(req.Secret)
	req.BranchFilter = normalizeOptionalString(req.BranchFilter)
	if req.Name == "" || req.TemplateId == "" || req.Secret == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository webhook fields"})
		return false
	}
	return true
}

func (s Server) parseWebhookEvent(w http.ResponseWriter, r *http.Request, webhook orbit.RepositoryWebhook, payload []byte, body map[string]any) (string, string, string, bool) {
	if signature := strings.TrimSpace(r.Header.Get("X-Hub-Signature-256")); signature != "" {
		if !verifyGithubWebhookSignature(payload, signature, webhook.EncryptedSecret) {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "Invalid signature"})
			return "", "", "", false
		}
		return strings.TrimPrefix(stringMapValue(body, "ref"), "refs/heads/"), stringMapValue(body, "after"), nestedStringMapValue(body, "pusher", "name"), true
	}
	if token := strings.TrimSpace(r.Header.Get("X-Gitlab-Token")); token != "" {
		if token != webhook.EncryptedSecret {
			writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "Invalid token"})
			return "", "", "", false
		}
		return strings.TrimPrefix(stringMapValue(body, "ref"), "refs/heads/"), stringMapValue(body, "checkout_sha"), stringMapValue(body, "user_name"), true
	}
	writeJSON(w, http.StatusUnauthorized, map[string]string{"detail": "Missing signature header"})
	return "", "", "", false
}

func verifyGithubWebhookSignature(payload []byte, signature string, secret string) bool {
	if secret == "" || !strings.HasPrefix(signature, "sha256=") {
		return false
	}
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write(payload)
	computed := hex.EncodeToString(mac.Sum(nil))
	return hmac.Equal([]byte(computed), []byte(strings.TrimPrefix(signature, "sha256=")))
}

func stringMapValue(values map[string]any, key string) string {
	value, _ := values[key].(string)
	return value
}

func nestedStringMapValue(values map[string]any, key string, nestedKey string) string {
	nested, _ := values[key].(map[string]any)
	return stringMapValue(nested, nestedKey)
}

func matchBranchFilter(branch string, filter string) bool {
	if !strings.Contains(filter, "*") {
		return branch == filter
	}
	parts := strings.Split(filter, "*")
	if len(parts) == 2 {
		return strings.HasPrefix(branch, parts[0]) && strings.HasSuffix(branch, parts[1])
	}
	position := 0
	for _, part := range parts {
		if part == "" {
			continue
		}
		index := strings.Index(branch[position:], part)
		if index < 0 {
			return false
		}
		position += index + len(part)
	}
	return true
}

func applyRepositoryWebhookUpdateReq(w http.ResponseWriter, r *http.Request, s Server, webhook *orbit.RepositoryWebhook, req map[string]json.RawMessage) bool {
	var value string
	if raw, exists := req["name"]; exists {
		if err := json.Unmarshal(raw, &value); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository webhook fields"})
			return false
		}
		value = strings.TrimSpace(value)
		if value == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository webhook fields"})
			return false
		}
		webhook.Name = value
	}
	if raw, exists := req["template_id"]; exists {
		if err := json.Unmarshal(raw, &value); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository webhook fields"})
			return false
		}
		value = strings.TrimSpace(value)
		if value == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository webhook fields"})
			return false
		}
		if !s.ensurePipelineTemplate(w, r, value) {
			return false
		}
		webhook.TemplateId = value
	}
	if raw, exists := req["secret"]; exists {
		if err := json.Unmarshal(raw, &value); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository webhook fields"})
			return false
		}
		value = strings.TrimSpace(value)
		if value == "" {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository webhook fields"})
			return false
		}
		webhook.EncryptedSecret = value
	}
	if raw, exists := req["branch_filter"]; exists {
		var branchFilter *string
		if string(raw) != "null" {
			if err := json.Unmarshal(raw, &value); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository webhook fields"})
				return false
			}
			branchFilter = &value
		}
		webhook.BranchFilter = normalizeOptionalString(branchFilter)
	}
	if raw, exists := req["enabled"]; exists {
		if err := json.Unmarshal(raw, &webhook.Enabled); err != nil {
			writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid repository webhook fields"})
			return false
		}
	}
	return true
}

func normalizeOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}

func marshalVariableOverrides(w http.ResponseWriter, variables []map[string]any) (string, bool) {
	if variables == nil {
		variables = []map[string]any{}
	}
	data, err := json.Marshal(variables)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid variable_overrides"})
		return "", false
	}
	return string(data), true
}

func repositoryVariables(w http.ResponseWriter, value string) ([]map[string]any, bool) {
	if strings.TrimSpace(value) == "" {
		return []map[string]any{}, true
	}
	var variables []map[string]any
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"detail": "Invalid repository variables"})
		return nil, false
	}
	return variables, true
}

func marshalTriggerVariables(w http.ResponseWriter, variables map[string]string) (string, bool) {
	if variables == nil {
		variables = map[string]string{}
	}
	data, err := json.Marshal(variables)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"detail": "Invalid variables"})
		return "", false
	}
	return string(data), true
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

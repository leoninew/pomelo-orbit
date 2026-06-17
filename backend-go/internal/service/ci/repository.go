package cisvc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"backend/internal/apperror"
	"backend/internal/repository"
	"backend/internal/repository/model"
	taskrepo "backend/internal/repository/task"
	"backend/internal/status"
)

var repositoryCodePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

type RepositoryStore interface {
	Project(ctx context.Context, id string) (model.Project, error)
	IsProjectMember(ctx context.Context, projectId string, userId string) (bool, error)
	ListRepositories(ctx context.Context, projectId *string, page int, perPage int, search string) (repository.Page[repository.Repository], error)
	Repository(ctx context.Context, id string) (repository.Repository, error)
	RepositoryByCode(ctx context.Context, projectId *string, code string) (repository.Repository, error)
	ListCredentials(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[repository.Credential], error)
	Credential(ctx context.Context, id string) (repository.Credential, error)
	CredentialByName(ctx context.Context, projectId string, name string) (repository.Credential, error)
	CredentialExists(ctx context.Context, id string) (bool, error)
	CredentialName(ctx context.Context, id string) (*string, error)
	CreateCredential(ctx context.Context, credential repository.Credential) error
	UpdateCredential(ctx context.Context, credential repository.Credential) error
	DeleteCredential(ctx context.Context, id string) error
	CredentialReferencedByRepositories(ctx context.Context, projectId string, credentialId string) (bool, error)
	CreateRepository(ctx context.Context, repo repository.Repository) error
	UpdateRepository(ctx context.Context, repo repository.Repository) error
	DeleteRepository(ctx context.Context, id string) error
	RepositoryHasRunningPipelines(ctx context.Context, repositoryId string) (bool, error)
	ListRepositoryWebhooks(ctx context.Context, repositoryId string) ([]repository.RepositoryWebhook, error)
	RepositoryWebhook(ctx context.Context, id string) (repository.RepositoryWebhook, error)
	CreateRepositoryWebhook(ctx context.Context, webhook repository.RepositoryWebhook) error
	UpdateRepositoryWebhook(ctx context.Context, webhook repository.RepositoryWebhook) error
	DeleteRepositoryWebhook(ctx context.Context, id string) error
	PipelineTemplate(ctx context.Context, id string) (repository.PipelineTemplate, error)
	ListPipelineTemplates(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[repository.PipelineTemplate], error)
	PipelineTemplateByName(ctx context.Context, projectId string, name string) (repository.PipelineTemplate, error)
	PipelineTemplateStages(ctx context.Context, templateId string) ([]repository.PipelineTemplateStage, error)
	ListBuildStages(ctx context.Context, projectId string, page int, perPage int, search string) (repository.Page[repository.BuildStage], error)
	BuildStage(ctx context.Context, id string) (repository.BuildStage, error)
	BuildStageByName(ctx context.Context, projectId string, name string) (repository.BuildStage, error)
	BuildStagesByIds(ctx context.Context, projectId string, ids []string) ([]repository.BuildStage, error)
	CreateBuildStage(ctx context.Context, stage repository.BuildStage) error
	UpdateBuildStage(ctx context.Context, stage repository.BuildStage) error
	DeleteBuildStage(ctx context.Context, id string) error
	BuildStageReferencedByTemplates(ctx context.Context, projectId string, stageId string) (bool, error)
	CreatePipelineTemplate(ctx context.Context, template repository.PipelineTemplate) error
	UpdatePipelineTemplate(ctx context.Context, template repository.PipelineTemplate) error
	UpdatePipelineTemplateWithStages(ctx context.Context, template repository.PipelineTemplate, stages []repository.PipelineTemplateStage) error
	DuplicatePipelineTemplate(ctx context.Context, template repository.PipelineTemplate, stages []repository.PipelineTemplateStage) error
	PipelineTemplateReferencedByWebhooks(ctx context.Context, templateId string) (bool, error)
	DeletePipelineTemplate(ctx context.Context, id string) error
	LatestPipelineSnapshot(ctx context.Context, templateId string) (repository.PipelineSnapshot, error)
	PipelineSnapshot(ctx context.Context, id string) (repository.PipelineSnapshot, error)
	ListPipelineRuns(ctx context.Context, projectId string, repositoryId string, templateId string, dateFrom *time.Time, dateTo *time.Time, page int, perPage int) (repository.Page[repository.PipelineRun], error)
	ListPipelineRunsByRepository(ctx context.Context, repositoryId string, page int, perPage int) (repository.Page[repository.PipelineRun], error)
	PipelineRun(ctx context.Context, id string) (repository.PipelineRun, error)
	ListStageRuns(ctx context.Context, runId string) ([]repository.StageRun, error)
	StageRun(ctx context.Context, id string) (repository.StageRun, error)
	ListArtifacts(ctx context.Context, projectId string, repositoryId string, templateId string, page int, perPage int, search string) (repository.Page[repository.Artifact], error)
	ListArtifactsByRun(ctx context.Context, projectId *string, runId string) ([]repository.Artifact, error)
	CreatePipelineRun(ctx context.Context, run repository.PipelineRun) error
	CancelPipelineRun(ctx context.Context, id string) error
}

type TaskService interface {
	EnqueueTyped(ctx context.Context, taskType string, payload any) (*taskrepo.Task, error)
}

type Service struct {
	store     RepositoryStore
	tasks     TaskService
	dataRoot  string
	secretKey string
}

type RepositoryCreateInput struct {
	ProjectId         string
	Name              string
	Code              string
	RepositoryURL     string
	GitCredentialId   *string
	VariableOverrides []map[string]any
	DefaultBranch     string
}

type RepositoryUpdateInput struct {
	Name              *string
	RepositoryURL     *string
	GitCredentialId   *string
	VariableOverrides *[]map[string]any
	DefaultBranch     *string
}

type RepositoryDetail struct {
	Repository           repository.Repository
	GitCredentialName    *string
	VariableDeclarations []map[string]any
}

type WebhookCreateInput struct {
	Name         string
	TemplateId   string
	Secret       string
	BranchFilter *string
}

type WebhookUpdateInput struct {
	Name         *string
	TemplateId   *string
	Secret       *string
	BranchFilter *string
	BranchSet    bool
	Enabled      *bool
}

type WebhookReceiveInput struct {
	WebhookId string
	Headers   map[string]string
	Payload   []byte
}

type WebhookReceiveResult struct {
	Status string
	Reason string
	RunId  string
}

func New(store RepositoryStore, tasks TaskService, dataRoot string, secretKey string) Service {
	return Service{store: store, tasks: tasks, dataRoot: dataRoot, secretKey: secretKey}
}

func (s Service) ListRepositories(ctx context.Context, userId string, projectId *string, page int, perPage int, search string) (repository.Page[repository.Repository], error) {
	if projectId != nil {
		if err := s.ensureProjectMembership(ctx, *projectId, userId); err != nil {
			return repository.Page[repository.Repository]{}, err
		}
	}
	items, err := s.store.ListRepositories(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[repository.Repository]{}, apperror.Wrap(apperror.KindInternal, "Failed to list repositories", err)
	}
	return items, nil
}

func (s Service) CreateRepository(ctx context.Context, userId string, input RepositoryCreateInput) (RepositoryDetail, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return RepositoryDetail{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return RepositoryDetail{}, err
	}
	name, code, repositoryURL, defaultBranch, gitCredentialId, err := normalizeRepositoryCreateInput(input)
	if err != nil {
		return RepositoryDetail{}, err
	}
	if err := s.ensureRepositoryCodeAvailable(ctx, &projectId, code); err != nil {
		return RepositoryDetail{}, err
	}
	if err := s.ensureCredential(ctx, gitCredentialId); err != nil {
		return RepositoryDetail{}, err
	}
	overrides, err := marshalVariableOverrides(input.VariableOverrides)
	if err != nil {
		return RepositoryDetail{}, err
	}
	repo := repository.Repository{Id: repository.NewId(), ProjectId: &projectId, Name: name, Code: code, RepositoryURL: repositoryURL, GitCredentialId: gitCredentialId, VariableOverrides: overrides, DefaultBranch: defaultBranch}
	if err := s.store.CreateRepository(ctx, repo); err != nil {
		return RepositoryDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create repository", err)
	}
	created, err := s.store.Repository(ctx, repo.Id)
	if err != nil {
		return RepositoryDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	return s.repositoryDetail(ctx, created)
}

func (s Service) RepositoryForUser(ctx context.Context, userId string, repositoryId string) (RepositoryDetail, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return RepositoryDetail{}, err
	}
	return s.repositoryDetail(ctx, repo)
}

func (s Service) UpdateRepository(ctx context.Context, userId string, repositoryId string, input RepositoryUpdateInput) (RepositoryDetail, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return RepositoryDetail{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return RepositoryDetail{}, apperror.New(apperror.KindValidation, "Invalid repository fields")
		}
		repo.Name = name
	}
	if input.RepositoryURL != nil {
		repositoryURL := strings.TrimSpace(*input.RepositoryURL)
		if repositoryURL == "" {
			return RepositoryDetail{}, apperror.New(apperror.KindValidation, "Invalid repository fields")
		}
		repo.RepositoryURL = repositoryURL
	}
	if input.GitCredentialId != nil {
		repo.GitCredentialId = normalizeOptionalString(input.GitCredentialId)
		if err := s.ensureCredential(ctx, repo.GitCredentialId); err != nil {
			return RepositoryDetail{}, err
		}
	}
	if input.VariableOverrides != nil {
		overrides, err := marshalVariableOverrides(*input.VariableOverrides)
		if err != nil {
			return RepositoryDetail{}, err
		}
		repo.VariableOverrides = overrides
	}
	if input.DefaultBranch != nil {
		branch := strings.TrimSpace(*input.DefaultBranch)
		if branch == "" {
			return RepositoryDetail{}, apperror.New(apperror.KindValidation, "Invalid repository fields")
		}
		repo.DefaultBranch = branch
	}
	if err := s.store.UpdateRepository(ctx, repo); err != nil {
		return RepositoryDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update repository", err)
	}
	updated, err := s.store.Repository(ctx, repo.Id)
	if err != nil {
		return RepositoryDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	return s.repositoryDetail(ctx, updated)
}

func (s Service) DeleteRepository(ctx context.Context, userId string, repositoryId string) error {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return err
	}
	running, err := s.store.RepositoryHasRunningPipelines(ctx, repo.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check repository pipelines", err)
	}
	if running {
		return apperror.New(apperror.KindConflict, "Repository has running pipelines, cannot delete")
	}
	if err := s.store.DeleteRepository(ctx, repo.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete repository", err)
	}
	return nil
}

func (s Service) ListRepositoryWebhooks(ctx context.Context, userId string, repositoryId string) ([]repository.RepositoryWebhook, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return nil, err
	}
	items, err := s.store.ListRepositoryWebhooks(ctx, repo.Id)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to list repository webhooks", err)
	}
	return items, nil
}

func (s Service) CreateRepositoryWebhook(ctx context.Context, userId string, repositoryId string, input WebhookCreateInput) (repository.RepositoryWebhook, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return repository.RepositoryWebhook{}, err
	}
	name, templateId, secret, branchFilter, err := normalizeWebhookCreateInput(input)
	if err != nil {
		return repository.RepositoryWebhook{}, err
	}
	if err := s.ensurePipelineTemplate(ctx, templateId); err != nil {
		return repository.RepositoryWebhook{}, err
	}
	webhook := repository.RepositoryWebhook{Id: repository.NewId(), RepositoryId: repo.Id, Name: name, TemplateId: templateId, BranchFilter: branchFilter, EncryptedSecret: secret, Enabled: true}
	if err := s.store.CreateRepositoryWebhook(ctx, webhook); err != nil {
		return repository.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to create repository webhook", err)
	}
	created, err := s.store.RepositoryWebhook(ctx, webhook.Id)
	if err != nil {
		return repository.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository webhook", err)
	}
	return created, nil
}

func (s Service) RepositoryWebhookForUser(ctx context.Context, userId string, webhookId string) (repository.RepositoryWebhook, error) {
	webhook, err := s.loadRepositoryWebhook(ctx, webhookId)
	if err != nil {
		return repository.RepositoryWebhook{}, err
	}
	repo, err := s.store.Repository(ctx, webhook.RepositoryId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.RepositoryWebhook{}, apperror.New(apperror.KindNotFound, "Repository "+webhook.RepositoryId+" not found")
		}
		return repository.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	if repo.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *repo.ProjectId, userId); err != nil {
			return repository.RepositoryWebhook{}, err
		}
	}
	return webhook, nil
}

func (s Service) UpdateRepositoryWebhook(ctx context.Context, userId string, repositoryId string, webhookId string, input WebhookUpdateInput) (repository.RepositoryWebhook, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return repository.RepositoryWebhook{}, err
	}
	webhook, err := s.loadRepositoryWebhook(ctx, webhookId)
	if err != nil {
		return repository.RepositoryWebhook{}, err
	}
	if webhook.RepositoryId != repo.Id {
		return repository.RepositoryWebhook{}, apperror.New(apperror.KindNotFound, "Webhook "+webhook.Id+" not found")
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return repository.RepositoryWebhook{}, apperror.New(apperror.KindValidation, "Invalid repository webhook fields")
		}
		webhook.Name = name
	}
	if input.TemplateId != nil {
		templateId := strings.TrimSpace(*input.TemplateId)
		if templateId == "" {
			return repository.RepositoryWebhook{}, apperror.New(apperror.KindValidation, "Invalid repository webhook fields")
		}
		if err := s.ensurePipelineTemplate(ctx, templateId); err != nil {
			return repository.RepositoryWebhook{}, err
		}
		webhook.TemplateId = templateId
	}
	if input.Secret != nil {
		secret := strings.TrimSpace(*input.Secret)
		if secret == "" {
			return repository.RepositoryWebhook{}, apperror.New(apperror.KindValidation, "Invalid repository webhook fields")
		}
		webhook.EncryptedSecret = secret
	}
	if input.BranchSet {
		webhook.BranchFilter = normalizeOptionalString(input.BranchFilter)
	}
	if input.Enabled != nil {
		webhook.Enabled = *input.Enabled
	}
	if err := s.store.UpdateRepositoryWebhook(ctx, webhook); err != nil {
		return repository.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to update repository webhook", err)
	}
	updated, err := s.store.RepositoryWebhook(ctx, webhook.Id)
	if err != nil {
		return repository.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository webhook", err)
	}
	return updated, nil
}

func (s Service) DeleteRepositoryWebhook(ctx context.Context, userId string, repositoryId string, webhookId string) error {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return err
	}
	webhook, err := s.loadRepositoryWebhook(ctx, webhookId)
	if err != nil {
		return err
	}
	if webhook.RepositoryId != repo.Id {
		return apperror.New(apperror.KindNotFound, "Webhook "+webhook.Id+" not found")
	}
	if err := s.store.DeleteRepositoryWebhook(ctx, webhook.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete repository webhook", err)
	}
	return nil
}

func (s Service) ReceiveRepositoryWebhook(ctx context.Context, input WebhookReceiveInput) (WebhookReceiveResult, error) {
	webhook, err := s.loadRepositoryWebhook(ctx, input.WebhookId)
	if err != nil {
		return WebhookReceiveResult{}, err
	}
	if !webhook.Enabled {
		return WebhookReceiveResult{Status: "ignored", Reason: "webhook disabled"}, nil
	}
	var body map[string]any
	if err := json.Unmarshal(input.Payload, &body); err != nil {
		return WebhookReceiveResult{}, apperror.New(apperror.KindValidation, "Invalid JSON payload")
	}
	branch, commitSha, author, err := parseWebhookEvent(input.Headers, webhook, input.Payload, body)
	if err != nil {
		return WebhookReceiveResult{}, err
	}
	if webhook.BranchFilter == nil || (*webhook.BranchFilter != "*" && !matchBranchFilter(branch, *webhook.BranchFilter)) {
		return WebhookReceiveResult{Status: "ignored", Reason: "branch filtered"}, nil
	}
	repo, err := s.store.Repository(ctx, webhook.RepositoryId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WebhookReceiveResult{}, apperror.New(apperror.KindNotFound, "Repository "+webhook.RepositoryId+" not found")
		}
		return WebhookReceiveResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	template, err := s.store.PipelineTemplate(ctx, webhook.TemplateId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WebhookReceiveResult{}, apperror.New(apperror.KindNotFound, "Pipeline template "+webhook.TemplateId+" not found")
		}
		return WebhookReceiveResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	snapshot, err := s.store.LatestPipelineSnapshot(ctx, webhook.TemplateId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return WebhookReceiveResult{}, apperror.New(apperror.KindNotFound, "Pipeline snapshot for template "+webhook.TemplateId+" not found")
		}
		return WebhookReceiveResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline snapshot", err)
	}
	variables := map[string]string{"commit_sha": commitSha, "author": author, "event_type": "push"}
	variablesSnapshot, err := marshalTriggerVariables(variables)
	if err != nil {
		return WebhookReceiveResult{}, err
	}
	triggerRef := branch
	if triggerRef == "" {
		triggerRef = commitSha
	}
	run := repository.PipelineRun{Id: repository.NewId(), ProjectId: repo.ProjectId, RepositoryId: repo.Id, RepositoryName: repo.Name, SnapshotId: snapshot.Id, TemplateId: template.Id, TemplateName: template.Name, TemplateVersion: snapshot.Version, Trigger: "webhook", TriggerRef: triggerRef, VariablesSnapshot: variablesSnapshot, Status: repository.WorkStatusWaitingToRun}
	if err := s.store.CreatePipelineRun(ctx, run); err != nil {
		return WebhookReceiveResult{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline run", err)
	}
	if _, err := s.tasks.EnqueueTyped(ctx, status.TaskTypeCIPipelineRunExecute, map[string]any{"pipeline_run_id": run.Id, "variables": variables}); err != nil {
		return WebhookReceiveResult{}, apperror.Wrap(apperror.KindInternal, "Failed to enqueue pipeline run", err)
	}
	return WebhookReceiveResult{Status: "triggered", RunId: run.Id}, nil
}

func (s Service) loadRepositoryForUser(ctx context.Context, userId string, repositoryId string) (repository.Repository, error) {
	repositoryId = strings.TrimSpace(repositoryId)
	repo, err := s.store.Repository(ctx, repositoryId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.Repository{}, apperror.New(apperror.KindNotFound, "Repository "+repositoryId+" not found")
		}
		return repository.Repository{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	if repo.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *repo.ProjectId, userId); err != nil {
			return repository.Repository{}, err
		}
	}
	return repo, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.store.Project(ctx, projectId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.store.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func (s Service) ensureRepositoryCodeAvailable(ctx context.Context, projectId *string, code string) error {
	existing, err := s.store.RepositoryByCode(ctx, projectId, code)
	if err == nil {
		return apperror.New(apperror.KindConflict, "Repository code '"+existing.Code+"' already exists")
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check repository code", err)
	}
	return nil
}

func (s Service) ensureCredential(ctx context.Context, credentialId *string) error {
	if credentialId == nil || strings.TrimSpace(*credentialId) == "" {
		return nil
	}
	exists, err := s.store.CredentialExists(ctx, *credentialId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check credential", err)
	}
	if !exists {
		return apperror.New(apperror.KindNotFound, "Credential "+*credentialId+" not found")
	}
	return nil
}

func (s Service) loadRepositoryWebhook(ctx context.Context, webhookId string) (repository.RepositoryWebhook, error) {
	webhookId = strings.TrimSpace(webhookId)
	webhook, err := s.store.RepositoryWebhook(ctx, webhookId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return repository.RepositoryWebhook{}, apperror.New(apperror.KindNotFound, "Webhook "+webhookId+" not found")
		}
		return repository.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository webhook", err)
	}
	return webhook, nil
}

func (s Service) ensurePipelineTemplate(ctx context.Context, templateId string) error {
	if _, err := s.store.PipelineTemplate(ctx, templateId); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return apperror.New(apperror.KindNotFound, "Pipeline template "+templateId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	return nil
}

func (s Service) repositoryDetail(ctx context.Context, repo repository.Repository) (RepositoryDetail, error) {
	credentialName, err := s.repositoryCredentialName(ctx, repo.GitCredentialId)
	if err != nil {
		return RepositoryDetail{}, err
	}
	variables, err := repositoryVariables(repo.VariableOverrides)
	if err != nil {
		return RepositoryDetail{}, err
	}
	return RepositoryDetail{Repository: repo, GitCredentialName: credentialName, VariableDeclarations: variables}, nil
}

func (s Service) repositoryCredentialName(ctx context.Context, credentialId *string) (*string, error) {
	if credentialId == nil || strings.TrimSpace(*credentialId) == "" {
		return nil, nil
	}
	name, err := s.store.CredentialName(ctx, *credentialId)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load repository credential", err)
	}
	return name, nil
}

func normalizeRepositoryCreateInput(input RepositoryCreateInput) (string, string, string, string, *string, error) {
	name := strings.TrimSpace(input.Name)
	code := strings.TrimSpace(input.Code)
	repositoryURL := strings.TrimSpace(input.RepositoryURL)
	defaultBranch := strings.TrimSpace(input.DefaultBranch)
	if defaultBranch == "" {
		defaultBranch = "master"
	}
	gitCredentialId := normalizeOptionalString(input.GitCredentialId)
	if name == "" || code == "" || !repositoryCodePattern.MatchString(code) || repositoryURL == "" || defaultBranch == "" {
		return "", "", "", "", nil, apperror.New(apperror.KindValidation, "Invalid repository fields")
	}
	return name, code, repositoryURL, defaultBranch, gitCredentialId, nil
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

func marshalVariableOverrides(variables []map[string]any) (string, error) {
	if variables == nil {
		variables = []map[string]any{}
	}
	data, err := json.Marshal(variables)
	if err != nil {
		return "", apperror.Wrap(apperror.KindValidation, "Invalid variable_overrides", err)
	}
	return string(data), nil
}

func repositoryVariables(value string) ([]map[string]any, error) {
	if strings.TrimSpace(value) == "" {
		return []map[string]any{}, nil
	}
	var variables []map[string]any
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Invalid repository variables", err)
	}
	return variables, nil
}

func normalizeWebhookCreateInput(input WebhookCreateInput) (string, string, string, *string, error) {
	name := strings.TrimSpace(input.Name)
	templateId := strings.TrimSpace(input.TemplateId)
	secret := strings.TrimSpace(input.Secret)
	branchFilter := normalizeOptionalString(input.BranchFilter)
	if name == "" || templateId == "" || secret == "" {
		return "", "", "", nil, apperror.New(apperror.KindValidation, "Invalid repository webhook fields")
	}
	return name, templateId, secret, branchFilter, nil
}

func parseWebhookEvent(headers map[string]string, webhook repository.RepositoryWebhook, payload []byte, body map[string]any) (string, string, string, error) {
	if signature := strings.TrimSpace(headers["X-Hub-Signature-256"]); signature != "" {
		if !verifyGithubWebhookSignature(payload, signature, webhook.EncryptedSecret) {
			return "", "", "", apperror.New(apperror.KindUnauthorized, "Invalid signature")
		}
		return strings.TrimPrefix(stringMapValue(body, "ref"), "refs/heads/"), stringMapValue(body, "after"), nestedStringMapValue(body, "pusher", "name"), nil
	}
	if token := strings.TrimSpace(headers["X-Gitlab-Token"]); token != "" {
		if token != webhook.EncryptedSecret {
			return "", "", "", apperror.New(apperror.KindUnauthorized, "Invalid token")
		}
		return strings.TrimPrefix(stringMapValue(body, "ref"), "refs/heads/"), stringMapValue(body, "checkout_sha"), stringMapValue(body, "user_name"), nil
	}
	return "", "", "", apperror.New(apperror.KindUnauthorized, "Missing signature header")
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

func marshalTriggerVariables(variables map[string]string) (string, error) {
	if variables == nil {
		variables = map[string]string{}
	}
	data, err := json.Marshal(variables)
	if err != nil {
		return "", apperror.Wrap(apperror.KindValidation, "Invalid variables", err)
	}
	return string(data), nil
}

package cisvc

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"log/slog"
	"maps"
	"regexp"
	"strings"

	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	ciport "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/port"
	civariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/rule/civariable"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var repositoryCodePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

type Service struct {
	store             repository.CIStore
	executionStore    repository.PipelineExecutionStore
	dispatcher        ciport.PipelineRunDispatcher
	workspace         ciport.Workspace
	logStore          ciport.LogReader
	executionLogStore ciport.ExecutionLogStore
	secretKey         string
	logger            *slog.Logger
	runner            ciport.ContainerRunner
}

func New(store repository.CIStore, dispatcher ciport.PipelineRunDispatcher, workspace ciport.Workspace, secretKey string, logger *slog.Logger, runner ciport.ContainerRunner, logStore ciport.LogReader) Service {
	executionStore, ok := store.(repository.PipelineExecutionStore)
	if !ok {
		panic("ci service store must implement PipelineExecutionStore")
	}
	return Service{store: store, executionStore: executionStore, dispatcher: dispatcher, workspace: workspace, logStore: logStore, secretKey: secretKey, logger: logger, runner: runner}
}

func NewExecutionService(store repository.PipelineExecutionStore, workspace ciport.Workspace, secretKey string, logger *slog.Logger, runner ciport.ContainerRunner, logStore ciport.ExecutionLogStore) Service {
	return Service{executionStore: store, workspace: workspace, logStore: logStore, executionLogStore: logStore, secretKey: secretKey, logger: logger, runner: runner}
}

func (s Service) ListRepositories(ctx context.Context, userId string, projectId *string, page int, perPage int, search string) (repository.Page[model.Repository], error) {
	if projectId != nil {
		if err := s.ensureProjectMembership(ctx, *projectId, userId); err != nil {
			return repository.Page[model.Repository]{}, err
		}
	}
	items, err := s.store.ListRepositories(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[model.Repository]{}, apperror.Wrap(apperror.KindInternal, "Failed to list repositories", err)
	}
	return items, nil
}

func (s Service) CreateRepository(ctx context.Context, userId string, input cidto.RepositoryCreateInput) (cidto.RepositoryDetail, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	if projectId == "" {
		return cidto.RepositoryDetail{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return cidto.RepositoryDetail{}, err
	}
	name, code, repositoryURL, defaultBranch, gitCredentialId, err := normalizeRepositoryCreateInput(input)
	if err != nil {
		return cidto.RepositoryDetail{}, err
	}
	if err := s.ensureRepositoryCodeAvailable(ctx, &projectId, code); err != nil {
		return cidto.RepositoryDetail{}, err
	}
	if err := s.ensureCredential(ctx, gitCredentialId); err != nil {
		return cidto.RepositoryDetail{}, err
	}
	overrides, err := marshalVariableOverrides(sanitizeRepositoryVariables(input.VariableOverrides))
	if err != nil {
		return cidto.RepositoryDetail{}, err
	}
	repo := model.Repository{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Code: code, RepositoryURL: repositoryURL, GitCredentialId: gitCredentialId, VariableOverrides: overrides, DefaultBranch: defaultBranch}
	if err := s.store.CreateRepository(ctx, repo); err != nil {
		return cidto.RepositoryDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create repository", err)
	}
	created, err := s.store.Repository(ctx, repo.Id)
	if err != nil {
		return cidto.RepositoryDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	return s.repositoryDetail(ctx, created)
}

func (s Service) RepositoryForUser(ctx context.Context, userId string, repositoryId string) (cidto.RepositoryDetail, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return cidto.RepositoryDetail{}, err
	}
	return s.repositoryDetail(ctx, repo)
}

func (s Service) UpdateRepository(ctx context.Context, userId string, repositoryId string, input cidto.RepositoryUpdateInput) (cidto.RepositoryDetail, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return cidto.RepositoryDetail{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return cidto.RepositoryDetail{}, apperror.New(apperror.KindValidation, "Invalid repository fields")
		}
		repo.Name = name
	}
	if input.RepositoryURL != nil {
		repositoryURL := strings.TrimSpace(*input.RepositoryURL)
		if repositoryURL == "" {
			return cidto.RepositoryDetail{}, apperror.New(apperror.KindValidation, "Invalid repository fields")
		}
		repo.RepositoryURL = repositoryURL
	}
	if input.GitCredentialId != nil {
		repo.GitCredentialId = normalizeOptionalString(input.GitCredentialId)
		if err := s.ensureCredential(ctx, repo.GitCredentialId); err != nil {
			return cidto.RepositoryDetail{}, err
		}
	}
	if input.VariableOverrides != nil {
		overrides, err := marshalVariableOverrides(sanitizeRepositoryVariables(*input.VariableOverrides))
		if err != nil {
			return cidto.RepositoryDetail{}, err
		}
		repo.VariableOverrides = overrides
	}
	if input.DefaultBranch != nil {
		branch := strings.TrimSpace(*input.DefaultBranch)
		if branch == "" {
			return cidto.RepositoryDetail{}, apperror.New(apperror.KindValidation, "Invalid repository fields")
		}
		repo.DefaultBranch = branch
	}
	if err := s.store.UpdateRepository(ctx, repo); err != nil {
		return cidto.RepositoryDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update repository", err)
	}
	updated, err := s.store.Repository(ctx, repo.Id)
	if err != nil {
		return cidto.RepositoryDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
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

func (s Service) ListRepositoryWebhooks(ctx context.Context, userId string, repositoryId string) ([]model.RepositoryWebhook, error) {
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

func (s Service) CreateRepositoryWebhook(ctx context.Context, userId string, repositoryId string, input cidto.WebhookCreateInput) (model.RepositoryWebhook, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return model.RepositoryWebhook{}, err
	}
	name, templateId, secret, branchFilter, err := normalizeWebhookCreateInput(input)
	if err != nil {
		return model.RepositoryWebhook{}, err
	}
	if err := s.ensurePipelineTemplate(ctx, templateId); err != nil {
		return model.RepositoryWebhook{}, err
	}
	webhook := model.RepositoryWebhook{Id: idutil.NewId(), RepositoryId: repo.Id, Name: name, TemplateId: templateId, BranchFilter: branchFilter, EncryptedSecret: secret, Enabled: true}
	if err := s.store.CreateRepositoryWebhook(ctx, webhook); err != nil {
		return model.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to create repository webhook", err)
	}
	created, err := s.store.RepositoryWebhook(ctx, webhook.Id)
	if err != nil {
		return model.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository webhook", err)
	}
	return created, nil
}

func (s Service) RepositoryWebhookForUser(ctx context.Context, userId string, webhookId string) (model.RepositoryWebhook, error) {
	webhook, err := s.loadRepositoryWebhook(ctx, webhookId)
	if err != nil {
		return model.RepositoryWebhook{}, err
	}
	repo, err := s.store.Repository(ctx, webhook.RepositoryId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.RepositoryWebhook{}, apperror.New(apperror.KindNotFound, "Repository "+webhook.RepositoryId+" not found")
		}
		return model.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	if repo.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *repo.ProjectId, userId); err != nil {
			return model.RepositoryWebhook{}, err
		}
	}
	return webhook, nil
}

func (s Service) UpdateRepositoryWebhook(ctx context.Context, userId string, repositoryId string, webhookId string, input cidto.WebhookUpdateInput) (model.RepositoryWebhook, error) {
	repo, err := s.loadRepositoryForUser(ctx, userId, repositoryId)
	if err != nil {
		return model.RepositoryWebhook{}, err
	}
	webhook, err := s.loadRepositoryWebhook(ctx, webhookId)
	if err != nil {
		return model.RepositoryWebhook{}, err
	}
	if webhook.RepositoryId != repo.Id {
		return model.RepositoryWebhook{}, apperror.New(apperror.KindNotFound, "Webhook "+webhook.Id+" not found")
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return model.RepositoryWebhook{}, apperror.New(apperror.KindValidation, "Invalid repository webhook fields")
		}
		webhook.Name = name
	}
	if input.TemplateId != nil {
		templateId := strings.TrimSpace(*input.TemplateId)
		if templateId == "" {
			return model.RepositoryWebhook{}, apperror.New(apperror.KindValidation, "Invalid repository webhook fields")
		}
		if err := s.ensurePipelineTemplate(ctx, templateId); err != nil {
			return model.RepositoryWebhook{}, err
		}
		webhook.TemplateId = templateId
	}
	if input.Secret != nil {
		secret := strings.TrimSpace(*input.Secret)
		if secret == "" {
			return model.RepositoryWebhook{}, apperror.New(apperror.KindValidation, "Invalid repository webhook fields")
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
		return model.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to update repository webhook", err)
	}
	updated, err := s.store.RepositoryWebhook(ctx, webhook.Id)
	if err != nil {
		return model.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository webhook", err)
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

func (s Service) ReceiveRepositoryWebhook(ctx context.Context, input cidto.WebhookReceiveInput) (cidto.WebhookReceiveResult, error) {
	webhook, err := s.loadRepositoryWebhook(ctx, input.WebhookId)
	if err != nil {
		return cidto.WebhookReceiveResult{}, err
	}
	if !webhook.Enabled {
		return cidto.WebhookReceiveResult{Status: "ignored", Reason: "webhook disabled"}, nil
	}
	var body map[string]any
	if err := json.Unmarshal(input.Payload, &body); err != nil {
		return cidto.WebhookReceiveResult{}, apperror.New(apperror.KindValidation, "Invalid JSON payload")
	}
	branch, commitSha, author, err := parseWebhookEvent(input.Headers, webhook, input.Payload, body)
	if err != nil {
		return cidto.WebhookReceiveResult{}, err
	}
	if webhook.BranchFilter == nil || (*webhook.BranchFilter != "*" && !matchBranchFilter(branch, *webhook.BranchFilter)) {
		return cidto.WebhookReceiveResult{Status: "ignored", Reason: "branch filtered"}, nil
	}
	repo, err := s.store.Repository(ctx, webhook.RepositoryId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return cidto.WebhookReceiveResult{}, apperror.New(apperror.KindNotFound, "Repository "+webhook.RepositoryId+" not found")
		}
		return cidto.WebhookReceiveResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	template, err := s.store.PipelineTemplate(ctx, webhook.TemplateId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return cidto.WebhookReceiveResult{}, apperror.New(apperror.KindNotFound, "Pipeline template "+webhook.TemplateId+" not found")
		}
		return cidto.WebhookReceiveResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	snapshot, err := s.getOrCreatePipelineSnapshot(ctx, template)
	if err != nil {
		return cidto.WebhookReceiveResult{}, err
	}
	variables := map[string]string{"commit_sha": commitSha, "author": author, "event_type": "push"}
	triggerRef := branch
	if triggerRef == "" {
		triggerRef = commitSha
	}
	variablesSnapshot, err := buildPipelineRunVariables(repo, template, snapshot, triggerRef, variables)
	if err != nil {
		return cidto.WebhookReceiveResult{}, err
	}
	run := model.PipelineRun{Id: idutil.NewId(), ProjectId: repo.ProjectId, RepositoryId: repo.Id, RepositoryName: repo.Name, SnapshotId: snapshot.Id, TemplateId: template.Id, TemplateName: template.Name, TemplateVersion: snapshot.Version, Trigger: "webhook", TriggerRef: triggerRef, VariablesSnapshot: variablesSnapshot, Status: status.WorkStatusWaitingToRun}
	if err := s.store.CreatePipelineRun(ctx, run); err != nil {
		return cidto.WebhookReceiveResult{}, apperror.Wrap(apperror.KindInternal, "Failed to create pipeline run", err)
	}
	if err := s.dispatcher.DispatchPipelineRun(ctx, cidto.PipelineRunDispatchInput{PipelineRunID: run.Id}); err != nil {
		return cidto.WebhookReceiveResult{}, apperror.Wrap(apperror.KindInternal, "Failed to enqueue pipeline run", err)
	}
	return cidto.WebhookReceiveResult{Status: "triggered", RunId: run.Id}, nil
}

func (s Service) loadRepositoryForUser(ctx context.Context, userId string, repositoryId string) (model.Repository, error) {
	repositoryId = strings.TrimSpace(repositoryId)
	repo, err := s.store.Repository(ctx, repositoryId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Repository{}, apperror.New(apperror.KindNotFound, "Repository "+repositoryId+" not found")
		}
		return model.Repository{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	if repo.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *repo.ProjectId, userId); err != nil {
			return model.Repository{}, err
		}
	}
	return repo, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.store.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
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
	if !errors.Is(err, repository.ErrNotFound) {
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

func (s Service) loadRepositoryWebhook(ctx context.Context, webhookId string) (model.RepositoryWebhook, error) {
	webhookId = strings.TrimSpace(webhookId)
	webhook, err := s.store.RepositoryWebhook(ctx, webhookId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.RepositoryWebhook{}, apperror.New(apperror.KindNotFound, "Webhook "+webhookId+" not found")
		}
		return model.RepositoryWebhook{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository webhook", err)
	}
	return webhook, nil
}

func (s Service) ensurePipelineTemplate(ctx context.Context, templateId string) error {
	if _, err := s.store.PipelineTemplate(ctx, templateId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Pipeline template "+templateId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load pipeline template", err)
	}
	return nil
}

func (s Service) repositoryDetail(ctx context.Context, repo model.Repository) (cidto.RepositoryDetail, error) {
	credentialName, err := s.repositoryCredentialName(ctx, repo.GitCredentialId)
	if err != nil {
		return cidto.RepositoryDetail{}, err
	}
	variables, err := repositoryVariables(repo)
	if err != nil {
		return cidto.RepositoryDetail{}, err
	}
	return cidto.RepositoryDetail{Repository: repo, GitCredentialName: credentialName, VariableDeclarations: variables}, nil
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

func normalizeRepositoryCreateInput(input cidto.RepositoryCreateInput) (string, string, string, string, *string, error) {
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

func repositoryVariables(repo model.Repository) ([]map[string]any, error) {
	custom, err := repositoryCustomVariables(repo.VariableOverrides)
	if err != nil {
		return nil, err
	}
	variables := repositoryBuiltinVariableDeclarations(repo)
	variables = append(variables, custom...)
	return variables, nil
}

func repositoryCustomVariables(value string) ([]map[string]any, error) {
	if strings.TrimSpace(value) == "" {
		return []map[string]any{}, nil
	}
	var variables []map[string]any
	if err := json.Unmarshal([]byte(value), &variables); err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Invalid repository variables", err)
	}
	return sanitizeRepositoryVariables(variables), nil
}

func repositoryBuiltinVariableDeclarations(repo model.Repository) []map[string]any {
	return []map[string]any{
		repositoryBuiltinVariableDeclaration("repository_id", repo.Id),
		repositoryBuiltinVariableDeclaration("repository_name", repo.Name),
		repositoryBuiltinVariableDeclaration("repository_code", repo.Code),
		repositoryBuiltinVariableDeclaration("repository_url", repo.RepositoryURL),
		repositoryBuiltinVariableDeclaration("repository_ref", repo.DefaultBranch),
	}
}

func repositoryBuiltinVariableDeclaration(name string, defaultValue any) map[string]any {
	return map[string]any{"name": name, "description": civariable.PipelineTemplateBuiltinVariableSpecs()[name], "default": defaultValue, "value": nil, "secret": false, "source": "repository", "editable": name == "repository_ref"}
}

func sanitizeRepositoryVariables(variables []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(variables))
	for _, variable := range variables {
		name, _ := variable["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" || civariable.IsPipelineTemplateBuiltinVariable(name) {
			continue
		}
		copy := map[string]any{}
		maps.Copy(copy, variable)
		copy["name"] = name
		copy["source"] = "repository_custom"
		copy["editable"] = true
		if _, exists := copy["secret"]; !exists {
			copy["secret"] = false
		}
		result = append(result, copy)
	}
	return result
}

func normalizeWebhookCreateInput(input cidto.WebhookCreateInput) (string, string, string, *string, error) {
	name := strings.TrimSpace(input.Name)
	templateId := strings.TrimSpace(input.TemplateId)
	secret := strings.TrimSpace(input.Secret)
	branchFilter := normalizeOptionalString(input.BranchFilter)
	if name == "" || templateId == "" || secret == "" {
		return "", "", "", nil, apperror.New(apperror.KindValidation, "Invalid repository webhook fields")
	}
	return name, templateId, secret, branchFilter, nil
}

func parseWebhookEvent(headers map[string]string, webhook model.RepositoryWebhook, payload []byte, body map[string]any) (string, string, string, error) {
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

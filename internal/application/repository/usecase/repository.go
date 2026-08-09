package repositorysvc

import (
	"context"
	"encoding/json"
	"errors"
	"maps"
	"regexp"
	"strings"

	pipelinevariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/rule/pipelinevariable"
	repositorydto "gitee.com/leoninew/PomeloOrbit-go/internal/application/repository/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

var repositoryCodePattern = regexp.MustCompile(`^[a-z0-9_-]+$`)

func (s Service) ListRepositories(ctx context.Context, userID string, projectID *string, page, perPage int, search string) (repository.Page[model.Repository], error) {
	if projectID != nil {
		if err := s.ensureProjectMembership(ctx, *projectID, userID); err != nil {
			return repository.Page[model.Repository]{}, err
		}
	}
	items, err := s.store.ListRepositories(ctx, projectID, page, perPage, search)
	if err != nil {
		return repository.Page[model.Repository]{}, apperror.Wrap(apperror.KindInternal, "Failed to list repositories", err)
	}
	return items, nil
}
func (s Service) CreateRepository(ctx context.Context, userID string, input repositorydto.RepositoryCreateInput) (repositorydto.RepositoryDetail, error) {
	projectID := strings.TrimSpace(input.ProjectId)
	if projectID == "" {
		return repositorydto.RepositoryDetail{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return repositorydto.RepositoryDetail{}, err
	}
	name, code, sourceType, sourceURL, branch, credentialID, err := normalizeRepositoryCreateInput(input)
	if err != nil {
		return repositorydto.RepositoryDetail{}, err
	}
	if sourceType == model.RepositoryTypeLocalDirectory {
		sourceURL, err = s.validateLocalSourcePath(ctx, sourceURL)
		if err != nil {
			return repositorydto.RepositoryDetail{}, err
		}
	}
	if err := s.ensureRepositoryCodeAvailable(ctx, &projectID, code); err != nil {
		return repositorydto.RepositoryDetail{}, err
	}
	if err := s.ensureCredential(ctx, credentialID); err != nil {
		return repositorydto.RepositoryDetail{}, err
	}
	variables, err := marshalVariableOverrides(sanitizeRepositoryVariables(input.VariableOverrides))
	if err != nil {
		return repositorydto.RepositoryDetail{}, err
	}
	item := model.Repository{Id: idutil.NewId(), ProjectId: &projectID, Name: name, Code: code, RepositoryType: sourceType, RepositoryUrl: sourceURL, GitCredentialId: credentialID, VariableOverrides: variables, DefaultBranch: branch}
	if err := s.store.CreateRepository(ctx, item); err != nil {
		return repositorydto.RepositoryDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to create repository", err)
	}
	return s.repositoryDetail(ctx, item)
}
func (s Service) RepositoryForUser(ctx context.Context, userID, repositoryID string) (repositorydto.RepositoryDetail, error) {
	item, err := s.loadRepositoryForUser(ctx, userID, repositoryID)
	if err != nil {
		return repositorydto.RepositoryDetail{}, err
	}
	return s.repositoryDetail(ctx, item)
}
func (s Service) UpdateRepository(ctx context.Context, userID, repositoryID string, input repositorydto.RepositoryUpdateInput) (repositorydto.RepositoryDetail, error) {
	item, err := s.loadRepositoryForUser(ctx, userID, repositoryID)
	if err != nil {
		return repositorydto.RepositoryDetail{}, err
	}
	if input.Name != nil {
		item.Name = strings.TrimSpace(*input.Name)
	}
	if input.RepositoryType != nil {
		item.RepositoryType = strings.TrimSpace(*input.RepositoryType)
	}
	if input.RepositoryUrl != nil {
		item.RepositoryUrl = strings.TrimSpace(*input.RepositoryUrl)
	}
	if input.GitCredentialId != nil {
		item.GitCredentialId = normalizeOptionalString(input.GitCredentialId)
		if err := s.ensureCredential(ctx, item.GitCredentialId); err != nil {
			return repositorydto.RepositoryDetail{}, err
		}
	}
	if input.VariableOverrides != nil {
		variables, err := marshalVariableOverrides(sanitizeRepositoryVariables(*input.VariableOverrides))
		if err != nil {
			return repositorydto.RepositoryDetail{}, err
		}
		item.VariableOverrides = variables
	}
	if input.DefaultBranch != nil {
		item.DefaultBranch = strings.TrimSpace(*input.DefaultBranch)
	}
	if item.Name == "" || item.DefaultBranch == "" {
		return repositorydto.RepositoryDetail{}, apperror.New(apperror.KindValidation, "Invalid repository fields")
	}
	if err := s.normalizeRepositorySource(ctx, &item); err != nil {
		return repositorydto.RepositoryDetail{}, err
	}
	if err := s.store.UpdateRepository(ctx, item); err != nil {
		return repositorydto.RepositoryDetail{}, apperror.Wrap(apperror.KindInternal, "Failed to update repository", err)
	}
	return s.repositoryDetail(ctx, item)
}
func (s Service) DeleteRepository(ctx context.Context, userID, repositoryID string) error {
	item, err := s.loadRepositoryForUser(ctx, userID, repositoryID)
	if err != nil {
		return err
	}
	running, err := s.store.RepositoryHasRunningPipelines(ctx, item.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check repository pipelines", err)
	}
	if running {
		return apperror.New(apperror.KindValidation, "Repository has running pipelines. Cancel or wait for them to finish before deleting it.")
	}
	if err := s.store.DeleteRepository(ctx, item.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete repository", err)
	}
	return nil
}
func (s Service) loadRepositoryForUser(ctx context.Context, userID, repositoryID string) (model.Repository, error) {
	item, err := s.store.Repository(ctx, strings.TrimSpace(repositoryID))
	if errors.Is(err, repository.ErrNotFound) {
		return model.Repository{}, apperror.New(apperror.KindNotFound, "Repository "+repositoryID+" not found")
	}
	if err != nil {
		return model.Repository{}, apperror.Wrap(apperror.KindInternal, "Failed to load repository", err)
	}
	if item.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *item.ProjectId, userID); err != nil {
			return model.Repository{}, err
		}
	}
	return item, nil
}
func (s Service) ensureProjectMembership(ctx context.Context, projectID, userID string) error {
	if _, err := s.store.Project(ctx, projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectID+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.store.IsProjectMember(ctx, projectID, userID)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}
func (s Service) ensureRepositoryCodeAvailable(ctx context.Context, projectID *string, code string) error {
	existing, err := s.store.RepositoryByCode(ctx, projectID, code)
	if err == nil {
		return apperror.New(apperror.KindConflict, "Repository code '"+existing.Code+"' already exists")
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check repository code", err)
	}
	return nil
}
func (s Service) ensureCredential(ctx context.Context, credentialID *string) error {
	if credentialID == nil {
		return nil
	}
	exists, err := s.store.CredentialExists(ctx, *credentialID)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check credential", err)
	}
	if !exists {
		return apperror.New(apperror.KindNotFound, "Credential "+*credentialID+" not found")
	}
	return nil
}
func (s Service) repositoryDetail(ctx context.Context, item model.Repository) (repositorydto.RepositoryDetail, error) {
	name, err := s.repositoryCredentialName(ctx, item.GitCredentialId)
	if err != nil {
		return repositorydto.RepositoryDetail{}, err
	}
	variables, err := repositoryVariables(item)
	if err != nil {
		return repositorydto.RepositoryDetail{}, err
	}
	return repositorydto.RepositoryDetail{Repository: item, GitCredentialName: name, VariableDeclarations: variables}, nil
}
func (s Service) repositoryCredentialName(ctx context.Context, credentialID *string) (*string, error) {
	if credentialID == nil {
		return nil, nil
	}
	value, err := s.store.CredentialName(ctx, *credentialID)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load repository credential", err)
	}
	return value, nil
}
func normalizeRepositoryCreateInput(input repositorydto.RepositoryCreateInput) (string, string, string, string, string, *string, error) {
	name, code, sourceURL, sourceType, branch := strings.TrimSpace(input.Name), strings.TrimSpace(input.Code), strings.TrimSpace(input.RepositoryUrl), strings.TrimSpace(input.RepositoryType), strings.TrimSpace(input.DefaultBranch)
	if sourceType == "" {
		sourceType = model.RepositoryTypeRemoteGit
	}
	if branch == "" {
		branch = "master"
	}
	credentialID := normalizeOptionalString(input.GitCredentialId)
	if name == "" || code == "" || !repositoryCodePattern.MatchString(code) {
		return "", "", "", "", "", nil, apperror.New(apperror.KindValidation, "Invalid repository fields")
	}
	if sourceType == model.RepositoryTypeRemoteGit && sourceURL != "" {
		return name, code, sourceType, sourceURL, branch, credentialID, nil
	}
	if sourceType == model.RepositoryTypeLocalDirectory && sourceURL != "" && credentialID == nil {
		return name, code, sourceType, sourceURL, branch, nil, nil
	}
	return "", "", "", "", "", nil, apperror.New(apperror.KindValidation, "Invalid repository fields")
}
func (s Service) normalizeRepositorySource(ctx context.Context, item *model.Repository) error {
	if item.RepositoryType == model.RepositoryTypeRemoteGit {
		if strings.TrimSpace(item.RepositoryUrl) == "" {
			return apperror.New(apperror.KindValidation, "repository_url is required for remote Git repositories")
		}
		return nil
	}
	if item.RepositoryType == model.RepositoryTypeLocalDirectory {
		if item.GitCredentialId != nil {
			return apperror.New(apperror.KindValidation, "local directory repositories do not support Git credentials")
		}
		value, err := s.validateLocalSourcePath(ctx, item.RepositoryUrl)
		if err != nil {
			return err
		}
		item.RepositoryUrl = value
		return nil
	}
	return apperror.New(apperror.KindValidation, "Unsupported repository_type")
}
func (s Service) validateLocalSourcePath(ctx context.Context, path string) (string, error) {
	if s.localSource == nil {
		return "", apperror.New(apperror.KindValidation, "Local directory sources are disabled")
	}
	value, err := s.localSource.Validate(ctx, path)
	if err != nil {
		return "", apperror.New(apperror.KindValidation, "Invalid local source path: "+err.Error())
	}
	return value, nil
}
func normalizeOptionalString(value *string) *string {
	if value == nil || strings.TrimSpace(*value) == "" {
		return nil
	}
	result := strings.TrimSpace(*value)
	return &result
}
func marshalVariableOverrides(values []map[string]any) (string, error) {
	if values == nil {
		values = []map[string]any{}
	}
	data, err := json.Marshal(values)
	if err != nil {
		return "", apperror.Wrap(apperror.KindValidation, "Invalid variable_overrides", err)
	}
	return string(data), nil
}
func repositoryVariables(item model.Repository) ([]map[string]any, error) {
	custom, err := repositoryCustomVariables(item.VariableOverrides)
	if err != nil {
		return nil, err
	}
	return append(repositoryBuiltinVariableDeclarations(item), custom...), nil
}
func repositoryCustomVariables(value string) ([]map[string]any, error) {
	if strings.TrimSpace(value) == "" {
		return []map[string]any{}, nil
	}
	var values []map[string]any
	if err := json.Unmarshal([]byte(value), &values); err != nil {
		return nil, apperror.New(apperror.KindInternal, "Invalid repository variables")
	}
	return sanitizeRepositoryVariables(values), nil
}
func repositoryBuiltinVariableDeclarations(item model.Repository) []map[string]any {
	return []map[string]any{repositoryBuiltinVariableDeclaration("repository_id", item.Id), repositoryBuiltinVariableDeclaration("repository_name", item.Name), repositoryBuiltinVariableDeclaration("repository_code", item.Code), repositoryBuiltinVariableDeclaration("repository_url", item.RepositoryUrl), repositoryBuiltinVariableDeclaration("repository_ref", item.DefaultBranch)}
}
func repositoryBuiltinVariableDeclaration(name string, value any) map[string]any {
	return map[string]any{"name": name, "description": pipelinevariable.PipelineBuiltinVariableSpecs()[name], "default": value, "value": nil, "secret": false, "source": "repository", "editable": name == "repository_ref"}
}
func sanitizeRepositoryVariables(values []map[string]any) []map[string]any {
	result := make([]map[string]any, 0, len(values))
	for _, value := range values {
		name, _ := value["name"].(string)
		name = strings.TrimSpace(name)
		if name == "" || pipelinevariable.IsPipelineBuiltinVariable(name) {
			continue
		}
		copy := map[string]any{}
		maps.Copy(copy, value)
		copy["name"], copy["source"], copy["editable"] = name, "repository_custom", true
		if _, exists := copy["secret"]; !exists {
			copy["secret"] = false
		}
		result = append(result, copy)
	}
	return result
}

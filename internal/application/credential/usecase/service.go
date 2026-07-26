package credentialsvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"

	credentialdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/credential/dto"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"

	security "gitee.com/leoninew/PomeloOrbit-go/internal/common/crypto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type Service struct {
	project    repository.ProjectReader
	credential repository.CredentialStore
	secretKey  string
}

func New(project repository.ProjectReader, credential repository.CredentialStore, secretKey string) Service {
	return Service{project: project, credential: credential, secretKey: secretKey}
}

func (s Service) ListCredentials(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[model.Credential], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[model.Credential]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Credential]{}, err
	}
	items, err := s.credential.ListCredentials(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[model.Credential]{}, apperror.Wrap(apperror.KindInternal, "Failed to list credentials", err)
	}
	return items, nil
}

func (s Service) CreateCredential(ctx context.Context, userId string, input credentialdto.CredentialCreateInput) (model.Credential, error) {
	projectId, name, credentialType, data, err := normalizeCredentialCreateInput(input)
	if err != nil {
		return model.Credential{}, err
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Credential{}, err
	}
	return s.createCredentialRecord(ctx, projectId, name, credentialType, data)
}

func (s Service) ImportCredential(ctx context.Context, userId string, input credentialdto.CredentialCreateInput) (model.Credential, error) {
	return s.CreateCredential(ctx, userId, input)
}

func (s Service) CredentialForUser(ctx context.Context, userId string, credentialId string) (model.Credential, error) {
	return s.loadCredentialForUser(ctx, userId, credentialId)
}

func (s Service) CredentialDetailForUser(ctx context.Context, userId string, credentialId string) (credentialdto.CredentialDetail, error) {
	credential, err := s.loadCredentialForUser(ctx, userId, credentialId)
	if err != nil {
		return credentialdto.CredentialDetail{}, err
	}
	decrypted, err := s.decryptCredentialData(credential.EncryptedData)
	if err != nil {
		return credentialdto.CredentialDetail{}, err
	}
	return credentialdto.CredentialDetail{Credential: credential, Data: decrypted}, nil
}

func (s Service) UpdateCredential(ctx context.Context, userId string, credentialId string, input credentialdto.CredentialUpdateInput) (model.Credential, error) {
	credential, err := s.loadCredentialForUser(ctx, userId, credentialId)
	if err != nil {
		return model.Credential{}, err
	}
	if input.Name != nil {
		name := strings.TrimSpace(*input.Name)
		if name == "" {
			return model.Credential{}, apperror.New(apperror.KindValidation, "Invalid credential fields")
		}
		if err := s.ensureCredentialNameAvailable(ctx, credentialProjectId(credential), name, credential.Id); err != nil {
			return model.Credential{}, err
		}
		credential.Name = name
	}
	if input.Data != nil {
		if *input.Data == "" {
			return model.Credential{}, apperror.New(apperror.KindValidation, "Invalid credential fields")
		}
		if credential.Type == "runtime_env" {
			values, ok := runtimeEnvCredentialValues(*input.Data)
			if !ok {
				return model.Credential{}, apperror.New(apperror.KindValidation, "Invalid runtime_env credential data")
			}
			refs, err := s.credential.VersionComponentSecretEnvRefsByCredential(ctx, credential.Id)
			if err != nil {
				return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to load runtime_env credential references", err)
			}
			for _, ref := range refs {
				if _, exists := values[ref.DataKey]; !exists {
					return model.Credential{}, apperror.New(apperror.KindConflict, "runtime_env credential update would remove a referenced data_key")
				}
			}
		}
		encrypted, err := s.encryptCredentialData(*input.Data)
		if err != nil {
			return model.Credential{}, err
		}
		credential.EncryptedData = encrypted
	}
	if err := s.credential.UpdateCredential(ctx, credential); err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to update credential", err)
	}
	updated, err := s.credential.Credential(ctx, credential.Id)
	if err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to load credential", err)
	}
	return updated, nil
}

func (s Service) DeleteCredential(ctx context.Context, userId string, credentialId string) error {
	credential, err := s.loadCredentialForUser(ctx, userId, credentialId)
	if err != nil {
		return err
	}
	referenced, err := s.credential.CredentialReferencedByRepositories(ctx, credentialProjectId(credential), credential.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check credential references", err)
	}
	if referenced {
		return apperror.New(apperror.KindConflict, "Credential is referenced by projects, cannot delete")
	}
	runtimeReferenced, err := s.credential.CredentialReferencedByVersionComponents(ctx, credential.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check runtime_env credential references", err)
	}
	if runtimeReferenced {
		return apperror.New(apperror.KindConflict, "Credential is referenced by version components, cannot delete")
	}
	if err := s.credential.DeleteCredential(ctx, credential.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete credential", err)
	}
	return nil
}

func (s Service) ExportCredential(ctx context.Context, userId string, credentialId string) (credentialdto.CredentialExport, error) {
	credential, err := s.loadCredentialForUser(ctx, userId, credentialId)
	if err != nil {
		return credentialdto.CredentialExport{}, err
	}
	decrypted, err := s.decryptCredentialData(credential.EncryptedData)
	if err != nil {
		return credentialdto.CredentialExport{}, err
	}
	return credentialdto.CredentialExport{Version: credentialdto.CredentialExportVersion, Name: credential.Name, Type: credential.Type, Data: decrypted}, nil
}

func (s Service) createCredentialRecord(ctx context.Context, projectId string, name string, credentialType string, data string) (model.Credential, error) {
	if err := s.ensureCredentialNameAvailable(ctx, projectId, name, ""); err != nil {
		return model.Credential{}, err
	}
	if credentialType == "runtime_env" && !validRuntimeEnvCredentialData(data) {
		return model.Credential{}, apperror.New(apperror.KindValidation, "Invalid runtime_env credential data")
	}
	encrypted, err := s.encryptCredentialData(data)
	if err != nil {
		return model.Credential{}, err
	}
	credential := model.Credential{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Type: credentialType, EncryptedData: encrypted}
	if err := s.credential.CreateCredential(ctx, credential); err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to create credential", err)
	}
	created, err := s.credential.Credential(ctx, credential.Id)
	if err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to load credential", err)
	}
	return created, nil
}

func (s Service) encryptCredentialData(data string) (string, error) {
	encrypted, err := security.EncryptString(s.secretKey, data)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to encrypt credential", err)
	}
	return encrypted, nil
}

func (s Service) decryptCredentialData(data string) (string, error) {
	decrypted, err := security.DecryptString(s.secretKey, data)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to decrypt credential", err)
	}
	return decrypted, nil
}

func (s Service) loadCredentialForUser(ctx context.Context, userId string, credentialId string) (model.Credential, error) {
	credentialId = strings.TrimSpace(credentialId)
	credential, err := s.credential.Credential(ctx, credentialId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Credential{}, apperror.New(apperror.KindNotFound, "Credential "+credentialId+" not found")
		}
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to load credential", err)
	}
	if credential.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *credential.ProjectId, userId); err != nil {
			return model.Credential{}, err
		}
	}
	return credential, nil
}

func (s Service) ensureCredentialNameAvailable(ctx context.Context, projectId string, name string, excludeId string) error {
	existing, err := s.credential.CredentialByName(ctx, projectId, name)
	if err == nil {
		if existing.Id != excludeId {
			return apperror.New(apperror.KindConflict, "凭据名称 '"+name+"' 已存在")
		}
		return nil
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check credential name", err)
	}
	return nil
}

func normalizeCredentialCreateInput(input credentialdto.CredentialCreateInput) (string, string, string, string, error) {
	projectId := strings.TrimSpace(input.ProjectId)
	name := strings.TrimSpace(input.Name)
	credentialType := strings.TrimSpace(input.Type)
	if projectId == "" {
		return "", "", "", "", apperror.New(apperror.KindValidation, "project_id is required")
	}
	if name == "" || input.Data == "" || !validCredentialType(credentialType) {
		return "", "", "", "", apperror.New(apperror.KindValidation, "Invalid credential fields")
	}
	return projectId, name, credentialType, input.Data, nil
}

func credentialProjectId(item model.Credential) string {
	if item.ProjectId == nil {
		return ""
	}
	return *item.ProjectId
}

func validCredentialType(value string) bool {
	switch value {
	case "git_ssh", "github_token", "gitee_token", "registry_token", "runtime_env":
		return true
	default:
		return false
	}
}

func validRuntimeEnvCredentialData(raw string) bool {
	_, ok := runtimeEnvCredentialValues(raw)
	return ok
}

func runtimeEnvCredentialValues(raw string) (map[string]string, bool) {
	var values map[string]string
	if err := json.Unmarshal([]byte(raw), &values); err != nil || len(values) == 0 {
		return nil, false
	}
	for key, value := range values {
		if !runtimeEnvKeyPattern.MatchString(key) || value == "" {
			return nil, false
		}
	}
	return values, true
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.project.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.project.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

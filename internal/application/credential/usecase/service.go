package credentialsvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"golang.org/x/crypto/ssh"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
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

// CreateDeploymentSSHCredential is intentionally an internal application
// surface. Project bootstrap calls it inside its transaction; public credential
// routes cannot create this credential type.
func (s Service) CreateDeploymentSSHCredential(ctx context.Context, projectID string, name string, privateKey string, passphrase string) (model.Credential, error) {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return model.Credential{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if _, err := s.project.Project(ctx, projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Credential{}, apperror.New(apperror.KindNotFound, "Project "+projectID+" not found")
		}
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	payload, err := normalizeDeploymentSSHPrivateKey(privateKey, passphrase)
	if err != nil {
		return model.Credential{}, err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to encode deployment SSH credential", err)
	}
	return s.createCredentialRecord(ctx, projectID, strings.TrimSpace(name), model.CredentialTypeDeploymentSSHPrivateKey, string(encoded))
}

// UpdateDeploymentSSHCredential rotates the system-managed deploy key without
// ever returning its decrypted contents to an API caller.
func (s Service) UpdateDeploymentSSHCredential(ctx context.Context, credentialID string, privateKey *string, passphrase *string) (model.Credential, error) {
	credential, err := s.credential.Credential(ctx, strings.TrimSpace(credentialID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Credential{}, apperror.New(apperror.KindNotFound, "Deployment SSH credential not found")
		}
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment SSH credential", err)
	}
	if !credential.IsDeploymentSSHPrivateKey() {
		return model.Credential{}, apperror.New(apperror.KindValidation, "Credential is not a deployment SSH private key")
	}

	var payload credentialdto.DeploymentSSHPrivateKey
	if credential.RequiresDeploymentSSHCredentialReconfiguration() {
		if privateKey == nil {
			return model.Credential{}, apperror.New(apperror.KindValidation, "Deployment SSH private key must be configured before this environment can be used")
		}
		payload.PrivateKey = *privateKey
		if passphrase != nil {
			payload.Passphrase = *passphrase
		}
	} else {
		payload, err = s.deploymentPayload(credential)
		if err != nil {
			return model.Credential{}, err
		}
		if privateKey != nil {
			payload.PrivateKey = *privateKey
		}
		if passphrase != nil {
			payload.Passphrase = *passphrase
		}
	}
	payload, err = normalizeDeploymentSSHPrivateKey(payload.PrivateKey, payload.Passphrase)
	if err != nil {
		return model.Credential{}, err
	}
	encoded, err := json.Marshal(payload)
	if err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to encode deployment SSH credential", err)
	}
	encrypted, err := s.encryptCredentialData(string(encoded))
	if err != nil {
		return model.Credential{}, err
	}
	credential.EncryptedData = encrypted
	credential.Revision++
	if err := s.credential.UpdateCredential(ctx, credential); err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to update deployment SSH credential", err)
	}
	updated, err := s.credential.Credential(ctx, credential.Id)
	if err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment SSH credential", err)
	}
	return updated, nil
}

// DeploymentSSHCredential is deliberately not exposed through HTTP or MCP.
// SSH execution adapters use it only after Project/Environment resolution.
func (s Service) DeploymentSSHCredential(ctx context.Context, credentialID string) (model.Credential, credentialdto.DeploymentSSHPrivateKey, error) {
	credential, err := s.credential.Credential(ctx, strings.TrimSpace(credentialID))
	if err != nil {
		return model.Credential{}, credentialdto.DeploymentSSHPrivateKey{}, err
	}
	if !credential.IsDeploymentSSHPrivateKey() {
		return model.Credential{}, credentialdto.DeploymentSSHPrivateKey{}, apperror.New(apperror.KindValidation, "Credential is not a deployment SSH private key")
	}
	if credential.RequiresDeploymentSSHCredentialReconfiguration() {
		return model.Credential{}, credentialdto.DeploymentSSHPrivateKey{}, apperror.New(apperror.KindValidation, "Deployment SSH private key must be configured before this environment can be used")
	}
	payload, err := s.deploymentPayload(credential)
	if err != nil {
		return model.Credential{}, credentialdto.DeploymentSSHPrivateKey{}, err
	}
	return credential, payload, nil
}
func (s Service) CredentialForUser(ctx context.Context, userId string, credentialId string) (model.Credential, error) {
	return s.loadCredentialForUser(ctx, userId, credentialId)
}

func (s Service) CredentialDetailForUser(ctx context.Context, userId string, credentialId string) (credentialdto.CredentialDetail, error) {
	credential, err := s.loadCredentialForUser(ctx, userId, credentialId)
	if err != nil {
		return credentialdto.CredentialDetail{}, err
	}
	if credential.IsDeploymentSSHPrivateKey() {
		return credentialdto.CredentialDetail{}, managedDeploymentKeyError()
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
	if credential.IsDeploymentSSHPrivateKey() {
		return model.Credential{}, managedDeploymentKeyError()
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
		encrypted, err := s.encryptCredentialData(*input.Data)
		if err != nil {
			return model.Credential{}, err
		}
		credential.EncryptedData = encrypted
	}
	credential.Revision++
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
	if credential.IsDeploymentSSHPrivateKey() {
		return managedDeploymentKeyError()
	}
	referenced, err := s.credential.CredentialReferencedByRepositories(ctx, credentialProjectId(credential), credential.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check credential references", err)
	}
	if referenced {
		return apperror.New(apperror.KindValidation, "Credential is still used by repositories. Remove it from those repositories before deleting it.")
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
	if credential.IsDeploymentSSHPrivateKey() {
		return credentialdto.CredentialExport{}, managedDeploymentKeyError()
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
	encrypted, err := s.encryptCredentialData(data)
	if err != nil {
		return model.Credential{}, err
	}
	credential := model.Credential{Id: idutil.NewId(), ProjectId: &projectId, Name: name, Type: credentialType, EncryptedData: encrypted, Revision: 1, CreatedAt: time.Now().UTC()}
	if err := s.credential.CreateCredential(ctx, credential); err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to create credential", err)
	}
	created, err := s.credential.Credential(ctx, credential.Id)
	if err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to load credential", err)
	}
	return created, nil
}

func (s Service) deploymentPayload(credential model.Credential) (credentialdto.DeploymentSSHPrivateKey, error) {
	decrypted, err := s.decryptCredentialData(credential.EncryptedData)
	if err != nil {
		return credentialdto.DeploymentSSHPrivateKey{}, err
	}
	var payload credentialdto.DeploymentSSHPrivateKey
	if err := json.Unmarshal([]byte(decrypted), &payload); err != nil {
		return credentialdto.DeploymentSSHPrivateKey{}, apperror.Wrap(apperror.KindInternal, "Stored deployment SSH credential is invalid", err)
	}
	return normalizeDeploymentSSHPrivateKey(payload.PrivateKey, payload.Passphrase)
}

func normalizeDeploymentSSHPrivateKey(privateKey string, passphrase string) (credentialdto.DeploymentSSHPrivateKey, error) {
	privateKey = strings.TrimSpace(privateKey)
	if privateKey == "" {
		return credentialdto.DeploymentSSHPrivateKey{}, apperror.New(apperror.KindValidation, "Deployment SSH private key is required")
	}
	var err error
	if passphrase == "" {
		_, err = ssh.ParsePrivateKey([]byte(privateKey))
	} else {
		_, err = ssh.ParsePrivateKeyWithPassphrase([]byte(privateKey), []byte(passphrase))
	}
	if err != nil {
		return credentialdto.DeploymentSSHPrivateKey{}, apperror.New(apperror.KindValidation, "Deployment SSH private key is invalid")
	}
	return credentialdto.DeploymentSSHPrivateKey{PrivateKey: privateKey, Passphrase: passphrase}, nil
}

func managedDeploymentKeyError() error {
	return apperror.New(apperror.KindForbidden, "Deployment SSH credentials are managed by the project environment")
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
	if name == "" {
		return apperror.New(apperror.KindValidation, "Invalid credential fields")
	}
	existing, err := s.credential.CredentialByName(ctx, projectId, name)
	if err == nil {
		if existing.Id != excludeId {
			return apperror.New(apperror.KindConflict, "Credential name '"+name+"' already exists")
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
	case "git_ssh", "github_token", "gitee_token", "gitea_token", "registry_token":
		return true
	default:
		return false
	}
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

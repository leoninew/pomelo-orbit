package cisvc

import (
	"context"
	"database/sql"
	"errors"
	"strings"

	"backend/internal/apperror"
	"backend/internal/repository"
	"backend/internal/repository/model"
	"backend/internal/security"
)

const CredentialExportVersion = "1.0"

type CredentialCreateInput struct {
	ProjectId string
	Name      string
	Type      string
	Data      string
}

type CredentialUpdateInput struct {
	Name *string
	Data *string
}

type CredentialExport struct {
	Version string
	Name    string
	Type    string
	Data    string
}

type CredentialDetail struct {
	Credential model.Credential
	Data       string
}

func (s Service) ListCredentials(ctx context.Context, userId string, projectId string, page int, perPage int, search string) (repository.Page[model.Credential], error) {
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return repository.Page[model.Credential]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return repository.Page[model.Credential]{}, err
	}
	items, err := s.store.ListCredentials(ctx, projectId, page, perPage, search)
	if err != nil {
		return repository.Page[model.Credential]{}, apperror.Wrap(apperror.KindInternal, "Failed to list credentials", err)
	}
	return items, nil
}

func (s Service) CreateCredential(ctx context.Context, userId string, input CredentialCreateInput) (model.Credential, error) {
	projectId, name, credentialType, data, err := normalizeCredentialCreateInput(input)
	if err != nil {
		return model.Credential{}, err
	}
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Credential{}, err
	}
	return s.createCredentialRecord(ctx, projectId, name, credentialType, data)
}

func (s Service) ImportCredential(ctx context.Context, userId string, input CredentialCreateInput) (model.Credential, error) {
	return s.CreateCredential(ctx, userId, input)
}

func (s Service) CredentialForUser(ctx context.Context, userId string, credentialId string) (model.Credential, error) {
	return s.loadCredentialForUser(ctx, userId, credentialId)
}

func (s Service) CredentialDetailForUser(ctx context.Context, userId string, credentialId string) (CredentialDetail, error) {
	credential, err := s.loadCredentialForUser(ctx, userId, credentialId)
	if err != nil {
		return CredentialDetail{}, err
	}
	decrypted, err := s.decryptCredentialData(credential.EncryptedData)
	if err != nil {
		return CredentialDetail{}, err
	}
	return CredentialDetail{Credential: credential, Data: decrypted}, nil
}

func (s Service) UpdateCredential(ctx context.Context, userId string, credentialId string, input CredentialUpdateInput) (model.Credential, error) {
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
		encrypted, err := s.encryptCredentialData(*input.Data)
		if err != nil {
			return model.Credential{}, err
		}
		credential.EncryptedData = encrypted
	}
	if err := s.store.UpdateCredential(ctx, credential); err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to update credential", err)
	}
	updated, err := s.store.Credential(ctx, credential.Id)
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
	referenced, err := s.store.CredentialReferencedByRepositories(ctx, credentialProjectId(credential), credential.Id)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check credential references", err)
	}
	if referenced {
		return apperror.New(apperror.KindConflict, "Credential is referenced by projects, cannot delete")
	}
	if err := s.store.DeleteCredential(ctx, credential.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete credential", err)
	}
	return nil
}

func (s Service) ExportCredential(ctx context.Context, userId string, credentialId string) (CredentialExport, error) {
	credential, err := s.loadCredentialForUser(ctx, userId, credentialId)
	if err != nil {
		return CredentialExport{}, err
	}
	decrypted, err := s.decryptCredentialData(credential.EncryptedData)
	if err != nil {
		return CredentialExport{}, err
	}
	return CredentialExport{Version: CredentialExportVersion, Name: credential.Name, Type: credential.Type, Data: decrypted}, nil
}

func (s Service) createCredentialRecord(ctx context.Context, projectId string, name string, credentialType string, data string) (model.Credential, error) {
	if err := s.ensureCredentialNameAvailable(ctx, projectId, name, ""); err != nil {
		return model.Credential{}, err
	}
	encrypted, err := s.encryptCredentialData(data)
	if err != nil {
		return model.Credential{}, err
	}
	credential := model.Credential{Id: repository.NewId(), ProjectId: &projectId, Name: name, Type: credentialType, EncryptedData: encrypted}
	if err := s.store.CreateCredential(ctx, credential); err != nil {
		return model.Credential{}, apperror.Wrap(apperror.KindInternal, "Failed to create credential", err)
	}
	created, err := s.store.Credential(ctx, credential.Id)
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
	credential, err := s.store.Credential(ctx, credentialId)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
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
	existing, err := s.store.CredentialByName(ctx, projectId, name)
	if err == nil {
		if existing.Id != excludeId {
			return apperror.New(apperror.KindConflict, "凭据名称 '"+name+"' 已存在")
		}
		return nil
	}
	if !errors.Is(err, sql.ErrNoRows) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check credential name", err)
	}
	return nil
}

func normalizeCredentialCreateInput(input CredentialCreateInput) (string, string, string, string, error) {
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
	case "git_ssh", "github_token", "gitee_token", "registry_token":
		return true
	default:
		return false
	}
}

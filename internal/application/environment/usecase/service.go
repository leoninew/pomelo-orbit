package environmentsvc

import (
	"context"
	"errors"
	"path"
	"regexp"
	"strings"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var hostKeyFingerprintPattern = regexp.MustCompile(`^SHA256:[A-Za-z0-9+/]+={0,2}$`)
var windowsWorkspacePattern = regexp.MustCompile(`^[A-Za-z]:\\`)

type deploymentKeyUpdater interface {
	UpdateDeploymentSSHCredential(ctx context.Context, credentialID string, privateKey *string, passphrase *string) (model.Credential, error)
}

type Service struct {
	environments         repository.EnvironmentStore
	projects             repository.ProjectReader
	deploymentKey        deploymentKeyUpdater
	deploymentCredential deploymentCredentialReader
	prober               environmentProber
}

func New(environments repository.EnvironmentStore, projects repository.ProjectReader, deploymentKey deploymentKeyUpdater, deploymentCredential deploymentCredentialReader, prober environmentProber) Service {
	return Service{
		environments:         environments,
		projects:             projects,
		deploymentKey:        deploymentKey,
		deploymentCredential: deploymentCredential,
		prober:               prober,
	}
}

// BootstrapForProject creates the one Environment owned by a newly created
// Project. Its caller owns the surrounding Project/Credential transaction.
func (s Service) BootstrapForProject(ctx context.Context, project model.Project, credential model.Credential, input environmentdto.BootstrapInput) (model.Environment, error) {
	item := model.Environment{
		Id:                    idutil.NewId(),
		ProjectId:             project.Id,
		Code:                  project.Code,
		State:                 strings.TrimSpace(input.State),
		Platform:              strings.TrimSpace(input.Platform),
		Host:                  strings.TrimSpace(input.Host),
		Port:                  input.Port,
		Username:              strings.TrimSpace(input.Username),
		WorkspaceRoot:         strings.TrimSpace(input.WorkspaceRoot),
		SSHCredentialId:       credential.Id,
		SSHCredentialRevision: credential.Revision,
		HostKeyFingerprint:    strings.TrimSpace(input.HostKeyFingerprint),
		TargetRevision:        1,
		CreatedAt:             time.Now().UTC(),
		UpdatedAt:             time.Now().UTC(),
	}
	if err := validateBootstrap(project, credential, item); err != nil {
		return model.Environment{}, err
	}
	if err := s.environments.CreateEnvironment(ctx, item); err != nil {
		return model.Environment{}, apperror.Wrap(apperror.KindInternal, "Failed to create project environment", err)
	}
	return item, nil
}

func (s Service) EnvironmentForUser(ctx context.Context, userID string, projectID string) (model.Environment, error) {
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return model.Environment{}, err
	}
	return s.environmentForProject(ctx, projectID)
}

func (s Service) UpdateForUser(ctx context.Context, userID string, projectID string, input environmentdto.UpdateInput) (model.Environment, error) {
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return model.Environment{}, err
	}
	item, err := s.environmentForProject(ctx, projectID)
	if err != nil {
		return model.Environment{}, err
	}
	previous := item
	applyUpdate(&item, input)
	if input.DeploymentSSHPrivateKey != nil || input.DeploymentSSHKeyPassphrase != nil {
		if s.deploymentKey == nil {
			return model.Environment{}, apperror.New(apperror.KindInternal, "deployment SSH key manager is not configured")
		}
		credential, err := s.deploymentKey.UpdateDeploymentSSHCredential(
			ctx,
			item.SSHCredentialId,
			input.DeploymentSSHPrivateKey,
			input.DeploymentSSHKeyPassphrase,
		)
		if err != nil {
			return model.Environment{}, err
		}
		item.SSHCredentialRevision = credential.Revision
	}
	if item.IsActive() {
		if s.deploymentCredential == nil {
			return model.Environment{}, apperror.New(apperror.KindInternal, "deployment SSH credential reader is not configured")
		}
		credential, _, err := s.deploymentCredential.DeploymentSSHCredential(ctx, item.SSHCredentialId)
		if err != nil || !matchesEnvironmentCredential(item, credential) {
			return model.Environment{}, apperror.New(apperror.KindValidation, "Environment deployment SSH credential must be configured before it can be active")
		}
	}
	if err := validateEnvironment(item); err != nil {
		return model.Environment{}, err
	}
	if targetChanged(previous, item) {
		item.TargetRevision++
	}
	if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
		return model.Environment{}, apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
	}
	return s.environmentForProject(ctx, projectID)
}

func (s Service) environmentForProject(ctx context.Context, projectID string) (model.Environment, error) {
	item, err := s.environments.EnvironmentByProject(ctx, strings.TrimSpace(projectID))
	if errors.Is(err, repository.ErrNotFound) {
		return model.Environment{}, apperror.New(apperror.KindNotFound, "Project environment not found")
	}
	if err != nil {
		return model.Environment{}, apperror.Wrap(apperror.KindInternal, "Failed to load project environment", err)
	}
	return item, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectID string, userID string) error {
	projectID = strings.TrimSpace(projectID)
	if projectID == "" {
		return apperror.New(apperror.KindValidation, "project_id is required")
	}
	if _, err := s.projects.Project(ctx, projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectID+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.projects.IsProjectMember(ctx, projectID, userID)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func validateBootstrap(project model.Project, credential model.Credential, item model.Environment) error {
	if project.Id == "" || project.Code == "" || credential.Id == "" || credential.ProjectId == nil || *credential.ProjectId != project.Id || !credential.IsDeploymentSSHPrivateKey() {
		return apperror.New(apperror.KindInternal, "Project environment bootstrap binding is invalid")
	}
	return validateEnvironment(item)
}

func validateEnvironment(item model.Environment) error {
	if item.ProjectId == "" || item.Code == "" {
		return apperror.New(apperror.KindValidation, "Project environment identity is invalid")
	}
	if item.State != model.EnvironmentStateActive && item.State != model.EnvironmentStateDisabled {
		return apperror.New(apperror.KindValidation, "Environment state must be active or disabled")
	}
	if item.Platform != model.EnvironmentPlatformLinux && item.Platform != model.EnvironmentPlatformWindows {
		return apperror.New(apperror.KindValidation, "Environment platform must be linux or windows")
	}
	if item.Host == "" || strings.ContainsAny(item.Host, " \t\r\n") || item.Port < 1 || item.Port > 65535 || item.Username == "" || strings.ContainsAny(item.Username, "\r\n") {
		return apperror.New(apperror.KindValidation, "Environment SSH target is invalid")
	}
	if !validWorkspaceRoot(item.Platform, item.WorkspaceRoot) {
		return apperror.New(apperror.KindValidation, "Environment workspace_root is invalid for its platform")
	}
	if item.SSHCredentialId == "" || item.SSHCredentialRevision < 1 {
		return apperror.New(apperror.KindValidation, "Environment SSH credential binding is invalid")
	}
	if !hostKeyFingerprintPattern.MatchString(item.HostKeyFingerprint) {
		return apperror.New(apperror.KindValidation, "Environment host_key_fingerprint must use SHA256 format")
	}
	return nil
}

func applyUpdate(item *model.Environment, input environmentdto.UpdateInput) {
	if input.State != nil {
		item.State = strings.TrimSpace(*input.State)
	}
	if input.Platform != nil {
		item.Platform = strings.TrimSpace(*input.Platform)
	}
	if input.Host != nil {
		item.Host = strings.TrimSpace(*input.Host)
	}
	if input.Port != nil {
		item.Port = *input.Port
	}
	if input.Username != nil {
		item.Username = strings.TrimSpace(*input.Username)
	}
	if input.WorkspaceRoot != nil {
		item.WorkspaceRoot = strings.TrimSpace(*input.WorkspaceRoot)
	}
	if input.HostKeyFingerprint != nil {
		item.HostKeyFingerprint = strings.TrimSpace(*input.HostKeyFingerprint)
	}
}

func validWorkspaceRoot(platform string, workspaceRoot string) bool {
	if workspaceRoot == "" || strings.ContainsAny(workspaceRoot, "\r\n") {
		return false
	}
	switch platform {
	case model.EnvironmentPlatformLinux:
		return path.IsAbs(workspaceRoot)
	case model.EnvironmentPlatformWindows:
		return windowsWorkspacePattern.MatchString(workspaceRoot)
	default:
		return false
	}
}

func targetChanged(before, after model.Environment) bool {
	return before.Platform != after.Platform ||
		before.Host != after.Host ||
		before.Port != after.Port ||
		before.Username != after.Username ||
		before.WorkspaceRoot != after.WorkspaceRoot ||
		before.SSHCredentialId != after.SSHCredentialId ||
		before.SSHCredentialRevision != after.SSHCredentialRevision ||
		before.HostKeyFingerprint != after.HostKeyFingerprint
}

package environmentsvc

import (
	"context"
	"errors"
	"path"
	"regexp"
	"strings"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var hostKeyFingerprintPattern = regexp.MustCompile(`^SHA256:[A-Za-z0-9+/]+={0,2}$`)
var windowsWorkspacePattern = regexp.MustCompile(`^[A-Za-z]:\\`)

type deploymentKeyManager interface {
	CreateDeploymentSSHCredential(ctx context.Context, projectID string, name string) (model.Credential, error)
	EnsureGeneratedDeploymentSSHCredential(ctx context.Context, credentialID string) (model.Credential, error)
	DeploymentSSHPublicKey(ctx context.Context, credentialID string) (string, error)
}

type Service struct {
	environments         repository.EnvironmentStore
	projects             repository.ProjectReader
	deploymentKey        deploymentKeyManager
	deploymentCredential deploymentCredentialReader
	prober               environmentProber
	bootstrapper         environmentport.Bootstrapper
}

func New(environments repository.EnvironmentStore, projects repository.ProjectReader, deploymentKey deploymentKeyManager, deploymentCredential deploymentCredentialReader, prober environmentProber, bootstrapper environmentport.Bootstrapper) Service {
	return Service{
		environments: environments, projects: projects, deploymentKey: deploymentKey,
		deploymentCredential: deploymentCredential, prober: prober, bootstrapper: bootstrapper,
	}
}

// BootstrapForProject creates the active local Environment owned by a newly
// created Project. SSH configuration is managed from the Environment page.
func (s Service) BootstrapForProject(ctx context.Context, project model.Project) (model.Environment, error) {
	item := model.Environment{
		Id: idutil.NewId(), ProjectId: project.Id, Code: project.Code,
		State: model.EnvironmentStateActive, TargetType: model.EnvironmentTargetTypeLocal,
		TargetRevision: 1, CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
	}
	if err := validateBootstrap(project, item); err != nil {
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
	item, err := s.environmentForProject(ctx, projectID)
	if err != nil {
		return model.Environment{}, err
	}
	return s.hydrateEnvironment(ctx, item)
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
	if err := applyUpdate(&item, input); err != nil {
		return model.Environment{}, err
	}
	item, err = s.ensureGeneratedCredential(ctx, item)
	if err != nil {
		return model.Environment{}, err
	}
	if item.IsActive() && item.IsSSH() {
		if s.deploymentCredential == nil {
			return model.Environment{}, apperror.New(apperror.KindInternal, "deployment SSH credential reader is not configured")
		}
		credential, _, err := s.deploymentCredential.DeploymentSSHCredential(ctx, item.SSH.CredentialId)
		if err != nil || !matchesEnvironmentCredential(item, credential) {
			return model.Environment{}, apperror.New(apperror.KindValidation, "Environment deployment SSH credential must be configured before it can be active")
		}
	}
	if err := validateEnvironment(item); err != nil {
		return model.Environment{}, err
	}
	if environmentTargetChanged(previous, item) {
		if item.SSH != nil {
			item.SSH.HostKeyFingerprint = ""
		}
		item.TargetRevision++
	}
	if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
		return model.Environment{}, apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
	}
	item, err = s.environmentForProject(ctx, projectID)
	if err != nil {
		return model.Environment{}, err
	}
	return item, nil
}

func (s Service) hydrateEnvironment(ctx context.Context, item model.Environment) (model.Environment, error) {
	if !item.IsSSH() {
		return item, nil
	}
	previous := item
	item, err := s.ensureGeneratedCredential(ctx, item)
	if err != nil {
		return model.Environment{}, err
	}
	if environmentTargetChanged(previous, item) {
		item.SSH.HostKeyFingerprint = ""
		item.TargetRevision++
		if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
			return model.Environment{}, apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
		}
	}
	return item, nil
}

func (s Service) ensureGeneratedCredential(ctx context.Context, item model.Environment) (model.Environment, error) {
	if !item.IsSSH() || s.deploymentKey == nil {
		return item, nil
	}
	if strings.TrimSpace(item.SSH.CredentialId) == "" {
		credential, err := s.deploymentKey.CreateDeploymentSSHCredential(ctx, item.ProjectId, "")
		if err != nil {
			return model.Environment{}, err
		}
		item.SSH.CredentialId = credential.Id
		item.SSH.CredentialRevision = credential.Revision
		return item, nil
	}
	credential, err := s.deploymentKey.EnsureGeneratedDeploymentSSHCredential(ctx, item.SSH.CredentialId)
	if err != nil {
		return model.Environment{}, err
	}
	if credential.Revision != item.SSH.CredentialRevision {
		item.SSH.CredentialRevision = credential.Revision
		item.SSH.HostKeyFingerprint = ""
	}
	return item, nil
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

func validateBootstrap(project model.Project, item model.Environment) error {
	if project.Id == "" || project.Code == "" {
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
	switch item.TargetType {
	case model.EnvironmentTargetTypeLocal:
		if item.SSH != nil {
			return apperror.New(apperror.KindValidation, "Local environment must not include an SSH target")
		}
	case model.EnvironmentTargetTypeSSH:
		if item.SSH == nil {
			return apperror.New(apperror.KindValidation, "SSH environment target is required")
		}
		if item.SSH.Platform != model.EnvironmentPlatformLinux && item.SSH.Platform != model.EnvironmentPlatformWindows {
			return apperror.New(apperror.KindValidation, "Environment SSH platform must be linux or windows")
		}
		if item.SSH.Host == "" || strings.ContainsAny(item.SSH.Host, " \t\r\n") || item.SSH.Port < 1 || item.SSH.Port > 65535 || item.SSH.Username == "" || strings.ContainsAny(item.SSH.Username, "\r\n") {
			return apperror.New(apperror.KindValidation, "Environment SSH target is invalid")
		}
		if !validWorkspaceRoot(item.SSH.Platform, item.SSH.WorkspaceRoot) {
			return apperror.New(apperror.KindValidation, "Environment SSH workspace_root is invalid for its platform")
		}
		if item.SSH.CredentialId == "" || item.SSH.CredentialRevision < 1 {
			return apperror.New(apperror.KindValidation, "Environment SSH credential binding is invalid")
		}
		if item.SSH.HostKeyFingerprint != "" && !hostKeyFingerprintPattern.MatchString(item.SSH.HostKeyFingerprint) {
			return apperror.New(apperror.KindValidation, "Environment host_key_fingerprint must use SHA256 format")
		}
	default:
		return apperror.New(apperror.KindValidation, "Environment target_type must be local or ssh")
	}
	return nil
}

func applyUpdate(item *model.Environment, input environmentdto.UpdateInput) error {
	if input.State != nil {
		item.State = strings.TrimSpace(*input.State)
	}
	if input.TargetType != nil {
		targetType := strings.TrimSpace(*input.TargetType)
		switch targetType {
		case model.EnvironmentTargetTypeLocal:
			item.TargetType = targetType
			item.SSH = nil
		case model.EnvironmentTargetTypeSSH:
			if input.SSH == nil {
				return apperror.New(apperror.KindValidation, "SSH target is required when target_type is ssh")
			}
			ssh := sshTargetFromInput(input.SSH, nil)
			if item.IsSSH() {
				ssh.CredentialId = item.SSH.CredentialId
				ssh.CredentialRevision = item.SSH.CredentialRevision
			}
			item.TargetType = targetType
			item.SSH = ssh
		default:
			return apperror.New(apperror.KindValidation, "Environment target_type must be local or ssh")
		}
	} else if input.SSH != nil {
		if !item.IsSSH() {
			return apperror.New(apperror.KindValidation, "SSH target is only valid for an ssh environment")
		}
		credentialID, credentialRevision := item.SSH.CredentialId, item.SSH.CredentialRevision
		item.SSH = sshTargetFromInput(input.SSH, nil)
		item.SSH.CredentialId, item.SSH.CredentialRevision = credentialID, credentialRevision
	}
	return nil
}

func sshTargetFromInput(input *environmentdto.SSHTargetInput, credential *model.Credential) *model.EnvironmentSSHTarget {
	if input == nil {
		return nil
	}
	target := &model.EnvironmentSSHTarget{
		Platform: strings.TrimSpace(input.Platform), Host: strings.TrimSpace(input.Host), Port: input.Port,
		Username: strings.TrimSpace(input.Username), WorkspaceRoot: strings.TrimSpace(input.WorkspaceRoot),
	}
	if credential != nil {
		target.CredentialId = credential.Id
		target.CredentialRevision = credential.Revision
	}
	return target
}

func validWorkspaceRoot(platform string, workspaceRoot string) bool {
	if workspaceRoot == "" || strings.ContainsAny(workspaceRoot, "\r\n") {
		return false
	}
	if workspaceRoot == "~" || strings.HasPrefix(workspaceRoot, "~/") {
		return true
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

func environmentTargetChanged(before, after model.Environment) bool {
	if before.TargetType != after.TargetType {
		return true
	}
	if before.SSH == nil || after.SSH == nil {
		return before.SSH != after.SSH
	}
	return before.SSH.Platform != after.SSH.Platform ||
		before.SSH.Host != after.SSH.Host ||
		before.SSH.Port != after.SSH.Port ||
		before.SSH.Username != after.SSH.Username ||
		before.SSH.WorkspaceRoot != after.SSH.WorkspaceRoot ||
		before.SSH.CredentialId != after.SSH.CredentialId ||
		before.SSH.CredentialRevision != after.SSH.CredentialRevision
}

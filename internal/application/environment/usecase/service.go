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
	prober               environmentport.Prober
	bootstrapper         environmentport.Bootstrapper
	localDisplay         environmentdto.LocalDisplaySnapshot
}

func New(environments repository.EnvironmentStore, projects repository.ProjectReader, deploymentKey deploymentKeyManager, deploymentCredential deploymentCredentialReader, prober environmentport.Prober, bootstrapper environmentport.Bootstrapper) Service {
	return Service{
		environments: environments, projects: projects, deploymentKey: deploymentKey,
		deploymentCredential: deploymentCredential, prober: prober, bootstrapper: bootstrapper,
	}
}

func (s Service) WithLocalDisplay(snapshot environmentdto.LocalDisplaySnapshot) Service {
	s.localDisplay = snapshot
	return s
}

// SaveInitialization creates or updates the Project Environment during Wizard
// initialization. It is not used after Gateway creation.
func (s Service) SaveInitialization(ctx context.Context, userID string, projectID string, input environmentdto.UpdateInput) (environmentdto.View, error) {
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return environmentdto.View{}, err
	}
	project, err := s.projects.Project(ctx, strings.TrimSpace(projectID))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return environmentdto.View{}, apperror.New(apperror.KindNotFound, "Project "+projectID+" not found")
		}
		return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	item, err := s.environments.EnvironmentByProject(ctx, project.Id)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to load project environment", err)
	}
	creating := errors.Is(err, repository.ErrNotFound)
	if creating {
		item = model.Environment{
			Id: idutil.NewId(), ProjectId: project.Id, Code: project.Code,
			State: model.EnvironmentStateActive, TargetRevision: 1,
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
	}
	previous := item
	item.State = model.EnvironmentStateActive
	if err := applyUpdate(&item, input); err != nil {
		return environmentdto.View{}, err
	}
	item, err = s.ensureGeneratedCredential(ctx, item)
	if err != nil {
		return environmentdto.View{}, err
	}
	if err := validateEnvironment(item, s.localDisplay.Platform, false); err != nil {
		return environmentdto.View{}, err
	}
	if item.IsSSH() {
		if err := s.testSSHTarget(ctx, item.SSH.Host, item.SSH.Port, item.SSH.Username); err != nil {
			return environmentdto.View{}, err
		}
	}
	if !creating && environmentTargetChanged(previous, item) {
		if item.SSH != nil {
			item.SSH.HostKeyFingerprint = ""
		}
		item.TargetRevision++
	}
	if creating {
		if err := s.environments.CreateEnvironment(ctx, item); err != nil {
			return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to create project environment", err)
		}
		return s.toView(item), nil
	}
	if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
		return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
	}
	item, err = s.environmentForProject(ctx, project.Id)
	if err != nil {
		return environmentdto.View{}, err
	}
	return s.toView(item), nil
}

func (s Service) TestSSHReachability(ctx context.Context, userID string, projectID string, input environmentdto.SSHTargetInput) error {
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return err
	}
	return s.testSSHTarget(ctx, input.Host, input.Port, input.Username)
}

// DeploymentSSHPublicKeyForProject returns a Project's managed deployment key.
// It does not require an Environment to have been saved, allowing a Windows
// host to receive its key before SSH key authentication is configured.
func (s Service) DeploymentSSHPublicKeyForProject(ctx context.Context, userID string, projectID string) (string, error) {
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return "", err
	}
	if s.deploymentKey == nil {
		return "", apperror.New(apperror.KindInternal, "deployment SSH credential manager is not configured")
	}
	credentialID := ""
	item, err := s.environmentForProject(ctx, projectID)
	if err == nil && item.IsSSH() && item.SSH != nil {
		credentialID = strings.TrimSpace(item.SSH.CredentialId)
	} else if err != nil && !apperror.IsKind(err, apperror.KindNotFound) {
		return "", err
	}
	if credentialID == "" {
		credential, err := s.deploymentKey.CreateDeploymentSSHCredential(ctx, projectID, "")
		if err != nil {
			return "", err
		}
		credentialID = credential.Id
	}
	publicKey, err := s.deploymentKey.DeploymentSSHPublicKey(ctx, credentialID)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(publicKey) == "" {
		return "", apperror.New(apperror.KindInternal, "Generated deployment SSH public key is empty")
	}
	return strings.TrimSpace(publicKey), nil
}

func (s Service) testSSHTarget(ctx context.Context, host string, port int, username string) error {
	if s.prober == nil {
		return apperror.New(apperror.KindInternal, "SSH reachability tester is not configured")
	}
	if err := s.prober.TestSSH(ctx, host, port, username); err != nil {
		if diagnostic, ok := err.(probeDiagnosticError); ok && strings.TrimSpace(diagnostic.ProbeDiagnostic()) != "" {
			return apperror.New(apperror.KindValidation, diagnostic.ProbeDiagnostic())
		}
		return apperror.New(apperror.KindValidation, "Cannot connect to the configured SSH host.")
	}
	return nil
}

func (s Service) EnvironmentForUser(ctx context.Context, userID string, projectID string) (environmentdto.View, error) {
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return environmentdto.View{}, err
	}
	item, err := s.environmentForProject(ctx, projectID)
	if err != nil {
		return environmentdto.View{}, err
	}
	item, err = s.hydrateEnvironment(ctx, item)
	if err != nil {
		return environmentdto.View{}, err
	}
	return s.toView(item), nil
}

func (s Service) UpdateForUser(ctx context.Context, userID string, projectID string, input environmentdto.UpdateInput) (environmentdto.View, error) {
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return environmentdto.View{}, err
	}
	item, err := s.environmentForProject(ctx, projectID)
	if err != nil {
		return environmentdto.View{}, err
	}
	previous := item
	if err := applyUpdate(&item, input); err != nil {
		return environmentdto.View{}, err
	}
	item, err = s.ensureGeneratedCredential(ctx, item)
	if err != nil {
		return environmentdto.View{}, err
	}
	if item.IsActive() && item.IsSSH() {
		if s.deploymentCredential == nil {
			return environmentdto.View{}, apperror.New(apperror.KindInternal, "deployment SSH credential reader is not configured")
		}
		credential, _, err := s.deploymentCredential.DeploymentSSHCredential(ctx, item.SSH.CredentialId)
		if err != nil || !matchesEnvironmentCredential(item, credential) {
			return environmentdto.View{}, apperror.New(apperror.KindValidation, "Environment deployment SSH credential must be configured before it can be active")
		}
	}
	if err := validateEnvironment(item, s.localDisplay.Platform, false); err != nil {
		return environmentdto.View{}, err
	}
	if environmentTargetChanged(previous, item) {
		if item.SSH != nil {
			item.SSH.HostKeyFingerprint = ""
		}
		item.TargetRevision++
	}
	if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
		return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
	}
	item, err = s.environmentForProject(ctx, projectID)
	if err != nil {
		return environmentdto.View{}, err
	}
	return s.toView(item), nil
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

func validateEnvironment(item model.Environment, localPlatform string, allowEmptyLocalWorkspace bool) error {
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
		if item.WorkspaceRoot == "" && allowEmptyLocalWorkspace {
			return nil
		}
		if !validWorkspaceRoot(localPlatform, item.WorkspaceRoot) {
			return apperror.New(apperror.KindValidation, "Environment local workspace_root is invalid for the control-plane platform")
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
		if !validWorkspaceRoot(item.SSH.Platform, item.WorkspaceRoot) {
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
			if input.Local == nil || input.SSH != nil {
				return apperror.New(apperror.KindValidation, "Local target is required when target_type is local")
			}
			item.TargetType = targetType
			item.WorkspaceRoot = strings.TrimSpace(input.Local.WorkspaceRoot)
			item.SSH = nil
		case model.EnvironmentTargetTypeSSH:
			if input.SSH == nil || input.Local != nil {
				return apperror.New(apperror.KindValidation, "SSH target is required when target_type is ssh")
			}
			ssh := sshTargetFromInput(input.SSH, nil)
			if item.IsSSH() {
				ssh.CredentialId = item.SSH.CredentialId
				ssh.CredentialRevision = item.SSH.CredentialRevision
			}
			item.TargetType = targetType
			item.WorkspaceRoot = strings.TrimSpace(input.SSH.WorkspaceRoot)
			item.SSH = ssh
		default:
			return apperror.New(apperror.KindValidation, "Environment target_type must be local or ssh")
		}
	} else if item.IsLocal() {
		if input.SSH != nil {
			return apperror.New(apperror.KindValidation, "SSH target is only valid for an ssh environment")
		}
		if input.Local != nil {
			item.WorkspaceRoot = strings.TrimSpace(input.Local.WorkspaceRoot)
		}
	} else if item.IsSSH() {
		if input.Local != nil {
			return apperror.New(apperror.KindValidation, "Local target is only valid for a local environment")
		}
		if input.SSH != nil {
			credentialID, credentialRevision := item.SSH.CredentialId, item.SSH.CredentialRevision
			item.SSH = sshTargetFromInput(input.SSH, nil)
			item.SSH.CredentialId, item.SSH.CredentialRevision = credentialID, credentialRevision
			item.WorkspaceRoot = strings.TrimSpace(input.SSH.WorkspaceRoot)
		}
	}
	return nil
}

func sshTargetFromInput(input *environmentdto.SSHTargetInput, credential *model.Credential) *model.EnvironmentSSHTarget {
	if input == nil {
		return nil
	}
	target := &model.EnvironmentSSHTarget{
		Platform: strings.TrimSpace(input.Platform), Host: strings.TrimSpace(input.Host), Port: input.Port,
		Username: strings.TrimSpace(input.Username),
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
	if before.WorkspaceRoot != after.WorkspaceRoot {
		return true
	}
	if before.SSH == nil || after.SSH == nil {
		return before.SSH != after.SSH
	}
	return before.SSH.Platform != after.SSH.Platform ||
		before.SSH.Host != after.SSH.Host ||
		before.SSH.Port != after.SSH.Port ||
		before.SSH.Username != after.SSH.Username ||
		before.SSH.CredentialId != after.SSH.CredentialId ||
		before.SSH.CredentialRevision != after.SSH.CredentialRevision
}

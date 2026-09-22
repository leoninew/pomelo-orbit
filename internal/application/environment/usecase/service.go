package environmentsvc

import (
	"context"
	"errors"
	"log/slog"
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

type environmentTargetReader interface {
	EnvironmentByTarget(ctx context.Context, projectId, targetType, host string, port int) (model.Environment, error)
}

type Service struct {
	environments           repository.EnvironmentStore
	projects               repository.ProjectReader
	environmentCredentials repository.EnvironmentCredentialStore
	secretKey              string
	prober                 environmentport.Prober
	localDisplay           environmentdto.LocalDisplaySnapshot
	logger                 *slog.Logger
}

func New(environments repository.EnvironmentStore, projects repository.ProjectReader, environmentCredentials repository.EnvironmentCredentialStore, secretKey string, prober environmentport.Prober, logger *slog.Logger) Service {
	return Service{
		environments: environments, projects: projects, environmentCredentials: environmentCredentials,
		secretKey: secretKey, prober: prober, logger: logger,
	}
}

func (s Service) WithLocalDisplay(snapshot environmentdto.LocalDisplaySnapshot) Service {
	s.localDisplay = snapshot
	return s
}

// SaveInitialization creates or updates the Project Environment during Wizard
// initialization. It is not used after Gateway creation.
func (s Service) SaveInitialization(ctx context.Context, userId string, projectId string, input environmentdto.UpdateInput) (environmentdto.View, error) {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return environmentdto.View{}, err
	}
	project, err := s.projects.Project(ctx, projectId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return environmentdto.View{}, apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
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
			TargetRevision: 1,
			CreatedAt:      time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
	}
	previous := item
	if err := applyUpdate(&item, input); err != nil {
		return environmentdto.View{}, err
	}
	if err := validateEnvironment(item, s.localDisplay.Platform, false, false); err != nil {
		return environmentdto.View{}, err
	}
	if err := s.ensureTargetIsAvailable(ctx, project.Id, item); err != nil {
		return environmentdto.View{}, err
	}
	if !creating && environmentTargetChanged(previous, item) {
		if item.SSH != nil && environmentIdentityChanged(previous, item) {
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

func (s Service) EnvironmentForUser(ctx context.Context, userId string, projectId string) (environmentdto.View, error) {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return environmentdto.View{}, err
	}
	item, err := s.environmentForProject(ctx, projectId)
	if err != nil {
		return environmentdto.View{}, err
	}
	return s.toView(item), nil
}

func (s Service) UpdateForUser(ctx context.Context, userId string, projectId string, input environmentdto.UpdateInput) (environmentdto.View, error) {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return environmentdto.View{}, err
	}
	item, err := s.environmentForProject(ctx, projectId)
	if err != nil {
		return environmentdto.View{}, err
	}
	previous := item
	if err := applyUpdate(&item, input); err != nil {
		return environmentdto.View{}, err
	}
	if err := validateEnvironment(item, s.localDisplay.Platform, false, false); err != nil {
		return environmentdto.View{}, err
	}
	if environmentTargetChanged(previous, item) {
		if item.SSH != nil && environmentIdentityChanged(previous, item) {
			item.SSH.HostKeyFingerprint = ""
		}
		item.TargetRevision++
	}
	if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
		return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
	}
	item, err = s.environmentForProject(ctx, projectId)
	if err != nil {
		return environmentdto.View{}, err
	}
	return s.toView(item), nil
}

func (s Service) environmentForProject(ctx context.Context, projectId string) (model.Environment, error) {
	item, err := s.environments.EnvironmentByProject(ctx, projectId)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Environment{}, apperror.New(apperror.KindNotFound, "Project environment not found")
	}
	if err != nil {
		return model.Environment{}, apperror.Wrap(apperror.KindInternal, "Failed to load project environment", err)
	}
	return item, nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if projectId == "" {
		return apperror.New(apperror.KindValidation, "project_id is required")
	}
	if _, err := s.projects.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.projects.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func validateEnvironment(item model.Environment, localPlatform string, allowEmptyLocalWorkspace, allowLocalWorkspacePlatformMismatch bool) error {
	if item.ProjectId == "" || item.Code == "" {
		return apperror.New(apperror.KindValidation, "Project environment identity is invalid")
	}
	switch item.TargetType {
	case model.EnvironmentTargetTypeLocal:
		if item.SSH != nil {
			return apperror.New(apperror.KindValidation, "Local environment must not include an SSH target")
		}
		if item.WorkspaceRoot == "" && allowEmptyLocalWorkspace {
			return nil
		}
		if !allowLocalWorkspacePlatformMismatch && !validWorkspaceRoot(localPlatform, item.WorkspaceRoot) {
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
		if !hasSSHCredentialBinding(item) && (item.SSH.CredentialId != "" || item.SSH.CredentialRevision != 0) {
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

func hasSSHCredentialBinding(item model.Environment) bool {
	return item.IsSSH() && item.SSH.CredentialId != "" && item.SSH.CredentialRevision > 0
}

func applyUpdate(item *model.Environment, input environmentdto.UpdateInput) error {
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
			ssh := sshTargetFromInput(input.SSH)
			if item.IsSSH() {
				ssh.CredentialId = item.SSH.CredentialId
				ssh.CredentialRevision = item.SSH.CredentialRevision
				ssh.HostKeyFingerprint = item.SSH.HostKeyFingerprint
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
			credentialId, credentialRevision := item.SSH.CredentialId, item.SSH.CredentialRevision
			hostKeyFingerprint := item.SSH.HostKeyFingerprint
			item.SSH = sshTargetFromInput(input.SSH)
			item.SSH.CredentialId, item.SSH.CredentialRevision = credentialId, credentialRevision
			item.SSH.HostKeyFingerprint = hostKeyFingerprint
			item.WorkspaceRoot = strings.TrimSpace(input.SSH.WorkspaceRoot)
		}
	}
	return nil
}

func sshTargetFromInput(input *environmentdto.SSHTargetInput) *model.EnvironmentSSHTarget {
	if input == nil {
		return nil
	}
	target := &model.EnvironmentSSHTarget{
		Platform: strings.TrimSpace(input.Platform), Host: strings.TrimSpace(input.Host), Port: input.Port,
		Username: strings.TrimSpace(input.Username),
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

// environmentIdentityChanged distinguishes SSH connection identity from the
// workspace path. A workspace edit invalidates the probe revision, but the
// pinned host key remains valid and the existing deployment credential can be
// reused.
func environmentIdentityChanged(before, after model.Environment) bool {
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
		before.SSH.CredentialId != after.SSH.CredentialId ||
		before.SSH.CredentialRevision != after.SSH.CredentialRevision
}

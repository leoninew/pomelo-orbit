package environmentsvc

import (
	"context"
	"errors"
	"strings"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// PrepareWindowsEnvironment persists the current Windows SSH target and the
// deployment key that its generated PowerShell command installs. The HTTP
// mutation unit of work commits both writes together. It deliberately does no
// remote I/O: the user must run that command before testing SSH.
func (s Service) PrepareWindowsEnvironment(ctx context.Context, userId string, projectId string, input environmentdto.SSHTargetInput) (environmentdto.View, string, error) {
	environment, publicKey, err := s.prepareWindowsEnvironment(ctx, userId, projectId, input)
	if err != nil {
		return environmentdto.View{}, "", err
	}
	return s.toView(environment), publicKey, nil
}

func (s Service) prepareWindowsEnvironment(ctx context.Context, userId string, projectId string, input environmentdto.SSHTargetInput) (model.Environment, string, error) {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return model.Environment{}, "", err
	}
	project, err := s.projects.Project(ctx, strings.TrimSpace(projectId))
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Environment{}, "", apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return model.Environment{}, "", apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}

	item, err := s.environments.EnvironmentByProject(ctx, project.Id)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return model.Environment{}, "", apperror.Wrap(apperror.KindInternal, "Failed to load project environment", err)
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
	targetType := model.EnvironmentTargetTypeSSH
	if err := applyUpdate(&item, environmentdto.UpdateInput{TargetType: &targetType, SSH: &input}); err != nil {
		return model.Environment{}, "", err
	}
	if item.SSH.Platform != model.EnvironmentPlatformWindows {
		return model.Environment{}, "", apperror.New(apperror.KindValidation, "Windows initialization command is only available for Windows SSH targets")
	}
	credential, err := s.replaceWindowsCredential(ctx, item)
	if err != nil {
		return model.Environment{}, "", err
	}
	item.SSH.CredentialId = credential.Id
	item.SSH.CredentialRevision = credential.Revision
	item.SSH.HostKeyFingerprint = ""
	if err := validateEnvironment(item, s.localDisplay.Platform, false); err != nil {
		return model.Environment{}, "", err
	}
	if err := s.ensureTargetIsAvailable(ctx, project.Id, item); err != nil {
		return model.Environment{}, "", err
	}
	if !creating && environmentTargetChanged(previous, item) {
		item.TargetRevision++
	}
	if creating {
		if err := s.environments.CreateEnvironment(ctx, item); err != nil {
			return model.Environment{}, "", apperror.Wrap(apperror.KindInternal, "Failed to create project environment", err)
		}
	} else if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
		return model.Environment{}, "", apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
	}
	return item, strings.TrimSpace(credential.PublicKey), nil
}

func (s Service) replaceWindowsCredential(ctx context.Context, environment model.Environment) (model.EnvironmentCredential, error) {
	if environment.IsSSH() && strings.TrimSpace(environment.SSH.CredentialId) != "" {
		credential, err := s.environmentCredential(ctx, environment.SSH.CredentialId)
		if err == nil && strings.TrimSpace(credential.ProjectId) == environment.ProjectId {
			return s.replaceEnvironmentCredential(ctx, credential)
		}
		if err != nil && !apperror.IsKind(err, apperror.KindNotFound) {
			return model.EnvironmentCredential{}, err
		}
	}
	if s.environmentCredentials == nil {
		return model.EnvironmentCredential{}, apperror.New(apperror.KindInternal, "environment credential store is not configured")
	}
	credential, err := s.environmentCredentials.LatestEnvironmentCredentialByProject(ctx, environment.ProjectId)
	if err == nil {
		return s.replaceEnvironmentCredential(ctx, credential)
	}
	if !errors.Is(err, repository.ErrNotFound) {
		return model.EnvironmentCredential{}, apperror.Wrap(apperror.KindInternal, "Failed to load environment credential", err)
	}
	return s.createEnvironmentCredential(ctx, environment.ProjectId)
}

func (s Service) ensureTargetIsAvailable(ctx context.Context, projectId string, item model.Environment) error {
	reader, ok := s.environments.(environmentTargetReader)
	if !ok {
		return nil
	}
	host, port := "", 0
	if item.SSH != nil {
		host, port = strings.TrimSpace(item.SSH.Host), item.SSH.Port
	}
	if _, err := reader.EnvironmentByTarget(ctx, projectId, item.TargetType, host, port); err == nil {
		return apperror.New(apperror.KindConflict, "The Docker target is already bound to another project")
	} else if !errors.Is(err, repository.ErrNotFound) {
		return apperror.Wrap(apperror.KindInternal, "Failed to check environment target", err)
	}
	return nil
}

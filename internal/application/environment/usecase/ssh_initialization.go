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

// PrepareSSHEnvironment persists the submitted SSH target and creates an
// Environment-specific deployment key only when its current binding is absent
// or invalid. This contains no remote I/O and runs in the HTTP write UoW.
func (s Service) PrepareSSHEnvironment(ctx context.Context, userId string, projectId string, input environmentdto.UpdateInput) (environmentdto.View, string, error) {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return environmentdto.View{}, "", err
	}
	if input.TargetType == nil || strings.TrimSpace(*input.TargetType) != model.EnvironmentTargetTypeSSH || input.SSH == nil || input.Local != nil {
		return environmentdto.View{}, "", apperror.New(apperror.KindValidation, "SSH initialization command requires a complete SSH target")
	}
	project, err := s.projects.Project(ctx, projectId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return environmentdto.View{}, "", apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return environmentdto.View{}, "", apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	item, err := s.environments.EnvironmentByProject(ctx, project.Id)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return environmentdto.View{}, "", apperror.Wrap(apperror.KindInternal, "Failed to load project environment", err)
	}
	creating := errors.Is(err, repository.ErrNotFound)
	if creating {
		item = model.Environment{
			Id: idutil.NewId(), ProjectId: project.Id, Code: project.Code, TargetRevision: 1,
			CreatedAt: time.Now().UTC(), UpdatedAt: time.Now().UTC(),
		}
	}
	previous := item
	if err := applyUpdate(&item, input); err != nil {
		return environmentdto.View{}, "", err
	}
	if err := validateEnvironment(item, s.localDisplay.Platform, false, false); err != nil {
		return environmentdto.View{}, "", err
	}
	credential, err := s.ensureEnvironmentCredential(ctx, item)
	if err != nil {
		return environmentdto.View{}, "", err
	}
	item.SSH.CredentialId = credential.Id
	item.SSH.CredentialRevision = credential.Revision
	if environmentTargetChanged(previous, item) {
		if environmentIdentityChanged(previous, item) {
			item.SSH.HostKeyFingerprint = ""
		}
		if !creating {
			item.TargetRevision++
		}
	}
	if creating {
		if err := s.environments.CreateEnvironment(ctx, item); err != nil {
			return environmentdto.View{}, "", apperror.Wrap(apperror.KindInternal, "Failed to create project environment", err)
		}
	} else if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
		return environmentdto.View{}, "", apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
	}
	view, err := s.viewWithTargetWarning(ctx, item)
	if err != nil {
		return environmentdto.View{}, "", err
	}
	return view, strings.TrimSpace(credential.PublicKey), nil
}

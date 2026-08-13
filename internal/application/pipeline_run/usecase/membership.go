package pipelinerunsvc

import (
	"context"
	"errors"
	"strings"

	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func requiredProjectID(projectId *string, resource string) (string, error) {
	if projectId == nil || strings.TrimSpace(*projectId) == "" {
		return "", apperror.New(apperror.KindValidation, resource+" has no project")
	}
	return strings.TrimSpace(*projectId), nil
}

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.store.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.store.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

package deploymentsvc

import (
	"context"
	"fmt"

	logstream "github.com/leoninew/pomelo-orbit/internal/application/logstream/usecase"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

func (s Service) OpenDeploymentLogStream(ctx context.Context, userId, projectId, deploymentId, cursor string) (logstream.Subscription, error) {
	deployment, err := s.loadDeploymentForUser(ctx, userId, projectId, deploymentId)
	if err != nil {
		return logstream.Subscription{}, err
	}
	if deployment.ApplicationId == nil {
		return logstream.Subscription{}, apperror.New(apperror.KindNotFound, "Deployment application not found")
	}
	service, err := s.resolveServiceFromDeployment(ctx, projectId, *deployment.ApplicationId, deployment)
	if err != nil {
		return logstream.Subscription{}, apperror.New(apperror.KindNotFound, "Deployment service not found")
	}
	source := logstream.SourceId(fmt.Sprintf("cd:%s:%s:%s", projectId, deployment.Id, service.Code))
	reader, err := s.logStore.OpenReader(ctx, service.Code, deployment.Id)
	if err != nil {
		return logstream.Subscription{}, apperror.Wrap(apperror.KindUnavailable, "Failed to open deployment logs", err)
	}
	return logstream.File(source, cursor, deployment.Status, reader, func(ctx context.Context) (string, error) {
		current, err := s.loadDeploymentForUser(ctx, userId, projectId, deploymentId)
		if err != nil {
			return "", err
		}
		if current.ServiceId == nil || *current.ServiceId != service.Id {
			return "", apperror.New(apperror.KindConflict, "Deployment log source changed")
		}
		return current.Status, nil
	})
}

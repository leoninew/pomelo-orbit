package deploymentsvc

import (
	"context"
	"time"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
)

const (
	defaultDeploymentWaitTimeout = 300 * time.Second
	maxDeploymentWaitTimeout     = 30 * time.Minute
	deploymentWaitPollInterval   = time.Second
)

// WaitDeployment waits for the persisted deployment state to become terminal.
// It never drives the worker or replays a lifecycle command.
func (s Service) WaitDeployment(ctx context.Context, userId, deploymentId string, timeout *time.Duration) (deploymentdto.DeploymentWaitResult, error) {
	limit := defaultDeploymentWaitTimeout
	if timeout != nil {
		limit = *timeout
	}
	if limit <= 0 || limit > maxDeploymentWaitTimeout {
		return deploymentdto.DeploymentWaitResult{}, apperror.New(apperror.KindValidation, "timeout_seconds must be between 1 and 1800")
	}

	deadline := time.NewTimer(limit)
	defer deadline.Stop()
	ticker := time.NewTicker(deploymentWaitPollInterval)
	defer ticker.Stop()

	for {
		deployment, err := s.DeploymentForUser(ctx, userId, deploymentId)
		if err != nil {
			return deploymentdto.DeploymentWaitResult{}, err
		}
		if status.WorkStatusIsComplete(deployment.Status) {
			return deploymentdto.DeploymentWaitResult{Deployment: deployment}, nil
		}
		select {
		case <-ctx.Done():
			return deploymentdto.DeploymentWaitResult{}, ctx.Err()
		case <-deadline.C:
			return deploymentdto.DeploymentWaitResult{Deployment: deployment, TimedOut: true}, nil
		case <-ticker.C:
		}
	}
}

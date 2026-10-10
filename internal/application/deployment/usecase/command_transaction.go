package deploymentsvc

import (
	"context"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func (s Service) DeployService(ctx context.Context, userId, projectId, serviceId string, input deploymentdto.DeployServiceInput) (deploymentdto.DeployServiceResult, error) {
	var result deploymentdto.DeployServiceResult
	err := s.runCommandTransaction(ctx, func(ctx context.Context) error {
		var err error
		result, err = s.deployService(ctx, userId, projectId, serviceId, input)
		return err
	})
	if err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	return result, nil
}

func (s Service) StopApplication(ctx context.Context, userId, projectId, applicationId string, input deploymentdto.ServiceTargetInput) (string, error) {
	var id string
	err := s.runCommandTransaction(ctx, func(ctx context.Context) error {
		var err error
		id, err = s.stopApplication(ctx, userId, projectId, applicationId, input)
		return err
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s Service) RestartApplication(ctx context.Context, userId, projectId, applicationId string, input deploymentdto.ServiceTargetInput) (string, error) {
	var id string
	err := s.runCommandTransaction(ctx, func(ctx context.Context) error {
		var err error
		id, err = s.restartApplication(ctx, userId, projectId, applicationId, input)
		return err
	})
	if err != nil {
		return "", err
	}
	return id, nil
}

func (s Service) runCommandTransaction(ctx context.Context, fn func(context.Context) error) error {
	if s.transactionRunner == nil {
		return apperror.New(apperror.KindInternal, "deployment transaction runner is not configured")
	}
	return s.transactionRunner.RunInTransaction(ctx, fn)
}

func (s Service) createCurrentDeployment(ctx context.Context, projectId string, deployment model.Deployment) error {
	if err := s.commandStore.CreateDeployment(ctx, projectId, deployment); err != nil {
		return err
	}
	return s.commandStore.SetServiceCurrentDeployment(ctx, projectId, *deployment.ServiceId, deployment.Id)
}

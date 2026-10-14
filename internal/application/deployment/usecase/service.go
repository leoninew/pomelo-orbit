package deploymentsvc

import (
	"context"
	"errors"
	"os"

	"log/slog"
	"strings"
	"time"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type Service struct {
	versionSelector deploymentport.ServiceVersionSelector
	project         repository.ProjectReader
	application     repository.ApplicationStore
	service         repository.ServiceStore
	deployment      repository.DeploymentStore
	logStore        deploymentport.DeploymentLogs
	targetResolver  environmentport.TargetResolver
	runtime         deploymentport.WorkspaceReader
	inspector       deploymentport.RuntimeInspector
	commands        deploymentport.LifecycleCommands
	containerLogs   deploymentport.ContainerLogs
	dispatcher      deploymentport.Dispatcher

	logger            *slog.Logger
	transactionRunner deploymentport.TransactionRunner

	commandStore       deploymentport.CommandStore
	gatewayCoordinator deploymentport.GatewayDeploymentCoordinator
}

func (s Service) ListDeployments(ctx context.Context, userId string, input deploymentdto.DeploymentListInput) (repository.Page[model.Deployment], error) {
	if input.ProjectId == "" {
		return repository.Page[model.Deployment]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := ensureProjectMembership(ctx, s.project, input.ProjectId, userId); err != nil {
		return repository.Page[model.Deployment]{}, err
	}
	dateFrom, err := parseOptionalRunTime(input.DateFrom, "date_from")
	if err != nil {
		return repository.Page[model.Deployment]{}, err
	}
	dateTo, err := parseOptionalRunTime(input.DateTo, "date_to")
	if err != nil {
		return repository.Page[model.Deployment]{}, err
	}
	items, err := s.deployment.ListDeployments(ctx, input.ProjectId, input.ApplicationId, input.Status, input.Search, dateFrom, dateTo, input.Page, input.PerPage)
	if err != nil {
		return repository.Page[model.Deployment]{}, apperror.Wrap(apperror.KindInternal, "Failed to list deployments", err)
	}
	return items, nil
}

// ClearProjectDeploymentHistory removes persisted deployment records without
// touching runtime workspaces, containers, or execution logs.
func (s Service) ClearProjectDeploymentHistory(ctx context.Context, userId, projectId string) error {
	if err := ensureProjectMembership(ctx, s.project, projectId, userId); err != nil {
		return err
	}
	if err := s.deployment.ClearProjectDeploymentHistory(ctx, projectId); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to clear Project deployment history", err)
	}
	return nil
}

func (s Service) DeploymentForUser(ctx context.Context, userId string, projectId string, deploymentId string) (model.Deployment, error) {
	return s.loadDeploymentForUser(ctx, userId, projectId, deploymentId)
}

func (s Service) DeploymentLog(ctx context.Context, userId string, projectId string, deploymentId string, offset int) (deploymentdto.DeploymentLog, error) {
	if offset < 0 {
		return deploymentdto.DeploymentLog{}, apperror.New(apperror.KindValidation, "offset must be greater than or equal to 0")
	}
	deployment, err := s.loadDeploymentForUser(ctx, userId, projectId, deploymentId)
	if err != nil {
		return deploymentdto.DeploymentLog{}, err
	}
	logs, newOffset, err := s.readDeploymentLog(ctx, projectId, deployment, offset)
	if err != nil {
		return deploymentdto.DeploymentLog{}, err
	}
	return deploymentdto.DeploymentLog{Logs: logs, Offset: newOffset, IsComplete: status.WorkStatusIsComplete(deployment.Status), Status: deployment.Status}, nil
}

func (s Service) CancelDeployment(ctx context.Context, userId string, projectId string, deploymentId string) (model.Deployment, error) {
	deployment, err := s.loadDeploymentForUser(ctx, userId, projectId, deploymentId)
	if err != nil {
		return model.Deployment{}, err
	}
	if deployment.Status != status.WorkStatusWaitingToRun && deployment.Status != status.WorkStatusRunning {
		return model.Deployment{}, apperror.New(apperror.KindValidation, "Cannot cancel deployment with status "+deployment.Status)
	}
	canceled, err := s.deployment.CancelDeployment(ctx, projectId, deployment.Id)
	if err != nil {
		return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to cancel deployment", err)
	}
	if !canceled {
		return model.Deployment{}, apperror.New(apperror.KindValidation, "Cannot cancel deployment with status "+deployment.Status)
	}
	updated, err := s.deployment.Deployment(ctx, projectId, deployment.Id)
	if err != nil {
		return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment", err)
	}
	return updated, nil
}

func (s Service) DeleteDeployment(ctx context.Context, userId string, projectId string, deploymentId string) error {
	deployment, err := s.loadDeploymentForUser(ctx, userId, projectId, deploymentId)
	if err != nil {
		return err
	}
	if !status.WorkStatusIsComplete(deployment.Status) {
		return apperror.New(apperror.KindValidation, "Cannot delete deployment with status "+deployment.Status)
	}
	if err := s.removeDeploymentLog(ctx, projectId, deployment); err != nil {
		return err
	}
	if err := s.deployment.DeleteDeployment(ctx, projectId, deployment.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete deployment", err)
	}
	return nil
}

func (s Service) loadDeploymentForUser(ctx context.Context, userId string, projectId string, deploymentId string) (model.Deployment, error) {
	if projectId == "" {
		return model.Deployment{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := ensureProjectMembership(ctx, s.project, projectId, userId); err != nil {
		return model.Deployment{}, err
	}
	deployment, err := s.deployment.Deployment(ctx, projectId, deploymentId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Deployment{}, apperror.New(apperror.KindNotFound, "Deployment "+deploymentId+" not found")
		}
		return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment", err)
	}
	return deployment, nil
}

func (s Service) removeDeploymentLog(ctx context.Context, projectId string, deployment model.Deployment) error {
	if deployment.ServiceId == nil || *deployment.ServiceId == "" {
		s.warnDeploymentLogCleanupSkipped(deployment.Id, "deployment has no associated service")
		return nil
	}
	service, err := s.service.Service(ctx, projectId, *deployment.ServiceId)
	if errors.Is(err, repository.ErrNotFound) {
		s.warnDeploymentLogCleanupSkipped(deployment.Id, "associated service no longer exists")
		return nil
	}
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load deployment service", err)
	}
	if strings.TrimSpace(service.Code) == "" {
		s.warnDeploymentLogCleanupSkipped(deployment.Id, "associated service has no code")
		return nil
	}
	if s.logStore == nil {
		return apperror.New(apperror.KindInternal, "deployment log store is not configured")
	}
	if err := s.logStore.Remove(service.Code, deployment.Id); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			s.warnDeploymentLogCleanupSkipped(deployment.Id, "deployment log file does not exist")
			return nil
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to delete deployment log", err)
	}
	return nil
}
func (s Service) warnDeploymentLogCleanupSkipped(deploymentId string, reason string) {
	if s.logger != nil {
		s.logger.Warn("skipped deployment log cleanup", "deployment_id", deploymentId, "reason", reason)
	}
}

func ensureProjectMembership(ctx context.Context, project repository.ProjectReader, projectId string, userId string) error {
	if _, err := project.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := project.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func (s Service) readDeploymentLog(ctx context.Context, projectId string, deployment model.Deployment, offset int) (string, int, error) {
	if deployment.ApplicationId == nil || *deployment.ApplicationId == "" {
		return "", offset, apperror.New(apperror.KindValidation, "Deployment "+deployment.Id+" has no associated application")
	}
	app, err := s.application.Application(ctx, projectId, *deployment.ApplicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", offset, apperror.New(apperror.KindNotFound, "Application "+*deployment.ApplicationId+" not found")
		}
		return "", offset, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	svc, err := resolveServiceFromDeployment(ctx, s.commandStore, projectId, app.Id, deployment)
	if err != nil {
		return "", offset, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	content, newOffset, err := s.logStore.Read(svc.Code, deployment.Id, offset)
	if err != nil {
		return "", offset, apperror.Wrap(apperror.KindInternal, "Failed to read deployment log", err)
	}
	return string(content), newOffset, nil
}

func parseOptionalRunTime(value string, name string) (*time.Time, error) {
	if value == "" {
		return nil, nil
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return nil, apperror.New(apperror.KindValidation, name+" must be ISO 8601")
	}
	return &parsed, nil
}

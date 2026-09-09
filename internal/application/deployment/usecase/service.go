package deploymentsvc

import (
	"context"
	"errors"
	"fmt"
	"os"

	"log/slog"
	"strconv"
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
	project        repository.ProjectReader
	application    repository.ApplicationStore
	service        repository.ServiceStore
	deployment     repository.DeploymentStore
	logStore       deploymentport.ExecutionLogStore
	targetResolver environmentport.TargetResolver
	runtime        deploymentport.Runtime
	store          *stores
	executionStore deploymentport.ExecutionStore
	dispatcher     deploymentport.Dispatcher

	logger       *slog.Logger
	pollInterval time.Duration

	commandStore          deploymentport.CommandStore
	gatewayCoordinator    deploymentport.GatewayDeploymentCoordinator
	gatewayRoutePublisher deploymentport.GatewayRoutePublisher
}

func New(
	project repository.ProjectReader,
	application repository.ApplicationStore,
	service repository.ServiceStore,
	deployment repository.DeploymentStore,
	targetResolver environmentport.TargetResolver,
	runtime deploymentport.Runtime,
	logStore deploymentport.ExecutionLogStore,
	gatewayCoordinator deploymentport.GatewayDeploymentCoordinator,
) Service {
	store := &stores{
		project: project, application: application,
		service: service, deployment: deployment,
	}
	return Service{
		project: project, application: application,
		service: service, deployment: deployment,
		logStore: logStore, targetResolver: targetResolver, runtime: runtime, store: store, executionStore: store,
		gatewayCoordinator: gatewayCoordinator,
	}
}

func (s Service) ListDeployments(ctx context.Context, userId string, input deploymentdto.DeploymentListInput) (repository.Page[model.Deployment], error) {
	if input.ProjectId == "" {
		return repository.Page[model.Deployment]{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureProjectMembership(ctx, input.ProjectId, userId); err != nil {
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

func (s Service) DeploymentForUser(ctx context.Context, userId string, deploymentId string) (model.Deployment, error) {
	return s.loadDeploymentForUser(ctx, userId, deploymentId)
}

func (s Service) DeploymentLog(ctx context.Context, userId string, deploymentId string, offset int) (deploymentdto.DeploymentLog, error) {
	if offset < 0 {
		return deploymentdto.DeploymentLog{}, apperror.New(apperror.KindValidation, "offset must be greater than or equal to 0")
	}
	deployment, err := s.loadDeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return deploymentdto.DeploymentLog{}, err
	}
	logs, newOffset, err := s.readDeploymentLog(ctx, deployment, offset)
	if err != nil {
		return deploymentdto.DeploymentLog{}, err
	}
	return deploymentdto.DeploymentLog{Logs: logs, Offset: newOffset, IsComplete: status.WorkStatusIsComplete(deployment.Status), Status: deployment.Status}, nil
}

func (s Service) CancelDeployment(ctx context.Context, userId string, deploymentId string) (model.Deployment, error) {
	deployment, err := s.loadDeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return model.Deployment{}, err
	}
	if deployment.Status != status.WorkStatusWaitingToRun && deployment.Status != status.WorkStatusRunning {
		return model.Deployment{}, apperror.New(apperror.KindValidation, "Cannot cancel deployment with status "+deployment.Status)
	}
	canceled, err := s.deployment.CancelDeployment(ctx, deployment.Id)
	if err != nil {
		return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to cancel deployment", err)
	}
	if !canceled {
		return model.Deployment{}, apperror.New(apperror.KindValidation, "Cannot cancel deployment with status "+deployment.Status)
	}
	updated, err := s.deployment.Deployment(ctx, deployment.Id)
	if err != nil {
		return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment", err)
	}
	return updated, nil
}

func (s Service) DeleteDeployment(ctx context.Context, userId string, deploymentId string) error {
	deployment, err := s.loadDeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return err
	}
	if !status.WorkStatusIsComplete(deployment.Status) {
		return apperror.New(apperror.KindValidation, "Cannot delete deployment with status "+deployment.Status)
	}
	if err := s.removeDeploymentLog(ctx, deployment); err != nil {
		return err
	}
	if err := s.deployment.DeleteDeployment(ctx, deployment.Id); err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to delete deployment", err)
	}
	return nil
}

func (s Service) DeploymentContainerLog(ctx context.Context, userId string, deploymentId string, tail int) (deploymentdto.DeploymentContainerLog, error) {
	if tail < 1 || tail > 1000 {
		return deploymentdto.DeploymentContainerLog{}, apperror.New(apperror.KindValidation, "tail must be between 1 and 1000")
	}
	deployment, err := s.loadDeploymentForUser(ctx, userId, deploymentId)
	if err != nil {
		return deploymentdto.DeploymentContainerLog{}, err
	}
	if deployment.OperationType == "stop" {
		return deploymentdto.DeploymentContainerLog{}, apperror.New(apperror.KindValidation, "container logs are not available for stop deployments")
	}
	if deployment.Status == status.WorkStatusWaitingToRun {
		return deploymentdto.DeploymentContainerLog{Source: "pending", IsRealtimeSupported: true}, nil
	}
	if deployment.ApplicationId == nil || *deployment.ApplicationId == "" {
		return deploymentdto.DeploymentContainerLog{}, apperror.New(apperror.KindValidation, "Deployment "+deployment.Id+" has no associated application")
	}
	app, err := s.application.Application(ctx, *deployment.ApplicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return deploymentdto.DeploymentContainerLog{}, apperror.New(apperror.KindNotFound, "Application "+*deployment.ApplicationId+" not found")
		}
		return deploymentdto.DeploymentContainerLog{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
	if err != nil {
		return deploymentdto.DeploymentContainerLog{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	target, err := s.resolveProjectTarget(ctx, app)
	if err != nil {
		return deploymentdto.DeploymentContainerLog{}, err
	}
	projectName := composeProjectName(app.Code, svc.InstanceKey)
	sinceCommand := containerLogsSinceCommand(projectName, deployment.StartedAt.UTC().Format(time.RFC3339))
	output, err := s.runtime.Query(ctx, target, svc.Code, sinceCommand.Name, sinceCommand.Args...)
	if err == nil {
		return deploymentdto.DeploymentContainerLog{Logs: output, Source: "since", IsRealtimeSupported: true}, nil
	}
	tailCommand := containerLogsTailCommand(projectName, strconv.Itoa(tail))
	output, tailErr := s.runtime.Query(ctx, target, svc.Code, tailCommand.Name, tailCommand.Args...)
	if tailErr != nil {
		return deploymentdto.DeploymentContainerLog{}, apperror.New(apperror.KindInternal, outputOrError(output, tailErr))
	}
	return deploymentdto.DeploymentContainerLog{Logs: output, Source: "tail", IsRealtimeSupported: true}, nil
}

func (s Service) loadDeploymentForUser(ctx context.Context, userId string, deploymentId string) (model.Deployment, error) {
	deployment, err := s.deployment.Deployment(ctx, deploymentId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Deployment{}, apperror.New(apperror.KindNotFound, "Deployment "+deploymentId+" not found")
		}
		return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load deployment", err)
	}
	if deployment.ProjectId != nil {
		if err := s.ensureProjectMembership(ctx, *deployment.ProjectId, userId); err != nil {
			return model.Deployment{}, err
		}
		return deployment, nil
	}
	if deployment.ApplicationId != nil {
		app, err := s.application.Application(ctx, *deployment.ApplicationId)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return model.Deployment{}, apperror.New(apperror.KindNotFound, "Application "+*deployment.ApplicationId+" not found")
			}
			return model.Deployment{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
		}
		if app.ProjectId != nil {
			if err := s.ensureProjectMembership(ctx, *app.ProjectId, userId); err != nil {
				return model.Deployment{}, err
			}
		}
	}
	return deployment, nil
}

func (s Service) removeDeploymentLog(ctx context.Context, deployment model.Deployment) error {
	if deployment.ServiceId == nil || strings.TrimSpace(*deployment.ServiceId) == "" {
		s.warnDeploymentLogCleanupSkipped(deployment.Id, "deployment has no associated service")
		return nil
	}
	service, err := s.service.Service(ctx, *deployment.ServiceId)
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

func (s Service) ensureProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.project.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.project.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func (s Service) readDeploymentLog(ctx context.Context, deployment model.Deployment, offset int) (string, int, error) {
	if deployment.ApplicationId == nil || *deployment.ApplicationId == "" {
		return "", offset, apperror.New(apperror.KindValidation, "Deployment "+deployment.Id+" has no associated application")
	}
	app, err := s.application.Application(ctx, *deployment.ApplicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", offset, apperror.New(apperror.KindNotFound, "Application "+*deployment.ApplicationId+" not found")
		}
		return "", offset, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	opts, err := parseDeployOptions(deployment.OptionsJSON)
	if err != nil {
		return "", offset, apperror.New(apperror.KindValidation, "Deployment "+deployment.Id+" has invalid options: "+err.Error())
	}
	if opts.InstanceKey == "" {
		return "", offset, apperror.New(apperror.KindValidation, "Deployment "+deployment.Id+" has no associated instance")
	}
	svc, err := s.resolveServiceFromDeployment(ctx, app.Id, deployment)
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

func outputOrError(output string, err error) string {
	if output != "" {
		return output
	}
	return fmt.Sprint(err)
}

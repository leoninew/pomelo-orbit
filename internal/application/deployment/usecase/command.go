package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"
	"time"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	idutil "github.com/leoninew/pomelo-orbit/internal/common/util"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

// NewCommandService is the future HTTP-facing constructor. It keeps command
// creation and its asynchronous dispatch contract inside the deployment domain.
func NewCommandService(
	project repository.ProjectReader,
	application repository.ApplicationStore,
	service repository.ServiceStore,
	deployment repository.DeploymentStore,
	gateway repository.GatewayStore,
	dispatcher deploymentport.Dispatcher,
	logger *slog.Logger,
	targetResolver environmentport.TargetResolver,
	runtime deploymentport.Runtime,
	logStore deploymentport.ExecutionLogStore,
	gatewayCoordinator deploymentport.GatewayDeploymentCoordinator,
) Service {
	store := &stores{
		project: project, application: application,
		service: service, deployment: deployment, gateway: gateway,
	}
	return Service{
		project: project, application: application,
		service: service, deployment: deployment, store: store, executionStore: store,
		dispatcher: dispatcher, commandStore: store, logger: logger,
		targetResolver: targetResolver, runtime: runtime, logStore: logStore,
		gatewayCoordinator: gatewayCoordinator,
	}
}

// NewExecutionService constructs the worker-facing deployment service.
func NewExecutionService(
	project repository.ProjectReader,
	application repository.ApplicationStore,
	service repository.ServiceStore,
	deployment repository.DeploymentStore,
	gatewayCoordinator deploymentport.GatewayDeploymentCoordinator,
	logger *slog.Logger,
	targetResolver environmentport.TargetResolver,
	runtime deploymentport.Runtime,
	logStore deploymentport.ExecutionLogStore,
	pollInterval time.Duration,
	gatewayRoutePublisher deploymentport.GatewayRoutePublisher,
) Service {
	store := &stores{
		project: project, application: application,
		service: service, deployment: deployment,
	}
	return Service{
		project: project, application: application,
		service: service, deployment: deployment, store: store, executionStore: store,
		logger: logger, targetResolver: targetResolver, runtime: runtime, pollInterval: pollInterval,
		logStore:              logStore,
		gatewayCoordinator:    gatewayCoordinator,
		gatewayRoutePublisher: gatewayRoutePublisher,
	}
}

func (s Service) DeployService(ctx context.Context, userId string, projectId string, serviceId string, input deploymentdto.DeployServiceInput) (deploymentdto.DeployServiceResult, error) {
	service, app, err := s.serviceForUser(ctx, userId, projectId, serviceId)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	if err := s.ensureNoActiveDeployment(ctx, projectId, service.Id); err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	service, err = s.selectGatewayVersionForDeployment(ctx, projectId, app, service)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	version, err := s.commandStore.Version(ctx, projectId, service.VersionId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return deploymentdto.DeployServiceResult{}, apperror.New(apperror.KindNotFound, "Version "+service.VersionId+" not found")
		}
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	components, err := s.commandStore.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	for _, component := range components {
		if err := validateVersionComponent(component); err != nil {
			return deploymentdto.DeployServiceResult{}, apperror.New(apperror.KindValidation, err.Error())
		}
	}
	overlays, err := s.commandStore.ServiceComponentsByService(ctx, projectId, service.Id)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load service components", err)
	}
	env, err := s.commandStore.ServiceEnvByService(ctx, projectId, service.Id)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load service environment", err)
	}
	plan, _, err := BuildEffectiveServicePlan(app, version, service, components, overlays, env, nil)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.New(apperror.KindValidation, err.Error())
	}
	setPlanJoinTraefikNetwork(&plan, input.JoinTraefikNetwork)
	gateway, err := s.gatewayForDeployment(ctx, projectId, app, plan)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	plan.Gateway = gateway
	setPlanJoinTraefikNetwork(&plan, input.JoinTraefikNetwork)
	if err := enrichGatewayPlan(&plan); err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.New(apperror.KindValidation, err.Error())
	}
	target, err := s.resolveProjectTarget(ctx, projectId)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	planHash, err := EffectiveServicePlanHash(plan)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to hash deployment plan", err)
	}
	deployment := newDeployment(projectId, app, "deploy")
	deployment.VersionId = &version.Id
	opts := deploymentdto.DeployOptionsJSON{
		ForceRecreate:      input.ForceRecreate,
		JoinTraefikNetwork: deploymentJoinTraefikNetwork(plan), GatewayConfig: cloneGatewayConfig(gateway),
	}
	applyDeploymentTargetSnapshot(&deployment, target, gateway)
	if err := setDeploymentOptions(&deployment, opts); err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	if s.dispatcher == nil {
		return deploymentdto.DeployServiceResult{}, apperror.New(apperror.KindInternal, "deployment dispatcher is not configured")
	}
	deployment.ServiceId = &service.Id
	deployment.EffectivePlanHash = &planHash
	deployment.CommandText = deployComposeCommand(composeProjectName(service.Code), deploymentPullPolicy(plan), input.ForceRecreate).String()
	if err := s.commandStore.CreateDeployment(ctx, projectId, deployment); err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchDeploy(ctx, deploymentdto.DeployDispatchInput{ProjectId: projectId, ApplicationId: app.Id, DeploymentId: deployment.Id, ForceRecreate: input.ForceRecreate}); err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deploymentdto.DeployServiceResult{DeploymentId: deployment.Id}, nil
}

func (s Service) StopApplication(ctx context.Context, userId string, projectId string, applicationId string, input deploymentdto.ServiceTargetInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, projectId, applicationId)
	if err != nil {
		return "", err
	}
	service, err := s.resolveServiceTarget(ctx, projectId, app.Id, input)
	if err != nil {
		return "", err
	}
	if err := s.ensureNoActiveDeployment(ctx, projectId, service.Id); err != nil {
		return "", err
	}
	canRemoveStoppedVolumes := service.Status == status.ServiceStatusStopped && input.RemoveVolumes
	if service.Status != status.ServiceStatusRunning && service.Status != status.ServiceStatusFaulted && !canRemoveStoppedVolumes {
		return "", apperror.New(apperror.KindValidation, "应用未在运行中, 无法停止")
	}
	target, err := s.resolveProjectTarget(ctx, projectId)
	if err != nil {
		return "", err
	}
	deployment := newDeployment(projectId, app, "stop")
	deployment.ServiceId = &service.Id
	deployment.VersionId = &service.VersionId
	options := deploymentdto.DeployOptionsJSON{RemoveVolumes: input.RemoveVolumes}
	applyDeploymentTargetSnapshot(&deployment, target, nil)
	if err := setDeploymentOptions(&deployment, options); err != nil {
		return "", err
	}
	deployment.CommandText = stopComposeCommand(composeProjectName(service.Code), input.RemoveVolumes).String()
	if s.dispatcher == nil {
		return "", apperror.New(apperror.KindInternal, "deployment dispatcher is not configured")
	}
	if err := s.commandStore.CreateDeployment(ctx, projectId, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchStop(ctx, deploymentdto.StopDispatchInput{ProjectId: projectId, ApplicationId: app.Id, DeploymentId: deployment.Id, RemoveVolumes: input.RemoveVolumes}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) ensureNoActiveDeployment(ctx context.Context, projectId string, serviceId string) error {
	active, err := s.commandStore.HasActiveDeployment(ctx, projectId, serviceId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check active deployment", err)
	}
	if active {
		return apperror.New(apperror.KindConflict, "Service already has an active deployment")
	}
	return nil
}

func (s Service) RestartApplication(ctx context.Context, userId string, projectId string, applicationId string, input deploymentdto.ServiceTargetInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, projectId, applicationId)
	if err != nil {
		return "", err
	}
	service, err := s.resolveServiceTarget(ctx, projectId, app.Id, input)
	if err != nil {
		return "", err
	}
	if err := s.ensureNoActiveDeployment(ctx, projectId, service.Id); err != nil {
		return "", err
	}
	if service.Status != status.ServiceStatusRunning && service.Status != status.ServiceStatusFaulted {
		return "", apperror.New(apperror.KindValidation, "应用未在运行中, 无法重启")
	}
	version, err := s.commandStore.Version(ctx, projectId, service.VersionId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	components, err := s.commandStore.VersionComponentsByVersion(ctx, projectId, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	overlays, err := s.commandStore.ServiceComponentsByService(ctx, projectId, service.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load service components", err)
	}
	env, err := s.commandStore.ServiceEnvByService(ctx, projectId, service.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load service environment", err)
	}
	plan, _, err := BuildEffectiveServicePlan(app, version, service, components, overlays, env, nil)
	if err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	setPlanJoinTraefikNetwork(&plan, nil)
	gateway, err := s.gatewayForDeployment(ctx, projectId, app, plan)
	if err != nil {
		return "", err
	}
	plan.Gateway = gateway
	setPlanJoinTraefikNetwork(&plan, nil)
	if err := enrichGatewayPlan(&plan); err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	target, err := s.resolveProjectTarget(ctx, projectId)
	if err != nil {
		return "", err
	}
	planHash, err := EffectiveServicePlanHash(plan)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to hash deployment plan", err)
	}
	deployment := newDeployment(projectId, app, "restart")
	deployment.ServiceId = &service.Id
	deployment.VersionId = &version.Id
	restartOptions := deploymentdto.DeployOptionsJSON{
		JoinTraefikNetwork: deploymentJoinTraefikNetwork(plan), GatewayConfig: cloneGatewayConfig(gateway),
	}
	applyDeploymentTargetSnapshot(&deployment, target, gateway)
	if err := setDeploymentOptions(&deployment, restartOptions); err != nil {
		return "", err
	}
	deployment.EffectivePlanHash = &planHash
	deployment.CommandText = deployComposeCommand(composeProjectName(service.Code), deploymentPullPolicy(plan), false).String()
	if s.dispatcher == nil {
		return "", apperror.New(apperror.KindInternal, "deployment dispatcher is not configured")
	}
	if err := s.commandStore.CreateDeployment(ctx, projectId, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchRestart(ctx, deploymentdto.RestartDispatchInput{ProjectId: projectId, ApplicationId: app.Id, DeploymentId: deployment.Id}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) loadApplicationForUser(ctx context.Context, userId string, projectId string, applicationId string) (model.Application, error) {
	if s.commandStore == nil {
		return model.Application{}, apperror.New(apperror.KindInternal, "deployment command store is not configured")
	}
	projectId = strings.TrimSpace(projectId)
	if projectId == "" {
		return model.Application{}, apperror.New(apperror.KindValidation, "project_id is required")
	}
	if err := s.ensureCommandProjectMembership(ctx, projectId, userId); err != nil {
		return model.Application{}, err
	}
	app, err := s.commandStore.Application(ctx, projectId, applicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Application{}, apperror.New(apperror.KindNotFound, "Application "+applicationId+" not found")
		}
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	return app, nil
}

func (s Service) resolveServiceTarget(ctx context.Context, projectId, applicationId string, input deploymentdto.ServiceTargetInput) (model.Service, error) {
	serviceId := strings.TrimSpace(input.ServiceId)
	if serviceId == "" {
		return model.Service{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
	service, err := s.commandStore.Service(ctx, projectId, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	if service.ApplicationId != applicationId {
		return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
	}
	return service, nil
}

func (s Service) serviceForUser(ctx context.Context, userId string, projectId string, serviceId string) (model.Service, model.Application, error) {
	serviceId = strings.TrimSpace(serviceId)
	if serviceId == "" {
		return model.Service{}, model.Application{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
	service, err := s.commandStore.Service(ctx, projectId, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Service{}, model.Application{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return model.Service{}, model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	app, err := s.loadApplicationForUser(ctx, userId, projectId, service.ApplicationId)
	if err != nil {
		return model.Service{}, model.Application{}, err
	}
	return service, app, nil
}

func deploymentPullPolicy(plan model.EffectiveServicePlan) string {
	allNever := len(plan.Components) > 0
	for _, component := range plan.Components {
		if component.PullPolicy == "always" {
			return "always"
		}
		if component.PullPolicy != "never" {
			allNever = false
		}
	}
	if allNever {
		return "never"
	}
	return "missing"
}

func (s Service) ensureCommandProjectMembership(ctx context.Context, projectId string, userId string) error {
	if _, err := s.commandStore.Project(ctx, projectId); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectId+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.commandStore.IsProjectMember(ctx, projectId, userId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func newDeployment(projectId string, app model.Application, operationType string) model.Deployment {
	return model.Deployment{Id: idutil.NewId(), ProjectId: &projectId, ApplicationId: &app.Id, ApplicationName: app.Name, OperationType: operationType, TriggerType: "manual", Status: status.WorkStatusWaitingToRun}
}

func setDeploymentOptions(deployment *model.Deployment, options deploymentdto.DeployOptionsJSON) error {
	raw, err := json.Marshal(options)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to encode deployment options", err)
	}
	text := string(raw)
	deployment.OptionsJSON = &text
	return nil
}

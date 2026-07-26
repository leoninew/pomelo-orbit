package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	deploymentport "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/port"
	gatewayport "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

// NewCommandService is the future HTTP-facing constructor. It keeps command
// creation and its asynchronous dispatch contract inside the deployment domain.
func NewCommandService(
	project repository.ProjectReader,
	application repository.ApplicationStore,
	environment repository.EnvironmentStore,
	service repository.ServiceStore,
	deployment repository.DeploymentStore,
	gateway repository.GatewayStore,
	dispatcher deploymentport.Dispatcher,
	logger *slog.Logger,
	workspace deploymentport.Workspace,
	queryRunner deploymentport.CommandQueryRunner,
	logStore deploymentport.LogReader,
	gatewayCoordinator gatewayport.DeploymentCoordinator,
) Service {
	store := &stores{
		project: project, application: application, environment: environment,
		service: service, deployment: deployment, gateway: gateway,
	}
	return Service{
		project: project, application: application, environment: environment,
		service: service, deployment: deployment, store: store, executionStore: store,
		dispatcher: dispatcher, commandStore: store, logger: logger, workspace: workspace,
		queryRunner: queryRunner, logStore: logStore,
		gatewayCoordinator: gatewayCoordinator,
	}
}

// NewExecutionService constructs the worker-facing deployment service.
func NewExecutionService(
	project repository.ProjectReader,
	application repository.ApplicationStore,
	environment repository.EnvironmentStore,
	service repository.ServiceStore,
	deployment repository.DeploymentStore,
	gatewayCoordinator gatewayport.DeploymentCoordinator,
	logger *slog.Logger,
	workspace deploymentport.Workspace,
	runner deploymentport.CommandRunner,
	logStore deploymentport.ExecutionLogStore,
) Service {
	store := &stores{
		project: project, application: application, environment: environment,
		service: service, deployment: deployment,
	}
	return Service{
		project: project, application: application, environment: environment,
		service: service, deployment: deployment, store: store, executionStore: store,
		logger: logger, workspace: workspace, runner: runner,
		logStore: logStore, executionLogStore: logStore,
		gatewayCoordinator: gatewayCoordinator,
	}
}

func (s Service) DeployApplication(ctx context.Context, userId string, applicationId string, input deploymentdto.DeployInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	if input.VersionId == "" {
		return "", apperror.New(apperror.KindValidation, "version_id is required")
	}
	if input.EnvironmentId == "" {
		return "", apperror.New(apperror.KindValidation, "environment_id is required")
	}
	version, err := s.commandStore.Version(ctx, input.VersionId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", apperror.New(apperror.KindNotFound, "Version "+input.VersionId+" not found")
		}
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	if version.ApplicationId != app.Id {
		return "", apperror.New(apperror.KindValidation, "Version does not belong to this application")
	}
	env, err := s.commandStore.Environment(ctx, input.EnvironmentId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return "", apperror.New(apperror.KindNotFound, "Environment not found")
		}
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	if app.ProjectId == nil || *app.ProjectId != env.ProjectId {
		return "", apperror.New(apperror.KindValidation, "application and environment must belong to the same project")
	}
	components, err := s.commandStore.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	if err := validateVersionComponents(components); err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	if app.Kind == status.ApplicationKindGateway {
		active, err := s.commandStore.HasActiveGatewayService(ctx, app.Id)
		if err != nil {
			return "", apperror.Wrap(apperror.KindInternal, "Failed to check active gateway", err)
		}
		if active {
			return "", apperror.New(apperror.KindValidation, "another gateway is already deploying or running; multiple gateways are not supported")
		}
	}
	runtimeConfig, err := resolveDeployRuntimeConfig(version, components, input.RuntimeConfig)
	if err != nil {
		return "", err
	}
	if input.InstanceKey == "" {
		return "", apperror.New(apperror.KindValidation, "instance_key is required")
	}
	deployment := newDeployment(app, "deploy")
	deployment.VersionId = &version.Id
	deployment.EnvironmentId = &env.Id
	opts := deploymentdto.DeployOptionsJSON{ForceRecreate: input.ForceRecreate, InstanceKey: input.InstanceKey, RuntimeConfig: runtimeConfig}
	if err := setDeploymentOptions(&deployment, opts); err != nil {
		return "", err
	}
	if s.dispatcher == nil {
		return "", apperror.New(apperror.KindInternal, "deployment dispatcher is not configured")
	}
	service, err := s.commandStore.ServiceByKey(ctx, app.Id, env.Id, input.InstanceKey)
	if err != nil && !errors.Is(err, repository.ErrNotFound) {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	if errors.Is(err, repository.ErrNotFound) {
		service = model.Service{
			Id:            idutil.NewId(),
			ApplicationId: app.Id,
			EnvironmentId: env.Id,
			InstanceKey:   input.InstanceKey,
		}
	}
	service.VersionId = version.Id
	service.Status = status.ServiceStatusDeploying
	if err := s.commandStore.UpsertService(ctx, service); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to prepare service", err)
	}
	deployment.ServiceId = &service.Id
	deployment.CommandText = deployComposeCommand(composeProjectName(app.Code, env.Code, input.InstanceKey), app.ImagePullPolicy, input.ForceRecreate).String()
	if err := s.commandStore.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchDeploy(ctx, deploymentdto.DeployDispatchInput{ApplicationID: app.Id, DeploymentID: deployment.Id, ForceRecreate: input.ForceRecreate}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) StopApplication(ctx context.Context, userId string, applicationId string, input deploymentdto.ServiceTargetInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	service, err := s.resolveServiceTarget(ctx, app.Id, input)
	if err != nil {
		return "", err
	}
	if service.Status != status.ServiceStatusRunning && service.Status != status.ServiceStatusFaulted {
		return "", apperror.New(apperror.KindValidation, "应用未在运行中, 无法停止")
	}
	env, err := s.commandStore.Environment(ctx, service.EnvironmentId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	deployment := newDeployment(app, "stop")
	deployment.ServiceId = &service.Id
	deployment.VersionId = &service.VersionId
	deployment.EnvironmentId = &service.EnvironmentId
	if err := setDeploymentOptions(&deployment, deploymentdto.DeployOptionsJSON{InstanceKey: service.InstanceKey, RemoveVolumes: input.RemoveVolumes}); err != nil {
		return "", err
	}
	deployment.CommandText = stopComposeCommand(composeProjectName(app.Code, env.Code, service.InstanceKey), input.RemoveVolumes).String()
	if s.dispatcher == nil {
		return "", apperror.New(apperror.KindInternal, "deployment dispatcher is not configured")
	}
	if err := s.commandStore.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchStop(ctx, deploymentdto.StopDispatchInput{ApplicationID: app.Id, DeploymentID: deployment.Id, RemoveVolumes: input.RemoveVolumes}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) RestartApplication(ctx context.Context, userId string, applicationId string, input deploymentdto.ServiceTargetInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	service, err := s.resolveServiceTarget(ctx, app.Id, input)
	if err != nil {
		return "", err
	}
	if service.Status != status.ServiceStatusRunning && service.Status != status.ServiceStatusFaulted {
		return "", apperror.New(apperror.KindValidation, "应用未在运行中, 无法重启")
	}
	version, err := s.commandStore.Version(ctx, service.VersionId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	components, err := s.commandStore.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	if err := validateVersionComponents(components); err != nil {
		return "", apperror.New(apperror.KindValidation, err.Error())
	}
	env, err := s.commandStore.Environment(ctx, service.EnvironmentId)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to load environment", err)
	}
	deployment := newDeployment(app, "restart")
	deployment.ServiceId = &service.Id
	deployment.VersionId = &version.Id
	deployment.EnvironmentId = &service.EnvironmentId
	if err := setDeploymentOptions(&deployment, deploymentdto.DeployOptionsJSON{InstanceKey: service.InstanceKey}); err != nil {
		return "", err
	}
	deployment.CommandText = deployComposeCommand(composeProjectName(app.Code, env.Code, service.InstanceKey), app.ImagePullPolicy, false).String()
	if s.dispatcher == nil {
		return "", apperror.New(apperror.KindInternal, "deployment dispatcher is not configured")
	}
	if err := s.commandStore.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchRestart(ctx, deploymentdto.RestartDispatchInput{ApplicationID: app.Id, DeploymentID: deployment.Id}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) loadApplicationForUser(ctx context.Context, userID string, applicationID string) (model.Application, error) {
	if s.commandStore == nil {
		return model.Application{}, apperror.New(apperror.KindInternal, "deployment command store is not configured")
	}
	app, err := s.commandStore.Application(ctx, applicationID)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Application{}, apperror.New(apperror.KindNotFound, "Application "+applicationID+" not found")
		}
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	if app.ProjectId != nil {
		if err := s.ensureCommandProjectMembership(ctx, *app.ProjectId, userID); err != nil {
			return model.Application{}, err
		}
	}
	return app, nil
}

func (s Service) resolveServiceTarget(ctx context.Context, applicationID string, input deploymentdto.ServiceTargetInput) (model.Service, error) {
	if input.ServiceId != "" {
		service, err := s.commandStore.Service(ctx, input.ServiceId)
		if err != nil {
			if errors.Is(err, repository.ErrNotFound) {
				return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
			}
			return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
		}
		if service.ApplicationId != applicationID {
			return model.Service{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return service, nil
	}
	environmentID := input.EnvironmentId
	if environmentID == "" {
		services, err := s.commandStore.ListServicesByApplication(ctx, applicationID)
		if err != nil {
			return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to list services", err)
		}
		if len(services) == 0 {
			return model.Service{}, apperror.New(apperror.KindValidation, "应用未在运行中")
		}
		if len(services) == 1 {
			return services[0], nil
		}
		return model.Service{}, apperror.New(apperror.KindValidation, "environment_id is required when multiple services exist")
	}
	if input.InstanceKey == "" {
		return model.Service{}, apperror.New(apperror.KindValidation, "instance_key is required")
	}
	service, err := s.commandStore.ServiceByKey(ctx, applicationID, environmentID, input.InstanceKey)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Service{}, apperror.New(apperror.KindValidation, "应用未在运行中")
		}
		return model.Service{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	return service, nil
}

func (s Service) ensureCommandProjectMembership(ctx context.Context, projectID string, userID string) error {
	if _, err := s.commandStore.Project(ctx, projectID); err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return apperror.New(apperror.KindNotFound, "Project "+projectID+" not found")
		}
		return apperror.Wrap(apperror.KindInternal, "Failed to load project", err)
	}
	member, err := s.commandStore.IsProjectMember(ctx, projectID, userID)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to check project member", err)
	}
	if !member {
		return apperror.New(apperror.KindForbidden, "Permission denied")
	}
	return nil
}

func newDeployment(app model.Application, operationType string) model.Deployment {
	return model.Deployment{Id: idutil.NewId(), ProjectId: app.ProjectId, ApplicationId: &app.Id, ApplicationName: app.Name, OperationType: operationType, TriggerType: "manual", Status: status.WorkStatusWaitingToRun}
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

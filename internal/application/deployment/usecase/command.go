package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"sort"
	"strconv"
	"strings"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	deploymentport "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/port"
	gatewayport "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	runtimeconfig "gitee.com/leoninew/PomeloOrbit-go/internal/common/runtimeconfig"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
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
	workspace deploymentport.Workspace,
	queryRunner deploymentport.CommandQueryRunner,
	logStore deploymentport.LogReader,
	gatewayCoordinator gatewayport.DeploymentCoordinator,
) Service {
	store := &stores{
		project: project, application: application,
		service: service, deployment: deployment, gateway: gateway,
	}
	return Service{
		project: project, application: application,
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
	service repository.ServiceStore,
	deployment repository.DeploymentStore,
	gatewayCoordinator gatewayport.DeploymentCoordinator,
	logger *slog.Logger,
	workspace deploymentport.Workspace,
	runner deploymentport.CommandRunner,
	logStore deploymentport.ExecutionLogStore,
) Service {
	store := &stores{
		project: project, application: application,
		service: service, deployment: deployment,
	}
	return Service{
		project: project, application: application,
		service: service, deployment: deployment, store: store, executionStore: store,
		logger: logger, workspace: workspace, runner: runner,
		logStore: logStore, executionLogStore: logStore,
		gatewayCoordinator: gatewayCoordinator,
	}
}

func (s Service) DeployService(ctx context.Context, userId string, serviceId string, input deploymentdto.DeployServiceInput) (deploymentdto.DeployServiceResult, error) {
	service, app, err := s.serviceForUser(ctx, userId, serviceId)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	if err := s.ensureBusinessGatewayRunning(ctx, app); err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	version, err := s.commandStore.Version(ctx, service.VersionId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return deploymentdto.DeployServiceResult{}, apperror.New(apperror.KindNotFound, "Version "+service.VersionId+" not found")
		}
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load version", err)
	}
	components, err := s.commandStore.VersionComponentsByVersion(ctx, version.Id)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to list components", err)
	}
	if err := validateVersionComponents(components); err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.New(apperror.KindValidation, err.Error())
	}
	if app.Kind == status.ApplicationKindGateway {
		active, err := s.commandStore.HasActiveGatewayService(ctx, app.Id)
		if err != nil {
			return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to check active gateway", err)
		}
		if active && service.Status != status.ServiceStatusRunning && service.Status != status.ServiceStatusDeploying {
			return deploymentdto.DeployServiceResult{}, apperror.New(apperror.KindValidation, "another gateway is already deploying or running; multiple gateways are not supported")
		}
	}
	exposes, err := s.commandStore.ServiceExposesByService(ctx, service.Id)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load service exposes", err)
	}
	missing, _ := runtimeconfig.Validate(service.RuntimeConfig, components)
	if len(missing) > 0 {
		return deploymentdto.DeployServiceResult{}, apperror.New(apperror.KindValidation, "missing runtime config keys: "+strings.Join(missing, ", "))
	}
	warnings, err := s.publicTCPGatewayWarnings(ctx, app, exposes)
	if err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	deployment := newDeployment(app, "deploy")
	deployment.VersionId = &version.Id
	opts := deploymentdto.DeployOptionsJSON{ForceRecreate: input.ForceRecreate, InstanceKey: service.InstanceKey, RuntimeConfig: cloneRuntimeConfig(service.RuntimeConfig)}
	if err := setDeploymentOptions(&deployment, opts); err != nil {
		return deploymentdto.DeployServiceResult{}, err
	}
	if s.dispatcher == nil {
		return deploymentdto.DeployServiceResult{}, apperror.New(apperror.KindInternal, "deployment dispatcher is not configured")
	}
	if err := s.commandStore.UpdateServiceStatus(ctx, service.Id, status.ServiceStatusDeploying); err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to prepare service", err)
	}
	deployment.ServiceId = &service.Id
	deployment.CommandText = deployComposeCommand(composeProjectName(app.Code, service.InstanceKey), app.ImagePullPolicy, input.ForceRecreate).String()
	if err := s.commandStore.CreateDeployment(ctx, deployment); err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchDeploy(ctx, deploymentdto.DeployDispatchInput{ApplicationId: app.Id, DeploymentId: deployment.Id, ForceRecreate: input.ForceRecreate}); err != nil {
		return deploymentdto.DeployServiceResult{}, apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deploymentdto.DeployServiceResult{DeploymentId: deployment.Id, Warnings: warnings}, nil
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
	deployment := newDeployment(app, "stop")
	deployment.ServiceId = &service.Id
	deployment.VersionId = &service.VersionId
	if err := setDeploymentOptions(&deployment, deploymentdto.DeployOptionsJSON{InstanceKey: service.InstanceKey, RemoveVolumes: input.RemoveVolumes}); err != nil {
		return "", err
	}
	deployment.CommandText = stopComposeCommand(composeProjectName(app.Code, service.InstanceKey), input.RemoveVolumes).String()
	if s.dispatcher == nil {
		return "", apperror.New(apperror.KindInternal, "deployment dispatcher is not configured")
	}
	if err := s.commandStore.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchStop(ctx, deploymentdto.StopDispatchInput{ApplicationId: app.Id, DeploymentId: deployment.Id, RemoveVolumes: input.RemoveVolumes}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) ensureBusinessGatewayRunning(ctx context.Context, app model.Application) error {
	if app.Kind == status.ApplicationKindGateway {
		return nil
	}
	if s.gatewayCoordinator == nil {
		return apperror.New(apperror.KindInternal, "gateway deployment coordinator is not configured")
	}
	return s.gatewayCoordinator.EnsureGatewayRunning(ctx, app)
}

func (s Service) RestartApplication(ctx context.Context, userId string, applicationId string, input deploymentdto.ServiceTargetInput) (string, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return "", err
	}
	if err := s.ensureBusinessGatewayRunning(ctx, app); err != nil {
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
	missing, _ := runtimeconfig.Validate(service.RuntimeConfig, components)
	if len(missing) > 0 {
		return "", apperror.New(apperror.KindValidation, "missing runtime config keys: "+strings.Join(missing, ", "))
	}
	deployment := newDeployment(app, "restart")
	deployment.ServiceId = &service.Id
	deployment.VersionId = &version.Id
	if err := setDeploymentOptions(&deployment, deploymentdto.DeployOptionsJSON{InstanceKey: service.InstanceKey, RuntimeConfig: cloneRuntimeConfig(service.RuntimeConfig)}); err != nil {
		return "", err
	}
	deployment.CommandText = deployComposeCommand(composeProjectName(app.Code, service.InstanceKey), app.ImagePullPolicy, false).String()
	if s.dispatcher == nil {
		return "", apperror.New(apperror.KindInternal, "deployment dispatcher is not configured")
	}
	if err := s.commandStore.CreateDeployment(ctx, deployment); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to create deployment", err)
	}
	if err := s.dispatcher.DispatchRestart(ctx, deploymentdto.RestartDispatchInput{ApplicationId: app.Id, DeploymentId: deployment.Id}); err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to enqueue deployment", err)
	}
	return deployment.Id, nil
}

func (s Service) loadApplicationForUser(ctx context.Context, userId string, applicationId string) (model.Application, error) {
	if s.commandStore == nil {
		return model.Application{}, apperror.New(apperror.KindInternal, "deployment command store is not configured")
	}
	app, err := s.commandStore.Application(ctx, applicationId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Application{}, apperror.New(apperror.KindNotFound, "Application "+applicationId+" not found")
		}
		return model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load application", err)
	}
	if app.ProjectId != nil {
		if err := s.ensureCommandProjectMembership(ctx, *app.ProjectId, userId); err != nil {
			return model.Application{}, err
		}
	}
	return app, nil
}

func (s Service) resolveServiceTarget(ctx context.Context, applicationId string, input deploymentdto.ServiceTargetInput) (model.Service, error) {
	serviceId := strings.TrimSpace(input.ServiceId)
	if serviceId == "" {
		return model.Service{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
	service, err := s.commandStore.Service(ctx, serviceId)
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

func (s Service) serviceForUser(ctx context.Context, userId string, serviceId string) (model.Service, model.Application, error) {
	serviceId = strings.TrimSpace(serviceId)
	if serviceId == "" {
		return model.Service{}, model.Application{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
	service, err := s.commandStore.Service(ctx, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return model.Service{}, model.Application{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return model.Service{}, model.Application{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	app, err := s.loadApplicationForUser(ctx, userId, service.ApplicationId)
	if err != nil {
		return model.Service{}, model.Application{}, err
	}
	return service, app, nil
}

func (s Service) publicTCPGatewayWarnings(ctx context.Context, app model.Application, exposes []model.ServiceExpose) ([]string, error) {
	if app.Kind == status.ApplicationKindGateway {
		return nil, nil
	}
	required := map[int]struct{}{}
	for _, expose := range exposes {
		if expose.Access == "public" && expose.Protocol == "tcp" {
			required[effectiveListen(expose)] = struct{}{}
		}
	}
	if len(required) == 0 {
		return nil, nil
	}
	cfg, err := s.commandStore.ResolveActiveGatewayConfig(ctx)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to resolve active gateway", err)
	}
	services, err := s.commandStore.ListServicesByApplication(ctx, cfg.ApplicationId)
	if err != nil {
		return nil, apperror.Wrap(apperror.KindInternal, "Failed to load gateway services", err)
	}
	available := map[int]struct{}{}
	for _, gatewayService := range services {
		if gatewayService.Status != status.ServiceStatusRunning {
			continue
		}
		components, err := s.commandStore.VersionComponentsByVersion(ctx, gatewayService.VersionId)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindInternal, "Failed to load deployed gateway components", err)
		}
		for _, component := range components {
			for _, port := range component.Ports {
				if port.HostPort == port.ContainerPort {
					available[port.HostPort] = struct{}{}
				}
			}
		}
	}
	warnings := make([]string, 0, len(required))
	for listen := range required {
		if _, ok := available[listen]; !ok {
			warnings = append(warnings, "Gateway is running but does not expose tcp"+strconv.Itoa(listen)+"; configure port "+strconv.Itoa(listen)+" and deploy Gateway.")
		}
	}
	sort.Strings(warnings)
	return warnings, nil
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

func cloneRuntimeConfig(values map[string]string) map[string]string {
	clone := make(map[string]string, len(values))
	for key, value := range values {
		clone[key] = value
	}
	return clone
}

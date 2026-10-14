package deploymentsvc

import (
	"context"
	"errors"
	"regexp"
	"strings"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var runtimeSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// ResolveRuntimeTarget validates that a requested runtime target is an Orbit
// managed Service belonging to the configured actor.
func (s Service) ResolveRuntimeTarget(ctx context.Context, userId, projectId, serviceId string, allowGateway bool) (deploymentdto.RuntimeTarget, error) {
	if serviceId == "" {
		return deploymentdto.RuntimeTarget{}, apperror.New(apperror.KindValidation, "service_id is required")
	}
	service, err := s.service.Service(ctx, projectId, serviceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return deploymentdto.RuntimeTarget{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return deploymentdto.RuntimeTarget{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	app, err := s.loadApplicationForUser(ctx, userId, projectId, service.ApplicationId)
	if err != nil {
		return deploymentdto.RuntimeTarget{}, err
	}
	if app.Kind != status.ApplicationKindStandard && (!allowGateway || app.Kind != status.ApplicationKindGateway) {
		return deploymentdto.RuntimeTarget{}, apperror.New(apperror.KindValidation, "runtime tools only support managed standard Applications")
	}
	if !safeRuntimeSegment(service.Code) {
		return deploymentdto.RuntimeTarget{}, apperror.New(apperror.KindInternal, "invalid managed service code")
	}
	target, err := resolveProjectTarget(ctx, s.targetResolver, projectId)
	if err != nil {
		return deploymentdto.RuntimeTarget{}, err
	}
	location, err := runtimeServiceLocation(target, service)
	if err != nil {
		return deploymentdto.RuntimeTarget{}, err
	}
	workingDirectory, err := s.runtime.ResolveDirectory(ctx, target, location)
	if err != nil {
		return deploymentdto.RuntimeTarget{}, apperror.Wrap(apperror.KindInternal, "Failed to resolve remote runtime directory", err)
	}
	return deploymentdto.RuntimeTarget{ApplicationId: app.Id, ServiceId: service.Id, ServiceCode: service.Code, WorkingDirectory: workingDirectory, ComposeProject: service.Code, ProjectId: projectId}, nil
}

func safeRuntimeSegment(value string) bool {
	return runtimeSegmentPattern.MatchString(value) && value != "." && value != ".."
}

func (s Service) RuntimeComposeConfig(ctx context.Context, target deploymentdto.RuntimeTarget) (deploymentdto.RuntimeTextResult, error) {
	scope, err := s.runtimeScope(ctx, target)
	if err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	output, err := s.inspector.Config(ctx, scope)
	if err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	return deploymentdto.RuntimeTextResult{Target: target, Text: output}, nil
}

func (s Service) RuntimeComposePS(ctx context.Context, target deploymentdto.RuntimeTarget) (deploymentdto.RuntimeComposePSResult, error) {
	scope, err := s.runtimeScope(ctx, target)
	if err != nil {
		return deploymentdto.RuntimeComposePSResult{}, err
	}
	observation, err := s.inspector.Containers(ctx, scope)
	if err != nil {
		return deploymentdto.RuntimeComposePSResult{}, err
	}
	containers := observation.Containers
	service, err := s.service.Service(ctx, target.ProjectId, target.ServiceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return deploymentdto.RuntimeComposePSResult{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return deploymentdto.RuntimeComposePSResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load runtime service", err)
	}
	if err := s.populateContainerComponentIds(ctx, target.ProjectId, service.VersionId, containers); err != nil {
		return deploymentdto.RuntimeComposePSResult{}, err
	}
	return deploymentdto.RuntimeComposePSResult{Target: target, Containers: containers, Raw: observation.Raw}, nil
}

func (s Service) RuntimeComposeLogs(ctx context.Context, target deploymentdto.RuntimeTarget, tail int, since string, services []string) (deploymentdto.RuntimeTextResult, error) {
	if tail < 1 || tail > 1000 {
		return deploymentdto.RuntimeTextResult{}, apperror.New(apperror.KindValidation, "tail must be between 1 and 1000")
	}
	if err := s.ensureRuntimeServiceNames(ctx, target.ProjectId, target.ServiceId, services); err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	scope, err := s.runtimeScope(ctx, target)
	if err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	output, err := s.inspector.Logs(ctx, scope, tail, since, services)
	if err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	return deploymentdto.RuntimeTextResult{Target: target, Text: output}, nil
}

func (s Service) RuntimeContainerInspect(ctx context.Context, target deploymentdto.RuntimeTarget, containerId string) (deploymentdto.RuntimeInspectResult, error) {
	inspection, err := s.inspectManagedContainer(ctx, target, containerId)
	if err != nil {
		return deploymentdto.RuntimeInspectResult{}, err
	}
	return deploymentdto.RuntimeInspectResult{Target: target, Data: inspection.Raw}, nil
}

func (s Service) inspectManagedContainer(ctx context.Context, target deploymentdto.RuntimeTarget, containerId string) (deploymentport.ContainerInspection, error) {
	if !safeRuntimeSegment(containerId) {
		return deploymentport.ContainerInspection{}, apperror.New(apperror.KindValidation, "invalid container_id")
	}
	ps, err := s.RuntimeComposePS(ctx, target)
	if err != nil {
		return deploymentport.ContainerInspection{}, err
	}
	if !runtimeContainerIdExists(ps.Containers, containerId) {
		return deploymentport.ContainerInspection{}, apperror.New(apperror.KindNotFound, "container_id is not managed by this runtime target")
	}
	scope, err := s.runtimeScope(ctx, target)
	if err != nil {
		return deploymentport.ContainerInspection{}, err
	}
	return s.inspector.InspectContainer(ctx, scope, containerId)
}

func runtimeContainerIdExists(containers []deploymentdto.RuntimeContainer, containerId string) bool {
	for _, container := range containers {
		if container.Id == containerId {
			return true
		}
	}
	return false
}

func (s Service) RuntimeNetworkInspect(ctx context.Context, target deploymentdto.RuntimeTarget, networkName string) (deploymentdto.RuntimeInspectResult, error) {
	networkName = strings.TrimSpace(networkName)
	if !safeRuntimeSegment(networkName) {
		return deploymentdto.RuntimeInspectResult{}, apperror.New(apperror.KindValidation, "invalid network_name")
	}
	ps, err := s.RuntimeComposePS(ctx, target)
	if err != nil {
		return deploymentdto.RuntimeInspectResult{}, err
	}
	if len(ps.Containers) == 0 {
		return deploymentdto.RuntimeInspectResult{}, apperror.New(apperror.KindNotFound, "runtime target has no containers")
	}
	managedName := ""
	for _, container := range ps.Containers {
		inspect, err := s.inspectManagedContainer(ctx, target, container.Id)
		if err != nil {
			return deploymentdto.RuntimeInspectResult{}, err
		}
		if managedName = resolveRuntimeNetworkName(inspect.Networks, networkName); managedName != "" {
			break
		}
	}
	if managedName == "" {
		return deploymentdto.RuntimeInspectResult{}, apperror.New(apperror.KindNotFound, "network_name is not managed by this runtime target")
	}
	scope, err := s.runtimeScope(ctx, target)
	if err != nil {
		return deploymentdto.RuntimeInspectResult{}, err
	}
	data, err := s.inspector.InspectNetwork(ctx, scope, managedName)
	if err != nil {
		return deploymentdto.RuntimeInspectResult{}, err
	}
	return deploymentdto.RuntimeInspectResult{Target: target, Data: data}, nil
}

func resolveRuntimeNetworkName(networks []string, networkName string) string {
	for _, name := range networks {
		if name == networkName {
			return name
		}
	}
	for _, name := range networks {
		if strings.HasSuffix(name, "_"+networkName) || strings.HasSuffix(name, "-"+networkName) {
			return name
		}
	}
	return ""
}

func (s Service) RuntimeHTTPProbe(ctx context.Context, target deploymentdto.RuntimeTarget, componentName string, port int, path string) (deploymentdto.RuntimeTextResult, error) {
	if port < 1 || port > 65535 {
		return deploymentdto.RuntimeTextResult{}, apperror.New(apperror.KindValidation, "port must be between 1 and 65535")
	}
	componentName = strings.TrimSpace(componentName)
	if !safeRuntimeSegment(componentName) {
		return deploymentdto.RuntimeTextResult{}, apperror.New(apperror.KindValidation, "invalid component_name")
	}
	if !strings.HasPrefix(path, "/") || strings.ContainsAny(path, "\r\n") {
		return deploymentdto.RuntimeTextResult{}, apperror.New(apperror.KindValidation, "path must begin with /")
	}
	if err := s.ensureRuntimeServiceNames(ctx, target.ProjectId, target.ServiceId, []string{componentName}); err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	scope, err := s.runtimeScope(ctx, target)
	if err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	output, err := s.inspector.HTTPProbe(ctx, scope, componentName, port, path)
	if err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	return deploymentdto.RuntimeTextResult{Target: target, Text: output}, nil
}

func (s Service) RuntimeDoctor(ctx context.Context, target *deploymentdto.RuntimeTarget) (map[string]any, error) {
	if target == nil {
		return nil, apperror.New(apperror.KindValidation, "managed runtime target is required")
	}
	scope, err := s.runtimeScope(ctx, *target)
	if err != nil {
		return nil, err
	}
	output, err := s.inspector.Version(ctx, scope)
	if err != nil {
		return nil, err
	}
	return map[string]any{"docker": strings.TrimSpace(output)}, nil
}

func (s Service) runtimeScope(ctx context.Context, target deploymentdto.RuntimeTarget) (deploymentport.RuntimeScope, error) {
	if s.runtime == nil || s.targetResolver == nil || s.inspector == nil {
		return deploymentport.RuntimeScope{}, apperror.New(apperror.KindInternal, "remote deployment runtime is not configured")
	}
	sshTarget, err := s.targetResolver.ResolveProjectTarget(ctx, target.ProjectId)
	if err != nil {
		return deploymentport.RuntimeScope{}, err
	}
	exists, err := s.runtime.ServiceDirExists(ctx, sshTarget, deploymentport.ServiceLocation{Code: target.ServiceCode, Directory: target.WorkingDirectory})
	if err != nil {
		return deploymentport.RuntimeScope{}, apperror.Wrap(apperror.KindInternal, "Failed to inspect remote service workspace", err)
	}
	if !exists {
		return deploymentport.RuntimeScope{}, apperror.New(apperror.KindNotFound, "managed runtime workspace does not exist")
	}
	return deploymentport.RuntimeScope{Target: sshTarget, Location: deploymentport.ServiceLocation{Code: target.ServiceCode, Directory: target.WorkingDirectory}, Project: target.ComposeProject}, nil
}
func (s Service) ensureRuntimeServiceNames(ctx context.Context, projectId string, serviceId string, names []string) error {
	if len(names) == 0 {
		return nil
	}
	service, err := s.service.Service(ctx, projectId, serviceId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load runtime service", err)
	}
	components, err := s.application.VersionComponentsByVersion(ctx, projectId, service.VersionId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load version components", err)
	}
	allowed := make(map[string]struct{}, len(components))
	for _, component := range components {
		allowed[component.Name] = struct{}{}
	}
	for _, name := range names {
		if _, ok := allowed[name]; !ok {
			return apperror.New(apperror.KindValidation, "service is not declared by the runtime target")
		}
	}
	return nil
}

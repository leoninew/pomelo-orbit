package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

var runtimeSegmentPattern = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9_.-]*$`)

// ResolveRuntimeTarget validates that a requested runtime target is an Orbit
// managed Service belonging to the configured actor.
func (s Service) ResolveRuntimeTarget(ctx context.Context, userId, applicationId, instanceKey string, allowGateway bool) (deploymentdto.RuntimeTarget, error) {
	app, err := s.loadApplicationForUser(ctx, userId, applicationId)
	if err != nil {
		return deploymentdto.RuntimeTarget{}, err
	}
	if app.Kind != status.ApplicationKindStandard && (!allowGateway || app.Kind != status.ApplicationKindGateway) {
		return deploymentdto.RuntimeTarget{}, apperror.New(apperror.KindValidation, "runtime tools only support managed standard Applications")
	}
	instanceKey = strings.TrimSpace(instanceKey)
	if !safeRuntimeSegment(instanceKey) {
		return deploymentdto.RuntimeTarget{}, apperror.New(apperror.KindValidation, "invalid instance_key")
	}
	service, err := s.service.ServiceByKey(ctx, app.Id, instanceKey)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return deploymentdto.RuntimeTarget{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return deploymentdto.RuntimeTarget{}, apperror.Wrap(apperror.KindInternal, "Failed to load service", err)
	}
	if !safeRuntimeSegment(service.Code) {
		return deploymentdto.RuntimeTarget{}, apperror.New(apperror.KindInternal, "invalid managed service code")
	}
	target, err := s.resolveProjectTarget(ctx, app)
	if err != nil {
		return deploymentdto.RuntimeTarget{}, err
	}
	workingDirectory, err := s.runtime.ServiceDir(target, service.Code)
	if err != nil {
		return deploymentdto.RuntimeTarget{}, apperror.Wrap(apperror.KindInternal, "Failed to resolve remote runtime directory", err)
	}
	return deploymentdto.RuntimeTarget{ApplicationId: app.Id, ServiceId: service.Id, InstanceKey: service.InstanceKey, ServiceCode: service.Code, WorkingDirectory: workingDirectory, ComposeProject: composeProjectName(service.Code), ProjectId: *app.ProjectId}, nil
}

func safeRuntimeSegment(value string) bool {
	return runtimeSegmentPattern.MatchString(value) && value != "." && value != ".."
}

func (s Service) RuntimeComposeConfig(ctx context.Context, target deploymentdto.RuntimeTarget) (deploymentdto.RuntimeTextResult, error) {
	output, err := s.runRuntimeCommand(ctx, target, composeConfigCommand())
	if err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	return deploymentdto.RuntimeTextResult{Target: target, Text: output}, nil
}

func (s Service) RuntimeComposePS(ctx context.Context, target deploymentdto.RuntimeTarget) (deploymentdto.RuntimeComposePSResult, error) {
	output, err := s.runRuntimeCommand(ctx, target, containerPsCommand(target.ComposeProject))
	if err != nil {
		return deploymentdto.RuntimeComposePSResult{}, err
	}
	containers, err := parseComposePsOutput(output)
	if err != nil {
		return deploymentdto.RuntimeComposePSResult{}, apperror.Wrap(apperror.KindInternal, "Failed to parse compose status", err)
	}
	service, err := s.service.Service(ctx, target.ServiceId)
	if err != nil {
		if errors.Is(err, repository.ErrNotFound) {
			return deploymentdto.RuntimeComposePSResult{}, apperror.New(apperror.KindNotFound, "Service not found")
		}
		return deploymentdto.RuntimeComposePSResult{}, apperror.Wrap(apperror.KindInternal, "Failed to load runtime service", err)
	}
	if err := s.populateContainerComponentIds(ctx, service.VersionId, containers); err != nil {
		return deploymentdto.RuntimeComposePSResult{}, err
	}
	var raw any
	if strings.TrimSpace(output) != "" && json.Unmarshal([]byte(output), &raw) != nil {
		raw = output
	}
	return deploymentdto.RuntimeComposePSResult{Target: target, Containers: containers, Raw: raw}, nil
}

func (s Service) RuntimeComposeLogs(ctx context.Context, target deploymentdto.RuntimeTarget, tail int, since string, services []string) (deploymentdto.RuntimeTextResult, error) {
	if tail < 1 || tail > 1000 {
		return deploymentdto.RuntimeTextResult{}, apperror.New(apperror.KindValidation, "tail must be between 1 and 1000")
	}
	if err := s.ensureRuntimeServiceNames(ctx, target.ServiceId, services); err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	var command composeCommand
	if since != "" {
		command = containerLogsSinceCommand(target.ComposeProject, since)
		command.Args = append(command.Args, services...)
	} else {
		command = containerLogsTailCommand(target.ComposeProject, strconv.Itoa(tail), services...)
	}
	output, err := s.runRuntimeCommand(ctx, target, command)
	if err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	return deploymentdto.RuntimeTextResult{Target: target, Text: output}, nil
}

func (s Service) RuntimeContainerInspect(ctx context.Context, target deploymentdto.RuntimeTarget, containerId string) (deploymentdto.RuntimeInspectResult, error) {
	containerId = strings.TrimSpace(containerId)
	if !safeRuntimeSegment(containerId) {
		return deploymentdto.RuntimeInspectResult{}, apperror.New(apperror.KindValidation, "invalid container_id")
	}
	ps, err := s.RuntimeComposePS(ctx, target)
	if err != nil {
		return deploymentdto.RuntimeInspectResult{}, err
	}
	if !runtimeContainerIdExists(ps.Containers, containerId) {
		return deploymentdto.RuntimeInspectResult{}, apperror.New(apperror.KindNotFound, "container_id is not managed by this runtime target")
	}
	output, err := s.runRuntimeCommand(ctx, target, composeCommand{Name: "docker", Args: []string{"inspect", containerId}})
	if err != nil {
		return deploymentdto.RuntimeInspectResult{}, err
	}
	var data any
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return deploymentdto.RuntimeInspectResult{}, apperror.New(apperror.KindInternal, "invalid container inspect output")
	}
	return deploymentdto.RuntimeInspectResult{Target: target, Data: data}, nil
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
	managed := false
	for _, container := range ps.Containers {
		inspect, err := s.RuntimeContainerInspect(ctx, target, container.Id)
		if err != nil {
			return deploymentdto.RuntimeInspectResult{}, err
		}
		if inspectContainsNetwork(inspect.Data, networkName) {
			managed = true
			break
		}
	}
	if !managed {
		return deploymentdto.RuntimeInspectResult{}, apperror.New(apperror.KindNotFound, "network_name is not managed by this runtime target")
	}
	output, err := s.runRuntimeCommand(ctx, target, composeCommand{Name: "docker", Args: []string{"network", "inspect", networkName}})
	if err != nil {
		return deploymentdto.RuntimeInspectResult{}, err
	}
	var data any
	if err := json.Unmarshal([]byte(output), &data); err != nil {
		return deploymentdto.RuntimeInspectResult{}, apperror.New(apperror.KindInternal, "invalid network inspect output")
	}
	return deploymentdto.RuntimeInspectResult{Target: target, Data: data}, nil
}

func inspectContainsNetwork(value any, networkName string) bool {
	items, ok := value.([]any)
	if !ok || len(items) == 0 {
		return false
	}
	item, ok := items[0].(map[string]any)
	if !ok {
		return false
	}
	networkSettings, ok := item["NetworkSettings"].(map[string]any)
	if !ok {
		return false
	}
	networks, ok := networkSettings["Networks"].(map[string]any)
	if !ok {
		return false
	}
	for name := range networks {
		if name == networkName || strings.HasSuffix(name, "_"+networkName) || strings.HasSuffix(name, "-"+networkName) {
			return true
		}
	}
	return false
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
	if err := s.ensureRuntimeServiceNames(ctx, target.ServiceId, []string{componentName}); err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	url := fmt.Sprintf("http://127.0.0.1:%d%s", port, path)
	command := composeCommand{Name: "docker", Args: []string{"compose", "-p", target.ComposeProject, "-f", "docker-compose.yml", "exec", "-T", componentName, "curl", "-fsS", "--max-time", "10", url}}
	output, err := s.runRuntimeCommand(ctx, target, command)
	if err != nil {
		return deploymentdto.RuntimeTextResult{}, err
	}
	return deploymentdto.RuntimeTextResult{Target: target, Text: output}, nil
}

func (s Service) RuntimeDoctor(ctx context.Context, target *deploymentdto.RuntimeTarget) (map[string]any, error) {
	if target == nil {
		return nil, apperror.New(apperror.KindValidation, "managed runtime target is required")
	}
	output, err := s.runRuntimeCommand(ctx, *target, composeCommand{Name: "docker", Args: []string{"version", "--format", "json"}})
	if err != nil {
		return nil, err
	}
	return map[string]any{"docker": strings.TrimSpace(output)}, nil
}

func (s Service) runRuntimeCommand(ctx context.Context, target deploymentdto.RuntimeTarget, command composeCommand) (string, error) {
	if s.runtime == nil || s.targetResolver == nil {
		return "", apperror.New(apperror.KindInternal, "remote deployment runtime is not configured")
	}
	sshTarget, err := s.targetResolver.ResolveProjectTarget(ctx, target.ProjectId)
	if err != nil {
		return "", err
	}
	exists, err := s.runtime.ServiceDirExists(ctx, sshTarget, target.ServiceCode)
	if err != nil {
		return "", apperror.Wrap(apperror.KindInternal, "Failed to inspect remote service workspace", err)
	}
	if !exists {
		return "", apperror.New(apperror.KindNotFound, "managed runtime workspace does not exist")
	}
	output, err := s.runtime.Query(ctx, sshTarget, target.ServiceCode, command.Name, command.Args...)
	if err != nil {
		return output, apperror.New(apperror.KindInternal, outputOrError(output, err))
	}
	return output, nil
}
func (s Service) ensureRuntimeServiceNames(ctx context.Context, serviceId string, names []string) error {
	if len(names) == 0 {
		return nil
	}
	service, err := s.service.Service(ctx, serviceId)
	if err != nil {
		return apperror.Wrap(apperror.KindInternal, "Failed to load runtime service", err)
	}
	components, err := s.application.VersionComponentsByVersion(ctx, service.VersionId)
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

func composeConfigCommand() composeCommand {
	return composeCommand{Name: "docker", Args: []string{"compose", "-f", "docker-compose.yml", "config"}}
}

package delivery

import (
	"errors"
	"time"

	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	gatewaydto "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/dto"
	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/common/commandline"
	applicationv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/application"
	servicev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/service"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func componentInput(input *applicationv1.VersionComponentReq) (applicationdto.VersionComponentInput, error) {
	if input == nil {
		return applicationdto.VersionComponentInput{}, errors.New("component is required")
	}
	return applicationdto.VersionComponentInput{
		Name:          input.Name,
		Image:         input.Image,
		Command:       input.Command,
		PullPolicy:    input.PullPolicy,
		RestartPolicy: input.RestartPolicy, Env: componentEnvInput(input.Env), Endpoints: componentEndpointsInput(input.Endpoints),
		Mounts: componentMountsInput(input.Mounts), Dependencies: componentDependenciesInput(input.Dependencies),
		Healthcheck: healthcheckInput(input.Healthcheck), Resources: resourcesInput(input.Resources), Tmpfs: tmpfsInput(input.Tmpfs),
		Ulimits: ulimitsInput(input.Ulimits), Devices: devicesInput(input.Devices),
	}, nil
}

func componentEnvInput(values []*applicationv1.ComponentEnv) []model.VersionComponentEnv {
	result := make([]model.VersionComponentEnv, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, model.VersionComponentEnv{Key: value.Key, Value: value.Value})
	}
	return result
}

func componentEndpointsInput(values []*applicationv1.ComponentEndpoint) []model.VersionComponentEndpoint {
	result := make([]model.VersionComponentEndpoint, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, model.VersionComponentEndpoint{Name: value.Name, Protocol: value.Protocol, ContainerPort: int(value.ContainerPort), Mode: value.Mode, BindAddress: value.BindAddress, ListenPort: intPtr(value.ListenPort), Entrypoint: value.Entrypoint, PathPrefix: value.PathPrefix})
	}
	return result
}

func componentMountsInput(values []*applicationv1.ComponentMount) []model.VersionComponentMount {
	result := make([]model.VersionComponentMount, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, model.VersionComponentMount{SourceType: value.SourceType, Source: value.Source, Target: value.Target, ReadOnly: value.ReadOnly, SourceIsHostPath: value.SourceIsHostPath, Content: stringValue(value.Content), Mode: value.Mode, IgnoreIfExists: value.IgnoreIfExists})
	}
	return result
}

func componentDependenciesInput(values []*applicationv1.ComponentDependency) []model.VersionComponentDependency {
	result := make([]model.VersionComponentDependency, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, model.VersionComponentDependency{Name: value.Name, Condition: value.Condition})
	}
	return result
}

func healthcheckInput(value *applicationv1.ComponentHealthcheck) *applicationdto.VersionComponentHealthcheckInput {
	if value == nil {
		return nil
	}
	return &applicationdto.VersionComponentHealthcheckInput{TestMode: value.TestMode, Test: value.Test, Interval: value.Interval, Timeout: value.Timeout, Retries: intPtr(value.Retries), StartPeriod: value.StartPeriod, StartInterval: value.StartInterval, Disabled: value.Disabled}
}

func resourcesInput(value *applicationv1.ComponentResources) *model.VersionComponentResources {
	if value == nil {
		return nil
	}
	return &model.VersionComponentResources{LimitCPUs: value.LimitCpus, LimitMemory: value.LimitMemory, ReservationCPUs: value.ReservationCpus, ReservationMemory: value.ReservationMemory}
}

func tmpfsInput(values []*applicationv1.ComponentTmpfs) []model.VersionComponentTmpfs {
	result := make([]model.VersionComponentTmpfs, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, model.VersionComponentTmpfs{Target: value.Target, SizeBytes: value.SizeBytes, Mode: value.Mode})
	}
	return result
}

func ulimitsInput(values []*applicationv1.ComponentUlimit) []model.VersionComponentUlimit {
	result := make([]model.VersionComponentUlimit, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, model.VersionComponentUlimit{Name: value.Name, Soft: value.Soft, Hard: value.Hard})
	}
	return result
}

func devicesInput(values []*applicationv1.ComponentDeviceRequest) []model.VersionComponentDeviceRequest {
	result := make([]model.VersionComponentDeviceRequest, 0, len(values))
	for _, value := range values {
		if value == nil {
			continue
		}
		result = append(result, model.VersionComponentDeviceRequest{Driver: value.Driver, Count: value.Count, Capabilities: append([]string(nil), value.Capabilities...)})
	}
	return result
}

func serviceOverlayInput(input *servicev1.ServiceComponentOverlayUpdateReq) servicedto.ServiceComponentOverlayInput {
	if input == nil {
		return servicedto.ServiceComponentOverlayInput{}
	}
	result := servicedto.ServiceComponentOverlayInput{Resources: serviceResourcesInput(input.Resources)}
	for _, value := range input.Env {
		if value == nil {
			continue
		}
		result.Env = append(result.Env, model.ServiceComponentEnv{Key: value.Key, Value: value.Value, State: model.ServiceComponentOverlayState(value.State)})
	}
	for _, value := range input.Mounts {
		if value == nil {
			continue
		}
		result.Mounts = append(result.Mounts, model.ServiceComponentMount{Target: value.Target, Source: value.Source, State: model.ServiceComponentOverlayState(value.State)})
	}
	for _, value := range input.Endpoints {
		if value == nil {
			continue
		}
		result.Endpoints = append(result.Endpoints, model.ServiceComponentEndpoint{Name: value.Name, Mode: value.Mode, BindAddress: value.BindAddress, ListenPort: intPtr(value.ListenPort), Entrypoint: value.Entrypoint, PathPrefix: value.PathPrefix, State: model.ServiceComponentOverlayState(value.State)})
	}
	return result
}

func serviceResourcesInput(value *servicev1.ServiceComponentResourceOverlay) *model.ServiceComponentResources {
	if value == nil {
		return nil
	}
	return &model.ServiceComponentResources{LimitCPUs: value.LimitCpus, LimitMemory: value.LimitMemory, ReservationCPUs: value.ReservationCpus, ReservationMemory: value.ReservationMemory, State: model.ServiceComponentOverlayState(value.State)}
}

func serviceEnvInput(values []*servicev1.ServiceEnv) servicedto.ServiceEnvUpdateInput {
	result := servicedto.ServiceEnvUpdateInput{Env: make([]model.ServiceEnv, 0, len(values))}
	for _, value := range values {
		if value == nil {
			continue
		}
		result.Env = append(result.Env, model.ServiceEnv{Key: value.Key, Value: value.Value})
	}
	return result
}

func intPtr(value *int32) *int {
	if value == nil {
		return nil
	}
	result := int(*value)
	return &result
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func applicationOutput(value model.Application) map[string]any {
	projectId := ""
	if value.ProjectId != nil {
		projectId = *value.ProjectId
	}
	return map[string]any{"id": value.Id, "project_id": projectId, "name": value.Name, "code": value.Code, "kind": value.Kind, "created_at": formatTime(value.CreatedAt), "updated_at": formatTime(value.UpdatedAt)}
}

func projectOutput(value model.Project) map[string]any {
	return map[string]any{"id": value.Id, "name": value.Name, "code": value.Code, "is_active": value.IsActive, "created_at": formatTime(value.CreatedAt), "updated_at": formatTime(value.UpdatedAt)}
}

func versionOutput(value applicationdto.VersionView) map[string]any {
	components := make([]map[string]any, 0, len(value.Components))
	for _, component := range value.Components {
		components = append(components, componentOutput(component))
	}
	version := value.Version
	result := map[string]any{"id": version.Id, "application_id": version.ApplicationId, "label": version.Label, "status": version.Status, "component_summary": version.ComponentSummary, "components": components, "created_at": formatTime(version.CreatedAt), "updated_at": formatTime(version.UpdatedAt)}
	if version.CreatedFromVersionId != nil {
		result["created_from_version_id"] = *version.CreatedFromVersionId
	}
	if version.Note != nil {
		result["note"] = *version.Note
	}
	return result
}

func componentOutput(value model.VersionComponent) map[string]any {
	env := make([]map[string]any, 0, len(value.Env))
	for _, item := range value.Env {
		env = append(env, map[string]any{"key": item.Key, "value": item.Value})
	}
	endpoints := make([]map[string]any, 0, len(value.Endpoints))
	for _, item := range value.Endpoints {
		endpoints = append(endpoints, endpointOutput(item))
	}
	mounts := make([]map[string]any, 0, len(value.Mounts))
	for _, item := range value.Mounts {
		mounts = append(mounts, map[string]any{"source_type": item.SourceType, "source": item.Source, "target": item.Target, "read_only": item.ReadOnly, "source_is_host_path": item.SourceIsHostPath, "content": item.Content, "mode": item.Mode, "ignore_if_exists": item.IgnoreIfExists})
	}
	dependencies := make([]map[string]any, 0, len(value.Dependencies))
	for _, item := range value.Dependencies {
		dependencies = append(dependencies, map[string]any{"name": item.Name, "condition": item.Condition})
	}
	tmpfs := make([]map[string]any, 0, len(value.Tmpfs))
	for _, item := range value.Tmpfs {
		tmpfs = append(tmpfs, map[string]any{"target": item.Target, "size_bytes": item.SizeBytes, "mode": item.Mode})
	}
	ulimits := make([]map[string]any, 0, len(value.Ulimits))
	for _, item := range value.Ulimits {
		ulimits = append(ulimits, map[string]any{"name": item.Name, "soft": item.Soft, "hard": item.Hard})
	}
	devices := make([]map[string]any, 0, len(value.Devices))
	for _, item := range value.Devices {
		devices = append(devices, map[string]any{"driver": item.Driver, "count": item.Count, "capabilities": item.Capabilities})
	}
	result := map[string]any{"id": value.Id, "version_id": value.VersionId, "name": value.Name, "image": value.Image, "command": commandline.Format(value.Command), "env": env, "endpoints": endpoints, "mounts": mounts, "dependencies": dependencies, "pull_policy": value.PullPolicy, "restart_policy": value.RestartPolicy, "tmpfs": tmpfs, "ulimits": ulimits, "devices": devices, "created_at": formatTime(value.CreatedAt), "updated_at": formatTime(value.UpdatedAt)}
	if value.Healthcheck != nil {
		result["healthcheck"] = healthcheckOutput(*value.Healthcheck)
	}
	if value.Resources != nil {
		result["resources"] = resourcesOutput(*value.Resources)
	}
	if value.Artifact != nil {
		result["artifact"] = map[string]any{"artifact_id": value.Artifact.ArtifactId, "image_ref": value.Artifact.ImageRef, "local_image_sha256": value.Artifact.LocalImageSha256, "source_commit_sha": value.Artifact.SourceCommitSha}
	}
	return result
}

func endpointOutput(value model.VersionComponentEndpoint) map[string]any {
	return map[string]any{"name": value.Name, "protocol": value.Protocol, "container_port": value.ContainerPort, "mode": value.Mode, "bind_address": value.BindAddress, "listen_port": value.ListenPort, "entrypoint": value.Entrypoint, "path_prefix": value.PathPrefix}
}

func healthcheckOutput(value model.VersionComponentHealthcheck) map[string]any {
	return map[string]any{"test_mode": value.TestMode, "test": value.Test, "interval": value.Interval, "timeout": value.Timeout, "retries": value.Retries, "start_period": value.StartPeriod, "start_interval": value.StartInterval, "disabled": value.Disabled}
}

func resourcesOutput(value model.VersionComponentResources) map[string]any {
	return map[string]any{"limit_cpus": value.LimitCPUs, "limit_memory": value.LimitMemory, "reservation_cpus": value.ReservationCPUs, "reservation_memory": value.ReservationMemory}
}

func serviceOutput(value servicedto.ServiceView) map[string]any {
	env := make([]map[string]any, 0, len(value.Env))
	for _, item := range value.Env {
		env = append(env, map[string]any{"key": item.Key, "value": item.Value})
	}
	components := make([]map[string]any, 0, len(value.Components))
	for _, item := range value.Components {
		components = append(components, serviceComponentOutput(item))
	}
	return map[string]any{"id": value.Service.Id, "application_id": value.Service.ApplicationId, "instance_key": value.Service.InstanceKey, "code": value.Service.Code, "version_id": value.Service.VersionId, "status": value.Service.Status, "application_name": value.ApplicationName, "application_code": value.ApplicationCode, "application_kind": value.ApplicationKind, "version_label": value.VersionLabel, "pending_deploy": value.PendingDeploy, "effective_plan_hash": value.EffectivePlanHash, "env": env, "components": components, "created_at": formatTime(value.Service.CreatedAt), "updated_at": formatTime(value.Service.UpdatedAt)}
}

func serviceComponentOutput(value model.ServiceComponent) map[string]any {
	return map[string]any{"id": value.Id, "service_id": value.ServiceId, "source_version_component_id": value.SourceVersionComponentId, "component_name": value.ComponentName, "status": value.Status, "created_at": formatTime(value.CreatedAt), "updated_at": formatTime(value.UpdatedAt)}
}

func deploymentOutput(value model.Deployment) map[string]any {
	result := map[string]any{"id": value.Id, "project_id": value.ProjectId, "application_id": value.ApplicationId, "application_name": value.ApplicationName, "operation_type": value.OperationType, "trigger_type": value.TriggerType, "command_text": value.CommandText, "status": value.Status, "started_at": formatTime(value.StartedAt), "log_text": value.LogText, "error_message": value.ErrorMessage, "is_rollback": value.IsRollback, "rollback_from_deployment_id": value.RollbackFromDeploymentId, "version_id": value.VersionId, "service_id": value.ServiceId, "options_json": value.OptionsJSON}
	if value.FinishedAt != nil {
		result["finished_at"] = formatTime(*value.FinishedAt)
	}
	if value.DurationMs != nil {
		result["duration_ms"] = *value.DurationMs
	}
	return result
}

func deploymentLogsOutput(value deploymentdto.DeploymentLog) map[string]any {
	return map[string]any{"logs": value.Logs, "offset": value.Offset, "is_complete": value.IsComplete, "status": value.Status}
}

func gatewayOutput(value gatewaydto.GatewayView) map[string]any {
	projectId := ""
	if value.Application.ProjectId != nil {
		projectId = *value.Application.ProjectId
	}
	exposures := make([]map[string]any, 0, len(value.Exposures))
	for _, item := range value.Exposures {
		exposures = append(exposures, map[string]any{"application_id": item.ApplicationId, "application_code": item.ApplicationCode, "component_name": item.ComponentName, "protocol": item.Protocol, "access": item.Access, "container_port": item.ContainerPort, "listen_port": item.ListenPort, "public_host": item.PublicHost, "internal_dns": item.InternalDns, "client_hint": item.ClientHint})
	}
	return map[string]any{"id": value.Application.Id, "project_id": projectId, "code": value.Application.Code, "name": value.Application.Name, "kind": value.Application.Kind, "rest_api_url": value.Config.RestApiUrl, "base_domain": value.Config.BaseDomain, "default_entrypoint": value.Config.DefaultEntrypoint, "tls_mode": value.Config.TLSMode, "exposures": exposures, "created_at": formatTime(value.Application.CreatedAt), "updated_at": formatTime(value.Application.UpdatedAt), "config_updated_at": formatTime(value.Config.UpdatedAt)}
}

func provisionGatewayOutput(value gatewaydto.ProvisionGatewayResult) map[string]any {
	resourceIds := map[string]string{"gateway_id": value.Gateway.Application.Id, "application_id": value.Gateway.Application.Id, "version_id": value.Version.Id, "service_id": value.Service.Id, "deployment_id": value.Deployment.Id}
	return map[string]any{
		"operation":    "provision_gateway",
		"ready":        value.Ready,
		"resource_ids": resourceIds,
		"created":      map[string]any{"gateway": value.GatewayCreated, "service": value.ServiceCreated, "version_published": value.VersionPublished},
		"network":      map[string]any{"name": value.Network.Name, "driver": value.Network.Driver, "ready": value.Network.Ready, "status": value.Network.Status},
		"timed_out":    value.TimedOut,
		"steps":        value.Steps,
	}
}

func formatTime(value time.Time) string {
	if value.IsZero() {
		return ""
	}
	return value.UTC().Format(time.RFC3339)
}

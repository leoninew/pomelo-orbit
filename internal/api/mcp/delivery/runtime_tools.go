package delivery

import (
	"context"
	"strings"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerRuntimeTools(server *mcp.Server) {
	addTool(server, "runtime_doctor", "Check Docker prerequisites for one managed runtime target. Supply application_id and instance_key together for an Application target, or gateway_application_id and gateway_instance_key together for a Gateway target. Do not combine those target pairs.", func(ctx context.Context, input struct {
		ApplicationId        string `json:"application_id,omitempty"`
		InstanceKey          string `json:"instance_key,omitempty"`
		GatewayApplicationId string `json:"gateway_application_id,omitempty"`
		GatewayInstanceKey   string `json:"gateway_instance_key,omitempty"`
	}) (map[string]any, error) {
		hasApplication := input.ApplicationId != "" || input.InstanceKey != ""
		hasGateway := input.GatewayApplicationId != "" || input.GatewayInstanceKey != ""
		if hasApplication && (input.ApplicationId == "" || input.InstanceKey == "") {
			return nil, apperror.New(apperror.KindValidation, "application_id and instance_key must be supplied together")
		}
		if hasGateway && (input.GatewayApplicationId == "" || input.GatewayInstanceKey == "") {
			return nil, apperror.New(apperror.KindValidation, "gateway_application_id and gateway_instance_key must be supplied together")
		}
		if hasApplication && hasGateway {
			return nil, apperror.New(apperror.KindValidation, "application target and gateway target cannot be requested together")
		}
		if !hasApplication && !hasGateway {
			return nil, apperror.New(apperror.KindValidation, "a managed runtime target is required")
		}
		var target *deploymentdto.RuntimeTarget
		if hasApplication {
			if _, err := c.applicationInScope(ctx, input.ApplicationId); err != nil {
				return nil, err
			}
			resolved, err := c.deps.Deployment.ResolveRuntimeTarget(ctx, c.deps.ActorUserId, input.ApplicationId, input.InstanceKey, false)
			if err != nil {
				return nil, err
			}
			target = &resolved
		}
		if hasGateway {
			if _, err := c.gatewayInScope(ctx, input.GatewayApplicationId); err != nil {
				return nil, err
			}
			resolved, err := c.deps.Deployment.ResolveRuntimeTarget(ctx, c.deps.ActorUserId, input.GatewayApplicationId, input.GatewayInstanceKey, true)
			if err != nil {
				return nil, err
			}
			target = &resolved
		}
		result, err := c.deps.Deployment.RuntimeDoctor(ctx, target)
		if err != nil {
			return nil, err
		}
		if target != nil {
			result = runtimeOutput(*target, result)
		}
		return result, nil
	})

	addTool(server, "runtime_compose_config", "Read rendered Docker Compose configuration for one managed runtime target.", func(ctx context.Context, input runtimeTargetInput) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ApplicationId, input.InstanceKey)
		if err != nil {
			return nil, err
		}
		result, err := c.deps.Deployment.RuntimeComposeConfig(ctx, target)
		if err != nil {
			return nil, err
		}
		return runtimeOutput(result.Target, map[string]any{"config": result.Text}), nil
	})

	addTool(server, "runtime_compose_ps", "Read concise Compose container state; set detail=true for raw Compose JSON.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
		InstanceKey   string `json:"instance_key,omitempty"`
		Detail        bool   `json:"detail,omitempty"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ApplicationId, input.InstanceKey)
		if err != nil {
			return nil, err
		}
		result, err := c.deps.Deployment.RuntimeComposePS(ctx, target)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(result.Containers))
		for _, container := range result.Containers {
			items = append(items, runtimeContainerOutput(container))
		}
		data := map[string]any{"containers": items}
		if input.Detail {
			data["raw"] = result.Raw
		}
		return runtimeOutput(result.Target, data), nil
	})

	addTool(server, "runtime_compose_logs", "Read Compose logs for services derived from one managed runtime target.", func(ctx context.Context, input struct {
		ApplicationId string   `json:"application_id" jsonschema:"required"`
		InstanceKey   string   `json:"instance_key,omitempty"`
		Tail          int      `json:"tail,omitempty"`
		Since         string   `json:"since,omitempty"`
		Services      []string `json:"services,omitempty"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ApplicationId, input.InstanceKey)
		if err != nil {
			return nil, err
		}
		tail := input.Tail
		if tail == 0 {
			tail = 200
		}
		result, err := c.deps.Deployment.RuntimeComposeLogs(ctx, target, tail, input.Since, input.Services)
		if err != nil {
			return nil, err
		}
		return runtimeOutput(result.Target, map[string]any{"logs": result.Text}), nil
	})

	addTool(server, "runtime_container_inspect", "Inspect a container only when its Id was returned by this target's Compose ps output.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
		ContainerId   string `json:"container_id" jsonschema:"required"`
		InstanceKey   string `json:"instance_key,omitempty"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ApplicationId, input.InstanceKey)
		if err != nil {
			return nil, err
		}
		result, err := c.deps.Deployment.RuntimeContainerInspect(ctx, target, input.ContainerId)
		if err != nil {
			return nil, err
		}
		return runtimeOutput(result.Target, map[string]any{"inspect": result.Data}), nil
	})

	addTool(server, "runtime_network_inspect", "Inspect a network only when it is derived from a target-managed container inspect result.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
		NetworkName   string `json:"network_name" jsonschema:"required"`
		InstanceKey   string `json:"instance_key,omitempty"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ApplicationId, input.InstanceKey)
		if err != nil {
			return nil, err
		}
		result, err := c.deps.Deployment.RuntimeNetworkInspect(ctx, target, input.NetworkName)
		if err != nil {
			return nil, err
		}
		return runtimeOutput(result.Target, map[string]any{"inspect": result.Data}), nil
	})

	addTool(server, "runtime_http_probe", "Run a fixed local HTTP curl probe in one managed Compose component.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
		ComponentName string `json:"component_name" jsonschema:"required"`
		Port          int    `json:"port" jsonschema:"required"`
		Path          string `json:"path,omitempty"`
		InstanceKey   string `json:"instance_key,omitempty"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ApplicationId, input.InstanceKey)
		if err != nil {
			return nil, err
		}
		path := input.Path
		if path == "" {
			path = "/"
		}
		result, err := c.deps.Deployment.RuntimeHTTPProbe(ctx, target, input.ComponentName, input.Port, path)
		if err != nil {
			return nil, err
		}
		return runtimeOutput(result.Target, map[string]any{"output": result.Text}), nil
	})
}

type runtimeTargetInput struct {
	ApplicationId string `json:"application_id" jsonschema:"required"`
	InstanceKey   string `json:"instance_key,omitempty"`
}

func (c *core) runtimeTarget(ctx context.Context, applicationId, instanceKey string) (deploymentdto.RuntimeTarget, error) {
	if _, err := c.applicationInScope(ctx, applicationId); err != nil {
		return deploymentdto.RuntimeTarget{}, err
	}
	if strings.TrimSpace(instanceKey) == "" {
		instanceKey = "default"
	}
	return c.deps.Deployment.ResolveRuntimeTarget(ctx, c.deps.ActorUserId, applicationId, instanceKey, false)
}

func runtimeOutput(target deploymentdto.RuntimeTarget, data map[string]any) map[string]any {
	targetOutput := map[string]any{"project_id": target.ProjectId, "application_id": target.ApplicationId, "service_id": target.ServiceId, "instance_key": target.InstanceKey, "service_code": target.ServiceCode, "working_directory": target.WorkingDirectory, "compose_project": target.ComposeProject}
	result := map[string]any{"project_id": target.ProjectId, "target": targetOutput, "working_directory": target.WorkingDirectory}
	for key, value := range data {
		result[key] = value
	}
	return result
}

func runtimeContainerOutput(value deploymentdto.RuntimeContainer) map[string]any {
	return map[string]any{"id": value.Id, "name": value.Name, "service": value.Service, "state": value.State, "status": value.Status, "health": value.Health, "image": value.Image, "version_id": value.VersionId, "version_label": value.VersionLabel, "component_id": value.ComponentId}
}

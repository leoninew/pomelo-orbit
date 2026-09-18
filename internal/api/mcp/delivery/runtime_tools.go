package delivery

import (
	"context"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerRuntimeTools(server *mcp.Server) {
	addTool(server, "runtime_doctor", "Check Docker prerequisites for one managed runtime Service.", func(ctx context.Context, input struct {
		ServiceId string `json:"service_id" jsonschema:"required"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ServiceId)
		if err != nil {
			return nil, err
		}
		result, err := c.deps.Deployment.RuntimeDoctor(ctx, &target)
		if err != nil {
			return nil, err
		}
		return runtimeOutput(target, result), nil
	})

	addTool(server, "runtime_compose_config", "Read rendered Docker Compose configuration for one managed runtime target.", func(ctx context.Context, input runtimeTargetInput) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ServiceId)
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
		ServiceId string `json:"service_id" jsonschema:"required"`
		Detail    bool   `json:"detail,omitempty"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ServiceId)
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
		ServiceId string   `json:"service_id" jsonschema:"required"`
		Tail      int      `json:"tail,omitempty"`
		Since     string   `json:"since,omitempty"`
		Services  []string `json:"services,omitempty"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ServiceId)
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
		ServiceId   string `json:"service_id" jsonschema:"required"`
		ContainerId string `json:"container_id" jsonschema:"required"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ServiceId)
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
		ServiceId   string `json:"service_id" jsonschema:"required"`
		NetworkName string `json:"network_name" jsonschema:"required"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ServiceId)
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
		ServiceId     string `json:"service_id" jsonschema:"required"`
		ComponentName string `json:"component_name" jsonschema:"required"`
		Port          int    `json:"port" jsonschema:"required"`
		Path          string `json:"path,omitempty"`
	}) (map[string]any, error) {
		target, err := c.runtimeTarget(ctx, input.ServiceId)
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
	ServiceId string `json:"service_id" jsonschema:"required"`
}

func (c *core) runtimeTarget(ctx context.Context, serviceId string) (deploymentdto.RuntimeTarget, error) {
	if err := c.serviceInScope(ctx, serviceId); err != nil {
		return deploymentdto.RuntimeTarget{}, err
	}
	projectId, err := c.currentProjectId()
	if err != nil {
		return deploymentdto.RuntimeTarget{}, err
	}
	return c.deps.Deployment.ResolveRuntimeTarget(ctx, c.deps.ActorUserId, projectId, serviceId, true)
}

func runtimeOutput(target deploymentdto.RuntimeTarget, data map[string]any) map[string]any {
	targetOutput := map[string]any{"project_id": target.ProjectId, "application_id": target.ApplicationId, "service_id": target.ServiceId, "service_code": target.ServiceCode, "working_directory": target.WorkingDirectory, "compose_project": target.ComposeProject}
	result := map[string]any{"project_id": target.ProjectId, "target": targetOutput, "working_directory": target.WorkingDirectory}
	for key, value := range data {
		result[key] = value
	}
	return result
}

func runtimeContainerOutput(value deploymentdto.RuntimeContainer) map[string]any {
	return map[string]any{"id": value.Id, "name": value.Name, "service": value.Service, "state": value.State, "status": value.Status, "health": value.Health, "image": value.Image, "version_id": value.VersionId, "version_label": value.VersionLabel, "component_id": value.ComponentId}
}

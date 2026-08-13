package delivery

import (
	"context"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	applicationv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/application"
	"github.com/leoninew/pomelo-orbit/internal/model"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerVersionComponentTools(server *mcp.Server) {
	addTool(server, "orbit_update_version_component_basic", "Replace a Component's name, image, command, pull policy, and restart policy.", func(ctx context.Context, input struct {
		VersionId     string  `json:"version_id" jsonschema:"required"`
		ComponentId   string  `json:"component_id" jsonschema:"required"`
		Name          string  `json:"name" jsonschema:"required"`
		Image         string  `json:"image" jsonschema:"required"`
		Command       string  `json:"command,omitempty"`
		PullPolicy    string  `json:"pull_policy" jsonschema:"required"`
		RestartPolicy *string `json:"restart_policy,omitempty"`
	}) (map[string]any, error) {
		command, err := commandInput(input.Command)
		if err != nil {
			return nil, err
		}
		component, err := c.deps.Application.UpdateVersionComponentBasic(ctx, c.deps.ActorUserId, input.VersionId, input.ComponentId, applicationdto.VersionComponentBasicUpdateInput{Name: input.Name, Image: input.Image, Command: command, PullPolicy: input.PullPolicy, RestartPolicy: input.RestartPolicy})
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_basic", input.VersionId, input.ComponentId, "/basic", component), nil
	})

	addTool(server, "orbit_update_version_component_runtime", "Replace a Component's health check.", func(ctx context.Context, input struct {
		VersionId   string                              `json:"version_id" jsonschema:"required"`
		ComponentId string                              `json:"component_id" jsonschema:"required"`
		Healthcheck *applicationv1.ComponentHealthcheck `json:"healthcheck,omitempty"`
	}) (map[string]any, error) {
		component, err := c.deps.Application.UpdateVersionComponentRuntime(ctx, c.deps.ActorUserId, input.VersionId, input.ComponentId, applicationdto.VersionComponentRuntimeUpdateInput{Healthcheck: healthcheckInput(input.Healthcheck)})
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_runtime", input.VersionId, input.ComponentId, "/runtime", component), nil
	})

	// The following collection tools deliberately keep their collection at the
	// top level. Python's wrapper schema required mounts: { mounts: [...] };
	// this Go schema is mounts: [...], and the same applies to peer fields.
	addTool(server, "orbit_update_version_component_endpoints", "Replace a Component's declared endpoint collection. Each endpoint requires protocol=http|tcp, container_port (1-65535), and mode=internal|local|host|gateway. Use gateway only for an HTTP route exposed by the managed Gateway; use an internal TCP endpoint as a custom TCP Route target. Include bind_address, listen_port, entrypoint, or path_prefix only when that selected mode needs an override. This replaces the complete collection, so preserve unrelated endpoints from the preceding orbit_get_version read.", func(ctx context.Context, input struct {
		VersionId   string                             `json:"version_id" jsonschema:"required"`
		ComponentId string                             `json:"component_id" jsonschema:"required"`
		Endpoints   []*applicationv1.ComponentEndpoint `json:"endpoints" jsonschema:"required"`
	}) (map[string]any, error) {
		component, err := c.deps.Application.UpdateVersionComponentEndpoints(ctx, c.deps.ActorUserId, input.VersionId, input.ComponentId, applicationdto.VersionComponentEndpointsUpdateInput{Endpoints: componentEndpointsInput(input.Endpoints)})
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_endpoints", input.VersionId, input.ComponentId, "/endpoints", component), nil
	})

	addTool(server, "orbit_update_version_component_env", "Replace a Component's environment collection.", func(ctx context.Context, input struct {
		VersionId   string                        `json:"version_id" jsonschema:"required"`
		ComponentId string                        `json:"component_id" jsonschema:"required"`
		Env         []*applicationv1.ComponentEnv `json:"env" jsonschema:"required"`
	}) (map[string]any, error) {
		component, err := c.deps.Application.UpdateVersionComponentEnv(ctx, c.deps.ActorUserId, input.VersionId, input.ComponentId, applicationdto.VersionComponentEnvUpdateInput{Env: componentEnvInput(input.Env)})
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_env", input.VersionId, input.ComponentId, "/env", component), nil
	})

	addTool(server, "orbit_update_version_component_mounts", "Replace a Component's mount collection. Each item requires source_type, source, and target. source_type=directory or file uses a managed relative source by default; set source_is_host_path=true only for an absolute host directory/file. source_type=named_volume uses a bare volume name and no file options. source_type=controlled_file materializes content under a relative, non-host source: source_is_host_path must be false, source cannot be absolute, contain backslashes, or contain '..'; content may be an empty string; mode is required as four-digit Unix octal such as 0644; content is at most 262144 bytes; ignore_if_exists is allowed only here. This replaces the complete collection, so preserve unrelated mounts from the preceding orbit_get_version read.", func(ctx context.Context, input struct {
		VersionId   string                          `json:"version_id" jsonschema:"required"`
		ComponentId string                          `json:"component_id" jsonschema:"required"`
		Mounts      []*applicationv1.ComponentMount `json:"mounts" jsonschema:"required"`
	}) (map[string]any, error) {
		component, err := c.deps.Application.UpdateVersionComponentMounts(ctx, c.deps.ActorUserId, input.VersionId, input.ComponentId, applicationdto.VersionComponentMountsUpdateInput{Mounts: componentMountsInput(input.Mounts)})
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_mounts", input.VersionId, input.ComponentId, "/mounts", component), nil
	})

	addTool(server, "orbit_update_version_component_dependencies", "Replace a Component's dependency collection.", func(ctx context.Context, input struct {
		VersionId    string                               `json:"version_id" jsonschema:"required"`
		ComponentId  string                               `json:"component_id" jsonschema:"required"`
		Dependencies []*applicationv1.ComponentDependency `json:"dependencies" jsonschema:"required"`
	}) (map[string]any, error) {
		component, err := c.deps.Application.UpdateVersionComponentDependencies(ctx, c.deps.ActorUserId, input.VersionId, input.ComponentId, applicationdto.VersionComponentDependenciesUpdateInput{Dependencies: componentDependenciesInput(input.Dependencies)})
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_dependencies", input.VersionId, input.ComponentId, "/dependencies", component), nil
	})

	addTool(server, "orbit_update_version_component_devices", "Replace a Component's device requests without changing resources, tmpfs, or ulimits.", func(ctx context.Context, input struct {
		VersionId   string                                  `json:"version_id" jsonschema:"required"`
		ComponentId string                                  `json:"component_id" jsonschema:"required"`
		Devices     []*applicationv1.ComponentDeviceRequest `json:"devices" jsonschema:"required"`
	}) (map[string]any, error) {
		component, err := c.deps.Application.UpdateVersionComponentDevices(ctx, c.deps.ActorUserId, input.VersionId, input.ComponentId, applicationdto.VersionComponentDevicesUpdateInput{Devices: devicesInput(input.Devices)})
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_devices", input.VersionId, input.ComponentId, "/devices", component), nil
	})

	addTool(server, "orbit_update_version_component_advanced", "Replace a Component's resources, tmpfs, and ulimit settings.", func(ctx context.Context, input struct {
		VersionId   string                            `json:"version_id" jsonschema:"required"`
		ComponentId string                            `json:"component_id" jsonschema:"required"`
		Resources   *applicationv1.ComponentResources `json:"resources,omitempty"`
		Tmpfs       []*applicationv1.ComponentTmpfs   `json:"tmpfs,omitempty"`
		Ulimits     []*applicationv1.ComponentUlimit  `json:"ulimits,omitempty"`
	}) (map[string]any, error) {
		component, err := c.updateComponentAdvanced(ctx, input.VersionId, input.ComponentId, resourcesInput(input.Resources), tmpfsInput(input.Tmpfs), ulimitsInput(input.Ulimits))
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_advanced", input.VersionId, input.ComponentId, "/advanced", component), nil
	})

	addTool(server, "orbit_update_version_component_resources", "Replace a Component's resource constraints and preserve its tmpfs and ulimit settings.", func(ctx context.Context, input struct {
		VersionId   string                            `json:"version_id" jsonschema:"required"`
		ComponentId string                            `json:"component_id" jsonschema:"required"`
		Resources   *applicationv1.ComponentResources `json:"resources,omitempty"`
	}) (map[string]any, error) {
		previous, err := c.deps.Application.VersionComponentForUser(ctx, c.deps.ActorUserId, input.VersionId, input.ComponentId)
		if err != nil {
			return nil, err
		}
		component, err := c.updateComponentAdvanced(ctx, input.VersionId, input.ComponentId, resourcesInput(input.Resources), previous.Tmpfs, previous.Ulimits)
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_resources", input.VersionId, input.ComponentId, "/advanced", component), nil
	})

	addTool(server, "orbit_update_version_component_tmpfs", "Replace a Component's tmpfs collection and preserve its resource and ulimit settings.", func(ctx context.Context, input struct {
		VersionId   string                          `json:"version_id" jsonschema:"required"`
		ComponentId string                          `json:"component_id" jsonschema:"required"`
		Tmpfs       []*applicationv1.ComponentTmpfs `json:"tmpfs" jsonschema:"required"`
	}) (map[string]any, error) {
		previous, err := c.deps.Application.VersionComponentForUser(ctx, c.deps.ActorUserId, input.VersionId, input.ComponentId)
		if err != nil {
			return nil, err
		}
		component, err := c.updateComponentAdvanced(ctx, input.VersionId, input.ComponentId, previous.Resources, tmpfsInput(input.Tmpfs), previous.Ulimits)
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_tmpfs", input.VersionId, input.ComponentId, "/advanced", component), nil
	})

	addTool(server, "orbit_update_version_component_ulimits", "Replace a Component's ulimit collection and preserve its resource and tmpfs settings.", func(ctx context.Context, input struct {
		VersionId   string                           `json:"version_id" jsonschema:"required"`
		ComponentId string                           `json:"component_id" jsonschema:"required"`
		Ulimits     []*applicationv1.ComponentUlimit `json:"ulimits" jsonschema:"required"`
	}) (map[string]any, error) {
		previous, err := c.deps.Application.VersionComponentForUser(ctx, c.deps.ActorUserId, input.VersionId, input.ComponentId)
		if err != nil {
			return nil, err
		}
		component, err := c.updateComponentAdvanced(ctx, input.VersionId, input.ComponentId, previous.Resources, previous.Tmpfs, ulimitsInput(input.Ulimits))
		if err != nil {
			return nil, err
		}
		return componentWriteResult("update_version_component_ulimits", input.VersionId, input.ComponentId, "/advanced", component), nil
	})
}

func componentWriteResult(operation, versionId, componentId, suffix string, component model.VersionComponent) map[string]any {
	return writeResult(operation, map[string]string{"version_id": versionId, "component_id": componentId}, "PUT", "/api/version/"+versionId+"/component/"+componentId+suffix, map[string]any{"component": componentOutput(component)})
}

func commandInput(value string) (string, error) {
	return value, nil
}

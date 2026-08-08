package delivery

import (
	"context"

	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	servicev1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/service"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerServiceTools(server *mcp.Server) {
	addTool(server, "orbit_create_service", "Create a stopped Service whose Component overlays initially inherit the Version. code must be a globally unique lowercase DNS label (a-z, 0-9, and internal hyphens only; it cannot start or end with a hyphen and is 63 characters maximum) and cannot be changed after creation.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
		VersionId     string `json:"version_id" jsonschema:"required"`
		InstanceKey   string `json:"instance_key" jsonschema:"required"`
		Code          string `json:"code" jsonschema:"required"`
	}) (map[string]any, error) {
		service, err := c.deps.Service.CreateService(ctx, c.deps.ActorUserId, servicedto.ServiceCreateInput{ApplicationId: input.ApplicationId, VersionId: input.VersionId, InstanceKey: input.InstanceKey, Code: input.Code})
		if err != nil {
			return nil, err
		}
		return writeResult("create_service", map[string]string{"application_id": input.ApplicationId, "service_id": service.Service.Id}, "POST", "/api/service", map[string]any{"service": serviceOutput(service)}), nil
	})

	addTool(server, "orbit_update_service_component_overlay", "Replace one declared Service Component's sparse runtime overlay.", func(ctx context.Context, input struct {
		ServiceId   string                                      `json:"service_id" jsonschema:"required"`
		ComponentId string                                      `json:"component_id" jsonschema:"required"`
		Overlay     *servicev1.ServiceComponentOverlayUpdateReq `json:"overlay" jsonschema:"required"`
	}) (map[string]any, error) {
		component, err := c.deps.Service.UpdateServiceComponentOverlay(ctx, c.deps.ActorUserId, input.ServiceId, input.ComponentId, serviceOverlayInput(input.Overlay))
		if err != nil {
			return nil, err
		}
		return writeResult("update_service_component_overlay", map[string]string{"service_id": input.ServiceId, "component_id": input.ComponentId}, "PUT", "/api/service/"+input.ServiceId+"/component/"+input.ComponentId, map[string]any{"component": serviceComponentOutput(component)}), nil
	})

	// env is a top-level collection by design. This avoids the Python wrapper's
	// env: { env: [...] } shape while retaining the same application DTO.
	addTool(server, "orbit_update_service_env", "Replace the Service environment shared by declared Components.", func(ctx context.Context, input struct {
		ServiceId string                  `json:"service_id" jsonschema:"required"`
		Env       []*servicev1.ServiceEnv `json:"env" jsonschema:"required"`
	}) (map[string]any, error) {
		service, err := c.deps.Service.UpdateServiceEnv(ctx, c.deps.ActorUserId, input.ServiceId, serviceEnvInput(input.Env))
		if err != nil {
			return nil, err
		}
		return writeResult("update_service_env", map[string]string{"service_id": input.ServiceId}, "PUT", "/api/service/"+input.ServiceId+"/env", map[string]any{"service": serviceOutput(service)}), nil
	})

	addTool(server, "orbit_update_service_basic", "Replace a Service's selected Version and instance key.", func(ctx context.Context, input struct {
		ServiceId   string `json:"service_id" jsonschema:"required"`
		VersionId   string `json:"version_id" jsonschema:"required"`
		InstanceKey string `json:"instance_key" jsonschema:"required"`
	}) (map[string]any, error) {
		service, err := c.deps.Service.UpdateServiceBasic(ctx, c.deps.ActorUserId, input.ServiceId, servicedto.ServiceBasicUpdateInput{VersionId: input.VersionId, InstanceKey: input.InstanceKey})
		if err != nil {
			return nil, err
		}
		return writeResult("update_service_basic", map[string]string{"service_id": input.ServiceId}, "PUT", "/api/service/"+input.ServiceId+"/basic", map[string]any{"service": serviceOutput(service)}), nil
	})
}

package delivery

import (
	"context"

	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	servicev1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/service"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerServiceTools(server *mcp.Server) {
	addTool(server, "orbit_get_service_component", "Read a Service Component's complete sparse overlay and Version declaration before replacing the overlay. Optional mount values distinguish inheritance from explicit empty strings and false.", func(ctx context.Context, input struct {
		ServiceId   string `json:"service_id" jsonschema:"required"`
		ComponentId string `json:"component_id" jsonschema:"required"`
	}) (map[string]any, error) {
		if err := c.serviceInScope(ctx, input.ServiceId); err != nil {
			return nil, err
		}
		projectId, err := c.currentProjectId()
		if err != nil {
			return nil, err
		}
		detail, err := c.deps.Service.GetServiceComponent(ctx, c.deps.ActorUserId, projectId, input.ServiceId, input.ComponentId)
		if err != nil {
			return nil, err
		}
		return map[string]any{"service_component": serviceComponentOutput(detail.ServiceComponent), "version_component": componentOutput(detail.VersionComponent)}, nil
	})
	addWriteTool(server, c.deps.TransactionRunner, "orbit_create_service", "Create a stopped Service whose Component overlays initially inherit the Version. The Service code is immutable after creation and must be unique within the selected Project.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
		VersionId     string `json:"version_id" jsonschema:"required"`
		Code          string `json:"code" jsonschema:"required"`
	}) (map[string]any, error) {
		application, err := c.applicationInScope(ctx, input.ApplicationId)
		if err != nil {
			return nil, err
		}
		projectId, err := c.currentProjectId()
		if err != nil {
			return nil, err
		}
		service, err := c.deps.Service.CreateService(ctx, c.deps.ActorUserId, projectId, servicedto.ServiceCreateInput{ApplicationId: application.Id, VersionId: input.VersionId, Code: input.Code})
		if err != nil {
			return nil, err
		}
		return writeResult("create_service", map[string]string{"application_id": input.ApplicationId, "service_id": service.Service.Id}, "POST", "/api/service", map[string]any{"service": serviceOutput(service)}), nil
	})

	addWriteTool(server, c.deps.TransactionRunner, "orbit_update_service_component_overlay", "Replace the complete sparse overlay; first read orbit_get_service_component and preserve unrelated values. Omitted optional mount fields inherit; empty content and false are explicit overrides. Controlled-file content is limited to 256 KiB; read_only is not supported for controlled files. ignore_if_exists preserves an existing file and initializes a missing one. An empty entrypoint or command explicitly selects an empty argv.", func(ctx context.Context, input struct {
		ServiceId   string                                      `json:"service_id" jsonschema:"required"`
		ComponentId string                                      `json:"component_id" jsonschema:"required"`
		Overlay     *servicev1.ServiceComponentOverlayUpdateReq `json:"overlay" jsonschema:"required"`
	}) (map[string]any, error) {
		if err := c.serviceInScope(ctx, input.ServiceId); err != nil {
			return nil, err
		}
		projectId, err := c.currentProjectId()
		if err != nil {
			return nil, err
		}
		component, err := c.deps.Service.UpdateServiceComponentOverlay(ctx, c.deps.ActorUserId, projectId, input.ServiceId, input.ComponentId, serviceOverlayInput(input.Overlay))
		if err != nil {
			return nil, err
		}
		return writeResult("update_service_component_overlay", map[string]string{"service_id": input.ServiceId, "component_id": input.ComponentId}, "PUT", "/api/service/"+input.ServiceId+"/component/"+input.ComponentId, map[string]any{"component": serviceComponentOutput(component)}), nil
	})

	// env is a top-level collection by design. This avoids the Python wrapper's
	// env: { env: [...] } shape while retaining the same application DTO.
	addWriteTool(server, c.deps.TransactionRunner, "orbit_update_service_env", "Replace the Service environment shared by declared Components and resolve their Version ${KEY} placeholders. Generate credentials securely once and keep them stable. Database bootstrap credentials apply only to an empty data volume, so changing a value here and redeploying does not rotate an initialized database credential; require an explicit in-place rotation or volume-reset decision.", func(ctx context.Context, input struct {
		ServiceId string                  `json:"service_id" jsonschema:"required"`
		Env       []*servicev1.ServiceEnv `json:"env" jsonschema:"required"`
	}) (map[string]any, error) {
		if err := c.serviceInScope(ctx, input.ServiceId); err != nil {
			return nil, err
		}
		projectId, err := c.currentProjectId()
		if err != nil {
			return nil, err
		}
		service, err := c.deps.Service.UpdateServiceEnv(ctx, c.deps.ActorUserId, projectId, input.ServiceId, serviceEnvInput(input.Env))
		if err != nil {
			return nil, err
		}
		return writeResult("update_service_env", map[string]string{"service_id": input.ServiceId}, "PUT", "/api/service/"+input.ServiceId+"/env", map[string]any{"service": serviceOutput(service)}), nil
	})

	addWriteTool(server, c.deps.TransactionRunner, "orbit_update_service_basic", "Replace a Service's selected Version.", func(ctx context.Context, input struct {
		ServiceId string `json:"service_id" jsonschema:"required"`
		VersionId string `json:"version_id" jsonschema:"required"`
	}) (map[string]any, error) {
		if err := c.serviceInScope(ctx, input.ServiceId); err != nil {
			return nil, err
		}
		projectId, err := c.currentProjectId()
		if err != nil {
			return nil, err
		}
		service, err := c.deps.Service.UpdateServiceBasic(ctx, c.deps.ActorUserId, projectId, input.ServiceId, servicedto.ServiceBasicUpdateInput{VersionId: input.VersionId})
		if err != nil {
			return nil, err
		}
		return writeResult("update_service_basic", map[string]string{"service_id": input.ServiceId}, "PUT", "/api/service/"+input.ServiceId+"/basic", map[string]any{"service": serviceOutput(service)}), nil
	})
}

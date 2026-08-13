package delivery

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	applicationv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/application"
	"github.com/leoninew/pomelo-orbit/internal/model"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

const implementationVersion = "0.1.0"

// NewServer creates the shared Delivery MCP Core. Streamable HTTP supplies an
// actor when the connection is accepted; stdio can bind one lazily on its
// first tools/call request.
func NewServer(deps Dependencies) (*mcp.Server, error) {
	if strings.TrimSpace(deps.ActorUserId) == "" && deps.ActorAuthorizer == nil {
		return nil, errors.New("mcp actor_user_id or actor authorizer is required")
	}
	server := mcp.NewServer(&mcp.Implementation{Name: "pomelo-delivery", Version: implementationVersion}, nil)
	core := &core{deps: deps}
	if deps.ActorAuthorizer != nil {
		server.AddReceivingMiddleware(core.authorizeToolCalls)
	}
	core.registerOrbitTools(server)
	return server, nil
}

type core struct {
	deps            Dependencies
	authorizationMu sync.Mutex
}

func (c *core) authorizeToolCalls(next mcp.MethodHandler) mcp.MethodHandler {
	return func(ctx context.Context, method string, request mcp.Request) (mcp.Result, error) {
		if method == "tools/call" {
			if err := c.ensureActor(ctx); err != nil {
				return &mcp.CallToolResult{
					Content: []mcp.Content{&mcp.TextContent{Text: "unauthorized: MCP authorization is required before tools can be used"}},
					IsError: true,
				}, nil
			}
		}
		return next(ctx, method, request)
	}
}

func (c *core) ensureActor(ctx context.Context) error {
	c.authorizationMu.Lock()
	defer c.authorizationMu.Unlock()

	if strings.TrimSpace(c.deps.ActorUserId) != "" {
		return nil
	}
	actorUserId, err := c.deps.ActorAuthorizer(ctx)
	if err != nil {
		return err
	}
	if strings.TrimSpace(actorUserId) == "" {
		return errors.New("MCP actor authorizer returned an empty user ID")
	}
	c.deps.ActorUserId = actorUserId
	return nil
}

func addTool[In any](server *mcp.Server, name, description string, handler func(context.Context, In) (map[string]any, error)) {
	mcp.AddTool(server, &mcp.Tool{Name: name, Description: description}, func(ctx context.Context, _ *mcp.CallToolRequest, input In) (*mcp.CallToolResult, map[string]any, error) {
		output, err := handler(ctx, input)
		if err != nil {
			return nil, nil, toolError(err)
		}
		return nil, output, nil
	})
}

func toolError(err error) error {
	if appErr, ok := apperror.As(err); ok {
		classification := apperror.Classify(appErr)
		return fmt.Errorf("%s: %s", classification.Code, classification.Message)
	}
	return fmt.Errorf("internal_error: %s", err)
}

func (c *core) registerOrbitTools(server *mcp.Server) {
	addTool(server, "orbit_list_projects", "List Projects visible to the configured Orbit user.", func(ctx context.Context, _ struct{}) (map[string]any, error) {
		projects, err := c.deps.Project.ListByMember(ctx, c.deps.ActorUserId)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(projects))
		for _, project := range projects {
			items = append(items, projectOutput(project))
		}
		return map[string]any{"projects": items}, nil
	})

	addTool(server, "orbit_list_applications", "List Orbit Applications in a Project, optionally limited to one application kind.", func(ctx context.Context, input struct {
		ProjectId string `json:"project_id" jsonschema:"required"`
		Kind      string `json:"kind,omitempty"`
	}) (map[string]any, error) {
		kind := input.Kind
		if kind == "" {
			kind = "standard"
		}
		projectId := input.ProjectId
		apps, err := c.deps.Application.ListApplications(ctx, c.deps.ActorUserId, &projectId, 1, 10000, "", kind)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(apps.Items))
		for _, app := range apps.Items {
			items = append(items, applicationOutput(app))
		}
		return map[string]any{"project_id": projectId, "kind": kind, "applications": items}, nil
	})

	addTool(server, "orbit_list_application_services", "List non-sensitive Service summaries for an Orbit Application.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
	}) (map[string]any, error) {
		services, err := c.deps.Service.ListServicesByApplication(ctx, c.deps.ActorUserId, input.ApplicationId)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(services))
		for _, service := range services {
			items = append(items, map[string]any{"id": service.Id, "application_id": service.ApplicationId, "instance_key": service.InstanceKey, "code": service.Code, "version_id": service.VersionId, "status": service.Status, "created_at": formatTime(service.CreatedAt), "updated_at": formatTime(service.UpdatedAt)})
		}
		return map[string]any{"application_id": input.ApplicationId, "services": items}, nil
	})

	addTool(server, "orbit_list_gateways", "List Gateway metadata in a Project through Orbit.", func(ctx context.Context, input struct {
		ProjectId string `json:"project_id" jsonschema:"required"`
	}) (map[string]any, error) {
		gateways, err := c.deps.Gateway.ListGateways(ctx, c.deps.ActorUserId, input.ProjectId, 1, 10000, "")
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(gateways.Items))
		for _, gateway := range gateways.Items {
			items = append(items, gatewayOutput(gateway))
		}
		return map[string]any{"project_id": input.ProjectId, "gateways": items}, nil
	})

	addTool(server, "orbit_create_gateway", "Create an Orbit-managed Gateway; deploy it with the existing deployment tools.", func(ctx context.Context, input struct {
		ProjectId                  string  `json:"project_id" jsonschema:"required"`
		Code                       string  `json:"code,omitempty"`
		Name                       string  `json:"name,omitempty"`
		RestApiUrl                 string  `json:"rest_api_url,omitempty"`
		BaseDomain                 string  `json:"base_domain,omitempty"`
		InitialComponentImage      *string `json:"initial_component_image,omitempty"`
		InitialComponentPullPolicy string  `json:"initial_component_pull_policy,omitempty"`
		DefaultEntrypoint          *string `json:"default_entrypoint,omitempty"`
		TLSMode                    *string `json:"tls_mode,omitempty"`
	}) (map[string]any, error) {
		code, name, restApiUrl, baseDomain, pullPolicy := input.Code, input.Name, input.RestApiUrl, input.BaseDomain, input.InitialComponentPullPolicy
		if code == "" {
			code = "traefik"
		}
		if name == "" {
			name = "Traefik"
		}
		if restApiUrl == "" {
			restApiUrl = "http://localhost:8080"
		}
		if baseDomain == "" {
			baseDomain = "lvh.me"
		}
		if pullPolicy == "" {
			pullPolicy = "missing"
		}
		image := input.InitialComponentImage
		if image == nil {
			value := "traefik:3.6"
			image = &value
		}
		gateway, err := c.deps.Gateway.CreateGateway(ctx, c.deps.ActorUserId, gatewaydto.GatewayCreateInput{ProjectId: input.ProjectId, Code: code, Name: name, RestApiUrl: restApiUrl, BaseDomain: baseDomain, InitialComponentImage: image, InitialComponentPullPolicy: pullPolicy, DefaultEntrypoint: input.DefaultEntrypoint, TLSMode: input.TLSMode})
		if err != nil {
			return nil, err
		}
		return writeResult("create_gateway", map[string]string{"gateway_id": gateway.Application.Id, "application_id": gateway.Application.Id}, "POST", "/api/gateway", map[string]any{"gateway": gatewayOutput(gateway)}), nil
	})

	addTool(server, "orbit_provision_gateway", "Ensure one managed traefik Gateway is deployed and its external network is ready.", func(ctx context.Context, input struct {
		ProjectId      string `json:"project_id" jsonschema:"required"`
		InstanceKey    string `json:"instance_key,omitempty"`
		ForceRecreate  bool   `json:"force_recreate,omitempty"`
		TimeoutSeconds *int   `json:"timeout_seconds,omitempty"`
	}) (map[string]any, error) {
		instanceKey := input.InstanceKey
		if instanceKey == "" {
			instanceKey = "default"
		}
		result, err := c.deps.Gateway.ProvisionGateway(ctx, c.deps.ActorUserId, gatewaydto.ProvisionGatewayInput{ProjectId: input.ProjectId, InstanceKey: instanceKey, ForceRecreate: input.ForceRecreate, TimeoutSeconds: input.TimeoutSeconds})
		if err != nil {
			return nil, err
		}
		return provisionGatewayOutput(result), nil
	})

	addTool(server, "orbit_get_gateway", "Read one Gateway and its Application metadata through Orbit.", func(ctx context.Context, input struct {
		GatewayId string `json:"gateway_id" jsonschema:"required"`
	}) (map[string]any, error) {
		gateway, err := c.deps.Gateway.GatewayForUser(ctx, c.deps.ActorUserId, input.GatewayId)
		if err != nil {
			return nil, err
		}
		return map[string]any{"gateway": gatewayOutput(gateway)}, nil
	})

	addTool(server, "orbit_update_gateway", "Update an Orbit-managed Gateway configuration through Orbit.", func(ctx context.Context, input struct {
		GatewayId         string  `json:"gateway_id" jsonschema:"required"`
		Name              *string `json:"name,omitempty"`
		RestApiUrl        *string `json:"rest_api_url,omitempty"`
		BaseDomain        *string `json:"base_domain,omitempty"`
		DefaultEntrypoint *string `json:"default_entrypoint,omitempty"`
		TLSMode           *string `json:"tls_mode,omitempty"`
	}) (map[string]any, error) {
		if input.Name == nil && input.RestApiUrl == nil && input.BaseDomain == nil && input.DefaultEntrypoint == nil && input.TLSMode == nil {
			return nil, apperror.New(apperror.KindValidation, "at least one Gateway field must be supplied")
		}
		gateway, err := c.deps.Gateway.UpdateGateway(ctx, c.deps.ActorUserId, input.GatewayId, gatewaydto.GatewayUpdateInput{Name: input.Name, RestApiUrl: input.RestApiUrl, BaseDomain: input.BaseDomain, DefaultEntrypoint: input.DefaultEntrypoint, TLSMode: input.TLSMode})
		if err != nil {
			return nil, err
		}
		return writeResult("update_gateway", map[string]string{"gateway_id": input.GatewayId, "application_id": input.GatewayId}, "PUT", "/api/gateway/"+input.GatewayId, map[string]any{"gateway": gatewayOutput(gateway)}), nil
	})

	addTool(server, "orbit_create_application", "Create a standard Application through Orbit and return its initial draft Version.", func(ctx context.Context, input struct {
		ProjectId string `json:"project_id" jsonschema:"required"`
		Name      string `json:"name" jsonschema:"required"`
		Code      string `json:"code" jsonschema:"required"`
		Kind      string `json:"kind,omitempty"`
	}) (map[string]any, error) {
		kind := input.Kind
		if kind == "" {
			kind = "standard"
		}
		if kind != "standard" {
			return nil, apperror.New(apperror.KindValidation, "MCP creation only supports kind=standard")
		}
		app, err := c.deps.Application.CreateApplication(ctx, c.deps.ActorUserId, applicationdto.ApplicationCreateInput{ProjectId: input.ProjectId, Name: input.Name, Code: input.Code, Kind: kind})
		if err != nil {
			return nil, err
		}
		versions, err := c.deps.Application.ListVersions(ctx, c.deps.ActorUserId, app.Id)
		if err != nil {
			return nil, err
		}
		var initial *map[string]any
		for _, version := range versions {
			if version.Version.Status == "unpublished" {
				output := versionOutput(version)
				initial = &output
				break
			}
		}
		ids := map[string]string{"application_id": app.Id}
		if initial != nil {
			ids["initial_version_id"] = (*initial)["id"].(string)
		}
		data := map[string]any{"application": applicationOutput(app), "initial_version": initial}
		return writeResult("create_application", ids, "POST", "/api/application", data), nil
	})

	addTool(server, "orbit_get_application", "Read one Orbit Application, including its current service summary.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
	}) (map[string]any, error) {
		app, err := c.deps.Application.ApplicationForUser(ctx, c.deps.ActorUserId, input.ApplicationId)
		if err != nil {
			return nil, err
		}
		return map[string]any{"application": applicationOutput(app)}, nil
	})

	addTool(server, "orbit_delete_application", "Delete an Orbit Application after its Services and Versions have been removed.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
	}) (map[string]any, error) {
		if err := c.deps.Deployment.DeleteApplication(ctx, c.deps.ActorUserId, input.ApplicationId); err != nil {
			return nil, err
		}
		return writeResult("delete_application", map[string]string{"application_id": input.ApplicationId}, "DELETE", "/api/application/"+input.ApplicationId, nil), nil
	})

	addTool(server, "orbit_list_versions", "List Versions for an Orbit Application.", func(ctx context.Context, input struct {
		ApplicationId string `json:"application_id" jsonschema:"required"`
	}) (map[string]any, error) {
		versions, err := c.deps.Application.ListVersions(ctx, c.deps.ActorUserId, input.ApplicationId)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(versions))
		for _, version := range versions {
			items = append(items, versionOutput(version))
		}
		return map[string]any{"application_id": input.ApplicationId, "versions": items}, nil
	})

	addTool(server, "orbit_get_version", "Read a Version with its Components.", func(ctx context.Context, input struct {
		VersionId string `json:"version_id" jsonschema:"required"`
	}) (map[string]any, error) {
		version, err := c.deps.Application.VersionForUser(ctx, c.deps.ActorUserId, input.VersionId)
		if err != nil {
			return nil, err
		}
		return map[string]any{"version": versionOutput(version)}, nil
	})

	addTool(server, "orbit_create_version_component", "Add a Component with basic configuration to an unpublished Version.", func(ctx context.Context, input struct {
		VersionId string                             `json:"version_id" jsonschema:"required"`
		Component *applicationv1.VersionComponentReq `json:"component" jsonschema:"required"`
	}) (map[string]any, error) {
		componentInput, err := componentInput(input.Component)
		if err != nil {
			return nil, apperror.Wrap(apperror.KindValidation, "invalid component command", err)
		}
		component, err := c.deps.Application.CreateVersionComponent(ctx, c.deps.ActorUserId, input.VersionId, componentInput)
		if err != nil {
			return nil, err
		}
		return writeResult("create_version_component", map[string]string{"version_id": input.VersionId, "component_id": component.Id}, "POST", "/api/version/"+input.VersionId+"/component", map[string]any{"component": componentOutput(component)}), nil
	})

	addTool(server, "orbit_create_version", "Create a Version using a complete Component collection.", func(ctx context.Context, input struct {
		ApplicationId string                               `json:"application_id" jsonschema:"required"`
		Label         string                               `json:"label" jsonschema:"required"`
		Components    []*applicationv1.VersionComponentReq `json:"components" jsonschema:"required"`
		Note          *string                              `json:"note,omitempty"`
	}) (map[string]any, error) {
		components := make([]applicationdto.VersionComponentInput, 0, len(input.Components))
		for _, item := range input.Components {
			component, err := componentInput(item)
			if err != nil {
				return nil, apperror.Wrap(apperror.KindValidation, "invalid component command", err)
			}
			components = append(components, component)
		}
		version, err := c.deps.Application.CreateVersion(ctx, c.deps.ActorUserId, applicationdto.VersionCreateInput{ApplicationId: input.ApplicationId, Label: input.Label, Note: input.Note, Components: components})
		if err != nil {
			return nil, err
		}
		return writeResult("create_version", map[string]string{"application_id": input.ApplicationId, "version_id": version.Version.Id}, "POST", "/api/application/"+input.ApplicationId+"/version", map[string]any{"version": versionOutput(version)}), nil
	})

	addTool(server, "orbit_update_version", "Update Version metadata; Components have dedicated tools.", func(ctx context.Context, input struct {
		VersionId string  `json:"version_id" jsonschema:"required"`
		Label     *string `json:"label,omitempty"`
		Note      *string `json:"note,omitempty"`
	}) (map[string]any, error) {
		if input.Label == nil && input.Note == nil {
			return nil, apperror.New(apperror.KindValidation, "at least one Version field must be supplied")
		}
		version, err := c.deps.Application.UpdateVersion(ctx, c.deps.ActorUserId, input.VersionId, applicationdto.VersionUpdateInput{Label: input.Label, Note: input.Note})
		if err != nil {
			return nil, err
		}
		return writeResult("update_version", map[string]string{"version_id": input.VersionId}, "PUT", "/api/version/"+input.VersionId, map[string]any{"version": versionOutput(version)}), nil
	})

	c.registerVersionComponentTools(server)

	addTool(server, "orbit_publish_version", "Mark a Version published through Orbit; publication does not lock later edits or deletion.", func(ctx context.Context, input struct {
		VersionId string `json:"version_id" jsonschema:"required"`
	}) (map[string]any, error) {
		version, err := c.deps.Application.PublishVersion(ctx, c.deps.ActorUserId, input.VersionId)
		if err != nil {
			return nil, err
		}
		return writeResult("publish_version", map[string]string{"version_id": input.VersionId}, "POST", "/api/version/"+input.VersionId+"/publish", map[string]any{"version": versionOutput(version)}), nil
	})

	addTool(server, "orbit_delete_version", "Delete an unreferenced Version through Orbit regardless of its published marker.", func(ctx context.Context, input struct {
		VersionId string `json:"version_id" jsonschema:"required"`
	}) (map[string]any, error) {
		if err := c.deps.Application.DeleteVersion(ctx, c.deps.ActorUserId, input.VersionId); err != nil {
			return nil, err
		}
		return writeResult("delete_version", map[string]string{"version_id": input.VersionId}, "DELETE", "/api/version/"+input.VersionId, nil), nil
	})

	addTool(server, "orbit_preview_service", "Render a saved Service configuration without deploying it.", func(ctx context.Context, input struct {
		ServiceId string `json:"service_id" jsonschema:"required"`
	}) (map[string]any, error) {
		preview, err := c.deps.Deployment.PreviewService(ctx, c.deps.ActorUserId, input.ServiceId)
		if err != nil {
			return nil, err
		}
		return map[string]any{"service_id": input.ServiceId, "preview": preview}, nil
	})

	c.registerServiceTools(server)
	c.registerDeploymentTools(server)
	c.registerRuntimeTools(server)
	c.registerVerificationTools(server)
}

func writeResult(operation string, resourceIds map[string]string, method, path string, data map[string]any) map[string]any {
	result := map[string]any{"operation": operation, "resource_ids": resourceIds, "steps": []string{"Orbit application use case completed"}, "request_summary": map[string]any{"method": method, "path": path}}
	for key, value := range data {
		result[key] = value
	}
	return result
}

func (c *core) updateComponentAdvanced(ctx context.Context, versionId, componentId string, resources *model.VersionComponentResources, tmpfs []model.VersionComponentTmpfs, ulimits []model.VersionComponentUlimit) (model.VersionComponent, error) {
	return c.deps.Application.UpdateVersionComponentAdvanced(ctx, c.deps.ActorUserId, versionId, componentId, applicationdto.VersionComponentAdvancedUpdateInput{Resources: resources, Tmpfs: tmpfs, Ulimits: ulimits})
}

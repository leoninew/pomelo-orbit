package delivery

import (
	"context"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerRouteTools(server *mcp.Server) {
	addTool(server, "orbit_list_routes", "List custom HTTP and TCP Routes in the selected Project. Read a selected Route with orbit_get_route before changing it.", func(ctx context.Context, input struct {
		Page    int    `json:"page,omitempty"`
		PerPage int    `json:"per_page,omitempty"`
		Search  string `json:"search,omitempty"`
	}) (map[string]any, error) {
		projectId, _, err := c.requireReadyEnvironment(ctx)
		if err != nil {
			return nil, err
		}
		page, perPage := input.Page, input.PerPage
		if page == 0 {
			page = 1
		}
		if perPage == 0 {
			perPage = 100
		}
		routes, err := c.deps.Route.ListRoutes(ctx, c.deps.ActorUserId, projectId, page, perPage, input.Search)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(routes.Items))
		for _, route := range routes.Items {
			items = append(items, routeOutput(route))
		}
		return map[string]any{"routes": items, "total": routes.Total, "page": routes.Page, "per_page": routes.PerPage}, nil
	})

	addTool(server, "orbit_get_route", "Read one custom HTTP or TCP Route before editing, enabling, or disabling it.", func(ctx context.Context, input struct {
		RouteId string `json:"route_id" jsonschema:"required"`
	}) (map[string]any, error) {
		route, err := c.routeInScope(ctx, input.RouteId)
		if err != nil {
			return nil, err
		}
		return map[string]any{"route": routeOutput(route)}, nil
	})

	addTool(server, "orbit_create_route", "Create a custom Route using the Route form fields. HTTP accepts an optional path_prefix (default /) and requires either a managed HTTP target (service_id, component_name, endpoint_protocol, endpoint_container_port) or custom target_url. TCP requires a managed TCP target and listen_port; it cannot use path_prefix or target_url.", func(ctx context.Context, input struct {
		Name                  string `json:"name" jsonschema:"required"`
		Protocol              string `json:"protocol" jsonschema:"required"`
		Domain                string `json:"domain" jsonschema:"required"`
		PathPrefix            string `json:"path_prefix,omitempty"`
		TargetUrl             string `json:"target_url,omitempty"`
		ListenPort            *int   `json:"listen_port,omitempty"`
		ServiceId             string `json:"service_id,omitempty"`
		ComponentName         string `json:"component_name,omitempty"`
		EndpointProtocol      string `json:"endpoint_protocol,omitempty"`
		EndpointContainerPort *int   `json:"endpoint_container_port,omitempty"`
		Enabled               bool   `json:"enabled,omitempty"`
	}) (map[string]any, error) {
		projectId, _, err := c.requireReadyEnvironment(ctx)
		if err != nil {
			return nil, err
		}
		route, err := c.deps.Route.CreateRoute(ctx, c.deps.ActorUserId, projectId, routedto.RouteCreateInput{
			Name: input.Name, Protocol: input.Protocol, Domain: input.Domain, PathPrefix: input.PathPrefix,
			TargetUrl: input.TargetUrl, ListenPort: input.ListenPort, ServiceId: input.ServiceId,
			ComponentName: input.ComponentName, EndpointProtocol: input.EndpointProtocol,
			EndpointContainerPort: input.EndpointContainerPort, Enabled: input.Enabled,
		})
		if err != nil {
			return nil, err
		}
		return writeResult("create_route", map[string]string{"route_id": route.Id}, "POST", "/api/route", map[string]any{"route": routeOutput(route)}), nil
	})

	addTool(server, "orbit_update_route", "Update a custom Route with the Route form fields. Supply only fields that change. To change an HTTP Route from a managed target to custom target_url, set service_id, component_name, and endpoint_protocol to empty strings; to switch protocol, provide the required target fields for the destination protocol.", func(ctx context.Context, input struct {
		RouteId               string  `json:"route_id" jsonschema:"required"`
		Name                  *string `json:"name,omitempty"`
		Protocol              *string `json:"protocol,omitempty"`
		Domain                *string `json:"domain,omitempty"`
		PathPrefix            *string `json:"path_prefix,omitempty"`
		TargetUrl             *string `json:"target_url,omitempty"`
		ListenPort            *int    `json:"listen_port,omitempty"`
		ServiceId             *string `json:"service_id,omitempty"`
		ComponentName         *string `json:"component_name,omitempty"`
		EndpointProtocol      *string `json:"endpoint_protocol,omitempty"`
		EndpointContainerPort *int    `json:"endpoint_container_port,omitempty"`
		Enabled               *bool   `json:"enabled,omitempty"`
	}) (map[string]any, error) {
		projectId, _, err := c.requireReadyEnvironment(ctx)
		if err != nil {
			return nil, err
		}
		route, err := c.deps.Route.UpdateRoute(ctx, c.deps.ActorUserId, projectId, input.RouteId, routedto.RouteUpdateInput{
			Name: input.Name, Protocol: input.Protocol, Domain: input.Domain, PathPrefix: input.PathPrefix,
			TargetUrl: input.TargetUrl, ListenPort: input.ListenPort, ServiceId: input.ServiceId,
			ComponentName: input.ComponentName, EndpointProtocol: input.EndpointProtocol,
			EndpointContainerPort: input.EndpointContainerPort, Enabled: input.Enabled,
		})
		if err != nil {
			return nil, err
		}
		return writeResult("update_route", map[string]string{"route_id": route.Id}, "PUT", "/api/route/"+route.Id, map[string]any{"route": routeOutput(route)}), nil
	})

	addTool(server, "orbit_enable_route", "Enable one custom Route in business data. Explicitly preview and confirm Route sync to publish its owned file. For TCP, the target Gateway must already expose the selected listen_port.", func(ctx context.Context, input struct {
		RouteId string `json:"route_id" jsonschema:"required"`
	}) (map[string]any, error) {
		projectId, _, err := c.requireReadyEnvironment(ctx)
		if err != nil {
			return nil, err
		}
		route, err := c.deps.Route.EnableRoute(ctx, c.deps.ActorUserId, projectId, input.RouteId)
		if err != nil {
			return nil, err
		}
		return writeResult("enable_route", map[string]string{"route_id": route.Id}, "POST", "/api/route/"+route.Id+"/enable", map[string]any{"route": routeOutput(route)}), nil
	})

	addTool(server, "orbit_disable_route", "Disable one custom Route in business data. Explicitly preview and confirm Route sync to withdraw its owned file.", func(ctx context.Context, input struct {
		RouteId string `json:"route_id" jsonschema:"required"`
	}) (map[string]any, error) {
		projectId, _, err := c.requireReadyEnvironment(ctx)
		if err != nil {
			return nil, err
		}
		route, err := c.deps.Route.DisableRoute(ctx, c.deps.ActorUserId, projectId, input.RouteId)
		if err != nil {
			return nil, err
		}
		return writeResult("disable_route", map[string]string{"route_id": route.Id}, "POST", "/api/route/"+route.Id+"/disable", map[string]any{"route": routeOutput(route)}), nil
	})

	addTool(server, "orbit_preview_route_sync", "Preview a publish/withdraw/skip list for selected Route files or the explicit project scope. Review the list and pass the frozen route_ids, changes, business_hash and publication_hash to orbit_confirm_route_sync after approval. Unknown files remain untouched.", func(ctx context.Context, input struct {
		Scope    string                 `json:"scope" jsonschema:"required"`
		RouteIds []string               `json:"route_ids,omitempty"`
		Changes  []routeSyncChangeInput `json:"changes,omitempty"`
	}) (map[string]any, error) {
		projectId, _, err := c.requireReadyEnvironment(ctx)
		if err != nil {
			return nil, err
		}
		preview, err := c.deps.Route.PreviewRouteSync(ctx, c.deps.ActorUserId, projectId, routedto.RouteSyncPreviewInput{Scope: input.Scope, RouteIds: input.RouteIds, Changes: routeSyncChangesInput(input.Changes)})
		if err != nil {
			return nil, err
		}
		return routeSyncPreviewOutput(projectId, preview), nil
	})

	addTool(server, "orbit_confirm_route_sync", "Confirm reviewed Route files in the frozen preview order. Failed items are reported and subsequent items continue. Check the overall code and per-Route configuration and certificate results.", func(ctx context.Context, input struct {
		RouteIds        []string               `json:"route_ids" jsonschema:"required"`
		PublicationHash string                 `json:"publication_hash" jsonschema:"required"`
		Changes         []routeSyncChangeInput `json:"changes,omitempty"`
		BusinessHash    string                 `json:"business_hash" jsonschema:"required"`
	}) (map[string]any, error) {
		projectId, _, err := c.requireReadyEnvironment(ctx)
		if err != nil {
			return nil, err
		}
		result, err := c.deps.Route.ConfirmRouteSync(ctx, c.deps.ActorUserId, projectId, routedto.RouteSyncConfirmInput{
			RouteIds:        input.RouteIds,
			PublicationHash: input.PublicationHash,
			Changes:         routeSyncChangesInput(input.Changes),
			BusinessHash:    input.BusinessHash,
		})
		if err != nil {
			return nil, err
		}
		return writeResult("confirm_route_sync", map[string]string{"project_id": projectId}, "POST", "/api/route/sync/confirm", map[string]any{"code": result.Code, "results": result.Results}), nil
	})
}

type routeSyncChangeInput struct {
	RouteId string `json:"route_id" jsonschema:"required"`
	Enabled *bool  `json:"enabled" jsonschema:"required"`
}

func routeSyncChangesInput(items []routeSyncChangeInput) []routedto.RouteSyncChange {
	changes := make([]routedto.RouteSyncChange, 0, len(items))
	for _, item := range items {
		changes = append(changes, routedto.RouteSyncChange{RouteId: item.RouteId, Enabled: item.Enabled})
	}
	return changes
}

func routeSyncPreviewOutput(projectId string, preview routedto.RouteSyncPreview) map[string]any {
	return map[string]any{
		"route_ids": preview.RouteIds, "publication_hash": preview.PublicationHash,
		"project_id": projectId, "business_hash": preview.BusinessHash, "items": preview.Items,
	}
}

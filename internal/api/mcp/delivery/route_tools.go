package delivery

import (
	"context"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func (c *core) registerRouteTools(server *mcp.Server) {
	addTool(server, "orbit_list_routes", "List custom HTTP and TCP Routes in a Project. Read a selected Route with orbit_get_route before changing it.", func(ctx context.Context, input struct {
		ProjectId string `json:"project_id" jsonschema:"required"`
		Page      int    `json:"page,omitempty"`
		PerPage   int    `json:"per_page,omitempty"`
		Search    string `json:"search,omitempty"`
	}) (map[string]any, error) {
		page, perPage := input.Page, input.PerPage
		if page == 0 {
			page = 1
		}
		if perPage == 0 {
			perPage = 100
		}
		routes, err := c.deps.Route.ListRoutes(ctx, c.deps.ActorUserId, input.ProjectId, page, perPage, input.Search)
		if err != nil {
			return nil, err
		}
		items := make([]map[string]any, 0, len(routes.Items))
		for _, route := range routes.Items {
			items = append(items, routeOutput(route))
		}
		return map[string]any{"project_id": input.ProjectId, "routes": items, "total": routes.Total, "page": routes.Page, "per_page": routes.PerPage}, nil
	})

	addTool(server, "orbit_get_route", "Read one custom HTTP or TCP Route before editing, enabling, or disabling it.", func(ctx context.Context, input struct {
		RouteId string `json:"route_id" jsonschema:"required"`
	}) (map[string]any, error) {
		route, err := c.deps.Route.RouteForUser(ctx, c.deps.ActorUserId, input.RouteId)
		if err != nil {
			return nil, err
		}
		return map[string]any{"route": routeOutput(route)}, nil
	})

	addTool(server, "orbit_create_route", "Create a custom Route using the Route form fields. HTTP accepts an optional path_prefix (default /) and requires either a managed HTTP target (service_id, component_name, endpoint_protocol, endpoint_container_port) or custom target_url. TCP requires a managed TCP target and listen_port; it cannot use path_prefix or target_url.", func(ctx context.Context, input struct {
		ProjectId             string `json:"project_id" jsonschema:"required"`
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
		route, err := c.deps.Route.CreateRoute(ctx, c.deps.ActorUserId, input.ProjectId, routedto.RouteCreateInput{
			Name: input.Name, Protocol: input.Protocol, Domain: input.Domain, PathPrefix: input.PathPrefix,
			TargetUrl: input.TargetUrl, ListenPort: input.ListenPort, ServiceId: input.ServiceId,
			ComponentName: input.ComponentName, EndpointProtocol: input.EndpointProtocol,
			EndpointContainerPort: input.EndpointContainerPort, Enabled: input.Enabled,
		})
		if err != nil {
			return nil, err
		}
		return writeResult("create_route", map[string]string{"project_id": input.ProjectId, "route_id": route.Id}, "POST", "/api/route", map[string]any{"route": routeOutput(route)}), nil
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
		route, err := c.deps.Route.UpdateRoute(ctx, c.deps.ActorUserId, input.RouteId, routedto.RouteUpdateInput{
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

	addTool(server, "orbit_enable_route", "Enable one custom Route in business data. The complete Route snapshot is published through the Route sync flow. For TCP, the target Gateway must already expose the selected listen_port.", func(ctx context.Context, input struct {
		RouteId string `json:"route_id" jsonschema:"required"`
	}) (map[string]any, error) {
		route, err := c.deps.Route.EnableRoute(ctx, c.deps.ActorUserId, input.RouteId)
		if err != nil {
			return nil, err
		}
		return writeResult("enable_route", map[string]string{"route_id": route.Id}, "POST", "/api/route/"+route.Id+"/enable", map[string]any{"route": routeOutput(route)}), nil
	})

	addTool(server, "orbit_disable_route", "Disable one custom Route in business data. The complete Route snapshot is published through the Route sync flow.", func(ctx context.Context, input struct {
		RouteId string `json:"route_id" jsonschema:"required"`
	}) (map[string]any, error) {
		route, err := c.deps.Route.DisableRoute(ctx, c.deps.ActorUserId, input.RouteId)
		if err != nil {
			return nil, err
		}
		return writeResult("disable_route", map[string]string{"route_id": route.Id}, "POST", "/api/route/"+route.Id+"/disable", map[string]any{"route": routeOutput(route)}), nil
	})

	addTool(server, "orbit_preview_route_sync", "Preview the complete Route snapshot before publication. Review the returned differences and pass the unchanged changes and hashes to orbit_confirm_route_sync only after approval.", func(ctx context.Context, input struct {
		ProjectId string                 `json:"project_id" jsonschema:"required"`
		Changes   []routeSyncChangeInput `json:"changes,omitempty"`
	}) (map[string]any, error) {
		preview, err := c.deps.Route.PreviewRouteSync(ctx, c.deps.ActorUserId, input.ProjectId, routeSyncChangesInput(input.Changes))
		if err != nil {
			return nil, err
		}
		return routeSyncPreviewOutput(input.ProjectId, preview), nil
	})

	addTool(server, "orbit_confirm_route_sync", "Confirm a previously reviewed Route sync. This applies the pending enable/disable changes and replaces the complete Traefik REST snapshot only when both preview hashes still match.", func(ctx context.Context, input struct {
		ProjectId    string                 `json:"project_id" jsonschema:"required"`
		Changes      []routeSyncChangeInput `json:"changes,omitempty"`
		BusinessHash string                 `json:"business_hash" jsonschema:"required"`
		TraefikHash  string                 `json:"traefik_hash" jsonschema:"required"`
	}) (map[string]any, error) {
		if err := c.deps.Route.ConfirmRouteSync(ctx, c.deps.ActorUserId, input.ProjectId, routedto.RouteSyncConfirmInput{
			Changes:      routeSyncChangesInput(input.Changes),
			BusinessHash: input.BusinessHash,
			TraefikHash:  input.TraefikHash,
		}); err != nil {
			return nil, err
		}
		return writeResult("confirm_route_sync", map[string]string{"project_id": input.ProjectId}, "POST", "/api/route/sync/confirm", map[string]any{"message": "Routes synced successfully"}), nil
	})
}

type routeSyncChangeInput struct {
	RouteId string `json:"route_id" jsonschema:"required"`
	Enabled bool   `json:"enabled"`
}

func routeSyncChangesInput(items []routeSyncChangeInput) []routedto.RouteSyncChange {
	changes := make([]routedto.RouteSyncChange, 0, len(items))
	for _, item := range items {
		changes = append(changes, routedto.RouteSyncChange{RouteId: item.RouteId, Enabled: item.Enabled})
	}
	return changes
}

func routeSyncPreviewOutput(projectId string, preview routedto.RouteSyncPreview) map[string]any {
	differences := make([]map[string]any, 0, len(preview.Differences))
	for _, difference := range preview.Differences {
		differences = append(differences, map[string]any{
			"action": difference.Action, "route_name": difference.RouteName, "field": difference.Field,
			"business_value": difference.BusinessValue, "traefik_value": difference.TraefikValue,
		})
	}
	return map[string]any{
		"project_id": projectId, "business_hash": preview.BusinessHash, "traefik_hash": preview.TraefikHash,
		"matched": preview.Matched, "differences": differences,
	}
}

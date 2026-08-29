package delivery

import (
	"context"
	"testing"

	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestRouteToolsMapCustomRouteFormFields(t *testing.T) {
	routeService := &routeToolService{}
	server, err := NewServer(Dependencies{ActorUserId: "actor", Route: routeService})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	byName := make(map[string]*mcp.Tool, len(tools.Tools))
	for _, tool := range tools.Tools {
		byName[tool.Name] = tool
	}
	for _, name := range []string{"orbit_list_routes", "orbit_get_route", "orbit_create_route", "orbit_update_route", "orbit_enable_route", "orbit_disable_route", "orbit_preview_route_sync", "orbit_confirm_route_sync"} {
		if byName[name] == nil {
			t.Errorf("missing route tool %q", name)
		}
	}

	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_create_route", Arguments: map[string]any{
		"project_id": "project-1", "name": "api-route", "protocol": "http", "domain": "api.example.test",
		"path_prefix": "/api", "target_url": "https://origin.example.test:8443", "enabled": false,
	}})
	if err != nil {
		t.Fatalf("CallTool(create route) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(create route) returned tool error: %#v", result.Content)
	}
	if routeService.createUserId != "actor" || routeService.createProjectId != "project-1" {
		t.Fatalf("create actor/project = %q/%q", routeService.createUserId, routeService.createProjectId)
	}
	created := routeService.createInput
	if created.Protocol != "http" || created.PathPrefix != "/api" || created.TargetUrl != "https://origin.example.test:8443" || created.ServiceId != "" || created.Enabled {
		t.Fatalf("create input = %#v", created)
	}
	output := structuredOutput(t, result)
	route, ok := output["route"].(map[string]any)
	if !ok || route["target_url"] != "https://origin.example.test:8443" {
		t.Fatalf("create output route = %#v", output["route"])
	}

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_update_route", Arguments: map[string]any{
		"route_id": "route-1", "protocol": "tcp", "domain": "redis.example.test", "path_prefix": "", "target_url": "",
		"listen_port": 16379, "service_id": "service-1", "component_name": "redis", "endpoint_protocol": "tcp",
		"endpoint_container_port": 6379, "enabled": true,
	}})
	if err != nil {
		t.Fatalf("CallTool(update route) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(update route) returned tool error: %#v", result.Content)
	}
	updated := routeService.updateInput
	if updated.Protocol == nil || *updated.Protocol != "tcp" || updated.ListenPort == nil || *updated.ListenPort != 16379 || updated.ServiceId == nil || *updated.ServiceId != "service-1" || updated.ComponentName == nil || *updated.ComponentName != "redis" || updated.EndpointProtocol == nil || *updated.EndpointProtocol != "tcp" || updated.EndpointContainerPort == nil || *updated.EndpointContainerPort != 6379 || updated.Enabled == nil || !*updated.Enabled {
		t.Fatalf("update input = %#v", updated)
	}
	if updated.PathPrefix == nil || *updated.PathPrefix != "" || updated.TargetUrl == nil || *updated.TargetUrl != "" {
		t.Fatalf("update clearing fields = %#v", updated)
	}

	for _, toolName := range []string{"orbit_enable_route", "orbit_disable_route"} {
		result, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: toolName, Arguments: map[string]any{"route_id": "route-1"}})
		if err != nil {
			t.Fatalf("CallTool(%s) error = %v", toolName, err)
		}
		if result.IsError {
			t.Fatalf("CallTool(%s) returned tool error: %#v", toolName, result.Content)
		}
	}
	if routeService.enableRouteId != "route-1" || routeService.disableRouteId != "route-1" {
		t.Fatalf("enable/disable IDs = %q/%q", routeService.enableRouteId, routeService.disableRouteId)
	}

	syncChanges := []any{map[string]any{"route_id": "route-1", "enabled": true}}
	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_preview_route_sync", Arguments: map[string]any{
		"project_id": "project-1", "changes": syncChanges,
	}})
	if err != nil {
		t.Fatalf("CallTool(preview route sync) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(preview route sync) returned tool error: %#v", result.Content)
	}
	if routeService.previewUserId != "actor" || routeService.previewProjectId != "project-1" || len(routeService.previewChanges) != 1 || routeService.previewChanges[0] != (routedto.RouteSyncChange{RouteId: "route-1", Enabled: true}) {
		t.Fatalf("preview input = %q/%q/%#v", routeService.previewUserId, routeService.previewProjectId, routeService.previewChanges)
	}
	previewOutput := structuredOutput(t, result)
	if previewOutput["business_hash"] != "business-hash" || previewOutput["traefik_hash"] != "traefik-hash" || previewOutput["matched"] != false {
		t.Fatalf("preview output = %#v", previewOutput)
	}

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_confirm_route_sync", Arguments: map[string]any{
		"project_id": "project-1", "changes": syncChanges, "business_hash": "business-hash", "traefik_hash": "traefik-hash",
	}})
	if err != nil {
		t.Fatalf("CallTool(confirm route sync) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(confirm route sync) returned tool error: %#v", result.Content)
	}
	if routeService.confirmUserId != "actor" || routeService.confirmProjectId != "project-1" || routeService.confirmInput.BusinessHash != "business-hash" || routeService.confirmInput.TraefikHash != "traefik-hash" || len(routeService.confirmInput.Changes) != 1 || routeService.confirmInput.Changes[0] != (routedto.RouteSyncChange{RouteId: "route-1", Enabled: true}) {
		t.Fatalf("confirm input = %q/%q/%#v", routeService.confirmUserId, routeService.confirmProjectId, routeService.confirmInput)
	}

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_list_routes", Arguments: map[string]any{"project_id": "project-1"}})
	if err != nil {
		t.Fatalf("CallTool(list routes) error = %v", err)
	}
	if result.IsError || routeService.listProjectId != "project-1" || routeService.listPage != 1 || routeService.listPerPage != 100 {
		t.Fatalf("list result/input = %#v / %q / %d / %d", result, routeService.listProjectId, routeService.listPage, routeService.listPerPage)
	}
	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_get_route", Arguments: map[string]any{"route_id": "route-1"}})
	if err != nil {
		t.Fatalf("CallTool(get route) error = %v", err)
	}
	if result.IsError || routeService.getRouteId != "route-1" {
		t.Fatalf("get result/input = %#v / %q", result, routeService.getRouteId)
	}
}

type routeToolService struct {
	RouteService
	createUserId     string
	createProjectId  string
	createInput      routedto.RouteCreateInput
	updateInput      routedto.RouteUpdateInput
	enableRouteId    string
	disableRouteId   string
	listProjectId    string
	listPage         int
	listPerPage      int
	getRouteId       string
	previewUserId    string
	previewProjectId string
	previewChanges   []routedto.RouteSyncChange
	confirmUserId    string
	confirmProjectId string
	confirmInput     routedto.RouteSyncConfirmInput
}

func (s *routeToolService) ListRoutes(_ context.Context, _ string, projectId string, page, perPage int, _ string) (repository.Page[model.Route], error) {
	s.listProjectId, s.listPage, s.listPerPage = projectId, page, perPage
	return repository.Page[model.Route]{Items: []model.Route{s.route("route-1", false)}, Total: 1, Page: page, PerPage: perPage}, nil
}

func (s *routeToolService) CreateRoute(_ context.Context, userId, projectId string, input routedto.RouteCreateInput) (model.Route, error) {
	s.createUserId, s.createProjectId, s.createInput = userId, projectId, input
	route := s.route("route-1", input.Enabled)
	route.Name, route.Protocol, route.Domain, route.PathPrefix, route.TargetUrl = input.Name, input.Protocol, input.Domain, input.PathPrefix, input.TargetUrl
	return route, nil
}

func (s *routeToolService) RouteForUser(_ context.Context, _ string, routeId string) (model.Route, error) {
	s.getRouteId = routeId
	return s.route(routeId, false), nil
}

func (s *routeToolService) UpdateRoute(_ context.Context, _ string, _ string, input routedto.RouteUpdateInput) (model.Route, error) {
	s.updateInput = input
	return s.route("route-1", input.Enabled != nil && *input.Enabled), nil
}

func (s *routeToolService) EnableRoute(_ context.Context, _ string, routeId string) (model.Route, error) {
	s.enableRouteId = routeId
	return s.route(routeId, true), nil
}

func (s *routeToolService) DisableRoute(_ context.Context, _ string, routeId string) (model.Route, error) {
	s.disableRouteId = routeId
	return s.route(routeId, false), nil
}

func (s *routeToolService) PreviewRouteSync(_ context.Context, userId, projectId string, changes []routedto.RouteSyncChange) (routedto.RouteSyncPreview, error) {
	s.previewUserId, s.previewProjectId = userId, projectId
	s.previewChanges = append([]routedto.RouteSyncChange(nil), changes...)
	return routedto.RouteSyncPreview{
		BusinessHash: "business-hash",
		TraefikHash:  "traefik-hash",
		Differences: []routedto.RouteSyncDiff{{
			Action: "added", RouteName: "api-route", Field: "route", BusinessValue: "HTTP Host(`api.example.test`) -> https://origin.example.test:8443",
		}},
	}, nil
}

func (s *routeToolService) ConfirmRouteSync(_ context.Context, userId, projectId string, input routedto.RouteSyncConfirmInput) error {
	s.confirmUserId, s.confirmProjectId = userId, projectId
	s.confirmInput = routedto.RouteSyncConfirmInput{
		Changes:      append([]routedto.RouteSyncChange(nil), input.Changes...),
		BusinessHash: input.BusinessHash,
		TraefikHash:  input.TraefikHash,
	}
	return nil
}

func (s *routeToolService) route(routeId string, enabled bool) model.Route {
	projectId := "project-1"
	return model.Route{Id: routeId, ProjectId: &projectId, Name: "api-route", Protocol: "http", Domain: "api.example.test", PathPrefix: "/", TargetUrl: "https://origin.example.test:8443", Enabled: enabled, CertType: "manual"}
}

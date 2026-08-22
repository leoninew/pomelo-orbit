package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolListIncludesDeliverySurfaceAndFlatCollectionSchemas(t *testing.T) {
	server, err := NewServer(Dependencies{ActorUserId: "actor"})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) != 54 {
		t.Fatalf("tool count = %d, want 54", len(tools.Tools))
	}

	byName := make(map[string]*mcp.Tool, len(tools.Tools))
	for _, tool := range tools.Tools {
		byName[tool.Name] = tool
	}
	for _, name := range deliveryToolNames {
		if byName[name] == nil {
			t.Errorf("missing tool %q", name)
		}
	}
	for _, schema := range []struct {
		tool     string
		property string
	}{
		{"orbit_update_version_component_endpoints", "endpoints"},
		{"orbit_update_version_component_env", "env"},
		{"orbit_update_version_component_mounts", "mounts"},
		{"orbit_update_version_component_dependencies", "dependencies"},
		{"orbit_update_version_component_devices", "devices"},
		{"orbit_update_version_component_advanced", "tmpfs"},
		{"orbit_update_version_component_advanced", "ulimits"},
		{"orbit_update_version_component_tmpfs", "tmpfs"},
		{"orbit_update_version_component_ulimits", "ulimits"},
		{"orbit_update_service_env", "env"},
	} {
		assertFlatArrayProperty(t, byName[schema.tool], schema.property)
	}
	assertFlatObjectProperty(t, byName["orbit_update_version_component_advanced"], "resources")
	assertFlatObjectProperty(t, byName["orbit_update_version_component_resources"], "resources")
}

func TestApplicationErrorBecomesClassifiedMCPToolError(t *testing.T) {
	server, err := NewServer(Dependencies{ActorUserId: "actor", Application: errorApplicationService{}})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_get_application", Arguments: map[string]any{"application_id": "missing"}})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if !result.IsError {
		t.Fatalf("CallTool() IsError = false, want true")
	}
	if len(result.Content) != 1 {
		t.Fatalf("error content count = %d, want 1", len(result.Content))
	}
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("error content = %T, want *mcp.TextContent", result.Content[0])
	}
	if !strings.Contains(content.Text, "not_found: Application missing not found") {
		t.Fatalf("error text = %q, want classified not_found error", content.Text)
	}
}

func TestServiceCodeMCPContract(t *testing.T) {
	application := &serviceApplicationToolService{application: model.Application{Id: "application-1", Code: "ragflow"}}
	service := &serviceToolService{services: []model.Service{{Id: "service-1", ApplicationId: "application-1", InstanceKey: "default", Code: "ragflow-default", VersionId: "version-1", Status: "stopped"}}}
	server, err := NewServer(Dependencies{ActorUserId: "actor", Application: application, Service: service})
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
	createTool := byName["orbit_create_service"]
	if createTool == nil {
		t.Fatal("create service tool not found")
		return
	}
	var createSchema struct {
		Properties map[string]json.RawMessage `json:"properties"`
		Required   []string                   `json:"required"`
	}
	encodedSchema, err := json.Marshal(createTool.InputSchema)
	if err != nil {
		t.Fatalf("marshal create schema: %v", err)
	}
	if err := json.Unmarshal(encodedSchema, &createSchema); err != nil {
		t.Fatalf("unmarshal create schema: %v", err)
	}
	if _, ok := createSchema.Properties["code"]; ok || containsString(createSchema.Required, "code") {
		t.Fatalf("create schema must derive code: %s", encodedSchema)
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_create_service", Arguments: map[string]any{
		"application_id": "application-1", "version_id": "version-1", "instance_key": "default",
	}})
	if err != nil {
		t.Fatalf("CallTool(create service) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(create service) returned tool error: %#v", result.Content)
	}
	if service.createInput.Code != "ragflow-default" {
		t.Fatalf("create input code = %q", service.createInput.Code)
	}
	created := structuredOutput(t, result)
	createdService, ok := created["service"].(map[string]any)
	if !ok || createdService["code"] != "ragflow-default" {
		t.Fatalf("create output service = %#v", created["service"])
	}

	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_list_application_services", Arguments: map[string]any{"application_id": "application-1"}})
	if err != nil {
		t.Fatalf("CallTool(list services) error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool(list services) returned tool error: %#v", result.Content)
	}
	listed := structuredOutput(t, result)
	services, ok := listed["services"].([]any)
	if !ok || len(services) != 1 {
		t.Fatalf("list output services = %#v", listed["services"])
	}
	listedService, ok := services[0].(map[string]any)
	if !ok || listedService["code"] != "ragflow-default" {
		t.Fatalf("list output service = %#v", services[0])
	}
}

func TestServiceComponentOverlayToolMapsRuntimeAndHostPathFields(t *testing.T) {
	service := &serviceOverlayToolService{}
	server, err := NewServer(Dependencies{ActorUserId: "actor", Service: service})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	hostPath := true
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_update_service_component_overlay", Arguments: map[string]any{
		"service_id":   "service-1",
		"component_id": "component-1",
		"overlay": map[string]any{
			"entrypoint":     "/custom-entrypoint",
			"command":        "--serve",
			"pull_policy":    "always",
			"restart_policy": "no",
			"mounts": []any{map[string]any{
				"target":              "/data",
				"source":              "D:/data",
				"source_is_host_path": hostPath,
				"state":               string(model.ServiceComponentOverlayOverride),
			}},
		},
	}})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool() returned tool error: %#v", result.Content)
	}
	input := service.overlayInput
	if input.Entrypoint == nil || *input.Entrypoint != "/custom-entrypoint" || input.Command == nil || *input.Command != "--serve" {
		t.Fatalf("runtime command input = %#v", input)
	}
	if input.PullPolicy == nil || *input.PullPolicy != "always" || input.RestartPolicy == nil || *input.RestartPolicy != "no" {
		t.Fatalf("runtime policy input = %#v", input)
	}
	if len(input.Mounts) != 1 || input.Mounts[0].SourceIsHostPath == nil || !*input.Mounts[0].SourceIsHostPath {
		t.Fatalf("mount host path input = %#v", input.Mounts)
	}
	output := structuredOutput(t, result)
	component, ok := output["component"].(map[string]any)
	if !ok || component["entrypoint"] != "/custom-entrypoint" || component["command"] != "--serve" || component["pull_policy"] != "always" || component["restart_policy"] != "no" {
		t.Fatalf("component output = %#v", output["component"])
	}
}

func TestActorAuthorizerRunsOnlyForToolCallsAndBindsTheSession(t *testing.T) {
	project := &actorProjectService{}
	authorizations := 0
	server, err := NewServer(Dependencies{
		ActorAuthorizer: func(context.Context) (string, error) {
			authorizations++
			return "current-user", nil
		},
		Project: project,
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	if _, err := session.ListTools(context.Background(), nil); err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if authorizations != 0 {
		t.Fatalf("authorizations after ListTools() = %d, want 0", authorizations)
	}
	for range 2 {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_list_projects"})
		if err != nil {
			t.Fatalf("CallTool() error = %v", err)
		}
		if result.IsError {
			t.Fatalf("CallTool() returned tool error: %#v", result.Content)
		}
	}
	if authorizations != 1 {
		t.Fatalf("authorizations after tool calls = %d, want 1", authorizations)
	}
	if project.actorUserId != "current-user" || project.calls != 2 {
		t.Fatalf("project calls = actor %q, count %d; want current-user, 2", project.actorUserId, project.calls)
	}
}

func TestActorAuthorizationFailureStopsToolExecution(t *testing.T) {
	project := &actorProjectService{}
	server, err := NewServer(Dependencies{
		ActorAuthorizer: func(context.Context) (string, error) {
			return "", errors.New("browser authorization canceled")
		},
		Project: project,
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	result, err := connectInMemory(t, server).CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_list_projects"})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool() IsError = false, want true")
	}
	if project.calls != 0 {
		t.Fatalf("project calls = %d, want 0", project.calls)
	}
}

type actorProjectService struct {
	actorUserId string
	calls       int
}

func (s *actorProjectService) ListByMember(_ context.Context, actorUserId string) ([]model.Project, error) {
	s.actorUserId = actorUserId
	s.calls++
	return []model.Project{}, nil
}

var deliveryToolNames = []string{
	"orbit_list_projects", "orbit_list_applications", "orbit_list_application_services", "orbit_list_gateways",
	"orbit_create_gateway", "orbit_provision_gateway", "orbit_get_gateway", "orbit_update_gateway",
	"orbit_create_application", "orbit_get_application", "orbit_delete_application", "orbit_list_versions", "orbit_get_version",
	"orbit_create_version_component", "orbit_create_version", "orbit_update_version",
	"orbit_update_version_component_basic", "orbit_update_version_component_runtime", "orbit_update_version_component_endpoints",
	"orbit_update_version_component_env", "orbit_update_version_component_mounts", "orbit_update_version_component_dependencies",
	"orbit_update_version_component_devices", "orbit_update_version_component_advanced", "orbit_update_version_component_resources",
	"orbit_update_version_component_tmpfs", "orbit_update_version_component_ulimits", "orbit_publish_version", "orbit_delete_version",
	"orbit_list_routes", "orbit_get_route", "orbit_create_route", "orbit_update_route", "orbit_enable_route", "orbit_disable_route",
	"orbit_preview_service", "orbit_create_service", "orbit_update_service_component_overlay", "orbit_update_service_env",
	"orbit_update_service_basic", "orbit_deploy", "orbit_stop", "orbit_restart", "orbit_deployment_status", "orbit_deployment_logs",
	"orbit_wait_deployment", "runtime_doctor", "runtime_compose_config", "runtime_compose_ps", "runtime_compose_logs",
	"runtime_container_inspect", "runtime_network_inspect", "runtime_http_probe", "verify_deployment",
}

func TestFlatMountToolMapsCollectionToApplicationInput(t *testing.T) {
	application := &mountApplicationService{}
	server, err := NewServer(Dependencies{ActorUserId: "actor", Application: application})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_update_version_component_mounts", Arguments: map[string]any{
		"version_id": "version-1", "component_id": "component-1",
		"mounts": []any{map[string]any{"source_type": "directory", "source": "./data", "target": "/var/lib/app"}},
	}})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool() returned tool error: %#v", result.Content)
	}
	if len(application.mounts) != 1 || application.mounts[0].Target != "/var/lib/app" {
		t.Fatalf("mapped mounts = %#v", application.mounts)
	}
	if application.versionId != "version-1" || application.componentId != "component-1" {
		t.Fatalf("mapped IDs = %q, %q", application.versionId, application.componentId)
	}
}

func TestMountToolDocumentsAndMapsControlledFile(t *testing.T) {
	application := &mountApplicationService{}
	server, err := NewServer(Dependencies{ActorUserId: "actor", Application: application})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	var mountTool *mcp.Tool
	for _, tool := range tools.Tools {
		if tool.Name == "orbit_update_version_component_mounts" {
			mountTool = tool
			break
		}
	}
	if mountTool == nil {
		t.Fatal("mount tool not found")
	}
	for _, term := range []string{"controlled_file", "source_is_host_path", "./", "absolute path", "bare relative", "named_volume", "262144", "four-digit Unix octal", "empty string"} {
		if !strings.Contains(mountTool.Description, term) {
			t.Errorf("mount description does not document %q: %s", term, mountTool.Description)
		}
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_update_version_component_mounts", Arguments: map[string]any{
		"version_id": "version-1", "component_id": "component-1",
		"mounts": []any{map[string]any{"source_type": "controlled_file", "source": "./config/app.env", "target": "/app/.env", "content": "", "mode": "0644", "source_is_host_path": false, "ignore_if_exists": true}},
	}})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool() returned tool error: %#v", result.Content)
	}
	if len(application.mounts) != 1 {
		t.Fatalf("mounts = %#v", application.mounts)
	}
	mount := application.mounts[0]
	if mount.SourceType != "controlled_file" || mount.Source != "./config/app.env" || mount.Content != "" || mount.Mode != "0644" || mount.SourceIsHostPath || !mount.IgnoreIfExists {
		t.Fatalf("controlled file mount = %#v", mount)
	}
	result, err = session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_update_version_component_mounts", Arguments: map[string]any{
		"version_id": "version-1", "component_id": "component-1",
		"mounts": []any{map[string]any{"source_type": "controlled_file", "source": "/etc/orbit/app.env", "target": "/app/.env", "content": "", "mode": "0644", "source_is_host_path": false, "ignore_if_exists": true}},
	}})
	if err != nil {
		t.Fatalf("CallTool() error for absolute controlled file: %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool() returned tool error for absolute controlled file: %#v", result.Content)
	}
	if application.mounts[0].Source != "/etc/orbit/app.env" || application.mounts[0].SourceIsHostPath {
		t.Fatalf("absolute controlled file mount = %#v", application.mounts[0])
	}
}

func TestDeployToolDocumentsServiceBoundVersion(t *testing.T) {
	server, err := NewServer(Dependencies{ActorUserId: "actor"})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	tools, err := connectInMemory(t, server).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	for _, tool := range tools.Tools {
		if tool.Name == "orbit_deploy" {
			if !strings.Contains(tool.Description, "Service's currently selected Version") {
				t.Fatalf("deploy description = %q", tool.Description)
			}
			return
		}
	}
	t.Fatal("deploy tool not found")
}

func TestStreamableHTTPUsesTheSharedToolRegistry(t *testing.T) {
	server, err := NewServer(Dependencies{ActorUserId: "actor"})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	httpServer := httptest.NewServer(mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server { return server }, nil))
	defer httpServer.Close()
	client := mcp.NewClient(&mcp.Implementation{Name: "delivery-http-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), &mcp.StreamableClientTransport{Endpoint: httpServer.URL, DisableStandaloneSSE: true}, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	defer func() { _ = session.Close() }()
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) != 54 {
		t.Fatalf("HTTP tool count = %d, want 54", len(tools.Tools))
	}
}

func connectInMemory(t *testing.T, server *mcp.Server) *mcp.ClientSession {
	t.Helper()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()
	serverSession, err := server.Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server.Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })
	client := mcp.NewClient(&mcp.Implementation{Name: "delivery-test", Version: "1.0.0"}, nil)
	session, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client.Connect() error = %v", err)
	}
	t.Cleanup(func() { _ = session.Close() })
	return session
}

func assertFlatArrayProperty(t *testing.T, tool *mcp.Tool, property string) {
	t.Helper()
	if tool == nil {
		return
	}
	raw, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("marshal input schema: %v", err)
	}
	var schema struct {
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("unmarshal input schema: %v", err)
	}
	field, ok := schema.Properties[property]
	if !ok {
		t.Fatalf("schema property %q missing: %s", property, raw)
	}
	if !schemaTypeIncludes(field["type"], "array") {
		t.Fatalf("schema property %q type = %#v, want array: %s", property, field["type"], raw)
	}
	if properties, _ := field["properties"].(map[string]any); properties != nil {
		if _, nested := properties[property]; nested {
			t.Fatalf("schema property %q retains a nested %q wrapper: %s", property, property, raw)
		}
	}
	if strings.Contains(string(raw), `"`+property+`":{"properties":{"`+property+`"`) {
		t.Fatalf("schema property %q retains a nested %q wrapper: %s", property, property, raw)
	}
}

func assertFlatObjectProperty(t *testing.T, tool *mcp.Tool, property string) {
	t.Helper()
	if tool == nil {
		return
	}
	raw, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("marshal input schema: %v", err)
	}
	var schema struct {
		Properties map[string]map[string]any `json:"properties"`
	}
	if err := json.Unmarshal(raw, &schema); err != nil {
		t.Fatalf("unmarshal input schema: %v", err)
	}
	field, ok := schema.Properties[property]
	if !ok {
		t.Fatalf("schema property %q missing: %s", property, raw)
	}
	if !schemaTypeIncludes(field["type"], "object") {
		t.Fatalf("schema property %q type = %#v, want object: %s", property, field["type"], raw)
	}
	if properties, _ := field["properties"].(map[string]any); properties != nil {
		if _, nested := properties[property]; nested {
			t.Fatalf("schema property %q retains a nested %q wrapper: %s", property, property, raw)
		}
	}
}

func schemaTypeIncludes(value any, expected string) bool {
	switch value := value.(type) {
	case string:
		return value == expected
	case []any:
		for _, item := range value {
			if item == expected {
				return true
			}
		}
	}
	return false
}

func containsString(values []string, expected string) bool {
	for _, value := range values {
		if value == expected {
			return true
		}
	}
	return false
}

func structuredOutput(t *testing.T, result *mcp.CallToolResult) map[string]any {
	t.Helper()
	raw, err := json.Marshal(result.StructuredContent)
	if err != nil {
		t.Fatalf("marshal structured output: %v", err)
	}
	var output map[string]any
	if err := json.Unmarshal(raw, &output); err != nil {
		t.Fatalf("unmarshal structured output: %v", err)
	}
	return output
}

type mountApplicationService struct {
	ApplicationService
	versionId   string
	componentId string
	mounts      []model.VersionComponentMount
}

type serviceToolService struct {
	ServiceService
	createInput servicedto.ServiceCreateInput
	services    []model.Service
}

type serviceApplicationToolService struct {
	ApplicationService
	application model.Application
}

func (s *serviceApplicationToolService) ApplicationForUser(context.Context, string, string) (model.Application, error) {
	return s.application, nil
}

type serviceOverlayToolService struct {
	ServiceService
	overlayInput servicedto.ServiceComponentOverlayInput
}

func (s *serviceToolService) CreateService(_ context.Context, _ string, input servicedto.ServiceCreateInput) (servicedto.ServiceView, error) {
	s.createInput = input
	return servicedto.ServiceView{Service: model.Service{Id: "service-1", ApplicationId: input.ApplicationId, VersionId: input.VersionId, InstanceKey: input.InstanceKey, Code: input.Code, Status: "stopped"}}, nil
}

func (s *serviceToolService) ListServicesByApplication(context.Context, string, string) ([]model.Service, error) {
	return s.services, nil
}

func (s *serviceOverlayToolService) UpdateServiceComponentOverlay(_ context.Context, _ string, _ string, _ string, input servicedto.ServiceComponentOverlayInput) (model.ServiceComponent, error) {
	s.overlayInput = input
	return model.ServiceComponent{
		Id:            "component-1",
		Entrypoint:    []string{"/custom-entrypoint"},
		Command:       []string{"--serve"},
		PullPolicy:    input.PullPolicy,
		RestartPolicy: input.RestartPolicy,
	}, nil
}

type errorApplicationService struct{ ApplicationService }

func (errorApplicationService) ApplicationForUser(context.Context, string, string) (model.Application, error) {
	return model.Application{}, apperror.New(apperror.KindNotFound, "Application missing not found")
}

func (s *mountApplicationService) UpdateVersionComponentMounts(_ context.Context, _ string, versionId, componentId string, input applicationdto.VersionComponentMountsUpdateInput) (model.VersionComponent, error) {
	s.versionId, s.componentId, s.mounts = versionId, componentId, input.Mounts
	return model.VersionComponent{Id: componentId, VersionId: versionId, Mounts: input.Mounts}, nil
}

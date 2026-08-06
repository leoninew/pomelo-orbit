package delivery

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestToolListIncludesPythonSurfaceAndFlatCollectionSchemas(t *testing.T) {
	server, err := NewServer(Dependencies{ActorUserId: "actor"})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	if len(tools.Tools) != 48 {
		t.Fatalf("tool count = %d, want 48", len(tools.Tools))
	}

	byName := make(map[string]*mcp.Tool, len(tools.Tools))
	for _, tool := range tools.Tools {
		byName[tool.Name] = tool
	}
	for _, name := range pythonDeliveryToolNames {
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

var pythonDeliveryToolNames = []string{
	"orbit_list_projects", "orbit_list_applications", "orbit_list_application_services", "orbit_list_gateways",
	"orbit_create_gateway", "orbit_provision_gateway", "orbit_get_gateway", "orbit_update_gateway",
	"orbit_create_application", "orbit_get_application", "orbit_delete_application", "orbit_list_versions", "orbit_get_version",
	"orbit_create_version_component", "orbit_create_version", "orbit_update_version",
	"orbit_update_version_component_basic", "orbit_update_version_component_runtime", "orbit_update_version_component_endpoints",
	"orbit_update_version_component_env", "orbit_update_version_component_mounts", "orbit_update_version_component_dependencies",
	"orbit_update_version_component_devices", "orbit_update_version_component_advanced", "orbit_update_version_component_resources",
	"orbit_update_version_component_tmpfs", "orbit_update_version_component_ulimits", "orbit_publish_version", "orbit_delete_version",
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
		"mounts": []any{map[string]any{"source_type": "directory", "source": "data", "target": "/var/lib/app"}},
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
	if len(tools.Tools) != 48 {
		t.Fatalf("HTTP tool count = %d, want 48", len(tools.Tools))
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

type mountApplicationService struct {
	ApplicationService
	versionId   string
	componentId string
	mounts      []model.VersionComponentMount
}

type errorApplicationService struct{ ApplicationService }

func (errorApplicationService) ApplicationForUser(context.Context, string, string) (model.Application, error) {
	return model.Application{}, apperror.New(apperror.KindNotFound, "Application missing not found")
}

func (s *mountApplicationService) UpdateVersionComponentMounts(_ context.Context, _ string, versionId, componentId string, input applicationdto.VersionComponentMountsUpdateInput) (model.VersionComponent, error) {
	s.versionId, s.componentId, s.mounts = versionId, componentId, input.Mounts
	return model.VersionComponent{Id: componentId, VersionId: versionId, Mounts: input.Mounts}, nil
}

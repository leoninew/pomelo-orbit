package delivery

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
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
	if len(tools.Tools) != 59 {
		t.Fatalf("tool count = %d, want 59", len(tools.Tools))
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

func TestServerReportsPomeloMCPImplementation(t *testing.T) {
	server, err := NewServer(Dependencies{ActorUserId: "actor"})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}

	serverInfo := connectInMemory(t, server).InitializeResult().ServerInfo
	if serverInfo == nil || serverInfo.Name != "pomelo-orbit-mcp" {
		t.Fatalf("InitializeResult().ServerInfo = %#v, want name pomelo-orbit-mcp", serverInfo)
	}
}

func TestProjectEnvironmentToolsUseProjectScopeWithoutInitializationCredentials(t *testing.T) {
	environment := &environmentToolService{environment: environmentdto.View{
		Id: "environment-1", ProjectId: "project-1", Code: "project", State: model.EnvironmentStateActive,
		TargetType: model.EnvironmentTargetTypeSSH, TargetRevision: 3,
		SSH: &environmentdto.SSHTargetView{
			Platform: model.EnvironmentPlatformLinux, Host: "host.example.test", Port: 22, Username: "orbit", WorkspaceRoot: "/srv/orbit",
			HostKeyFingerprint: "SHA256:abc",
		},
	}}
	server, err := NewServer(Dependencies{ActorUserId: "actor", Environment: environment})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)

	getResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_get_project_environment", Arguments: map[string]any{"project_id": "project-1"}})
	if err != nil || getResult.IsError {
		t.Fatalf("CallTool(get environment) result=%#v err=%v", getResult, err)
	}
	getOutput := structuredOutput(t, getResult)
	if getOutput["project_id"] != "project-1" || environment.userID != "actor" || environment.projectID != "project-1" {
		t.Fatalf("get environment scope = output %#v service %#v", getOutput, environment)
	}

	workspaceRoot := "/srv/orbit-next"
	updateResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_update_project_environment", Arguments: map[string]any{"project_id": "project-1", "ssh": map[string]any{"platform": "linux", "host": "host.example.test", "port": 22, "username": "orbit", "workspace_root": workspaceRoot}}})
	if err != nil || updateResult.IsError {
		t.Fatalf("CallTool(update environment) result=%#v err=%v", updateResult, err)
	}
	if environment.update.SSH == nil || environment.update.SSH.WorkspaceRoot != workspaceRoot {
		t.Fatalf("update input = %#v", environment.update)
	}
	encodedUpdate, err := json.Marshal(structuredOutput(t, updateResult))
	if err != nil {
		t.Fatalf("marshal update output: %v", err)
	}
	if strings.Contains(string(encodedUpdate), "PRIVATE KEY") || strings.Contains(strings.ToLower(string(encodedUpdate)), "private_key") {
		t.Fatalf("update output exposed private key: %s", encodedUpdate)
	}
	getEncoded, err := json.Marshal(getOutput)
	if err != nil {
		t.Fatalf("marshal get output: %v", err)
	}
	if strings.Contains(string(getEncoded), "authorized_keys") || strings.Contains(strings.ToLower(string(getEncoded)), "private_key") {
		t.Fatalf("get output exposed initialization credentials: %s", getEncoded)
	}

	probeResult, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_probe_project_environment", Arguments: map[string]any{"project_id": "project-1"}})
	if err != nil || probeResult.IsError {
		t.Fatalf("CallTool(probe environment) result=%#v err=%v", probeResult, err)
	}
	if environment.probeCalls != 1 || environment.projectID != "project-1" || environment.userID != "actor" {
		t.Fatalf("probe scope = service %#v", environment)
	}
}

func TestProjectEnvironmentToolsReturnLocalWorkspaceWithoutSSHFields(t *testing.T) {
	environment := &environmentToolService{environment: environmentdto.View{
		Id: "environment-1", ProjectId: "project-1", Code: "project", State: model.EnvironmentStateActive,
		TargetType: model.EnvironmentTargetTypeLocal,
		Local:      &environmentdto.LocalTargetView{WorkspaceRoot: "/srv/orbit/deployment", Platform: "linux", Host: "orbit-host", Username: "orbit"},
	}}
	server, err := NewServer(Dependencies{
		ActorUserId: "actor", Environment: environment,
	})
	if err != nil {
		t.Fatal(err)
	}
	result, err := connectInMemory(t, server).CallTool(context.Background(), &mcp.CallToolParams{
		Name: "orbit_get_project_environment", Arguments: map[string]any{"project_id": "project-1"},
	})
	if err != nil || result.IsError {
		t.Fatalf("CallTool(get local environment) result=%#v err=%v", result, err)
	}
	output := structuredOutput(t, result)
	environmentOutput, ok := output["environment"].(map[string]any)
	if !ok {
		t.Fatalf("environment output = %#v", output["environment"])
	}
	local, ok := environmentOutput["local"].(map[string]any)
	if !ok || local["workspace_root"] != "/srv/orbit/deployment" {
		t.Fatalf("local output = %#v", environmentOutput["local"])
	}
	if _, found := environmentOutput["ssh"]; found {
		t.Fatalf("local environment exposed SSH fields: %#v", environmentOutput)
	}
}

func TestCreateVersionComponentPassesPolicies(t *testing.T) {
	application := &versionComponentApplicationService{}
	server, err := NewServer(Dependencies{ActorUserId: "actor", Application: application})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "orbit_create_version_component",
		Arguments: map[string]any{"version_id": "version-1", "component": map[string]any{"name": "api", "image": "nginx:1.27", "pull_policy": "missing", "restart_policy": "unless-stopped"}},
	})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if result.IsError {
		t.Fatalf("CallTool() returned tool error: %#v", result.Content)
	}
	if application.input.PullPolicy != "missing" || application.input.RestartPolicy == nil || *application.input.RestartPolicy != "unless-stopped" {
		t.Fatalf("component input policies = pull %q, restart %#v; want submitted values", application.input.PullPolicy, application.input.RestartPolicy)
	}
}

func TestServerInstructionsAndRuntimeConfigToolsDocumentStatefulServiceBoundary(t *testing.T) {
	server, err := NewServer(Dependencies{ActorUserId: "actor"})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	instructions := session.InitializeResult().Instructions
	for _, term := range []string{"${KEY}", "orbit_update_service_env", "empty data volume", "in-place rotation", "volume reset"} {
		if !strings.Contains(instructions, term) {
			t.Errorf("instructions do not document %q: %s", term, instructions)
		}
	}

	tools, err := session.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	byName := make(map[string]*mcp.Tool, len(tools.Tools))
	for _, tool := range tools.Tools {
		byName[tool.Name] = tool
	}
	for name, term := range map[string]string{
		"orbit_update_version_component_env": "${KEY}",
		"orbit_update_service_env":           "empty data volume",
	} {
		if tool := byName[name]; tool == nil || !strings.Contains(tool.Description, term) {
			t.Errorf("tool %q does not document %q", name, term)
		}
	}
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

func TestActorAuthenticatorRunsOnlyForToolCallsAndRevalidatesTheSession(t *testing.T) {
	project := &actorProjectService{}
	authentications := 0
	server, err := NewServer(Dependencies{
		ActorAuthenticator: func(context.Context) (string, error) {
			authentications++
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
	if authentications != 0 {
		t.Fatalf("authentications after ListTools() = %d, want 0", authentications)
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
	if authentications != 2 {
		t.Fatalf("authentications after tool calls = %d, want 2", authentications)
	}
	if project.actorUserId != "current-user" || project.calls != 2 {
		t.Fatalf("project calls = actor %q, count %d; want current-user, 2", project.actorUserId, project.calls)
	}
}

func TestActorAuthenticationFailureStopsToolExecution(t *testing.T) {
	project := &actorProjectService{}
	server, err := NewServer(Dependencies{
		ActorAuthenticator: func(context.Context) (string, error) {
			return "", apperror.New(apperror.KindUnauthorized, "Invalid token")
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

func TestActorAuthenticatorRejectsActorChangesWithinOneSession(t *testing.T) {
	project := &actorProjectService{}
	actorUserIds := []string{"current-user", "other-user"}
	server, err := NewServer(Dependencies{
		ActorAuthenticator: func(context.Context) (string, error) {
			actorUserId := actorUserIds[0]
			actorUserIds = actorUserIds[1:]
			return actorUserId, nil
		},
		Project: project,
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	for index := range 2 {
		result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_list_projects"})
		if err != nil {
			t.Fatalf("CallTool() error = %v", err)
		}
		if index == 0 && result.IsError {
			t.Fatalf("first CallTool() returned tool error: %#v", result.Content)
		}
		if index == 1 && !result.IsError {
			t.Fatal("second CallTool() IsError = false, want true")
		}
	}
	if project.calls != 1 || project.actorUserId != "current-user" {
		t.Fatalf("project calls = actor %q, count %d; want current-user, 1", project.actorUserId, project.calls)
	}
}

func TestActorAuthenticatorKeepsConcurrentCallsBoundToOneActor(t *testing.T) {
	project := &actorProjectService{}
	var mu sync.Mutex
	authentications := 0
	server, err := NewServer(Dependencies{
		ActorAuthenticator: func(context.Context) (string, error) {
			mu.Lock()
			authentications++
			mu.Unlock()
			return "current-user", nil
		},
		Project: project,
	})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	session := connectInMemory(t, server)
	var wait sync.WaitGroup
	errs := make(chan error, 2)
	for range 2 {
		wait.Add(1)
		go func() {
			defer wait.Done()
			result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_list_projects"})
			if err != nil {
				errs <- err
				return
			}
			if result.IsError {
				errs <- errors.New("concurrent tool call returned an error")
			}
		}()
	}
	wait.Wait()
	close(errs)
	for err := range errs {
		t.Fatal(err)
	}
	mu.Lock()
	gotAuthentications := authentications
	mu.Unlock()
	actorUserId, calls := project.snapshot()
	if gotAuthentications != 2 || calls != 2 || actorUserId != "current-user" {
		t.Fatalf("authentications=%d calls=%d actor=%q; want 2, 2, current-user", gotAuthentications, calls, actorUserId)
	}
}

type environmentToolService struct {
	EnvironmentService
	environment environmentdto.View
	userID      string
	projectID   string
	update      environmentdto.UpdateInput
	probeCalls  int
}

func (s *environmentToolService) EnvironmentForUser(_ context.Context, userID, projectID string) (environmentdto.View, error) {
	s.userID, s.projectID = userID, projectID
	return s.environment, nil
}

func (s *environmentToolService) UpdateForUser(_ context.Context, userID, projectID string, input environmentdto.UpdateInput) (environmentdto.View, error) {
	s.userID, s.projectID, s.update = userID, projectID, input
	return s.environment, nil
}

func (s *environmentToolService) ProbeForUser(_ context.Context, userID, projectID string) (environmentdto.View, error) {
	s.userID, s.projectID, s.probeCalls = userID, projectID, s.probeCalls+1
	return s.environment, nil
}

type actorProjectService struct {
	mu          sync.Mutex
	actorUserId string
	calls       int
}

func (s *actorProjectService) ListByMember(_ context.Context, actorUserId string) ([]model.Project, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.actorUserId = actorUserId
	s.calls++
	return []model.Project{}, nil
}

func (s *actorProjectService) snapshot() (string, int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.actorUserId, s.calls
}

var deliveryToolNames = []string{
	"orbit_list_projects", "orbit_get_project_environment", "orbit_update_project_environment", "orbit_probe_project_environment",
	"orbit_list_applications", "orbit_list_application_services", "orbit_list_gateways",
	"orbit_create_gateway", "orbit_provision_gateway", "orbit_get_gateway", "orbit_update_gateway",
	"orbit_create_application", "orbit_get_application", "orbit_delete_application", "orbit_list_versions", "orbit_get_version",
	"orbit_create_version_component", "orbit_create_version", "orbit_update_version",
	"orbit_update_version_component_basic", "orbit_update_version_component_runtime", "orbit_update_version_component_endpoints",
	"orbit_update_version_component_env", "orbit_update_version_component_mounts", "orbit_update_version_component_dependencies",
	"orbit_update_version_component_devices", "orbit_update_version_component_advanced", "orbit_update_version_component_resources",
	"orbit_update_version_component_tmpfs", "orbit_update_version_component_ulimits", "orbit_publish_version", "orbit_delete_version",
	"orbit_list_routes", "orbit_get_route", "orbit_create_route", "orbit_update_route", "orbit_enable_route", "orbit_disable_route", "orbit_preview_route_sync", "orbit_confirm_route_sync",
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

type versionComponentApplicationService struct {
	ApplicationService
	input applicationdto.VersionComponentInput
}

func (s *versionComponentApplicationService) CreateVersionComponent(_ context.Context, _ string, _ string, input applicationdto.VersionComponentInput) (model.VersionComponent, error) {
	s.input = input
	return model.VersionComponent{Id: "component-1", PullPolicy: input.PullPolicy, RestartPolicy: input.RestartPolicy}, nil
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

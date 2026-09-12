package delivery

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestListProjectsReturnsNameCodeAndActiveState(t *testing.T) {
	server := newScopedServer(t, Dependencies{Project: &readyProjectService{projects: []model.Project{
		{Id: "project-1", Name: "Demo", Code: "demo", IsActive: true},
		{Id: "project-2", Name: "Demo", Code: "demo-b", IsActive: false},
	}}})
	result, err := connectInMemory(t, server).CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_list_projects"})
	if err != nil || result.IsError {
		t.Fatalf("CallTool(list projects) result=%#v err=%v", result, err)
	}
	output := structuredOutput(t, result)
	projects, ok := output["projects"].([]any)
	if !ok || len(projects) != 2 {
		t.Fatalf("projects = %#v", output["projects"])
	}
	first, ok := projects[0].(map[string]any)
	if !ok || first["id"] != "project-1" || first["name"] != "Demo" || first["code"] != "demo" || first["is_active"] != true {
		t.Fatalf("first project = %#v", projects[0])
	}
}

func TestProjectLevelToolsRequireSelection(t *testing.T) {
	server, err := NewServer(Dependencies{ActorUserId: "actor"})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	result, err := connectInMemory(t, server).CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_get_current_project"})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool() IsError = false, want true")
	}
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(content.Text, "project_not_selected") {
		t.Fatalf("error text = %#v", result.Content)
	}
}

func TestSelectProjectConfirmsReadyScope(t *testing.T) {
	server := newReadyServer(t, Dependencies{SelectedProjectId: ""})
	session := connectInMemory(t, server)
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_select_project", Arguments: map[string]any{"project_id": "project-1"}})
	if err != nil || result.IsError {
		t.Fatalf("CallTool(select project) result=%#v err=%v", result, err)
	}
	output := structuredOutput(t, result)
	project, ok := output["project"].(map[string]any)
	if !ok || project["id"] != "project-1" || project["code"] != "demo" {
		t.Fatalf("select project = %#v", output["project"])
	}
	if _, ok := output["environment"].(map[string]any); !ok {
		t.Fatalf("select environment = %#v", output["environment"])
	}
	if _, ok := output["gateway"].(map[string]any); !ok {
		t.Fatalf("select gateway = %#v", output["gateway"])
	}
	current, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_get_current_project"})
	if err != nil || current.IsError {
		t.Fatalf("CallTool(current project) result=%#v err=%v", current, err)
	}
	currentOutput := structuredOutput(t, current)
	currentProject, ok := currentOutput["project"].(map[string]any)
	if !ok || currentProject["id"] != "project-1" {
		t.Fatalf("current project = %#v", currentOutput["project"])
	}
}

func TestSelectProjectRejectsUnreadyProject(t *testing.T) {
	server := newReadyServer(t, Dependencies{
		SelectedProjectId: "",
		Environment:       missingEnvironmentService{},
	})
	result, err := connectInMemory(t, server).CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_select_project", Arguments: map[string]any{"project_id": "project-1"}})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool() IsError = false, want true")
	}
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(content.Text, "project_not_ready") {
		t.Fatalf("error text = %#v", result.Content)
	}
}

func TestSelectProjectCanSwitchConnectionScope(t *testing.T) {
	projects := &readyProjectService{projects: []model.Project{
		{Id: "project-1", Name: "Demo", Code: "demo", IsActive: true},
		{Id: "project-2", Name: "Other", Code: "other", IsActive: true},
	}}
	server := newReadyServer(t, Dependencies{SelectedProjectId: "", Project: projects})
	session := connectInMemory(t, server)
	if _, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_select_project", Arguments: map[string]any{"project_id": "project-1"}}); err != nil {
		t.Fatalf("select project-1: %v", err)
	}
	result, err := session.CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_select_project", Arguments: map[string]any{"project_id": "project-2"}})
	if err != nil || result.IsError {
		t.Fatalf("CallTool(select project-2) result=%#v err=%v", result, err)
	}
	output := structuredOutput(t, result)
	project, ok := output["project"].(map[string]any)
	if !ok || project["id"] != "project-2" {
		t.Fatalf("switched project = %#v", output["project"])
	}
}

func TestFixedDialogueScopeRejectsProjectSwitch(t *testing.T) {
	projects := &readyProjectService{projects: []model.Project{
		{Id: "project-1", Name: "Demo", Code: "demo", IsActive: true},
		{Id: "project-2", Name: "Other", Code: "other", IsActive: true},
	}}
	server := newScopedServer(t, Dependencies{SelectedProjectId: "project-1", ScopeFixed: true, Project: projects})
	result, err := connectInMemory(t, server).CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_select_project", Arguments: map[string]any{"project_id": "project-2"}})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool() IsError = false, want true")
	}
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(content.Text, "project_scope_fixed") {
		t.Fatalf("error text = %#v", result.Content)
	}
}

func TestProjectLevelToolSchemasOmitProjectID(t *testing.T) {
	server, err := NewServer(Dependencies{ActorUserId: "actor"})
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	tools, err := connectInMemory(t, server).ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("ListTools() error = %v", err)
	}
	byName := map[string]*mcp.Tool{}
	for _, tool := range tools.Tools {
		byName[tool.Name] = tool
	}
	selectSchema := toolInputSchema(t, byName["orbit_select_project"])
	if _, ok := selectSchema.Properties["project_id"]; !ok {
		t.Fatal("orbit_select_project must require project_id")
	}
	for _, name := range []string{"orbit_list_applications", "orbit_create_application", "orbit_list_routes", "orbit_create_route", "orbit_list_gateways", "orbit_provision_gateway", "orbit_get_project_environment"} {
		schema := toolInputSchema(t, byName[name])
		if _, ok := schema.Properties["project_id"]; ok {
			t.Fatalf("%s schema still has project_id", name)
		}
	}
}

func TestApplicationOwnershipMustMatchSelectedProject(t *testing.T) {
	server := newScopedServer(t, Dependencies{Application: mismatchApplicationService{projectID: "project-2"}})
	result, err := connectInMemory(t, server).CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_get_application", Arguments: map[string]any{"application_id": "application-1"}})
	if err != nil {
		t.Fatalf("CallTool() error = %v", err)
	}
	if !result.IsError {
		t.Fatal("CallTool() IsError = false, want true")
	}
	content, ok := result.Content[0].(*mcp.TextContent)
	if !ok || !strings.Contains(content.Text, "project_scope_mismatch") {
		t.Fatalf("error text = %#v", result.Content)
	}
}

func TestProvisionGatewayDoesNotCreateMissingGateway(t *testing.T) {
	view := readyGatewayView("project-1")
	gateway := &readyGatewayService{provision: gatewaydto.ProvisionGatewayResult{
		Gateway: view, Service: *view.DefaultService, GatewayCreated: false, ServiceCreated: false, Steps: []string{"resolved existing gateway"},
	}}
	server := newScopedServer(t, Dependencies{Gateway: gateway})
	result, err := connectInMemory(t, server).CallTool(context.Background(), &mcp.CallToolParams{Name: "orbit_provision_gateway"})
	if err != nil || result.IsError {
		t.Fatalf("CallTool(provision gateway) result=%#v err=%v", result, err)
	}
	output := structuredOutput(t, result)
	created, _ := output["created"].(map[string]any)
	if created["gateway"] == true || created["service"] == true {
		t.Fatalf("provision created resources: %#v", output)
	}
}

type missingEnvironmentService struct{ EnvironmentService }

func (missingEnvironmentService) EnvironmentForUser(context.Context, string, string) (environmentdto.View, error) {
	return environmentdto.View{}, apperror.New(apperror.KindNotFound, "Environment not found")
}

type mismatchApplicationService struct {
	ApplicationService
	projectID string
}

func (s mismatchApplicationService) ApplicationForUser(_ context.Context, _ string, applicationID string) (model.Application, error) {
	projectID := s.projectID
	return model.Application{Id: applicationID, ProjectId: &projectID}, nil
}

func toolInputSchema(t *testing.T, tool *mcp.Tool) (schema struct {
	Properties map[string]json.RawMessage `json:"properties"`
	Required   []string                   `json:"required"`
}) {
	t.Helper()
	if tool == nil {
		t.Fatal("tool is nil")
	}
	encoded, err := json.Marshal(tool.InputSchema)
	if err != nil {
		t.Fatalf("marshal schema: %v", err)
	}
	if err := json.Unmarshal(encoded, &schema); err != nil {
		t.Fatalf("unmarshal schema: %v", err)
	}
	return schema
}

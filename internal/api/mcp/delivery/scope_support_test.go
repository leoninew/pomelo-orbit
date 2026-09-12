package delivery

import (
	"context"
	"testing"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func readyProjectID() *string {
	id := "project-1"
	return &id
}

func readyProbe(revision int64) (int64, string) {
	status := model.EnvironmentProbeStatusSucceeded
	return revision, status
}

func readyEnvironmentView(projectID string) environmentdto.View {
	revision, status := readyProbe(1)
	return environmentdto.View{
		Id: "environment-1", ProjectId: projectID, Code: "demo", State: model.EnvironmentStateActive,
		TargetType: model.EnvironmentTargetTypeLocal, TargetRevision: 1,
		LastProbeRevision: &revision, LastProbeStatus: &status,
		Local: &environmentdto.LocalTargetView{WorkspaceRoot: "/srv/orbit"},
	}
}

func readyGatewayView(projectID string) gatewaydto.GatewayView {
	return gatewaydto.GatewayView{
		Application:    model.Application{Id: "gateway-1", ProjectId: &projectID, Name: "Demo Gateway", Code: "demo-gateway"},
		DefaultService: &model.Service{Id: "gateway-service-1", ApplicationId: "gateway-1", InstanceKey: "default", Status: "stopped"},
	}
}

type readyProjectService struct {
	ProjectService
	projects []model.Project
}

func (s *readyProjectService) ListByMember(context.Context, string) ([]model.Project, error) {
	if len(s.projects) > 0 {
		return s.projects, nil
	}
	return []model.Project{{Id: "project-1", Name: "Demo", Code: "demo", IsActive: true}}, nil
}

type readyGatewayService struct {
	GatewayService
	gateway   gatewaydto.GatewayView
	provision gatewaydto.ProvisionGatewayResult
}

func (s *readyGatewayService) view() gatewaydto.GatewayView {
	if s.gateway.Application.Id != "" {
		return s.gateway
	}
	return readyGatewayView("project-1")
}

func (s *readyGatewayService) ListGateways(context.Context, string, string, int, int, string) (repository.Page[gatewaydto.GatewayView], error) {
	view := s.view()
	return repository.Page[gatewaydto.GatewayView]{Items: []gatewaydto.GatewayView{view}, Total: 1, Page: 1, PerPage: 1}, nil
}

func (s *readyGatewayService) GatewayForUser(_ context.Context, _ string, gatewayID string) (gatewaydto.GatewayView, error) {
	view := s.view()
	view.Application.Id = gatewayID
	return view, nil
}

func (s *readyGatewayService) ProvisionGateway(context.Context, string, gatewaydto.ProvisionGatewayInput) (gatewaydto.ProvisionGatewayResult, error) {
	if s.provision.Service.Id != "" {
		return s.provision, nil
	}
	view := s.view()
	return gatewaydto.ProvisionGatewayResult{Gateway: view, Service: *view.DefaultService, Steps: []string{"resolved existing gateway"}}, nil
}

func withReadyScope(deps Dependencies) Dependencies {
	if deps.ActorUserId == "" {
		deps.ActorUserId = "actor"
	}
	if deps.SelectedProjectId == "" {
		deps.SelectedProjectId = "project-1"
	}
	if deps.Project == nil {
		deps.Project = &readyProjectService{}
	}
	if deps.Environment == nil {
		deps.Environment = &environmentToolService{environment: readyEnvironmentView("project-1")}
	}
	if deps.Gateway == nil {
		deps.Gateway = &readyGatewayService{}
	}
	return deps
}

func newScopedServer(t *testing.T, deps Dependencies) *mcp.Server {
	t.Helper()
	server, err := NewServer(withReadyScope(deps))
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}

func newReadyServer(t *testing.T, deps Dependencies) *mcp.Server {
	t.Helper()
	selected := deps.SelectedProjectId
	fixed := deps.ScopeFixed
	scoped := withReadyScope(deps)
	scoped.SelectedProjectId = selected
	scoped.ScopeFixed = fixed
	server, err := NewServer(scoped)
	if err != nil {
		t.Fatalf("NewServer() error = %v", err)
	}
	return server
}

func (s *mountApplicationService) ApplicationForUser(_ context.Context, _ string, applicationID string) (model.Application, error) {
	return model.Application{Id: applicationID, ProjectId: readyProjectID()}, nil
}

func (s *mountApplicationService) VersionForUser(_ context.Context, _ string, versionID string) (applicationdto.VersionView, error) {
	return applicationdto.VersionView{Version: model.Version{Id: versionID, ApplicationId: "application-1"}}, nil
}

func (s *versionComponentApplicationService) ApplicationForUser(_ context.Context, _ string, applicationID string) (model.Application, error) {
	return model.Application{Id: applicationID, ProjectId: readyProjectID()}, nil
}

func (s *versionComponentApplicationService) VersionForUser(_ context.Context, _ string, versionID string) (applicationdto.VersionView, error) {
	return applicationdto.VersionView{Version: model.Version{Id: versionID, ApplicationId: "application-1"}}, nil
}

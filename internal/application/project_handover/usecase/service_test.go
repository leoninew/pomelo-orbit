package handoversvc

import (
	"context"
	"reflect"
	"testing"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	projectdto "github.com/leoninew/pomelo-orbit/internal/application/project/dto"
	handoverdto "github.com/leoninew/pomelo-orbit/internal/application/project_handover/dto"
	routedto "github.com/leoninew/pomelo-orbit/internal/application/route/dto"
	servicedto "github.com/leoninew/pomelo-orbit/internal/application/service/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestImportNewCreatesProjectBeforeRestoringEnvironment(t *testing.T) {
	calls := []string{}
	service := newTestService(&calls)

	project, err := service.Import(context.Background(), "operator", handoverdto.ImportInput{
		Mode: handoverdto.ImportModeNew, Package: testPackage(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.Id != "new-project" {
		t.Fatalf("unexpected imported Project: %+v", project)
	}
	want := []string{"project.create", "environment.save"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestImportReplaceClearsDomainConfigurationBeforeEnvironmentRestore(t *testing.T) {
	calls := []string{}
	service := newTestService(&calls)

	project, err := service.Import(context.Background(), "operator", handoverdto.ImportInput{
		Mode: handoverdto.ImportModeReplace, TargetProjectID: "target-project", Package: testPackage(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.Id != "target-project" {
		t.Fatalf("unexpected imported Project: %+v", project)
	}
	want := []string{
		"project.load",
		"pipeline.references",
		"pipeline-run.references",
		"gateway.list",
		"route.list",
		"service.list",
		"application.list",
		"deployment.clear",
		"project.update",
		"environment.save",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestRemapDefinitionsResetRuntimeStateAndProjectOwnership(t *testing.T) {
	projectID := "source-project"
	serviceID := "source-service"
	service := servicedto.ServiceDefinition{Service: model.Service{
		Id: serviceID, ProjectId: projectID, ApplicationId: "source-application", VersionId: "source-version", Status: "running",
	}, Components: []model.ServiceComponent{{
		Id: "source-component", ServiceId: serviceID, SourceVersionComponentId: "source-version-component",
	}}}
	maps := packageIDMaps{
		applications: map[string]string{"source-application": "target-application"},
		versions:     map[string]string{"source-version": "target-version"},
		components:   map[string]string{"source-version-component": "target-version-component"},
		services:     map[string]string{serviceID: "target-service"},
	}

	remappedService := remapServiceDefinition(service, maps)
	if remappedService.Service.Id != "" || remappedService.Service.ProjectId != "" || remappedService.Service.Status != "stopped" {
		t.Fatalf("Service was not normalized for import: %+v", remappedService.Service)
	}
	if remappedService.Components[0].Id != "" || remappedService.Components[0].ServiceId != "" || remappedService.Components[0].SourceVersionComponentId != "target-version-component" {
		t.Fatalf("Service Component was not remapped: %+v", remappedService.Components[0])
	}

	route := routedto.RouteDefinitionInput{Route: model.Route{Id: "source-route", ProjectId: &projectID, ServiceId: &serviceID, Enabled: true}}
	remappedRoute := remapRouteDefinition(route, maps)
	if remappedRoute.Route.Id != "" || remappedRoute.Route.ProjectId != nil || remappedRoute.Route.Enabled || remappedRoute.Route.ServiceId == nil || *remappedRoute.Route.ServiceId != "target-service" {
		t.Fatalf("Route was not normalized for import: %+v", remappedRoute.Route)
	}
}

func testPackage() handoverdto.Package {
	return handoverdto.Package{
		Format:       handoverdto.Format,
		Version:      handoverdto.FormatVersion,
		Project:      projectdto.ProjectDefinition{Name: "Source", Code: "source", IsActive: true},
		Environment:  environmentdto.TargetDefinition{TargetType: "local", WorkspaceRoot: "/srv/orbit", TargetRevision: 1},
		Applications: []applicationdto.ApplicationDefinition{},
		Services:     []servicedto.ServiceDefinition{},
		Routes:       []routedto.RouteDefinitionInput{},
	}
}

func newTestService(calls *[]string) Service {
	return New(
		&testProjectDomain{calls: calls},
		&testEnvironmentDomain{calls: calls},
		&testApplicationDomain{calls: calls},
		&testServiceDomain{calls: calls},
		&testRouteDomain{calls: calls},
		&testGatewayDomain{calls: calls},
		&testPipelineDomain{calls: calls},
		&testPipelineRunDomain{calls: calls},
		&testDeploymentDomain{calls: calls},
	)
}

type testProjectDomain struct{ calls *[]string }

func (d *testProjectDomain) LoadForUser(_ context.Context, projectID string, _ string) (model.Project, error) {
	*d.calls = append(*d.calls, "project.load")
	return model.Project{Id: projectID}, nil
}
func (d *testProjectDomain) CreateFromDefinition(_ context.Context, _ string, _ projectdto.ProjectDefinition) (model.Project, error) {
	*d.calls = append(*d.calls, "project.create")
	return model.Project{Id: "new-project"}, nil
}
func (d *testProjectDomain) UpdateFromDefinition(_ context.Context, _ string, projectID string, _ projectdto.ProjectDefinition) (model.Project, error) {
	*d.calls = append(*d.calls, "project.update")
	return model.Project{Id: projectID}, nil
}

type testEnvironmentDomain struct{ calls *[]string }

func (d *testEnvironmentDomain) TargetDefinitionForUser(context.Context, string, string) (environmentdto.TargetDefinition, error) {
	return environmentdto.TargetDefinition{}, nil
}
func (d *testEnvironmentDomain) SaveTargetDefinitionForUser(_ context.Context, _ string, _ string, definition environmentdto.TargetDefinition) (environmentdto.TargetDefinition, error) {
	*d.calls = append(*d.calls, "environment.save")
	return definition, nil
}

type testApplicationDomain struct{ calls *[]string }

func (d *testApplicationDomain) ListApplications(context.Context, string, string, int, int, string, string) (repository.Page[model.Application], error) {
	*d.calls = append(*d.calls, "application.list")
	return repository.Page[model.Application]{}, nil
}
func (*testApplicationDomain) ApplicationDefinitionForUser(context.Context, string, string, string) (applicationdto.ApplicationDefinition, error) {
	return applicationdto.ApplicationDefinition{}, nil
}
func (*testApplicationDomain) CreateApplicationFromDefinition(context.Context, string, string, applicationdto.ApplicationDefinition) (applicationdto.ApplicationDefinition, error) {
	return applicationdto.ApplicationDefinition{}, nil
}
func (d *testApplicationDomain) RemoveApplication(context.Context, string, string, string) error {
	*d.calls = append(*d.calls, "application.remove")
	return nil
}

type testServiceDomain struct{ calls *[]string }

func (d *testServiceDomain) ListServices(context.Context, string, servicedto.ServiceListInput) (repository.Page[servicedto.ServiceView], error) {
	*d.calls = append(*d.calls, "service.list")
	return repository.Page[servicedto.ServiceView]{}, nil
}
func (*testServiceDomain) ServiceDefinitionForUser(context.Context, string, string, string) (servicedto.ServiceDefinition, error) {
	return servicedto.ServiceDefinition{}, nil
}
func (*testServiceDomain) CreateServiceFromDefinition(context.Context, string, string, servicedto.ServiceDefinition) (servicedto.ServiceView, error) {
	return servicedto.ServiceView{}, nil
}
func (d *testServiceDomain) RemoveService(context.Context, string, string, string) error {
	*d.calls = append(*d.calls, "service.remove")
	return nil
}

type testRouteDomain struct{ calls *[]string }

func (d *testRouteDomain) ListRoutes(context.Context, string, string, int, int, string) (repository.Page[model.Route], error) {
	*d.calls = append(*d.calls, "route.list")
	return repository.Page[model.Route]{}, nil
}
func (*testRouteDomain) CreateRouteFromDefinition(context.Context, string, string, routedto.RouteDefinitionInput) (model.Route, error) {
	return model.Route{}, nil
}
func (d *testRouteDomain) RemoveRoute(context.Context, string, string, string) error {
	*d.calls = append(*d.calls, "route.remove")
	return nil
}

type testGatewayDomain struct{ calls *[]string }

func (d *testGatewayDomain) ListGateways(context.Context, string, string, int, int, string) (repository.Page[gatewaydto.GatewayView], error) {
	*d.calls = append(*d.calls, "gateway.list")
	return repository.Page[gatewaydto.GatewayView]{}, nil
}
func (*testGatewayDomain) GatewayDefinitionForUser(context.Context, string, string, string) (gatewaydto.GatewayDefinition, error) {
	return gatewaydto.GatewayDefinition{}, nil
}
func (*testGatewayDomain) CreateGatewayFromDefinition(context.Context, string, string, gatewaydto.GatewayDefinition) (gatewaydto.GatewayDefinition, error) {
	return gatewaydto.GatewayDefinition{}, nil
}
func (d *testGatewayDomain) RemoveGateway(context.Context, string, string, string) error {
	*d.calls = append(*d.calls, "gateway.remove")
	return nil
}

type testPipelineDomain struct{ calls *[]string }

func (d *testPipelineDomain) EnsureNoCDConfigurationReferences(context.Context, string, string) error {
	*d.calls = append(*d.calls, "pipeline.references")
	return nil
}

type testPipelineRunDomain struct{ calls *[]string }

func (d *testPipelineRunDomain) EnsureNoCDConfigurationReferences(context.Context, string, string) error {
	*d.calls = append(*d.calls, "pipeline-run.references")
	return nil
}

type testDeploymentDomain struct{ calls *[]string }

func (d *testDeploymentDomain) ClearProjectDeploymentHistory(context.Context, string, string) error {
	*d.calls = append(*d.calls, "deployment.clear")
	return nil
}

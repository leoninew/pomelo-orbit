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
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestImportNewCreatesProjectWithSubmittedIdentity(t *testing.T) {
	calls := []string{}
	created := projectdto.ProjectDefinition{}
	service := newTestServiceWithProject(&calls, &testProjectDomain{calls: &calls, created: &created})

	project, err := service.Import(context.Background(), "operator", handoverdto.ImportInput{
		Mode: handoverdto.ImportModeNew, Name: "Taken Over", Code: "taken-over", OverrideEnvironment: true, Package: testPackage(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if project.Id != "new-project" {
		t.Fatalf("unexpected imported Project: %+v", project)
	}
	if created.Name != "Taken Over" || created.Code != "taken-over" || !created.IsActive {
		t.Fatalf("created Project identity = %+v", created)
	}
	want := []string{"project.create", "environment.handover.save"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestImportDocumentReportsSafeDecodeError(t *testing.T) {
	service := newTestService(&[]string{})

	_, err := service.ImportDocument(
		context.Background(),
		"operator",
		handoverdto.ImportInput{Mode: handoverdto.ImportModeNew, Name: "Taken Over", Code: "taken-over"},
		[]byte(`{"unexpected":true}`),
	)
	if err == nil {
		t.Fatal("ImportDocument() error = nil")
	}
	if got, want := apperror.Classify(err).Message, "Handover package contains unsupported or invalid fields"; got != want {
		t.Fatalf("error message = %q, want %q", got, want)
	}
}

func TestImportReplaceKeepsTargetProjectIdentity(t *testing.T) {
	calls := []string{}
	service := newTestService(&calls)

	project, err := service.Import(context.Background(), "operator", handoverdto.ImportInput{
		Mode: handoverdto.ImportModeReplace, TargetProjectId: "target-project", OverrideEnvironment: true, Package: testPackage(),
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
		"environment.handover.save",
	}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestRestoreProjectConfigurationCreatesGatewayBeforeRoutes(t *testing.T) {
	calls := []string{}
	service := newTestService(&calls)
	applicationId, versionId, serviceId := "source-application", "source-version", "source-service"
	item := testPackage()
	item.Applications = []applicationdto.ApplicationDefinition{{
		Application: model.Application{Id: applicationId},
		Versions:    []applicationdto.VersionDefinition{{Version: model.Version{Id: versionId}}},
	}}
	item.Services = []servicedto.ServiceDefinition{{
		Service: model.Service{Id: serviceId, ApplicationId: applicationId, VersionId: versionId},
	}}
	item.Gateway = &gatewaydto.GatewayDefinition{}
	item.Routes = []routedto.RouteDefinitionInput{{Route: model.Route{ServiceId: &serviceId}}}

	if _, err := service.restoreProjectConfiguration(context.Background(), "operator", model.Project{Id: "target-project"}, item); err != nil {
		t.Fatal(err)
	}
	want := []string{"application.create", "service.create", "gateway.create", "route.create"}
	if !reflect.DeepEqual(calls, want) {
		t.Fatalf("calls = %v, want %v", calls, want)
	}
}

func TestRestoreProjectConfigurationRemapsGatewayRouteService(t *testing.T) {
	calls := []string{}
	gatewayServiceId := "source-gateway-service"
	targetGatewayServiceId := "target-gateway-service"
	routeDomain := &testRouteDomain{calls: &calls}
	gatewayDomain := &testGatewayDomain{
		calls: &calls,
		created: gatewaydto.GatewayDefinition{RuntimeService: servicedto.ServiceDefinition{
			Service: model.Service{Id: targetGatewayServiceId},
		}},
	}
	service := New(
		&testProjectDomain{calls: &calls},
		&testEnvironmentDomain{calls: &calls},
		&testApplicationDomain{calls: &calls},
		&testServiceDomain{calls: &calls},
		routeDomain,
		gatewayDomain,
		&testPipelineDomain{calls: &calls},
		&testPipelineRunDomain{calls: &calls},
		&testDeploymentDomain{calls: &calls},
		testHandoverSecretKey,
	)
	item := testPackage()
	item.Gateway = &gatewaydto.GatewayDefinition{RuntimeService: servicedto.ServiceDefinition{
		Service: model.Service{Id: gatewayServiceId},
	}}
	item.Routes = []routedto.RouteDefinitionInput{{Route: model.Route{ServiceId: &gatewayServiceId}}}

	if _, err := service.restoreProjectConfiguration(context.Background(), "operator", model.Project{Id: "target-project"}, item); err != nil {
		t.Fatal(err)
	}
	if len(routeDomain.created) != 1 || routeDomain.created[0].Route.ServiceId == nil || *routeDomain.created[0].Route.ServiceId != targetGatewayServiceId {
		t.Fatalf("Gateway Route ServiceId = %+v, want %q", routeDomain.created, targetGatewayServiceId)
	}
}

func TestRemapDefinitionsPreserveServiceAndRouteState(t *testing.T) {
	projectId := "source-project"
	serviceId := "source-service"
	service := servicedto.ServiceDefinition{Service: model.Service{
		Id: serviceId, ProjectId: projectId, ApplicationId: "source-application", VersionId: "source-version", Status: "running",
	}, Components: []model.ServiceComponent{{
		Id: "source-component", ServiceId: serviceId, SourceVersionComponentId: "source-version-component",
	}}}
	maps := packageIdMaps{
		applications: map[string]string{"source-application": "target-application"},
		versions:     map[string]string{"source-version": "target-version"},
		components:   map[string]string{"source-version-component": "target-version-component"},
		services:     map[string]string{serviceId: "target-service"},
	}

	remappedService := remapServiceDefinition(service, maps)
	if remappedService.Service.Id != "" || remappedService.Service.ProjectId != "" || remappedService.Service.Status != "running" {
		t.Fatalf("Service was not normalized for import: %+v", remappedService.Service)
	}
	if remappedService.Components[0].Id != "" || remappedService.Components[0].ServiceId != "" || remappedService.Components[0].SourceVersionComponentId != "target-version-component" {
		t.Fatalf("Service Component was not remapped: %+v", remappedService.Components[0])
	}

	route := routedto.RouteDefinitionInput{Route: model.Route{Id: "source-route", ProjectId: &projectId, ServiceId: &serviceId, Enabled: true}}
	remappedRoute := remapRouteDefinition(route, maps)
	if remappedRoute.Route.Id != "" || remappedRoute.Route.ProjectId != nil || !remappedRoute.Route.Enabled || remappedRoute.Route.ServiceId == nil || *remappedRoute.Route.ServiceId != "target-service" {
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
	return newTestServiceWithProject(calls, &testProjectDomain{calls: calls})
}

func newTestServiceWithProject(calls *[]string, project *testProjectDomain) Service {
	return New(
		project,
		&testEnvironmentDomain{calls: calls},
		&testApplicationDomain{calls: calls},
		&testServiceDomain{calls: calls},
		&testRouteDomain{calls: calls},
		&testGatewayDomain{calls: calls},
		&testPipelineDomain{calls: calls},
		&testPipelineRunDomain{calls: calls},
		&testDeploymentDomain{calls: calls},
		testHandoverSecretKey,
	)
}

func newTestServiceWithEnvironment(calls *[]string, environment *testEnvironmentDomain) Service {
	return New(
		&testProjectDomain{calls: calls},
		environment,
		&testApplicationDomain{calls: calls},
		&testServiceDomain{calls: calls},
		&testRouteDomain{calls: calls},
		&testGatewayDomain{calls: calls},
		&testPipelineDomain{calls: calls},
		&testPipelineRunDomain{calls: calls},
		&testDeploymentDomain{calls: calls},
		testHandoverSecretKey,
	)
}

const testHandoverSecretKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestImportDocumentDecryptsEnvironmentCredentialWithSystemKey(t *testing.T) {
	assertImportDocumentDecryptsCredential(t, "", testHandoverSecretKey)
}

func TestImportDocumentDecryptsEnvironmentCredentialWithCustomKey(t *testing.T) {
	customKey := "BBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBBB="
	assertImportDocumentDecryptsCredential(t, customKey, customKey)
}

func TestImportDocumentRejectsInvalidEnvironmentCredentialKey(t *testing.T) {
	calls := []string{}
	environment := &testEnvironmentDomain{calls: &calls}
	service := newTestServiceWithEnvironment(&calls, environment)
	item := testPackage()
	encrypted, err := security.EncryptString(testHandoverSecretKey, "PRIVATE KEY")
	if err != nil {
		t.Fatal(err)
	}
	item.Environment.Credential = &environmentdto.SSHCredentialDefinition{EncryptedPrivateKey: encrypted}
	document, err := handoverdto.Encode(item)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ImportDocument(context.Background(), "operator", handoverdto.ImportInput{
		Mode: handoverdto.ImportModeNew, Name: "Taken Over", Code: "taken-over", DecryptionKey: "invalid", OverrideEnvironment: true,
	}, document)
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("ImportDocument() error = %v, want validation error", err)
	}
	if len(calls) != 0 || environment.saved != nil {
		t.Fatalf("ImportDocument() continued after decryption failure: %v", calls)
	}
}

func assertImportDocumentDecryptsCredential(t *testing.T, decryptionKey string, encryptionKey string) {
	t.Helper()
	calls := []string{}
	environment := &testEnvironmentDomain{calls: &calls}
	service := newTestServiceWithEnvironment(&calls, environment)
	item := testPackage()
	encrypted, err := security.EncryptString(encryptionKey, "PRIVATE KEY")
	if err != nil {
		t.Fatal(err)
	}
	item.Environment.Credential = &environmentdto.SSHCredentialDefinition{EncryptedPrivateKey: encrypted}
	document, err := handoverdto.Encode(item)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ImportDocument(context.Background(), "operator", handoverdto.ImportInput{
		Mode: handoverdto.ImportModeNew, Name: "Taken Over", Code: "taken-over", DecryptionKey: decryptionKey, OverrideEnvironment: true,
	}, document)
	if err != nil {
		t.Fatal(err)
	}
	if len(calls) < 2 || calls[1] != "environment.handover.save" {
		t.Fatalf("ImportDocument() calls = %v, want environment save", calls)
	}
	if environment.saved == nil || environment.saved.Credential == nil || environment.saved.Credential.PrivateKey != "PRIVATE KEY" || environment.saved.Credential.EncryptedPrivateKey != "" {
		t.Fatalf("saved environment credential = %+v", environment.saved)
	}
}

func TestEncryptPackagePrivateKeyRemovesPlaintext(t *testing.T) {
	item := testPackage()
	item.Environment.Credential = &environmentdto.SSHCredentialDefinition{PrivateKey: "PRIVATE KEY"}
	if err := encryptPackagePrivateKey(testHandoverSecretKey, &item); err != nil {
		t.Fatal(err)
	}
	if item.Environment.Credential.PrivateKey != "" || item.Environment.Credential.EncryptedPrivateKey == "" {
		t.Fatalf("encrypted package credential = %+v", item.Environment.Credential)
	}
	plain, err := security.DecryptString(testHandoverSecretKey, item.Environment.Credential.EncryptedPrivateKey)
	if err != nil || plain != "PRIVATE KEY" {
		t.Fatalf("encrypted package credential decrypted = %q, %v", plain, err)
	}
}

func TestImportDocumentSkipsEnvironmentAndCredentialWhenNotOverridden(t *testing.T) {
	calls := []string{}
	environment := &testEnvironmentDomain{calls: &calls}
	service := newTestServiceWithEnvironment(&calls, environment)
	item := testPackage()
	item.Environment.Credential = &environmentdto.SSHCredentialDefinition{EncryptedPrivateKey: "not-a-valid-token"}
	document, err := handoverdto.Encode(item)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.ImportDocument(context.Background(), "operator", handoverdto.ImportInput{
		Mode: handoverdto.ImportModeNew, Name: "Taken Over", Code: "taken-over",
	}, document)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(calls, []string{"project.create"}) {
		t.Fatalf("ImportDocument() calls = %v, want only project creation", calls)
	}
	if environment.saved != nil {
		t.Fatalf("environment was saved despite override disabled: %+v", environment.saved)
	}
}

type testProjectDomain struct {
	calls   *[]string
	created *projectdto.ProjectDefinition
}

func (d *testProjectDomain) LoadForUser(_ context.Context, projectId string, _ string) (model.Project, error) {
	*d.calls = append(*d.calls, "project.load")
	return model.Project{Id: projectId}, nil
}
func (d *testProjectDomain) CreateFromDefinition(_ context.Context, _ string, input projectdto.ProjectDefinition) (model.Project, error) {
	*d.calls = append(*d.calls, "project.create")
	if d.created != nil {
		*d.created = input
	}
	return model.Project{Id: "new-project"}, nil
}

type testEnvironmentDomain struct {
	calls *[]string
	saved *environmentdto.TargetDefinition
}

func (d *testEnvironmentDomain) TargetDefinitionForUser(context.Context, string, string) (environmentdto.TargetDefinition, error) {
	return environmentdto.TargetDefinition{}, nil
}
func (d *testEnvironmentDomain) SaveTargetDefinitionForHandover(_ context.Context, _ string, _ string, definition environmentdto.TargetDefinition) (environmentdto.TargetDefinition, error) {
	*d.calls = append(*d.calls, "environment.handover.save")
	d.saved = &definition
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
func (d *testApplicationDomain) CreateApplicationFromDefinition(_ context.Context, _ string, _ string, definition applicationdto.ApplicationDefinition) (applicationdto.ApplicationDefinition, error) {
	*d.calls = append(*d.calls, "application.create")
	return definition, nil
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
func (d *testServiceDomain) CreateServiceFromDefinition(_ context.Context, _ string, _ string, definition servicedto.ServiceDefinition) (servicedto.ServiceView, error) {
	*d.calls = append(*d.calls, "service.create")
	return servicedto.ServiceView{Service: definition.Service}, nil
}
func (d *testServiceDomain) RemoveService(context.Context, string, string, string) error {
	*d.calls = append(*d.calls, "service.remove")
	return nil
}

func (d *testRouteDomain) ListAllRoutes(context.Context, string, string) ([]model.Route, error) {
	*d.calls = append(*d.calls, "route.list")
	return nil, nil
}

type testRouteDomain struct {
	calls   *[]string
	created []routedto.RouteDefinitionInput
}

func (d *testRouteDomain) CreateRouteFromDefinition(_ context.Context, _ string, _ string, definition routedto.RouteDefinitionInput) (model.Route, error) {
	*d.calls = append(*d.calls, "route.create")
	d.created = append(d.created, definition)
	return model.Route{}, nil
}
func (d *testRouteDomain) RemoveRoute(context.Context, string, string, string) error {
	*d.calls = append(*d.calls, "route.remove")
	return nil
}

type testGatewayDomain struct {
	calls   *[]string
	created gatewaydto.GatewayDefinition
}

func (d *testGatewayDomain) ListGateways(context.Context, string, string, int, int, string) (repository.Page[gatewaydto.GatewayView], error) {
	*d.calls = append(*d.calls, "gateway.list")
	return repository.Page[gatewaydto.GatewayView]{}, nil
}
func (*testGatewayDomain) GatewayDefinitionForUser(context.Context, string, string, string) (gatewaydto.GatewayDefinition, error) {
	return gatewaydto.GatewayDefinition{}, nil
}
func (d *testGatewayDomain) CreateGatewayFromDefinition(context.Context, string, string, gatewaydto.GatewayDefinition) (gatewaydto.GatewayDefinition, error) {
	*d.calls = append(*d.calls, "gateway.create")
	return d.created, nil
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

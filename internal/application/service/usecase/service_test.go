package servicesvc

import (
	"context"
	"strings"
	"testing"

	deploymentsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/usecase"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type serviceApplicationFake struct {
	app          model.Application
	version      model.Version
	declarations []model.VersionComponent
}

func (f serviceApplicationFake) Application(_ context.Context, _ string) (model.Application, error) {
	return f.app, nil
}

func (f serviceApplicationFake) Version(_ context.Context, _ string) (model.Version, error) {
	return f.version, nil
}

func (f serviceApplicationFake) VersionComponentsByVersion(_ context.Context, _ string) ([]model.VersionComponent, error) {
	return f.declarations, nil
}

type serviceStoreFake struct {
	repository.ServiceStore
	service    model.Service
	component  model.ServiceComponent
	components []model.ServiceComponent
	env        []model.ServiceEnv
}

func (f serviceStoreFake) Service(_ context.Context, _ string) (model.Service, error) {
	return f.service, nil
}

func (f serviceStoreFake) ServiceComponent(_ context.Context, _ string) (model.ServiceComponent, error) {
	return f.component, nil
}

func (f serviceStoreFake) ServiceComponentsByService(_ context.Context, _ string) ([]model.ServiceComponent, error) {
	return f.components, nil
}

func (f serviceStoreFake) ServiceEnvByService(_ context.Context, _ string) ([]model.ServiceEnv, error) {
	return f.env, nil
}

type deploymentStoreFake struct {
	repository.DeploymentStore
	planHash *string
	active   bool
}

func (f deploymentStoreFake) HasActiveDeployment(_ context.Context, _ string) (bool, error) {
	return f.active, nil
}

func (f deploymentStoreFake) LatestSuccessfulDeploymentPlanHash(_ context.Context, _ string) (*string, error) {
	return f.planHash, nil
}

func TestServiceViewPendingDeployComparesEffectivePlanHash(t *testing.T) {
	service, app, version, declaration, component := serviceViewFixture()
	plan, hash, err := buildFixturePlan(app, version, service, declaration, component)
	if err != nil {
		t.Fatalf("build fixture plan: %v", err)
	}
	if len(plan.Components) != 1 {
		t.Fatalf("fixture plan has %d components", len(plan.Components))
	}
	for _, test := range []struct {
		name     string
		deployed *string
		pending  bool
	}{
		{name: "matches latest successful deployment", deployed: &hash, pending: false},
		{name: "changed since latest successful deployment", deployed: stringPtr("outdated"), pending: true},
		{name: "has never been deployed", deployed: nil, pending: true},
	} {
		t.Run(test.name, func(t *testing.T) {
			usecase := Service{
				application: serviceApplicationFake{app: app, version: version, declarations: []model.VersionComponent{declaration}},
				service:     serviceStoreFake{components: []model.ServiceComponent{component}},
				deployment:  deploymentStoreFake{planHash: test.deployed},
			}
			view, err := usecase.serviceView(context.Background(), model.ServiceListItem{
				Id: service.Id, ApplicationId: service.ApplicationId, VersionId: service.VersionId, InstanceKey: service.InstanceKey,
				Status: service.Status, ApplicationName: app.Name, ApplicationCode: app.Code, ApplicationKind: app.Kind, VersionLabel: version.Label,
			})
			if err != nil {
				t.Fatal(err)
			}
			if view.EffectivePlanHash != hash || view.PendingDeploy != test.pending {
				t.Fatalf("service view = %#v, want hash %q and pending=%t", view, hash, test.pending)
			}
			if len(view.ComponentDefinitions) != 1 || view.ComponentDefinitions[0].Image != declaration.Image {
				t.Fatalf("component definitions = %#v, want declaration image %q", view.ComponentDefinitions, declaration.Image)
			}
		})
	}
}

func TestServiceViewReportsActiveDeploymentSeparatelyFromServiceStatus(t *testing.T) {
	service, app, version, declaration, component := serviceViewFixture()
	usecase := Service{
		application: serviceApplicationFake{app: app, version: version, declarations: []model.VersionComponent{declaration}},
		service:     serviceStoreFake{components: []model.ServiceComponent{component}},
		deployment:  deploymentStoreFake{active: true},
	}
	view, err := usecase.serviceView(context.Background(), model.ServiceListItem{
		Id: service.Id, ApplicationId: service.ApplicationId, VersionId: service.VersionId, InstanceKey: service.InstanceKey,
		Status: service.Status, ApplicationName: app.Name, ApplicationCode: app.Code, ApplicationKind: app.Kind, VersionLabel: version.Label,
	})
	if err != nil {
		t.Fatal(err)
	}
	if !view.ActiveDeployment || view.Service.Status != status.ServiceStatusStopped {
		t.Fatalf("service view must expose operation state separately: %#v", view)
	}
}

func TestServiceViewWithGatewayEndpointDoesNotRequireGateway(t *testing.T) {
	service, app, version, declaration, component := serviceViewFixture()
	declaration.Endpoints = []model.VersionComponentEndpoint{{Name: "http", Protocol: "http", ContainerPort: 80, Mode: "gateway_http"}}
	_, hash, err := buildFixturePlan(app, version, service, declaration, component)
	if err != nil {
		t.Fatalf("build fixture plan: %v", err)
	}
	usecase := Service{
		application: serviceApplicationFake{app: app, version: version, declarations: []model.VersionComponent{declaration}},
		service:     serviceStoreFake{components: []model.ServiceComponent{component}},
		deployment:  deploymentStoreFake{planHash: &hash},
	}

	view, err := usecase.serviceView(context.Background(), model.ServiceListItem{
		Id: service.Id, ApplicationId: service.ApplicationId, VersionId: service.VersionId, InstanceKey: service.InstanceKey,
		Status: service.Status, ApplicationName: app.Name, ApplicationCode: app.Code, ApplicationKind: app.Kind, VersionLabel: version.Label,
	})
	if err != nil {
		t.Fatalf("service view must not require a gateway: %v", err)
	}
	if view.EffectivePlanHash != hash || view.PendingDeploy {
		t.Fatalf("service view = %#v, want hash %q and pending=false", view, hash)
	}
}

func TestGetServiceComponentReturnsDeclarationOverlayAndEffectiveValues(t *testing.T) {
	service, app, version, declaration, component := serviceViewFixture()
	usecase := Service{
		application: serviceApplicationFake{app: app, version: version, declarations: []model.VersionComponent{declaration}},
		service:     serviceStoreFake{service: service, component: component},
	}
	detail, err := usecase.GetServiceComponent(context.Background(), "user-1", service.Id, component.Id)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Component.Id != component.Id || detail.Declaration.Id != declaration.Id {
		t.Fatalf("component detail lost source views: %#v", detail)
	}
	if detail.Effective == nil || len(detail.Effective.Env) != 1 || detail.Effective.Env[0].Value != "runtime-value" {
		t.Fatalf("effective component = %#v", detail.Effective)
	}
}

func TestGetServiceComponentResolvesServiceEnvironmentWithoutCreatingOverlay(t *testing.T) {
	service, app, version, declaration, component := serviceViewFixture()
	declaration.Env[0].Value = "${SHARED_VALUE}"
	component.Env = nil
	usecase := Service{
		application: serviceApplicationFake{app: app, version: version, declarations: []model.VersionComponent{declaration}},
		service: serviceStoreFake{
			service: service, component: component,
			env: []model.ServiceEnv{{Key: "SHARED_VALUE", Value: "from-service"}},
		},
	}
	detail, err := usecase.GetServiceComponent(context.Background(), "user-1", service.Id, component.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(detail.Component.Env) != 0 {
		t.Fatalf("service value must not become a component overlay: %#v", detail.Component.Env)
	}
	if detail.Effective == nil || len(detail.Effective.Env) != 1 || detail.Effective.Env[0].Value != "from-service" {
		t.Fatalf("effective component = %#v", detail.Effective)
	}
}

func TestGetServiceComponentReturnsSourceValuesForIncompleteServiceEnvironment(t *testing.T) {
	service, app, version, declaration, component := serviceViewFixture()
	declaration.Env[0].Value = "${MYSQL_DATABASE:?required}"
	component.Env = nil
	usecase := Service{
		application: serviceApplicationFake{app: app, version: version, declarations: []model.VersionComponent{declaration}},
		service:     serviceStoreFake{service: service, component: component},
	}

	detail, err := usecase.GetServiceComponent(context.Background(), "user-1", service.Id, component.Id)
	if err != nil {
		t.Fatal(err)
	}
	if detail.Component.Id != component.Id || detail.Declaration.Id != declaration.Id || detail.Effective != nil {
		t.Fatalf("component detail = %#v", detail)
	}
}

func TestServiceViewAllowsMissingRequiredServiceEnvironment(t *testing.T) {
	service, app, version, declaration, component := serviceViewFixture()
	declaration.Env[0].Value = "${REQUIRED_VALUE:?required}"
	component.Env = nil
	usecase := Service{
		application: serviceApplicationFake{app: app, version: version, declarations: []model.VersionComponent{declaration}},
		service:     serviceStoreFake{components: []model.ServiceComponent{component}},
		deployment:  deploymentStoreFake{},
	}
	view, err := usecase.serviceView(context.Background(), model.ServiceListItem{
		Id: service.Id, ApplicationId: service.ApplicationId, VersionId: service.VersionId, InstanceKey: service.InstanceKey,
		Status: service.Status, ApplicationName: app.Name, ApplicationCode: app.Code, ApplicationKind: app.Kind, VersionLabel: version.Label,
	})
	if err != nil {
		t.Fatalf("service view must remain accessible: %v", err)
	}
	if !view.PendingDeploy || !strings.Contains(view.EffectiveError, "requires service environment REQUIRED_VALUE") {
		t.Fatalf("service view must report the missing environment: %#v", view)
	}
}

func TestNormalizeServiceEnv(t *testing.T) {
	items, err := normalizeServiceEnv([]model.ServiceEnv{{Key: " SHARED_VALUE ", Value: "value"}})
	if err != nil || len(items) != 1 || items[0].Key != "SHARED_VALUE" {
		t.Fatalf("normalized environment = %#v, %v", items, err)
	}
	for _, input := range [][]model.ServiceEnv{
		{{Key: ""}},
		{{Key: "invalid.key"}},
		{{Key: "VALUE"}, {Key: "VALUE"}},
	} {
		if _, err := normalizeServiceEnv(input); err == nil {
			t.Fatalf("environment %#v must be rejected", input)
		}
	}
}

func TestRemapServiceComponentsKeepsMountOverlayWithItsTargetAfterReorder(t *testing.T) {
	overrideSource := "/srv/runtime-data"
	mappings := []model.ServiceComponent{{
		Id: "service-component-1", ServiceId: "service-1", SourceVersionComponentId: "component-v1", ComponentName: "web",
		Mounts: []model.ServiceComponentMount{{Target: "/data", Source: &overrideSource, State: model.ServiceComponentOverlayOverride}},
	}}
	declarations := []model.VersionComponent{{
		Id: "component-v2", Name: "web", Image: "nginx:latest",
		Mounts: []model.VersionComponentMount{
			{SourceType: "directory", Source: "cache", Target: "/cache"},
			{SourceType: "directory", Source: "data", Target: "/data"},
		},
	}}

	if err := remapServiceComponents(mappings, declarations); err != nil {
		t.Fatal(err)
	}
	if mappings[0].SourceVersionComponentId != "component-v2" {
		t.Fatalf("component was not remapped: %#v", mappings[0])
	}
	plan, _, err := deploymentsvc.BuildEffectiveServicePlan(
		model.Application{Id: "app-1", Code: "example", Kind: status.ApplicationKindStandard},
		model.Version{Id: "version-v2", ApplicationId: "app-1"},
		model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-v2"},
		declarations,
		mappings,
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Components[0].Mounts[0].Target != "/cache" || plan.Components[0].Mounts[0].Source != "cache" {
		t.Fatalf("cache mount received the data overlay: %#v", plan.Components[0].Mounts)
	}
	if plan.Components[0].Mounts[1].Target != "/data" || plan.Components[0].Mounts[1].Source != overrideSource {
		t.Fatalf("data mount lost its overlay: %#v", plan.Components[0].Mounts)
	}
}

func TestRemapServiceComponentsRejectsMissingMountTarget(t *testing.T) {
	deleted := model.ServiceComponentOverlayDeleted
	mappings := []model.ServiceComponent{{
		Id: "service-component-1", ServiceId: "service-1", SourceVersionComponentId: "component-v1", ComponentName: "web",
		Mounts: []model.ServiceComponentMount{{Target: "/removed", State: deleted}},
	}}
	declarations := []model.VersionComponent{{
		Id: "component-v2", Name: "web", Image: "nginx:latest",
		Mounts: []model.VersionComponentMount{{SourceType: "directory", Source: "data", Target: "/data"}},
	}}

	err := remapServiceComponents(mappings, declarations)
	if err == nil || !strings.Contains(err.Error(), "mount target /removed is not declared") {
		t.Fatalf("remap error = %v, want missing target rejection", err)
	}
	if mappings[0].SourceVersionComponentId != "component-v1" {
		t.Fatalf("failed remap must not update source declaration: %#v", mappings[0])
	}
}

func serviceViewFixture() (model.Service, model.Application, model.Version, model.VersionComponent, model.ServiceComponent) {
	baseValue, overrideValue := "version-value", "runtime-value"
	service := model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-1", InstanceKey: "default", Status: status.ServiceStatusStopped}
	app := model.Application{Id: "app-1", Name: "Example", Code: "example", Kind: status.ApplicationKindStandard}
	version := model.Version{Id: "version-1", ApplicationId: "app-1", Label: "v1"}
	declaration := model.VersionComponent{Id: "component-1", VersionId: version.Id, Name: "web", Image: "nginx:latest", Env: []model.VersionComponentEnv{{Key: "APP_VALUE", Value: baseValue}}}
	component := model.ServiceComponent{
		Id: "service-component-1", ServiceId: service.Id, SourceVersionComponentId: declaration.Id, ComponentName: declaration.Name,
		Env: []model.ServiceComponentEnv{{Key: "APP_VALUE", Value: &overrideValue, State: model.ServiceComponentOverlayOverride}},
	}
	return service, app, version, declaration, component
}

func buildFixturePlan(app model.Application, version model.Version, service model.Service, declaration model.VersionComponent, component model.ServiceComponent) (model.EffectiveServicePlan, string, error) {
	return deploymentsvc.BuildEffectiveServicePlan(app, version, service, []model.VersionComponent{declaration}, []model.ServiceComponent{component}, nil, nil)
}

func stringPtr(value string) *string {
	return &value
}

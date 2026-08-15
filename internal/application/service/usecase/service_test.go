package servicesvc

import (
	"context"
	"strings"
	"testing"

	deploymentsvc "github.com/leoninew/pomelo-orbit/internal/application/deployment/usecase"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
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

type deleteServiceStoreFake struct {
	repository.ServiceStore
	service   model.Service
	deletedId string
}

func (f *deleteServiceStoreFake) Service(_ context.Context, _ string) (model.Service, error) {
	return f.service, nil
}

func (f *deleteServiceStoreFake) DeleteService(_ context.Context, id string) error {
	f.deletedId = id
	return nil
}

func (f deploymentStoreFake) HasActiveDeployment(_ context.Context, _ string) (bool, error) {
	return f.active, nil
}

func (f deploymentStoreFake) LatestSuccessfulDeploymentPlanHash(_ context.Context, _ string) (*string, error) {
	return f.planHash, nil
}

func TestDeleteServiceAllowsStoppedAndFaultedService(t *testing.T) {
	for _, serviceStatus := range []string{status.ServiceStatusStopped, status.ServiceStatusFaulted} {
		t.Run(serviceStatus, func(t *testing.T) {
			store := &deleteServiceStoreFake{service: model.Service{Id: "service-1", ApplicationId: "application-1", Status: serviceStatus}}
			usecase := Service{
				application: serviceApplicationFake{app: model.Application{Id: "application-1"}},
				service:     store,
			}

			if err := usecase.DeleteService(context.Background(), "user-1", "service-1"); err != nil {
				t.Fatalf("DeleteService() error = %v", err)
			}
			if store.deletedId != "service-1" {
				t.Fatalf("deleted service = %q, want service-1", store.deletedId)
			}
		})
	}
}

func TestDeleteServiceRejectsRunningService(t *testing.T) {
	store := &deleteServiceStoreFake{service: model.Service{Id: "service-1", ApplicationId: "application-1", Status: status.ServiceStatusRunning}}
	usecase := Service{
		application: serviceApplicationFake{app: model.Application{Id: "application-1"}},
		service:     store,
	}

	if err := usecase.DeleteService(context.Background(), "user-1", "service-1"); err == nil {
		t.Fatal("DeleteService() error = nil, want validation error")
	}
	if store.deletedId != "" {
		t.Fatalf("deleted service = %q, want none", store.deletedId)
	}
}

func TestServiceViewPendingDeployComparesEffectivePlanHash(t *testing.T) {
	service, app, version, declaration, component := serviceViewFixture()
	listenPort := 18080
	mode := "local"
	declaration.Endpoints = []model.VersionComponentEndpoint{{Protocol: "http", ContainerPort: 8080, Mode: "internal"}}
	component.Endpoints = []model.ServiceComponentEndpoint{{Protocol: "http", ContainerPort: 8080, Mode: &mode, ListenPort: &listenPort, State: model.ServiceComponentOverlayOverride}}
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
			if len(view.EffectiveComponents) != 1 || len(view.EffectiveComponents[0].Endpoints) != 1 {
				t.Fatalf("effective components = %#v", view.EffectiveComponents)
			}
			endpoint := view.EffectiveComponents[0].Endpoints[0]
			if endpoint.Protocol != "http" || endpoint.ContainerPort != 8080 || endpoint.Mode != mode || endpoint.ListenPort == nil || *endpoint.ListenPort != listenPort {
				t.Fatalf("effective endpoint = %#v", endpoint)
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
	declaration.Endpoints = []model.VersionComponentEndpoint{{Protocol: "http", ContainerPort: 80, Mode: "gateway"}}
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

func TestGetServiceComponentReturnsVersionAndServiceComponentValues(t *testing.T) {
	service, app, version, declaration, component := serviceViewFixture()
	usecase := Service{
		application: serviceApplicationFake{app: app, version: version, declarations: []model.VersionComponent{declaration}},
		service:     serviceStoreFake{service: service, component: component},
	}
	detail, err := usecase.GetServiceComponent(context.Background(), "user-1", service.Id, component.Id)
	if err != nil {
		t.Fatal(err)
	}
	if detail.ServiceComponent.Id != component.Id || detail.VersionComponent.Id != declaration.Id {
		t.Fatalf("component detail lost source views: %#v", detail)
	}
	if len(detail.ServiceComponent.Env) != 1 || detail.ServiceComponent.Env[0].Value == nil || *detail.ServiceComponent.Env[0].Value != "runtime-value" {
		t.Fatalf("service component values = %#v", detail.ServiceComponent)
	}
}

func TestGetServiceComponentDoesNotResolveServiceEnvironment(t *testing.T) {
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
	if len(detail.ServiceComponent.Env) != 0 {
		t.Fatalf("service value must not become a component overlay: %#v", detail.ServiceComponent.Env)
	}
	if len(detail.VersionComponent.Env) != 1 || detail.VersionComponent.Env[0].Value != "${SHARED_VALUE}" {
		t.Fatalf("version component values = %#v", detail.VersionComponent.Env)
	}
}

func TestGetServiceComponentReturnsSourceValuesWithoutResolvingEnvironment(t *testing.T) {
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
	if detail.ServiceComponent.Id != component.Id || detail.VersionComponent.Id != declaration.Id {
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

func TestNormalizeOverlayAcceptsAbsoluteHostPathMountSource(t *testing.T) {
	source := "D:/SourceCodes/mywork/PomeloOrbit-go/data/backup/bge-m3"
	sourceIsHostPath := true
	component := model.ServiceComponent{Mounts: []model.ServiceComponentMount{{
		Target: "/models", Source: &source, SourceIsHostPath: &sourceIsHostPath, State: model.ServiceComponentOverlayOverride,
	}}}
	declaration := model.VersionComponent{Name: "tei", Mounts: []model.VersionComponentMount{{
		SourceType: "directory", Source: "models/bge-m3", Target: "/models",
	}}}
	if err := normalizeOverlay(&component, declaration); err != nil {
		t.Fatalf("normalizeOverlay() error = %v", err)
	}
	if len(component.Mounts) != 1 || component.Mounts[0].SourceIsHostPath == nil || !*component.Mounts[0].SourceIsHostPath {
		t.Fatalf("mount overlay = %#v", component.Mounts)
	}
}

func TestNormalizeOverlayRejectsAbsoluteLogicalMountSource(t *testing.T) {
	source := "D:/SourceCodes/mywork/PomeloOrbit-go/data/backup/bge-m3"
	sourceIsHostPath := false
	component := model.ServiceComponent{Mounts: []model.ServiceComponentMount{{
		Target: "/models", Source: &source, SourceIsHostPath: &sourceIsHostPath, State: model.ServiceComponentOverlayOverride,
	}}}
	declaration := model.VersionComponent{Name: "tei", Mounts: []model.VersionComponentMount{{
		SourceType: "directory", Source: "models/bge-m3", Target: "/models",
	}}}
	err := normalizeOverlay(&component, declaration)
	if err == nil || !strings.Contains(err.Error(), "component tei mount /models: source must be a relative path") {
		t.Fatalf("normalizeOverlay() error = %v", err)
	}
}

func TestRemapServiceComponentsKeepsMountOverlayWithItsTargetAfterReorder(t *testing.T) {
	overrideSource := "/srv/runtime-data"
	overrideSourceIsHostPath := true
	mappings := []model.ServiceComponent{{
		Id: "service-component-1", ServiceId: "service-1", SourceVersionComponentId: "component-v1", ComponentName: "web",
		Mounts: []model.ServiceComponentMount{{Target: "/data", Source: &overrideSource, SourceIsHostPath: &overrideSourceIsHostPath, State: model.ServiceComponentOverlayOverride}},
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

func TestAlignServiceComponentsRebuildsMissingMappings(t *testing.T) {
	declarations := []model.VersionComponent{{
		Id: "component-new", Name: "traefik", Image: "traefik:3.6",
	}}
	aligned := alignServiceComponents("service-1", nil, declarations)
	if len(aligned) != 1 {
		t.Fatalf("aligned = %#v", aligned)
	}
	if aligned[0].ServiceId != "service-1" || aligned[0].SourceVersionComponentId != "component-new" || aligned[0].ComponentName != "traefik" {
		t.Fatalf("aligned mapping = %#v", aligned[0])
	}
	if aligned[0].Id == "" || aligned[0].Status != "active" {
		t.Fatalf("aligned mapping identity = %#v", aligned[0])
	}
}

func TestAlignServiceComponentsKeepsOverlayWhenSourceIdChanges(t *testing.T) {
	override := "runtime"
	existing := []model.ServiceComponent{{
		Id: "service-component-1", ServiceId: "service-1", SourceVersionComponentId: "component-old", ComponentName: "traefik",
		Env: []model.ServiceComponentEnv{{Key: "FOO", Value: &override, State: model.ServiceComponentOverlayOverride}},
	}}
	declarations := []model.VersionComponent{{
		Id: "component-new", Name: "traefik", Image: "traefik:3.6",
	}}
	aligned := alignServiceComponents("service-1", existing, declarations)
	if len(aligned) != 1 {
		t.Fatalf("aligned = %#v", aligned)
	}
	if aligned[0].Id != "service-component-1" || aligned[0].SourceVersionComponentId != "component-new" {
		t.Fatalf("aligned mapping = %#v", aligned[0])
	}
	if len(aligned[0].Env) != 1 || aligned[0].Env[0].Value == nil || *aligned[0].Env[0].Value != override {
		t.Fatalf("overlay lost: %#v", aligned[0].Env)
	}
}

func TestAlignComponentMappingsToVersionRewritesBoundServices(t *testing.T) {
	store := &alignServiceStoreFake{
		services: []model.Service{
			{Id: "service-bound", ApplicationId: "app-1", VersionId: "version-1"},
			{Id: "service-other", ApplicationId: "app-1", VersionId: "version-2"},
		},
		components: map[string][]model.ServiceComponent{
			"service-bound": nil,
			"service-other": {{Id: "keep", ServiceId: "service-other", SourceVersionComponentId: "other", ComponentName: "web"}},
		},
	}
	usecase := Service{
		application: serviceApplicationFake{
			version: model.Version{Id: "version-1", ApplicationId: "app-1"},
			declarations: []model.VersionComponent{{
				Id: "component-1", Name: "traefik", Image: "traefik:3.6",
			}},
		},
		service: store,
	}
	if err := usecase.AlignComponentMappingsToVersion(context.Background(), "app-1", "version-1"); err != nil {
		t.Fatal(err)
	}
	if len(store.updated) != 1 || store.updated[0].service.Id != "service-bound" {
		t.Fatalf("updated services = %#v", store.updated)
	}
	if len(store.updated[0].components) != 1 || store.updated[0].components[0].SourceVersionComponentId != "component-1" {
		t.Fatalf("updated mappings = %#v", store.updated[0].components)
	}
	if _, touched := store.components["service-other"]; !touched {
		t.Fatal("unrelated service mapping must remain")
	}
}

type alignServiceStoreFake struct {
	repository.ServiceStore
	services   []model.Service
	components map[string][]model.ServiceComponent
	updated    []alignServiceUpdate
}

type alignServiceUpdate struct {
	service    model.Service
	components []model.ServiceComponent
}

func (f *alignServiceStoreFake) ListServicesByApplication(context.Context, string) ([]model.Service, error) {
	return append([]model.Service(nil), f.services...), nil
}

func (f *alignServiceStoreFake) ServiceComponentsByService(_ context.Context, serviceId string) ([]model.ServiceComponent, error) {
	return append([]model.ServiceComponent(nil), f.components[serviceId]...), nil
}

func (f *alignServiceStoreFake) UpdateServiceConfiguration(_ context.Context, svc model.Service, components []model.ServiceComponent) error {
	f.updated = append(f.updated, alignServiceUpdate{service: svc, components: append([]model.ServiceComponent(nil), components...)})
	f.components[svc.Id] = append([]model.ServiceComponent(nil), components...)
	return nil
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

package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestBuildEffectiveServicePlanMergesSparseOverrides(t *testing.T) {
	baseCPUs := "1"
	overrideCPUs := "2"
	baseSource := "/var/lib/postgres"
	overrideSource := "/srv/postgres"
	overrideSourceIsHostPath := true
	basePort := 5432
	overridePort := 15432
	mode := "host"
	declaration := model.VersionComponent{
		Id: "component-db", VersionId: "version-1", Name: "db", Image: "postgres:16",
		Env:       []model.VersionComponentEnv{{Key: "POSTGRES_DB", Value: "orbit"}},
		Mounts:    []model.VersionComponentMount{{SourceType: mountSourceDirectory, Source: baseSource, Target: "/var/lib/postgresql/data"}},
		Resources: &model.VersionComponentResources{LimitCPUs: &baseCPUs},
		Endpoints: []model.VersionComponentEndpoint{{Protocol: "tcp", ContainerPort: basePort, Mode: "local", ListenPort: &basePort}},
	}
	overrideValue := "orbit-runtime"
	plan, hash, err := BuildEffectiveServicePlan(
		model.Application{Id: "app-1", Code: "demo", Kind: status.ApplicationKindStandard},
		model.Version{Id: "version-1", ApplicationId: "app-1", Label: "v1"},
		model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-1", InstanceKey: "default"},
		[]model.VersionComponent{declaration},
		[]model.ServiceComponent{{
			Id: "service-component-db", ServiceId: "service-1", SourceVersionComponentId: declaration.Id, ComponentName: declaration.Name,
			Env:       []model.ServiceComponentEnv{{Key: "POSTGRES_DB", Value: &overrideValue, State: model.ServiceComponentOverlayOverride}},
			Mounts:    []model.ServiceComponentMount{{Target: "/var/lib/postgresql/data", Source: &overrideSource, SourceIsHostPath: &overrideSourceIsHostPath, State: model.ServiceComponentOverlayOverride}},
			Resources: &model.ServiceComponentResources{LimitCPUs: &overrideCPUs, State: model.ServiceComponentOverlayOverride},
			Endpoints: []model.ServiceComponentEndpoint{{Protocol: "tcp", ContainerPort: basePort, Mode: &mode, ListenPort: &overridePort, State: model.ServiceComponentOverlayOverride}},
		}},
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if hash == "" {
		t.Fatal("effective plan hash must be populated")
	}
	component := plan.Components[0]
	if component.Env[0].Value != overrideValue || component.Mounts[0].Source != overrideSource {
		t.Fatalf("unexpected merged component: %#v", component)
	}
	if !component.Mounts[0].SourceIsHostPath {
		t.Fatal("mount source must be rendered as a host path")
	}
	if component.Resources.LimitCPUs == nil || *component.Resources.LimitCPUs != overrideCPUs {
		t.Fatalf("resource override = %#v", component.Resources)
	}
	endpoint := component.Endpoints[0]
	if endpoint.Protocol != "tcp" || endpoint.ContainerPort != basePort || endpoint.Mode != mode || endpoint.ListenPort == nil || *endpoint.ListenPort != overridePort {
		t.Fatalf("endpoint overlay must retain the declaration contract: %#v", endpoint)
	}
}

func TestEffectiveServicePlanHashExcludesGatewayConfiguration(t *testing.T) {
	plan := model.EffectiveServicePlan{
		Application: model.Application{Code: "demo", Kind: status.ApplicationKindStandard},
		Version:     model.Version{Label: "v1"},
		Service:     model.Service{InstanceKey: "default"},
		Components: []model.EffectiveServiceComponent{{
			Name: "web", Image: "nginx:latest",
			Endpoints: []model.VersionComponentEndpoint{{Protocol: "http", ContainerPort: 80, Mode: "gateway"}},
		}},
	}
	withoutGateway, err := EffectiveServicePlanHash(plan)
	if err != nil {
		t.Fatal(err)
	}
	plan.Gateway = &model.GatewayConfig{RestApiUrl: "http://127.0.0.1:8080", BaseDomain: "example.com", DefaultEntrypoint: "websecure", TLSMode: "letsencrypt"}
	withGateway, err := EffectiveServicePlanHash(plan)
	if err != nil {
		t.Fatal(err)
	}
	if withGateway != withoutGateway {
		t.Fatalf("gateway configuration changed plan hash: without=%s with=%s", withoutGateway, withGateway)
	}
}

func TestBuildVersionPreviewPlanUsesVersionDeclarations(t *testing.T) {
	listenPort := 8080
	app := model.Application{Id: "app-1", Code: "demo", Kind: status.ApplicationKindStandard}
	version := model.Version{Id: "version-1", ApplicationId: app.Id, Label: "v1"}
	declarations := []model.VersionComponent{{
		Id: "component-web", VersionId: version.Id, Name: "web", Image: "nginx:latest",
		Env:       []model.VersionComponentEnv{{Key: "MODE", Value: "version"}},
		Endpoints: []model.VersionComponentEndpoint{{Protocol: "http", ContainerPort: 80, Mode: "local", ListenPort: &listenPort}},
	}}

	plan, err := BuildVersionPreviewPlan(app, version, declarations, nil)
	if err != nil {
		t.Fatalf("BuildVersionPreviewPlan returned error: %v", err)
	}
	if plan.Service.Id != "" || plan.Service.InstanceKey != versionPreviewInstanceKey || plan.Service.Code != "demo-preview" {
		t.Fatalf("preview must not use a persisted service: %+v", plan.Service)
	}
	if len(plan.Components) != 1 {
		t.Fatalf("components = %+v", plan.Components)
	}
	component := plan.Components[0]
	if component.Image != "nginx:latest" || len(component.Env) != 1 || component.Env[0].Value != "version" {
		t.Fatalf("preview component = %+v", component)
	}
	if len(component.Endpoints) != 1 || component.Endpoints[0].ListenPort == nil || *component.Endpoints[0].ListenPort != listenPort {
		t.Fatalf("preview endpoints = %+v", component.Endpoints)
	}

	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: plan})
	if err != nil {
		t.Fatalf("RenderCompose returned error: %v", err)
	}
	if !strings.Contains(compose, "MODE: version") || !strings.Contains(compose, "127.0.0.1:8080:80") {
		t.Fatalf("version declaration was not rendered:\n%s", compose)
	}
}

func TestBuildVersionPreviewPlanRendersGatewayHTTPHost(t *testing.T) {
	app := model.Application{Id: "app-1", Code: "demo", Kind: status.ApplicationKindStandard}
	version := model.Version{Id: "version-1", ApplicationId: app.Id, Label: "v1"}
	declarations := []model.VersionComponent{{
		Id: "component-web", VersionId: version.Id, Name: "web", Image: "nginx:latest",
		Endpoints: []model.VersionComponentEndpoint{{Protocol: "http", ContainerPort: 80, Mode: "gateway"}},
	}}

	plan, err := BuildVersionPreviewPlan(app, version, declarations, &model.GatewayConfig{BaseDomain: "example.test", DefaultEntrypoint: "web"})
	if err != nil {
		t.Fatalf("BuildVersionPreviewPlan returned error: %v", err)
	}
	compose, err := Service{}.RenderCompose(context.Background(), RenderInput{Plan: plan})
	if err != nil {
		t.Fatalf("RenderCompose returned error: %v", err)
	}
	if !strings.Contains(compose, "Host(`web.demo-preview.example.test`)") {
		t.Fatalf("gateway host was not rendered:\n%s", compose)
	}
}

func TestBuildEffectiveServicePlanAppliesTombstones(t *testing.T) {
	value := "value"
	source := "/srv/data"
	cpus := "1"
	port := 8080
	declaration := model.VersionComponent{
		Id: "component-web", VersionId: "version-1", Name: "web", Image: "nginx",
		Env:       []model.VersionComponentEnv{{Key: "FEATURE", Value: value}},
		Mounts:    []model.VersionComponentMount{{SourceType: mountSourceDirectory, Source: source, Target: "/data"}},
		Resources: &model.VersionComponentResources{LimitCPUs: &cpus},
		Endpoints: []model.VersionComponentEndpoint{{Protocol: "http", ContainerPort: port, Mode: "host", ListenPort: &port}},
	}
	plan, _, err := BuildEffectiveServicePlan(
		model.Application{Id: "app-1", Code: "demo", Kind: status.ApplicationKindStandard},
		model.Version{Id: "version-1", ApplicationId: "app-1"},
		model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-1"},
		[]model.VersionComponent{declaration},
		[]model.ServiceComponent{{
			Id: "service-component-web", ServiceId: "service-1", SourceVersionComponentId: declaration.Id, ComponentName: declaration.Name,
			Env:       []model.ServiceComponentEnv{{Key: "FEATURE", State: model.ServiceComponentOverlayDeleted}},
			Mounts:    []model.ServiceComponentMount{{Target: "/data", State: model.ServiceComponentOverlayDeleted}},
			Resources: &model.ServiceComponentResources{State: model.ServiceComponentOverlayDeleted},
			Endpoints: []model.ServiceComponentEndpoint{{Protocol: "http", ContainerPort: port, State: model.ServiceComponentOverlayDeleted}},
		}},
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	component := plan.Components[0]
	if len(component.Env) != 0 || len(component.Mounts) != 0 || component.Resources != nil {
		t.Fatalf("tombstones must suppress declared runtime values: %#v", component)
	}
	endpoint := component.Endpoints[0]
	if endpoint.Mode != "internal" || endpoint.BindAddress != nil || endpoint.ListenPort != nil || endpoint.Entrypoint != nil || endpoint.PathPrefix != nil {
		t.Fatalf("deleted endpoint = %#v, want internal without runtime values", endpoint)
	}
}

func TestBuildEffectiveServicePlanRejectsUndeclaredOverlay(t *testing.T) {
	_, _, err := BuildEffectiveServicePlan(
		model.Application{Id: "app-1", Code: "demo", Kind: status.ApplicationKindStandard},
		model.Version{Id: "version-1", ApplicationId: "app-1"},
		model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-1"},
		[]model.VersionComponent{{Id: "component-web", VersionId: "version-1", Name: "web", Image: "nginx"}},
		[]model.ServiceComponent{{
			Id: "service-component-web", ServiceId: "service-1", SourceVersionComponentId: "component-web", ComponentName: "web",
			Env: []model.ServiceComponentEnv{{Key: "UNDECLARED", State: model.ServiceComponentOverlayDeleted}},
		}},
		nil,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "undeclared environment") {
		t.Fatalf("error = %v, want undeclared environment rejection", err)
	}
}

func TestBuildEffectiveServicePlanKeepsGatewayApiEndpointConfiguration(t *testing.T) {
	defaultPort := 18080
	listenPort := 8080
	mode := "host"
	plan, _, err := BuildEffectiveServicePlan(
		model.Application{Id: "gateway-app", Code: "traefik", Kind: status.ApplicationKindGateway},
		model.Version{Id: "version-1", ApplicationId: "gateway-app"},
		model.Service{Id: "service-1", ApplicationId: "gateway-app", VersionId: "version-1"},
		[]model.VersionComponent{{
			Id: "component-traefik", VersionId: "version-1", Name: "traefik", Image: "traefik:3.6",
			Endpoints: []model.VersionComponentEndpoint{{
				Protocol: "tcp", ContainerPort: 8080, Mode: "local", ListenPort: &defaultPort,
			}},
		}},
		[]model.ServiceComponent{{
			Id: "service-component-traefik", ServiceId: "service-1", SourceVersionComponentId: "component-traefik", ComponentName: "traefik",
			Endpoints: []model.ServiceComponentEndpoint{{
				Protocol: "tcp", ContainerPort: 8080, Mode: &mode, ListenPort: &listenPort, State: model.ServiceComponentOverlayOverride,
			}},
		}},
		nil,
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	endpoint := plan.Components[0].Endpoints[0]
	if endpoint.Mode != "host" || endpoint.ListenPort == nil || *endpoint.ListenPort != listenPort {
		t.Fatalf("gateway api endpoint configuration changed: %#v", endpoint)
	}
}

func TestBuildEffectiveServicePlanResolvesSharedServiceEnvironment(t *testing.T) {
	declarations := []model.VersionComponent{
		{Id: "component-api", VersionId: "version-1", Name: "api", Image: "api:latest", Env: []model.VersionComponentEnv{{Key: "DATABASE_PASSWORD", Value: "${SHARED_PASSWORD:?required}"}}},
		{Id: "component-worker", VersionId: "version-1", Name: "worker", Image: "worker:latest", Env: []model.VersionComponentEnv{{Key: "DATABASE_PASSWORD", Value: "${SHARED_PASSWORD}"}}},
		{Id: "component-sidecar", VersionId: "version-1", Name: "sidecar", Image: "sidecar:latest", Env: []model.VersionComponentEnv{{Key: "DATABASE_PASSWORD", Value: "${SHARED_PASSWORD}"}}},
	}
	override := "component-only"
	plan, _, err := BuildEffectiveServicePlan(
		model.Application{Id: "app-1", Code: "demo", Kind: status.ApplicationKindStandard},
		model.Version{Id: "version-1", ApplicationId: "app-1"},
		model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-1"},
		declarations,
		[]model.ServiceComponent{
			{Id: "service-component-api", ServiceId: "service-1", SourceVersionComponentId: "component-api", ComponentName: "api"},
			{Id: "service-component-worker", ServiceId: "service-1", SourceVersionComponentId: "component-worker", ComponentName: "worker"},
			{Id: "service-component-sidecar", ServiceId: "service-1", SourceVersionComponentId: "component-sidecar", ComponentName: "sidecar", Env: []model.ServiceComponentEnv{{Key: "DATABASE_PASSWORD", Value: &override, State: model.ServiceComponentOverlayOverride}}},
		},
		[]model.ServiceEnv{{Key: "SHARED_PASSWORD", Value: "shared-value"}},
		nil,
	)
	if err != nil {
		t.Fatal(err)
	}
	if plan.Components[0].Env[0].Value != "shared-value" || plan.Components[1].Env[0].Value != "shared-value" {
		t.Fatalf("shared service environment was not applied: %#v", plan.Components)
	}
	if plan.Components[2].Env[0].Value != override {
		t.Fatalf("component override must take precedence: %#v", plan.Components[2].Env)
	}
}

func TestBuildEffectiveServicePlanRejectsMissingRequiredServiceEnvironment(t *testing.T) {
	_, _, err := BuildEffectiveServicePlan(
		model.Application{Id: "app-1", Code: "demo", Kind: status.ApplicationKindStandard},
		model.Version{Id: "version-1", ApplicationId: "app-1"},
		model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-1"},
		[]model.VersionComponent{{Id: "component-api", VersionId: "version-1", Name: "api", Image: "api:latest", Env: []model.VersionComponentEnv{{Key: "DATABASE_PASSWORD", Value: "${SHARED_PASSWORD:?required}"}}}},
		[]model.ServiceComponent{{Id: "service-component-api", ServiceId: "service-1", SourceVersionComponentId: "component-api", ComponentName: "api"}},
		nil,
		nil,
	)
	if err == nil || !strings.Contains(err.Error(), "requires service environment SHARED_PASSWORD") {
		t.Fatalf("error = %v, want missing service environment", err)
	}
}

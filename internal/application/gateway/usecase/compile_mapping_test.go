package gatewaysvc

import (
	"context"
	"testing"
	"time"

	gatewayport "github.com/leoninew/pomelo-orbit/internal/application/gateway/port"
	servicesvc "github.com/leoninew/pomelo-orbit/internal/application/service/usecase"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func testTraefikConfig() config.TraefikConfig {
	return config.TraefikConfig{
		CertDir:          "data/deployment/traefik/data/certs",
		Image:            "traefik:3.6",
		RestApiUrl:       "http://localhost:8080",
		BaseDomain:       "lvh.me",
		RestReadyTimeout: 20 * time.Second,
	}
}

func TestCompileGatewayToVersionRestoresServiceComponentMappings(t *testing.T) {
	app := model.Application{Id: "gateway-1", Code: "traefik", Kind: status.ApplicationKindGateway}
	version := model.Version{Id: "version-1", ApplicationId: app.Id, Label: "traefik:3.6", Status: status.VersionStatusUnpublished}
	component := model.VersionComponent{
		Id: "component-1", VersionId: version.Id, Name: "traefik", Image: "traefik:3.6", PullPolicy: "missing",
		Endpoints: []model.VersionComponentEndpoint{
			{Protocol: "tcp", ContainerPort: 80, Mode: "host", ListenPort: intRef(80)},
		},
	}
	svc := model.Service{Id: "service-1", ApplicationId: app.Id, VersionId: version.Id, InstanceKey: "default", Code: "traefik-default", Status: status.ServiceStatusStopped}
	application := &compileApplicationStore{
		application: app,
		versions:    []model.Version{version},
		components:  map[string][]model.VersionComponent{version.Id: {component}},
	}
	serviceStore := &compileServiceStore{
		services:   []model.Service{svc},
		components: map[string][]model.ServiceComponent{svc.Id: nil},
	}
	gateway := Service{
		application: application,
		service:     serviceStore,
		serviceCommands: servicesvc.New(
			nil,
			application,
			serviceStore,
			nil,
		),
		traefik: testTraefikConfig(),
	}

	versionId, err := gateway.CompileGatewayToVersion(context.Background(), app, model.GatewayConfig{ApplicationId: app.Id}, 16379)
	if err != nil {
		t.Fatal(err)
	}
	if versionId != version.Id {
		t.Fatalf("version id = %q, want %q", versionId, version.Id)
	}
	if len(application.replaced[version.Id]) == 0 {
		t.Fatal("expected compiled components to be written")
	}
	mappings := serviceStore.components[svc.Id]
	if len(mappings) != 1 {
		t.Fatalf("service mappings = %#v", mappings)
	}
	if mappings[0].ComponentName != "traefik" || mappings[0].SourceVersionComponentId == "" {
		t.Fatalf("restored mapping = %#v", mappings[0])
	}
	compiled := application.components[version.Id]
	if len(compiled) != 1 || mappings[0].SourceVersionComponentId != compiled[0].Id {
		t.Fatalf("mapping source %q does not match compiled component %#v", mappings[0].SourceVersionComponentId, compiled)
	}
}

type compileApplicationStore struct {
	gatewayport.ApplicationStore
	application model.Application
	versions    []model.Version
	components  map[string][]model.VersionComponent
	replaced    map[string][]model.VersionComponent
}

func (s *compileApplicationStore) Application(context.Context, string) (model.Application, error) {
	return s.application, nil
}

func (s *compileApplicationStore) ListVersions(context.Context, string) ([]model.Version, error) {
	return append([]model.Version(nil), s.versions...), nil
}

func (s *compileApplicationStore) Version(_ context.Context, id string) (model.Version, error) {
	for _, version := range s.versions {
		if version.Id == id {
			return version, nil
		}
	}
	return model.Version{}, repository.ErrNotFound
}

func (s *compileApplicationStore) VersionComponentsByVersion(_ context.Context, versionId string) ([]model.VersionComponent, error) {
	return append([]model.VersionComponent(nil), s.components[versionId]...), nil
}

func (s *compileApplicationStore) ReplaceVersionComponents(_ context.Context, versionId string, components []model.VersionComponent) error {
	if s.replaced == nil {
		s.replaced = map[string][]model.VersionComponent{}
	}
	copied := append([]model.VersionComponent(nil), components...)
	s.replaced[versionId] = copied
	s.components[versionId] = copied
	return nil
}

type compileServiceStore struct {
	repository.ServiceStore
	services   []model.Service
	components map[string][]model.ServiceComponent
}

func (s *compileServiceStore) ListServicesByApplication(context.Context, string) ([]model.Service, error) {
	return append([]model.Service(nil), s.services...), nil
}

func (s *compileServiceStore) ServiceComponentsByService(_ context.Context, serviceId string) ([]model.ServiceComponent, error) {
	return append([]model.ServiceComponent(nil), s.components[serviceId]...), nil
}

func (s *compileServiceStore) UpdateServiceConfiguration(_ context.Context, svc model.Service, components []model.ServiceComponent) error {
	s.components[svc.Id] = append([]model.ServiceComponent(nil), components...)
	return nil
}

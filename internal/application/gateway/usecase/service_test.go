package gatewaysvc

import (
	"context"
	"strings"
	"testing"

	gatewaydto "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type gatewayTestStore struct {
	applications map[string]model.Application
	versions     map[string][]model.Version
	components   map[string][]model.VersionComponent
	exposes      map[string][]model.VersionExpose
	configs      map[string]model.GatewayConfig
	services     map[string][]model.Service
	removedDirs  []string
}

func newGatewayTestStore() *gatewayTestStore {
	return &gatewayTestStore{
		applications: map[string]model.Application{},
		versions:     map[string][]model.Version{},
		components:   map[string][]model.VersionComponent{},
		exposes:      map[string][]model.VersionExpose{},
		configs:      map[string]model.GatewayConfig{},
		services:     map[string][]model.Service{},
	}
}

func newGatewayTestService() (Service, *gatewayTestStore) {
	store := newGatewayTestStore()
	return New(store, store, store, store, store), store
}

func (s *gatewayTestStore) Project(context.Context, string) (model.Project, error) {
	return model.Project{}, nil
}

func (s *gatewayTestStore) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

func (s *gatewayTestStore) ListApplications(_ context.Context, projectId *string, page int, perPage int, search string, kind string) (repository.Page[model.Application], error) {
	items := make([]model.Application, 0, len(s.applications))
	for _, app := range s.applications {
		if projectId != nil && (app.ProjectId == nil || *app.ProjectId != *projectId) {
			continue
		}
		if kind != "" && app.Kind != kind {
			continue
		}
		if search != "" && !strings.Contains(app.Name, search) && !strings.Contains(app.Code, search) {
			continue
		}
		items = append(items, app)
	}
	return repository.Page[model.Application]{Items: items, Total: len(items), Page: page, PerPage: perPage}, nil
}

func (s *gatewayTestStore) Application(_ context.Context, id string) (model.Application, error) {
	app, ok := s.applications[id]
	if !ok {
		return model.Application{}, repository.ErrNotFound
	}
	return app, nil
}

func (s *gatewayTestStore) ApplicationByName(_ context.Context, name string) (model.Application, error) {
	for _, app := range s.applications {
		if app.Name == name {
			return app, nil
		}
	}
	return model.Application{}, repository.ErrNotFound
}

func (s *gatewayTestStore) ApplicationByCode(_ context.Context, code string) (model.Application, error) {
	for _, app := range s.applications {
		if app.Code == code {
			return app, nil
		}
	}
	return model.Application{}, repository.ErrNotFound
}

func (s *gatewayTestStore) CreateApplication(_ context.Context, app model.Application) error {
	s.applications[app.Id] = app
	return nil
}

func (s *gatewayTestStore) UpdateApplication(_ context.Context, app model.Application) error {
	if _, ok := s.applications[app.Id]; !ok {
		return repository.ErrNotFound
	}
	s.applications[app.Id] = app
	return nil
}

func (s *gatewayTestStore) DeleteApplication(_ context.Context, id string) error {
	delete(s.applications, id)
	delete(s.versions, id)
	delete(s.services, id)
	delete(s.configs, id)
	return nil
}

func (s *gatewayTestStore) ListVersions(_ context.Context, applicationId string) ([]model.Version, error) {
	return append([]model.Version(nil), s.versions[applicationId]...), nil
}

func (s *gatewayTestStore) CreateVersion(_ context.Context, version model.Version) error {
	s.versions[version.ApplicationId] = append(s.versions[version.ApplicationId], version)
	return nil
}

func (s *gatewayTestStore) VersionComponentsByVersion(_ context.Context, versionId string) ([]model.VersionComponent, error) {
	return append([]model.VersionComponent(nil), s.components[versionId]...), nil
}

func (s *gatewayTestStore) ReplaceVersionComponents(_ context.Context, versionId string, components []model.VersionComponent) error {
	s.components[versionId] = append([]model.VersionComponent(nil), components...)
	return nil
}

func (s *gatewayTestStore) VersionExposesByVersion(_ context.Context, versionId string) ([]model.VersionExpose, error) {
	return append([]model.VersionExpose(nil), s.exposes[versionId]...), nil
}

func (s *gatewayTestStore) GatewayConfig(_ context.Context, applicationId string) (model.GatewayConfig, error) {
	config, ok := s.configs[applicationId]
	if !ok {
		return model.GatewayConfig{}, repository.ErrNotFound
	}
	return config, nil
}

func (s *gatewayTestStore) ResolveActiveGatewayConfig(_ context.Context) (model.GatewayConfig, error) {
	for _, config := range s.configs {
		return config, nil
	}
	return model.GatewayConfig{}, repository.ErrNotFound
}

func (s *gatewayTestStore) UpsertGatewayConfig(_ context.Context, config model.GatewayConfig) error {
	s.configs[config.ApplicationId] = config
	return nil
}

func (s *gatewayTestStore) ListServicesByApplication(_ context.Context, applicationId string) ([]model.Service, error) {
	return append([]model.Service(nil), s.services[applicationId]...), nil
}

func (s *gatewayTestStore) RemoveAppDir(appCode string) error {
	s.removedDirs = append(s.removedDirs, appCode)
	return nil
}

func TestGatewayCRUDCompilesManagedVersionAndProjectsExposures(t *testing.T) {
	service, store := newGatewayTestService()
	context := context.Background()
	image := "traefik:v3"
	view, err := service.CreateGateway(context, "user-1", gatewaydto.GatewayCreateInput{
		ProjectId: "project-1", Code: "gateway", Name: "Gateway", RestApiUrl: "https://gateway.example.com/",
		BaseDomain: "example.com", Image: &image, ImagePullPolicy: "always",
	})
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}
	if view.Application.Kind != status.ApplicationKindGateway || view.Config.RestApiUrl != "https://gateway.example.com" {
		t.Fatalf("created gateway = %+v", view)
	}
	versions := store.versions[view.Application.Id]
	if len(versions) != 1 || versions[0].Status != status.VersionStatusUnpublished {
		t.Fatalf("managed version = %+v", versions)
	}
	if components := store.components[versions[0].Id]; len(components) != 1 || components[0].Name != gatewayManagedComponentName {
		t.Fatalf("managed components = %+v", components)
	}

	projectId := "project-1"
	store.applications["api-app"] = model.Application{Id: "api-app", ProjectId: &projectId, Code: "api", Name: "API", Kind: status.ApplicationKindStandard}
	store.services["api-app"] = []model.Service{{Id: "service-1", ApplicationId: "api-app", VersionId: "api-version", Status: status.ServiceStatusRunning}}
	store.exposes["api-version"] = []model.VersionExpose{
		{ComponentName: "web", Protocol: "http", Access: "public", ContainerPort: 80},
		{ComponentName: "postgres", Protocol: "tcp", Access: "local", ContainerPort: 5432},
	}
	view, err = service.GatewayForUser(context, "user-1", view.Application.Id)
	if err != nil {
		t.Fatalf("load gateway: %v", err)
	}
	if len(view.Exposures) != 2 || view.Exposures[0].ClientHint != "http://api.example.com" || view.Exposures[1].ClientHint != "127.0.0.1:5432" {
		t.Fatalf("gateway exposures = %+v", view.Exposures)
	}

	updatedName := "Gateway Updated"
	updatedImage := "traefik:v3.1"
	updated, err := service.UpdateGateway(context, "user-1", view.Application.Id, gatewaydto.GatewayUpdateInput{Name: &updatedName, Image: &updatedImage})
	if err != nil {
		t.Fatalf("update gateway: %v", err)
	}
	if updated.Application.Name != updatedName || updated.Config.Image == nil || *updated.Config.Image != updatedImage {
		t.Fatalf("updated gateway = %+v", updated)
	}
	if err := service.DeleteGateway(context, "user-1", updated.Application.Id, true); err != nil {
		t.Fatalf("delete gateway: %v", err)
	}
	if _, ok := store.applications[updated.Application.Id]; ok || len(store.removedDirs) != 1 {
		t.Fatalf("gateway was not fully deleted: apps=%+v dirs=%+v", store.applications, store.removedDirs)
	}
}

func TestCompileGatewayToVersionCreatesUniqueDraftAfterManagedVersionIsPublished(t *testing.T) {
	service, store := newGatewayTestService()
	context := context.Background()
	image := "traefik:v3"
	view, err := service.CreateGateway(context, "user-1", gatewaydto.GatewayCreateInput{
		ProjectId: "project-1", Code: "gateway", Name: "Gateway", RestApiUrl: "http://localhost:8080",
		BaseDomain: "example.com", Image: &image, ImagePullPolicy: "missing",
	})
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}
	versions := store.versions[view.Application.Id]
	versions[0].Status = status.VersionStatusPublished
	store.versions[view.Application.Id] = versions

	draftVersionId, err := service.CompileGatewayToVersion(context, view.Application, view.Config)
	if err != nil {
		t.Fatalf("compile gateway after publish: %v", err)
	}
	versions = store.versions[view.Application.Id]
	if len(versions) != 2 || versions[1].Id != draftVersionId || versions[1].Status != status.VersionStatusUnpublished {
		t.Fatalf("unexpected versions after compile: %+v", versions)
	}
	if !strings.HasPrefix(versions[1].Label, gatewayCompileVersionLabel+"-") {
		t.Fatalf("expected unique managed draft label, got %q", versions[1].Label)
	}
}

func TestGatewayRejectsActivePortConflicts(t *testing.T) {
	service, store := newGatewayTestService()
	projectId := "project-1"
	store.applications["existing"] = model.Application{Id: "existing", ProjectId: &projectId, Code: "existing", Name: "Existing", Kind: status.ApplicationKindStandard}
	store.services["existing"] = []model.Service{{Id: "service-existing", ApplicationId: "existing", VersionId: "existing-version", Status: status.ServiceStatusRunning}}
	store.exposes["existing-version"] = []model.VersionExpose{{ComponentName: "db", Protocol: "tcp", Access: "public", ContainerPort: 5432}}
	store.configs["gateway"] = model.GatewayConfig{ApplicationId: "gateway", BaseDomain: "example.com", TLSMode: "none"}

	ports, err := service.ActivePublicTCPListens(context.Background(), "")
	if err != nil {
		t.Fatalf("list active TCP ports: %v", err)
	}
	if len(ports) != 1 || ports[0] != 5432 {
		t.Fatalf("active TCP ports = %v", ports)
	}
	err = service.ValidateDeploymentExposureConflicts(context.Background(), model.Application{Id: "candidate", Code: "candidate"}, []model.VersionExpose{{
		ComponentName: "db", Protocol: "tcp", Access: "public", ContainerPort: 5432,
	}}, &model.GatewayConfig{BaseDomain: "example.com", TLSMode: "none"})
	if err == nil || !strings.Contains(err.Error(), "public TCP listen port 5432") {
		t.Fatalf("expected public TCP conflict, got %v", err)
	}
}

func TestPrepareDeploymentCompilesGatewayForChangedPublicTCPListeners(t *testing.T) {
	service, store := newGatewayTestService()
	projectID := "project-1"
	image := "traefik:v3"
	store.applications["gateway"] = model.Application{Id: "gateway", ProjectId: &projectID, Code: "gateway", Name: "Gateway", Kind: status.ApplicationKindGateway}
	store.applications["candidate"] = model.Application{Id: "candidate", ProjectId: &projectID, Code: "candidate", Name: "Candidate", Kind: status.ApplicationKindStandard}
	store.configs["gateway"] = model.GatewayConfig{ApplicationId: "gateway", BaseDomain: "example.com", TLSMode: "none", Image: &image}
	store.versions["gateway"] = []model.Version{{Id: "gateway-version", ApplicationId: "gateway", Status: status.VersionStatusUnpublished}}

	preparation, err := service.PrepareDeployment(context.Background(), store.applications["candidate"], []model.VersionExpose{{
		ComponentName: "database", Protocol: "tcp", Access: "public", ContainerPort: 5432,
	}})
	if err != nil {
		t.Fatalf("PrepareDeployment returned error: %v", err)
	}
	if preparation.RenderConfig == nil || preparation.RolloutConfig == nil {
		t.Fatalf("expected render and rollout configs, got %+v", preparation)
	}
	if listens := CompiledTCPListens(store.components["gateway-version"]); len(listens) != 1 || listens[0] != 5432 {
		t.Fatalf("compiled gateway TCP listens = %v", listens)
	}
}

func TestPrepareDeploymentSkipsGatewayRolloutWhenListenersMatch(t *testing.T) {
	service, store := newGatewayTestService()
	projectID := "project-1"
	image := "traefik:v3"
	gateway := model.Application{Id: "gateway", ProjectId: &projectID, Code: "gateway", Name: "Gateway", Kind: status.ApplicationKindGateway}
	candidate := model.Application{Id: "candidate", ProjectId: &projectID, Code: "candidate", Name: "Candidate", Kind: status.ApplicationKindStandard}
	store.applications[gateway.Id] = gateway
	store.applications[candidate.Id] = candidate
	store.configs[gateway.Id] = model.GatewayConfig{ApplicationId: gateway.Id, BaseDomain: "example.com", TLSMode: "none", Image: &image}
	store.versions[gateway.Id] = []model.Version{{Id: "gateway-version", ApplicationId: gateway.Id, Status: status.VersionStatusUnpublished}}
	if _, err := service.CompileGatewayToVersion(context.Background(), gateway, store.configs[gateway.Id], 5432); err != nil {
		t.Fatalf("compile gateway: %v", err)
	}

	preparation, err := service.PrepareDeployment(context.Background(), candidate, []model.VersionExpose{{
		ComponentName: "database", Protocol: "tcp", Access: "public", ContainerPort: 5432,
	}})
	if err != nil {
		t.Fatalf("PrepareDeployment returned error: %v", err)
	}
	if preparation.RolloutConfig != nil {
		t.Fatalf("expected no rollout, got %+v", preparation.RolloutConfig)
	}
}

func TestPrepareDeploymentReturnsExposureConflict(t *testing.T) {
	service, store := newGatewayTestService()
	projectID := "project-1"
	store.applications["existing"] = model.Application{Id: "existing", ProjectId: &projectID, Code: "existing", Name: "Existing", Kind: status.ApplicationKindStandard}
	store.applications["candidate"] = model.Application{Id: "candidate", ProjectId: &projectID, Code: "candidate", Name: "Candidate", Kind: status.ApplicationKindStandard}
	store.services["existing"] = []model.Service{{Id: "service-existing", ApplicationId: "existing", VersionId: "existing-version", Status: status.ServiceStatusRunning}}
	store.exposes["existing-version"] = []model.VersionExpose{{ComponentName: "database", Protocol: "tcp", Access: "public", ContainerPort: 5432}}
	store.configs["gateway"] = model.GatewayConfig{ApplicationId: "gateway", BaseDomain: "example.com", TLSMode: "none"}

	_, err := service.PrepareDeployment(context.Background(), store.applications["candidate"], []model.VersionExpose{{
		ComponentName: "database", Protocol: "tcp", Access: "public", ContainerPort: 5432,
	}})
	if err == nil || !strings.Contains(err.Error(), "public TCP listen port 5432") {
		t.Fatalf("expected public TCP conflict, got %v", err)
	}
}

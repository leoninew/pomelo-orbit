package deploymentsvc

import (
	"context"
	"strings"
	"testing"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type statusQueryRunner struct {
	called bool
}

func (r *statusQueryRunner) Run(context.Context, string, string, ...string) (string, error) {
	r.called = true
	return "", context.Canceled
}

func TestApplicationStatusReturnsNoContainersBeforeFirstDeployment(t *testing.T) {
	service, store := newRuntimeQueryService()
	store.service.Status = status.ServiceStatusStopped
	workspace := testWorkspace(t.TempDir())
	runner := &statusQueryRunner{}
	service.workspace = workspace
	service.queryRunner = runner

	containers, err := service.ApplicationStatus(context.Background(), "user-1", "app-1", deploymentdto.ServiceTargetInput{
		ServiceId: "service-1",
	})
	if err != nil {
		t.Fatalf("ApplicationStatus returned error: %v", err)
	}
	if len(containers) != 0 {
		t.Fatalf("containers = %+v, want none", containers)
	}
	if runner.called {
		t.Fatal("status query must not run without a deployment workspace")
	}
}

func TestDeleteApplicationRequiresServicesAndVersionsToBeRemoved(t *testing.T) {
	projectId := "project-1"
	commandStore := &runtimeQueryStore{
		application: model.Application{Id: "app-1", ProjectId: &projectId, Code: "demo"},
	}
	applicationStore := &deleteApplicationStore{
		versions: []model.Version{{Id: "version-1", ApplicationId: "app-1"}},
	}
	serviceStore := &deleteApplicationServiceStore{
		services: []model.Service{{Id: "service-1", ApplicationId: "app-1"}},
	}
	workspace := testWorkspace(t.TempDir())
	service := Service{
		commandStore: commandStore,
		application:  applicationStore,
		service:      serviceStore,
		workspace:    workspace,
	}

	err := service.DeleteApplication(context.Background(), "user-1", "app-1", true)
	if err == nil || !strings.Contains(err.Error(), "服务") {
		t.Fatalf("DeleteApplication error = %v, want service validation error", err)
	}
	if applicationStore.deleted || len(workspace.removedApps) != 0 {
		t.Fatal("application data and workspace must remain while services exist")
	}

	serviceStore.services = nil
	err = service.DeleteApplication(context.Background(), "user-1", "app-1", true)
	if err == nil || !strings.Contains(err.Error(), "版本") {
		t.Fatalf("DeleteApplication error = %v, want version validation error", err)
	}
	if applicationStore.deleted || len(workspace.removedApps) != 0 {
		t.Fatal("application data and workspace must remain while versions exist")
	}

	applicationStore.versions = nil
	if err := service.DeleteApplication(context.Background(), "user-1", "app-1", true); err != nil {
		t.Fatalf("DeleteApplication returned error: %v", err)
	}
	if !applicationStore.deleted {
		t.Fatal("application was not deleted after services and versions were removed")
	}
	if len(workspace.removedApps) != 1 || workspace.removedApps[0] != "demo" {
		t.Fatalf("removed workspaces = %#v, want [demo]", workspace.removedApps)
	}
}

type deleteApplicationStore struct {
	repository.ApplicationStore
	versions []model.Version
	deleted  bool
}

func (s *deleteApplicationStore) ListVersions(_ context.Context, _ string) ([]model.Version, error) {
	return s.versions, nil
}

func (s *deleteApplicationStore) DeleteApplication(context.Context, string) error {
	s.deleted = true
	return nil
}

type deleteApplicationServiceStore struct {
	repository.ServiceStore
	services []model.Service
}

func (s *deleteApplicationServiceStore) ListServicesByApplication(_ context.Context, _ string) ([]model.Service, error) {
	return s.services, nil
}

func newRuntimeQueryService() (Service, *runtimeQueryStore) {
	projectID := "project-1"
	store := &runtimeQueryStore{
		application: model.Application{Id: "app-1", ProjectId: &projectID, Code: "demo", Kind: status.ApplicationKindStandard},
		service:     model.Service{Id: "service-1", ApplicationId: "app-1", InstanceKey: "default", VersionId: "version-1"},
	}
	return Service{commandStore: store}, store
}

type runtimeQueryStore struct {
	application model.Application
	service     model.Service
}

func (s *runtimeQueryStore) ServiceEnvByService(_ context.Context, _ string) ([]model.ServiceEnv, error) {
	return nil, nil
}

func (s *runtimeQueryStore) Project(context.Context, string) (model.Project, error) {
	return model.Project{Id: "project-1"}, nil
}

func (s *runtimeQueryStore) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

func (s *runtimeQueryStore) Application(context.Context, string) (model.Application, error) {
	return s.application, nil
}

func (s *runtimeQueryStore) Version(context.Context, string) (model.Version, error) {
	return model.Version{}, repository.ErrNotFound
}

func (s *runtimeQueryStore) VersionComponentsByVersion(context.Context, string) ([]model.VersionComponent, error) {
	return nil, nil
}

func (s *runtimeQueryStore) Service(context.Context, string) (model.Service, error) {
	return s.service, nil
}

func (s *runtimeQueryStore) ListServicesByApplication(context.Context, string) ([]model.Service, error) {
	return []model.Service{s.service}, nil
}

func (s *runtimeQueryStore) ServiceComponentsByService(context.Context, string) ([]model.ServiceComponent, error) {
	return nil, nil
}

func (s *runtimeQueryStore) UpdateServiceStatus(context.Context, string, string) error {
	return nil
}

func (s *runtimeQueryStore) CreateDeployment(context.Context, model.Deployment) error {
	return nil
}

func (s *runtimeQueryStore) HasActiveDeployment(context.Context, string) (bool, error) {
	return false, nil
}

func (s *runtimeQueryStore) HasActiveGatewayService(context.Context, string) (bool, error) {
	return false, nil
}

func (s *runtimeQueryStore) ResolveActiveGatewayConfig(context.Context) (model.GatewayConfig, error) {
	return model.GatewayConfig{}, repository.ErrNotFound
}

func TestApplyContainerComponentIds(t *testing.T) {
	t.Parallel()
	containers := []deploymentdto.RuntimeContainer{
		{Service: "web"},
		{Service: "worker"},
	}

	applyContainerComponentIds(containers, "version-1", map[string]string{
		"web": "component-web",
	})

	if got, want := containers[0].VersionId, "version-1"; got != want {
		t.Fatalf("VersionID: got %q want %q", got, want)
	}
	if got, want := containers[0].ComponentId, "component-web"; got != want {
		t.Fatalf("ComponentID: got %q want %q", got, want)
	}
	if got, want := containers[1].VersionId, "version-1"; got != want {
		t.Fatalf("VersionID: got %q want %q", got, want)
	}
	if containers[1].ComponentId != "" {
		t.Fatalf("unexpected ComponentID for %+v", containers[1])
	}
}

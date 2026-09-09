package deploymentsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestApplicationStatusReturnsNoContainersBeforeFirstDeployment(t *testing.T) {
	service, store := newRuntimeQueryService()
	store.service.Status = status.ServiceStatusStopped
	workspace := testWorkspace(t.TempDir())
	service.runtime = workspace
	service.targetResolver = staticTargetResolver{target: testSSHTarget("project-1")}

	containers, err := service.ApplicationStatus(context.Background(), "user-1", "app-1", deploymentdto.ServiceTargetInput{
		ServiceId: "service-1",
	})
	if err != nil {
		t.Fatalf("ApplicationStatus returned error: %v", err)
	}
	if len(containers) != 0 {
		t.Fatalf("containers = %+v, want none", containers)
	}
	if workspace.queryCalled {
		t.Fatal("status query must not run without a deployment workspace")
	}
}

func TestComposePreviewsDoNotRequireConfiguredProjectEnvironment(t *testing.T) {
	projectID := "project-1"
	store := &runtimeQueryStore{
		application: model.Application{Id: "app-1", ProjectId: &projectID, Code: "demo", Kind: status.ApplicationKindStandard},
		service:     model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-1", InstanceKey: "default", Code: "demo-default"},
		version:     model.Version{Id: "version-1", ApplicationId: "app-1"},
		components: []model.VersionComponent{{
			Id: "component-1", VersionId: "version-1", Name: "web", Image: "nginx:latest",
			Mounts: []model.VersionComponentMount{{SourceType: "directory", Source: "./data", Target: "/var/lib/app"}},
		}},
		serviceComponents: []model.ServiceComponent{{
			Id: "service-component-1", ServiceId: "service-1", SourceVersionComponentId: "component-1", ComponentName: "web",
		}},
	}
	service := Service{
		commandStore:       store,
		executionStore:     store,
		targetResolver:     staticTargetResolver{err: errors.New("Project environment is disabled")},
		gatewayCoordinator: &gatewayDeploymentCoordinatorFake{gateway: &model.GatewayConfig{NetworkName: "traefik"}},
	}

	servicePreview, err := service.PreviewService(context.Background(), "user-1", "service-1", deploymentdto.PreviewComposeInput{})
	if err != nil {
		t.Fatalf("PreviewService returned error: %v", err)
	}
	if !strings.Contains(servicePreview, "./data:/var/lib/app") {
		t.Fatalf("PreviewService did not render relative mount:\n%s", servicePreview)
	}

	versionPreview, err := service.PreviewVersion(context.Background(), "user-1", "version-1", deploymentdto.PreviewComposeInput{})
	if err != nil {
		t.Fatalf("PreviewVersion returned error: %v", err)
	}
	if !strings.Contains(versionPreview, "./data:/var/lib/app") {
		t.Fatalf("PreviewVersion did not render relative mount:\n%s", versionPreview)
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
	service := Service{
		commandStore: commandStore,
		application:  applicationStore,
		service:      serviceStore,
	}

	err := service.DeleteApplication(context.Background(), "user-1", "app-1")
	if err == nil || !strings.Contains(err.Error(), "服务") {
		t.Fatalf("DeleteApplication error = %v, want service validation error", err)
	}
	if applicationStore.deleted {
		t.Fatal("application must remain while services exist")
	}

	serviceStore.services = nil
	err = service.DeleteApplication(context.Background(), "user-1", "app-1")
	if err == nil || !strings.Contains(err.Error(), "版本") {
		t.Fatalf("DeleteApplication error = %v, want version validation error", err)
	}
	if applicationStore.deleted {
		t.Fatal("application must remain while versions exist")
	}

	applicationStore.versions = nil
	if err := service.DeleteApplication(context.Background(), "user-1", "app-1"); err != nil {
		t.Fatalf("DeleteApplication returned error: %v", err)
	}
	if !applicationStore.deleted {
		t.Fatal("application was not deleted after services and versions were removed")
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
	deploymentport.ExecutionStore
	application       model.Application
	service           model.Service
	version           model.Version
	components        []model.VersionComponent
	serviceComponents []model.ServiceComponent
	serviceEnv        []model.ServiceEnv
}

func (s *runtimeQueryStore) ServiceEnvByService(_ context.Context, _ string) ([]model.ServiceEnv, error) {
	return s.serviceEnv, nil
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

func (s *runtimeQueryStore) Version(_ context.Context, id string) (model.Version, error) {
	if s.version.Id != id {
		return model.Version{}, repository.ErrNotFound
	}
	return s.version, nil
}

func (s *runtimeQueryStore) VersionComponentsByVersion(_ context.Context, versionID string) ([]model.VersionComponent, error) {
	if s.version.Id != versionID {
		return nil, repository.ErrNotFound
	}
	return s.components, nil
}

func (s *runtimeQueryStore) Service(context.Context, string) (model.Service, error) {
	return s.service, nil
}

func (s *runtimeQueryStore) ListServicesByApplication(context.Context, string) ([]model.Service, error) {
	return []model.Service{s.service}, nil
}

func (s *runtimeQueryStore) ServiceComponentsByService(context.Context, string) ([]model.ServiceComponent, error) {
	return s.serviceComponents, nil
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

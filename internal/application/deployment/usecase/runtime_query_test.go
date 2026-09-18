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

	containers, err := service.ApplicationStatus(context.Background(), "user-1", "project-1", "app-1", deploymentdto.ServiceTargetInput{
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

func TestApplicationLogsReturnsEmptyBeforeFirstDeployment(t *testing.T) {
	service, store := newRuntimeQueryService()
	store.service.Status = status.ServiceStatusStopped
	workspace := testWorkspace(t.TempDir())
	service.runtime = workspace
	service.targetResolver = staticTargetResolver{target: testSSHTarget("project-1")}

	logs, err := service.ApplicationLogs(context.Background(), "user-1", "project-1", "app-1", 200, deploymentdto.ServiceTargetInput{
		ServiceId: "service-1",
	}, "traefik")
	if err != nil {
		t.Fatalf("ApplicationLogs returned error: %v", err)
	}
	if logs != "" {
		t.Fatalf("logs = %q, want empty", logs)
	}
	if workspace.queryCalled {
		t.Fatal("log query must not run without a deployment workspace")
	}
}

func TestComposePreviewsDoNotRequireConfiguredProjectEnvironment(t *testing.T) {
	projectId := "project-1"
	store := &runtimeQueryStore{
		application: model.Application{Id: "app-1", ProjectId: &projectId, Code: "demo", Kind: status.ApplicationKindStandard},
		service:     model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-1", Code: "demo-default"},
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
		targetResolver:     staticTargetResolver{err: errors.New("Project environment is unavailable")},
		gatewayCoordinator: &gatewayDeploymentCoordinatorFake{gateway: &model.GatewayConfig{NetworkName: "traefik"}},
	}

	servicePreview, err := service.PreviewService(context.Background(), "user-1", projectId, "service-1", deploymentdto.PreviewComposeInput{})
	if err != nil {
		t.Fatalf("PreviewService returned error: %v", err)
	}
	if !strings.Contains(servicePreview, "./data:/var/lib/app") {
		t.Fatalf("PreviewService did not render relative mount:\n%s", servicePreview)
	}

	versionPreview, err := service.PreviewVersion(context.Background(), "user-1", projectId, "version-1", deploymentdto.PreviewComposeInput{})
	if err != nil {
		t.Fatalf("PreviewVersion returned error: %v", err)
	}
	if !strings.Contains(versionPreview, "./data:/var/lib/app") {
		t.Fatalf("PreviewVersion did not render relative mount:\n%s", versionPreview)
	}
}

func newRuntimeQueryService() (Service, *runtimeQueryStore) {
	projectId := "project-1"
	store := &runtimeQueryStore{
		application: model.Application{Id: "app-1", ProjectId: &projectId, Code: "demo", Kind: status.ApplicationKindStandard},
		service:     model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-1", Code: "demo-default"},
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

func (s *runtimeQueryStore) ServiceEnvByService(_ context.Context, _, _ string) ([]model.ServiceEnv, error) {
	return s.serviceEnv, nil
}

func (s *runtimeQueryStore) Project(context.Context, string) (model.Project, error) {
	return model.Project{Id: "project-1"}, nil
}

func (s *runtimeQueryStore) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

func (s *runtimeQueryStore) Application(context.Context, string, string) (model.Application, error) {
	return s.application, nil
}

func (s *runtimeQueryStore) Version(_ context.Context, _ string, id string) (model.Version, error) {
	if s.version.Id != id {
		return model.Version{}, repository.ErrNotFound
	}
	return s.version, nil
}

func (s *runtimeQueryStore) VersionComponentsByVersion(_ context.Context, _ string, versionId string) ([]model.VersionComponent, error) {
	if s.version.Id != versionId {
		return nil, repository.ErrNotFound
	}
	return s.components, nil
}

func (s *runtimeQueryStore) Service(context.Context, string, string) (model.Service, error) {
	return s.service, nil
}

func (s *runtimeQueryStore) ListServicesByApplication(context.Context, string, string) ([]model.Service, error) {
	return []model.Service{s.service}, nil
}

func (s *runtimeQueryStore) ServiceComponentsByService(context.Context, string, string) ([]model.ServiceComponent, error) {
	return s.serviceComponents, nil
}

func (s *runtimeQueryStore) UpdateServiceStatus(context.Context, string, string, string) error {
	return nil
}

func (s *runtimeQueryStore) CreateDeployment(context.Context, string, model.Deployment) error {
	return nil
}

func (s *runtimeQueryStore) HasActiveDeployment(context.Context, string, string) (bool, error) {
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
		t.Fatalf("VersionId: got %q want %q", got, want)
	}
	if got, want := containers[0].ComponentId, "component-web"; got != want {
		t.Fatalf("ComponentId: got %q want %q", got, want)
	}
	if got, want := containers[1].VersionId, "version-1"; got != want {
		t.Fatalf("VersionId: got %q want %q", got, want)
	}
	if containers[1].ComponentId != "" {
		t.Fatalf("unexpected ComponentId for %+v", containers[1])
	}
}

package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func TestDeployServiceCreatesAndDispatchesDeployment(t *testing.T) {
	service, store, dispatcher := newCommandTestService()

	result, err := service.DeployService(context.Background(), "user-1", "service-1", deploymentdto.DeployServiceInput{ForceRecreate: true})
	if err != nil {
		t.Fatalf("DeployService returned error: %v", err)
	}
	if result.DeploymentId == "" || len(store.deployments) != 1 {
		t.Fatalf("expected one deployment, result=%+v deployments=%d", result, len(store.deployments))
	}
	deployment := store.deployments[0]
	if deployment.ServiceId == nil || *deployment.ServiceId != store.service.Id || deployment.VersionId == nil || *deployment.VersionId != store.service.VersionId {
		t.Fatalf("deployment must persist the saved service target: %+v", deployment)
	}
	if store.service.Status != status.ServiceStatusDeploying || dispatcher.deploy.DeploymentId != result.DeploymentId || !dispatcher.deploy.ForceRecreate {
		t.Fatalf("service or dispatch was not prepared: service=%+v dispatch=%+v", store.service, dispatcher.deploy)
	}
	var options deploymentdto.DeployOptionsJSON
	if err := json.Unmarshal([]byte(*deployment.OptionsJSON), &options); err != nil {
		t.Fatal(err)
	}
	if !options.ForceRecreate || options.InstanceKey != "default" || options.RuntimeConfig["UNUSED"] != "value" {
		t.Fatalf("deployment must snapshot saved service configuration: %+v", options)
	}
}

func TestDeployServiceRejectsGatewayNotRunningBeforePersisting(t *testing.T) {
	service, store, dispatcher := newCommandTestService()
	service.gatewayCoordinator = commandGatewayCoordinator{ensureErr: errors.New("gateway traefik is configured but not running")}

	_, err := service.DeployService(context.Background(), "user-1", "service-1", deploymentdto.DeployServiceInput{})
	if err == nil || !strings.Contains(err.Error(), "gateway traefik is configured but not running") {
		t.Fatalf("expected gateway readiness error, got %v", err)
	}
	if len(store.deployments) != 0 || dispatcher.deploy.DeploymentId != "" {
		t.Fatalf("gateway preflight must reject before persistence: deployments=%+v dispatch=%+v", store.deployments, dispatcher.deploy)
	}
}

func TestDeployServiceWarnsForMissingPublicTCPGatewayEntrypoint(t *testing.T) {
	service, store, dispatcher := newCommandTestService()
	store.exposes = []model.ServiceExpose{{ServiceId: store.service.Id, ComponentName: "redis", Protocol: "tcp", Access: "public", ContainerPort: 6379}}
	store.gatewayConfig = model.GatewayConfig{ApplicationId: "gateway-1"}
	store.gatewayServices = []model.Service{{Id: "gateway-service", ApplicationId: "gateway-1", VersionId: "gateway-version", Status: status.ServiceStatusRunning}}
	store.gatewayComponents = []model.VersionComponent{{Id: "gateway-component", VersionId: "gateway-version", Name: "traefik", Image: "traefik", Ports: []model.VersionComponentPort{{HostPort: 80, ContainerPort: 80}}}}

	result, err := service.DeployService(context.Background(), "user-1", "service-1", deploymentdto.DeployServiceInput{})
	if err != nil {
		t.Fatalf("DeployService returned error: %v", err)
	}
	if len(result.Warnings) != 1 || !strings.Contains(result.Warnings[0], "tcp6379") {
		t.Fatalf("expected missing TCP entrypoint warning, got %+v", result.Warnings)
	}
	if len(store.deployments) != 1 || dispatcher.deploy.DeploymentId != result.DeploymentId {
		t.Fatalf("warning must not block business deployment: deployments=%+v dispatch=%+v", store.deployments, dispatcher.deploy)
	}
}

func TestRestartAndStopRemainServiceTargeted(t *testing.T) {
	service, store, dispatcher := newCommandTestService()

	restartID, err := service.RestartApplication(context.Background(), "user-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1"})
	if err != nil {
		t.Fatalf("RestartApplication returned error: %v", err)
	}
	if dispatcher.restart.DeploymentId != restartID {
		t.Fatalf("unexpected restart dispatch: %+v", dispatcher.restart)
	}
	store.service.Status = status.ServiceStatusRunning
	stopID, err := service.StopApplication(context.Background(), "user-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1", RemoveVolumes: true})
	if err != nil {
		t.Fatalf("StopApplication returned error: %v", err)
	}
	if dispatcher.stop.DeploymentId != stopID || !dispatcher.stop.RemoveVolumes {
		t.Fatalf("unexpected stop dispatch: %+v", dispatcher.stop)
	}
}

func newCommandTestService() (Service, *commandStoreFake, *commandDispatcherFake) {
	projectID := "project-1"
	store := &commandStoreFake{
		application: model.Application{Id: "app-1", ProjectId: &projectID, Code: "demo", Name: "Demo", Kind: status.ApplicationKindStandard, ImagePullPolicy: "missing"},
		version:     model.Version{Id: "version-1", ApplicationId: "app-1", Label: "v1"},
		components:  []model.VersionComponent{{Id: "component-1", VersionId: "version-1", Name: "redis", Image: "redis:7"}},
		service:     model.Service{Id: "service-1", ApplicationId: "app-1", InstanceKey: "default", VersionId: "version-1", RuntimeConfig: map[string]string{"UNUSED": "value"}, Status: status.ServiceStatusRunning},
	}
	dispatcher := &commandDispatcherFake{}
	return Service{commandStore: store, dispatcher: dispatcher, gatewayCoordinator: commandGatewayCoordinator{}}, store, dispatcher
}

type commandStoreFake struct {
	application       model.Application
	version           model.Version
	components        []model.VersionComponent
	service           model.Service
	exposes           []model.ServiceExpose
	deployments       []model.Deployment
	gatewayConfig     model.GatewayConfig
	gatewayServices   []model.Service
	gatewayComponents []model.VersionComponent
}

func (s *commandStoreFake) Project(_ context.Context, id string) (model.Project, error) {
	return model.Project{Id: id}, nil
}
func (s *commandStoreFake) IsProjectMember(_ context.Context, _, _ string) (bool, error) {
	return true, nil
}
func (s *commandStoreFake) Application(_ context.Context, _ string) (model.Application, error) {
	return s.application, nil
}
func (s *commandStoreFake) Version(_ context.Context, id string) (model.Version, error) {
	if id == s.version.Id {
		return s.version, nil
	}
	if id == "gateway-version" {
		return model.Version{Id: id, ApplicationId: "gateway-1"}, nil
	}
	return model.Version{}, repository.ErrNotFound
}
func (s *commandStoreFake) VersionComponentsByVersion(_ context.Context, id string) ([]model.VersionComponent, error) {
	if id == "gateway-version" {
		return s.gatewayComponents, nil
	}
	return s.components, nil
}
func (s *commandStoreFake) Service(_ context.Context, id string) (model.Service, error) {
	if id == s.service.Id {
		return s.service, nil
	}
	return model.Service{}, repository.ErrNotFound
}
func (s *commandStoreFake) ListServicesByApplication(_ context.Context, applicationID string) ([]model.Service, error) {
	if applicationID == "gateway-1" {
		return s.gatewayServices, nil
	}
	return []model.Service{s.service}, nil
}
func (s *commandStoreFake) ServiceExposesByService(_ context.Context, _ string) ([]model.ServiceExpose, error) {
	return s.exposes, nil
}
func (s *commandStoreFake) UpdateServiceStatus(_ context.Context, id string, serviceStatus string) error {
	if id != s.service.Id {
		return repository.ErrNotFound
	}
	s.service.Status = serviceStatus
	return nil
}
func (s *commandStoreFake) CreateDeployment(_ context.Context, deployment model.Deployment) error {
	s.deployments = append(s.deployments, deployment)
	return nil
}
func (s *commandStoreFake) HasActiveGatewayService(_ context.Context, _ string) (bool, error) {
	return false, nil
}
func (s *commandStoreFake) ResolveActiveGatewayConfig(_ context.Context) (model.GatewayConfig, error) {
	if s.gatewayConfig.ApplicationId == "" {
		return model.GatewayConfig{}, repository.ErrNotFound
	}
	return s.gatewayConfig, nil
}

type commandDispatcherFake struct {
	deploy  deploymentdto.DeployDispatchInput
	restart deploymentdto.RestartDispatchInput
	stop    deploymentdto.StopDispatchInput
}

func (d *commandDispatcherFake) DispatchDeploy(_ context.Context, input deploymentdto.DeployDispatchInput) error {
	d.deploy = input
	return nil
}
func (d *commandDispatcherFake) DispatchRestart(_ context.Context, input deploymentdto.RestartDispatchInput) error {
	d.restart = input
	return nil
}
func (d *commandDispatcherFake) DispatchStop(_ context.Context, input deploymentdto.StopDispatchInput) error {
	d.stop = input
	return nil
}

type commandGatewayCoordinator struct{ ensureErr error }

func (commandGatewayCoordinator) GatewayForDeployment(context.Context, model.Application, []model.ServiceExpose) (*model.GatewayConfig, error) {
	return nil, nil
}
func (c commandGatewayCoordinator) EnsureGatewayRunning(context.Context, model.Application) error {
	return c.ensureErr
}

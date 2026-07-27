package deploymentsvc

import (
	"context"
	"encoding/json"
	"testing"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func TestDeployApplicationCreatesAndDispatchesDeployment(t *testing.T) {
	service, store, dispatcher := newCommandTestService()

	deploymentID, err := service.DeployApplication(context.Background(), "user-1", "app-1", deploymentdto.DeployInput{
		VersionId: "version-1", ServiceId: "service-1", ForceRecreate: true,
	})
	if err != nil {
		t.Fatalf("DeployApplication returned error: %v", err)
	}
	if deploymentID == "" || len(store.deployments) != 1 {
		t.Fatalf("expected one deployment, id=%q deployments=%d", deploymentID, len(store.deployments))
	}
	deployment := store.deployments[0]
	if deployment.OperationType != "deploy" || deployment.Status != "waiting_to_run" {
		t.Fatalf("unexpected deployment: %+v", deployment)
	}
	if deployment.ServiceId == nil || *deployment.ServiceId != store.service.Id {
		t.Fatalf("deployment must persist the service id: deployment=%+v service=%+v", deployment, store.service)
	}
	if store.service.Status != status.ServiceStatusDeploying || store.service.VersionId != "version-1" {
		t.Fatalf("service was not prepared for deployment: %+v", store.service)
	}
	if dispatcher.deploy.DeploymentID != deploymentID || !dispatcher.deploy.ForceRecreate {
		t.Fatalf("unexpected dispatch input: %+v", dispatcher.deploy)
	}
	var options deploymentdto.DeployOptionsJSON
	if err := json.Unmarshal([]byte(*deployment.OptionsJSON), &options); err != nil {
		t.Fatal(err)
	}
	if !options.ForceRecreate || options.InstanceKey != "default" {
		t.Fatalf("unexpected deployment options: %+v", options)
	}
	if options.RuntimeConfig["UNUSED"] != "value" {
		t.Fatalf("deployment must snapshot the complete service config: %+v", options.RuntimeConfig)
	}
}

func TestDeployApplicationRejectsVersionWithoutComponents(t *testing.T) {
	service, store, dispatcher := newCommandTestService()
	store.components = nil

	_, err := service.DeployApplication(context.Background(), "user-1", "app-1", deploymentdto.DeployInput{
		VersionId: "version-1", ServiceId: "service-1",
	})
	if err == nil {
		t.Fatal("expected deployment validation error")
	}
	if len(store.deployments) != 0 || len(store.operations) != 0 {
		t.Fatalf("deployment must not be persisted when the version has no components: deployments=%+v operations=%v", store.deployments, store.operations)
	}
	if dispatcher.deploy.DeploymentID != "" {
		t.Fatalf("deployment must not be dispatched when the version has no components: %+v", dispatcher.deploy)
	}
}

func TestRestartApplicationCreatesAndDispatchesDeployment(t *testing.T) {
	service, store, dispatcher := newCommandTestService()

	deploymentID, err := service.RestartApplication(context.Background(), "user-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1"})
	if err != nil {
		t.Fatalf("RestartApplication returned error: %v", err)
	}
	if store.deployments[0].OperationType != "restart" || dispatcher.restart.DeploymentID != deploymentID {
		t.Fatalf("unexpected restart command: deployment=%+v dispatch=%+v", store.deployments[0], dispatcher.restart)
	}
	var options deploymentdto.DeployOptionsJSON
	if err := json.Unmarshal([]byte(*store.deployments[0].OptionsJSON), &options); err != nil {
		t.Fatal(err)
	}
	if options.RuntimeConfig["UNUSED"] != "value" {
		t.Fatalf("restart must snapshot the complete service config: %+v", options.RuntimeConfig)
	}
}

func TestRestartApplicationRejectsMissingRuntimeConfig(t *testing.T) {
	service, store, dispatcher := newCommandTestService()
	required := `[{"key":"API_TOKEN","value":"${API_TOKEN}"}]`
	store.version.EnvJSON = &required
	store.service.RuntimeConfig = map[string]string{}

	_, err := service.RestartApplication(context.Background(), "user-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1"})
	if err == nil {
		t.Fatal("expected restart runtime config validation error")
	}
	if len(store.deployments) != 0 || dispatcher.restart.DeploymentID != "" {
		t.Fatalf("restart must not be persisted or dispatched with missing config: deployments=%+v dispatch=%+v", store.deployments, dispatcher.restart)
	}
}

func TestStopApplicationCreatesAndDispatchesDeployment(t *testing.T) {
	service, store, dispatcher := newCommandTestService()

	deploymentID, err := service.StopApplication(context.Background(), "user-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1", RemoveVolumes: true})
	if err != nil {
		t.Fatalf("StopApplication returned error: %v", err)
	}
	if store.deployments[0].OperationType != "stop" || dispatcher.stop.DeploymentID != deploymentID || !dispatcher.stop.RemoveVolumes {
		t.Fatalf("unexpected stop command: deployment=%+v dispatch=%+v", store.deployments[0], dispatcher.stop)
	}
}

func TestDeployApplicationRejectsMissingDispatcherBeforePersisting(t *testing.T) {
	service, store, _ := newCommandTestService()
	service.dispatcher = nil

	_, err := service.DeployApplication(context.Background(), "user-1", "app-1", deploymentdto.DeployInput{
		VersionId: "version-1", ServiceId: "service-1",
	})
	if err == nil {
		t.Fatal("expected deployment dispatcher error")
	}
	if len(store.deployments) != 0 {
		t.Fatalf("deployment must not be persisted without a dispatcher: %+v", store.deployments)
	}
	if len(store.operations) != 0 {
		t.Fatalf("service must not be persisted without a dispatcher: %v", store.operations)
	}
}

func newCommandTestService() (Service, *commandStoreFake, *commandDispatcherFake) {
	projectID := "project-1"
	store := &commandStoreFake{
		application: model.Application{Id: "app-1", ProjectId: &projectID, Code: "demo", Name: "Demo", ImagePullPolicy: "missing"},
		version:     model.Version{Id: "version-1", ApplicationId: "app-1", Label: "v1"},
		components:  []model.VersionComponent{{Id: "component-1", VersionId: "version-1", Name: "web", Image: "nginx"}},
		service: model.Service{
			Id: "service-1", ApplicationId: "app-1", InstanceKey: "default", VersionId: "version-1", RuntimeConfig: map[string]string{"UNUSED": "value"}, Status: "running",
		},
	}
	dispatcher := &commandDispatcherFake{}
	return Service{commandStore: store, dispatcher: dispatcher}, store, dispatcher
}

type commandStoreFake struct {
	application model.Application
	version     model.Version
	components  []model.VersionComponent
	service     model.Service
	deployments []model.Deployment
	operations  []string
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

func (s *commandStoreFake) Version(_ context.Context, _ string) (model.Version, error) {
	return s.version, nil
}

func (s *commandStoreFake) VersionComponentsByVersion(_ context.Context, _ string) ([]model.VersionComponent, error) {
	return s.components, nil
}

func (s *commandStoreFake) ServiceByKey(_ context.Context, _, _ string) (model.Service, error) {
	if s.service.Id == "" {
		return model.Service{}, repository.ErrNotFound
	}
	return s.service, nil
}

func (s *commandStoreFake) Service(_ context.Context, _ string) (model.Service, error) {
	return s.service, nil
}

func (s *commandStoreFake) ListServicesByApplication(_ context.Context, _ string) ([]model.Service, error) {
	return []model.Service{s.service}, nil
}

func (s *commandStoreFake) UpsertService(_ context.Context, service model.Service) error {
	s.service = service
	s.operations = append(s.operations, "upsert_service")
	return nil
}

func (s *commandStoreFake) CreateDeployment(_ context.Context, deployment model.Deployment) error {
	s.deployments = append(s.deployments, deployment)
	s.operations = append(s.operations, "create_deployment")
	return nil
}

func (s *commandStoreFake) HasActiveGatewayService(_ context.Context, _ string) (bool, error) {
	return false, nil
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

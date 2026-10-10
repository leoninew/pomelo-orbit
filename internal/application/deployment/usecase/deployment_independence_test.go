package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestDeploymentAndRestartDoNotRequireRoutePublication(t *testing.T) {
	for _, operation := range []string{"deploy", "restart"} {
		for _, gateway := range []bool{false, true} {
			name := operation + "/ordinary-service"
			if gateway {
				name = operation + "/gateway"
			}
			t.Run(name, func(t *testing.T) {
				service, store, runtime, readiness := independentDeploymentTestService(t, gateway)
				if !gateway {
					readiness.err = errors.New("Gateway is unavailable")
				}
				if err := executeIndependentDeployment(service, operation); err != nil {
					t.Fatal(err)
				}
				if store.deployment.Status != status.WorkStatusRanToCompletion || store.service.Status != status.ServiceStatusRunning {
					t.Fatalf("deployment status=%s, service status=%s", store.deployment.Status, store.service.Status)
				}
				if !strings.Contains(runtime.command, "up") || len(runtime.staged) != 1 {
					t.Fatalf("deployment did not execute Compose: command=%s staged=%d", runtime.command, len(runtime.staged))
				}
				if runtime.queryCalled {
					t.Fatal("deployment queried runtime state outside Gateway readiness")
				}
				if !gateway && readiness.calls != 0 {
					t.Fatal("ordinary Service waited for Gateway readiness")
				}
				if gateway && (readiness.calls != 1 || readiness.gateway.ApplicationId != store.application.Id || readiness.timeout != 30*time.Second) {
					t.Fatalf("Gateway readiness did not use the deployment snapshot: %+v", readiness)
				}
			})
		}
	}
}

func TestGatewayDeploymentAndRestartStillRequireGatewayReadiness(t *testing.T) {
	for _, operation := range []string{"deploy", "restart"} {
		t.Run(operation, func(t *testing.T) {
			service, store, runtime, readiness := independentDeploymentTestService(t, true)
			readiness.err = errors.New("Traefik API is unavailable")
			err := executeIndependentDeployment(service, operation)
			if !errors.Is(err, readiness.err) {
				t.Fatalf("Gateway readiness error = %v", err)
			}
			if !strings.Contains(runtime.command, "up") || store.deployment.Status != status.WorkStatusFaulted || store.service.Status != status.ServiceStatusFaulted {
				t.Fatalf("command=%s deployment status=%s service status=%s", runtime.command, store.deployment.Status, store.service.Status)
			}
		})
	}
}

func executeIndependentDeployment(service Service, operation string) error {
	deployment, err := service.executionStore.Deployment(context.Background(), "project-1", "deployment-1")
	if err != nil {
		return err
	}
	if operation == "restart" {
		return service.ExecuteApplicationRestart(context.Background(), "project-1", "app-1", deployment.Id)
	}
	return service.ExecuteApplicationDeploy(context.Background(), "project-1", "app-1", deployment.Id, false)
}

func independentDeploymentTestService(t *testing.T, gateway bool) (Service, *independentDeploymentStore, *gatewayDeploymentRuntimeFake, *gatewayReadinessFake) {
	t.Helper()
	projectId := "project-1"
	store := &independentDeploymentStore{runtimeQueryStore: &runtimeQueryStore{
		application: model.Application{Id: "app-1", ProjectId: &projectId, Code: "demo", Kind: status.ApplicationKindStandard},
		service:     model.Service{Id: "service-1", ApplicationId: "app-1", VersionId: "version-base", Code: "demo-default", ProjectId: projectId, DeploymentDirectory: "/custom/demo", DirectoryTargetRevision: 1},
		version:     model.Version{Id: "version-base", ApplicationId: "app-1"},
		components:  []model.VersionComponent{{Id: "component-1", VersionId: "version-base", Name: "web", Image: "nginx:latest"}},
		serviceComponents: []model.ServiceComponent{{
			Id: "overlay-1", ServiceId: "service-1", SourceVersionComponentId: "component-1", ComponentName: "web",
		}},
	}}
	config := &model.GatewayConfig{
		ApplicationId: "gateway-1", NetworkName: "traefik", RestReadyTimeoutSeconds: 30,
		InternalDomain: "example.test", DefaultEntrypoint: "web",
		VersionBindings: []model.GatewayVersionBinding{{Profile: "base", VersionId: "version-base"}},
	}
	if gateway {
		config.ApplicationId = store.application.Id
		store.application.Code = "traefik"
		store.application.Kind = status.ApplicationKindGateway
		store.service.Code = "traefik-default"
		store.components[0].Name, store.components[0].Image = "traefik", "traefik:3.6"
		store.serviceComponents[0].ComponentName = "traefik"
		store.components[0].Mounts = testGatewayEnrichmentPlan("", "", "").Components[0].Mounts
	}
	plan, _, err := BuildEffectiveServicePlan(store.application, store.version, store.service, store.components, store.serviceComponents, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan.Gateway = config
	if err := enrichGatewayPlan(&plan); err != nil {
		t.Fatal(err)
	}
	planHash, err := EffectiveServicePlanHash(plan)
	if err != nil {
		t.Fatal(err)
	}
	options, err := json.Marshal(deploymentdto.DeployOptionsJSON{GatewayConfig: config})
	if err != nil {
		t.Fatal(err)
	}
	optionsJSON := string(options)
	workingDirectory := store.service.DeploymentDirectory
	store.deployment = model.Deployment{
		WorkingDirectory: &workingDirectory,
		Id:               "deployment-1", ServiceId: &store.service.Id, VersionId: &store.version.Id,
		OptionsJSON: &optionsJSON, EffectivePlanHash: &planHash,
	}
	target := deploymentTestTarget(1)
	applyDeploymentTargetSnapshot(&store.deployment, target, config)
	store.service.CurrentDeploymentId = &store.deployment.Id
	runtime := &gatewayDeploymentRuntimeFake{workspaceFake: testWorkspace(t.TempDir())}
	readiness := &gatewayReadinessFake{}
	service := Service{executionStore: store, targetResolver: staticTargetResolver{target: target}, runtime: runtime, logStore: runtime, gatewayReadiness: readiness, executionTimeout: time.Minute, cancelTimeout: time.Second, transactionRunner: testDeploymentTransaction{}}
	return service, store, runtime, readiness
}

type independentDeploymentStore struct {
	*runtimeQueryStore
	deployment model.Deployment
}

func (s *independentDeploymentStore) CreateDeployment(_ context.Context, _ string, deployment model.Deployment) error {
	s.deployment = deployment
	return nil
}

func (s *independentDeploymentStore) BeginDeployment(context.Context, string, string) (bool, error) {
	s.deployment.Status = status.WorkStatusRunning
	return true, nil
}

func (s *independentDeploymentStore) Deployment(context.Context, string, string) (model.Deployment, error) {
	return s.deployment, nil
}

func (s *independentDeploymentStore) CompleteDeployment(_ context.Context, _, _, value, _ string) (bool, error) {
	s.deployment.Status = value
	return true, nil
}

func (s *independentDeploymentStore) UpdateServiceDeploymentResult(_ context.Context, _, _, deploymentId, value string, versionId *string) (bool, error) {
	if s.service.CurrentDeploymentId == nil || *s.service.CurrentDeploymentId != deploymentId || s.deployment.Status != status.WorkStatusRunning {
		return false, nil
	}
	s.service.Status = value
	if versionId != nil {
		s.service.VersionId = *versionId
	}
	return true, nil
}

func (s *independentDeploymentStore) CancelSupersededDeployments(context.Context, string, string, string) ([]string, error) {
	return nil, nil
}

func (s *independentDeploymentStore) ReconcileServiceAfterCancellation(_ context.Context, _, _, deploymentId, value string) (bool, error) {
	if s.service.CurrentDeploymentId == nil || *s.service.CurrentDeploymentId != deploymentId || s.deployment.Status != status.WorkStatusCanceled {
		return false, nil
	}
	s.service.Status = value
	return true, nil
}
func (s *independentDeploymentStore) CancelObsoleteDeployment(context.Context, string, string) (bool, error) {
	if s.service.CurrentDeploymentId != nil && *s.service.CurrentDeploymentId == s.deployment.Id {
		return false, nil
	}
	if status.WorkStatusIsComplete(s.deployment.Status) {
		return false, nil
	}
	s.deployment.Status = status.WorkStatusCanceled
	return true, nil
}

type testDeploymentTransaction struct{}

func (testDeploymentTransaction) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return fn(ctx)
}

func (s *independentDeploymentStore) UpdateServiceStatus(_ context.Context, _, _, value string) error {
	s.service.Status = value
	return nil
}

type gatewayReadinessFake struct {
	calls   int
	gateway model.GatewayConfig
	timeout time.Duration
	err     error
}

func (f *gatewayReadinessFake) WaitUntilReady(_ context.Context, _ string, gateway model.GatewayConfig, timeout time.Duration) error {
	f.calls++
	f.gateway, f.timeout = gateway, timeout
	return f.err
}

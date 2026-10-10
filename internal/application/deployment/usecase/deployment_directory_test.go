package deploymentsvc

import (
	"context"
	"errors"
	"io"
	"path/filepath"
	"strings"
	"testing"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type directoryFailureRuntime struct {
	*workspaceFake
	stageErr, runErr error
	ran              bool
	location         deploymentport.ServiceLocation
	onStage, onRun   func()
}

func (r *directoryFailureRuntime) StageWorkspace(ctx context.Context, target environmentport.Target, workspace deploymentport.Workspace) error {
	if r.onStage != nil {
		r.onStage()
	}
	if r.stageErr != nil {
		return r.stageErr
	}
	return r.workspaceFake.StageWorkspace(ctx, target, workspace)
}
func (r *directoryFailureRuntime) Run(_ context.Context, _ environmentport.Target, location deploymentport.ServiceLocation, _ io.Writer, _ string, _ ...string) error {
	r.ran, r.location = true, location
	if r.onRun != nil {
		r.onRun()
	}
	return r.runErr
}

type directoryDispatcher struct{ deploymentport.Dispatcher }

func (directoryDispatcher) DispatchDeploy(context.Context, deploymentdto.DeployDispatchInput) error {
	return nil
}
func (directoryDispatcher) DispatchRestart(context.Context, deploymentdto.RestartDispatchInput) error {
	return nil
}
func (directoryDispatcher) DispatchStop(context.Context, deploymentdto.StopDispatchInput) error {
	return nil
}

type directoryVersionSelector struct {
	store *independentDeploymentStore
	calls int
}

func (s *directoryVersionSelector) SelectServiceDeploymentVersion(_ context.Context, _, _, _, versionId string) (model.Service, error) {
	s.calls++
	s.store.service.VersionId = versionId
	return s.store.service, nil
}

func TestDeploymentRequestFreezesVersionDirectoryAndRevision(t *testing.T) {
	directory := filepath.ToSlash(filepath.Join(t.TempDir(), "api"))
	service, store, _, _ := independentDeploymentTestService(t, false)
	service.commandStore, service.dispatcher = store, directoryDispatcher{}
	service.gatewayCoordinator = &gatewayDeploymentCoordinatorFake{gateway: &model.GatewayConfig{ApplicationId: "gateway-1", NetworkName: "traefik"}}
	selector := &directoryVersionSelector{store: store}
	service.versionSelector = selector
	store.version.Id = "selected-version"
	store.components[0].VersionId = store.version.Id
	result, err := service.DeployService(context.Background(), "user-1", "project-1", "service-1", deploymentdto.DeployServiceInput{
		VersionId: &store.version.Id, DeploymentDirectory: directory, EnvironmentTargetRevision: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if selector.calls != 1 || result.DeploymentId != store.deployment.Id || *store.deployment.VersionId != "selected-version" || *store.deployment.WorkingDirectory != directory || *store.deployment.EnvironmentTargetRevision != 1 {
		t.Fatalf("deployment=%+v selector calls=%d", store.deployment, selector.calls)
	}
	store.service.DeploymentDirectory = "/later/edit"
	store.service.VersionId = "later-version"
	if err := executeIndependentDeployment(service, "deploy"); err != nil {
		t.Fatal(err)
	}
	if filepath.ToSlash(store.service.RuntimeDirectory) != directory {
		t.Fatalf("runtime directory=%s", store.service.RuntimeDirectory)
	}
}

func TestDeploymentRejectsStaleEnvironmentBeforeChangingConfiguration(t *testing.T) {
	service, store, _, _ := independentDeploymentTestService(t, false)
	service.commandStore, service.dispatcher = store, directoryDispatcher{}
	selector := &directoryVersionSelector{store: store}
	service.versionSelector = selector
	_, err := service.DeployService(context.Background(), "user-1", "project-1", "service-1", deploymentdto.DeployServiceInput{
		VersionId: &store.version.Id, DeploymentDirectory: "/new/api", EnvironmentTargetRevision: 2,
	})
	if err == nil || selector.calls != 0 || store.service.DeploymentDirectory != "/custom/demo" {
		t.Fatalf("error=%v selector calls=%d directory=%s", err, selector.calls, store.service.DeploymentDirectory)
	}
}

func TestRestartFreezesRuntimeDirectoryInsteadOfPendingDirectory(t *testing.T) {
	service, store, _, _ := independentDeploymentTestService(t, false)
	service.commandStore, service.dispatcher = store, directoryDispatcher{}
	service.gatewayCoordinator = &gatewayDeploymentCoordinatorFake{gateway: &model.GatewayConfig{ApplicationId: "gateway-1", NetworkName: "traefik"}}
	store.service.Status = status.ServiceStatusRunning
	store.service.RuntimeDirectory, store.service.RuntimeTargetRevision = "/running/api", 1
	if _, err := service.RestartApplication(context.Background(), "user-1", "project-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1"}); err != nil {
		t.Fatal(err)
	}
	if *store.deployment.WorkingDirectory != "/running/api" || store.service.DeploymentDirectory != "/custom/demo" {
		t.Fatalf("deployment=%+v service=%+v", store.deployment, store.service)
	}
	if err := executeIndependentDeployment(service, "restart"); err != nil {
		t.Fatal(err)
	}
	if store.service.RuntimeDirectory != "/running/api" {
		t.Fatalf("runtime directory=%s", store.service.RuntimeDirectory)
	}
}

func TestRuntimeOperationsRejectMissingServiceDirectoryWithActionableError(t *testing.T) {
	for _, operation := range []string{"stop", "restart"} {
		t.Run(operation, func(t *testing.T) {
			service, store, _, _ := independentDeploymentTestService(t, false)
			service.commandStore, service.dispatcher = store, directoryDispatcher{}
			store.service.Status = status.ServiceStatusFaulted
			store.service.DeploymentDirectory = ""
			input := deploymentdto.ServiceTargetInput{ServiceId: store.service.Id}
			var err error
			if operation == "stop" {
				_, err = service.StopApplication(context.Background(), "user-1", "project-1", "app-1", input)
			} else {
				_, err = service.RestartApplication(context.Background(), "user-1", "project-1", "app-1", input)
			}
			if !apperror.IsKind(err, apperror.KindConflict) || !errors.Is(err, errRuntimeDirectoryNotReady) {
				t.Fatalf("expected a missing runtime directory conflict, got %v", err)
			}
			classification := apperror.Classify(err)
			if classification.Code != "service_runtime_directory_missing" || !strings.Contains(classification.Message, "Confirm the service directory") {
				t.Fatalf("unhelpful public error: %+v", classification)
			}
			for _, context := range []string{"project_id=project-1", "application_id=app-1", "service_id=service-1", "service_code=demo-default", "service_status=faulted", "environment_target_revision=1", "runtime_target_revision=0"} {
				if !strings.Contains(err.Error(), context) {
					t.Fatalf("error is missing %q: %v", context, err)
				}
				if strings.Contains(classification.Message, context) {
					t.Fatalf("internal context leaked into public message: %s", classification.Message)
				}
			}
			if store.deployment.Id != "deployment-1" {
				t.Fatal("rejected operation created a deployment")
			}
		})
	}
}

func TestLifecycleOperationsUseConfirmedDirectoryWithoutRuntimeRecord(t *testing.T) {
	for _, operation := range []string{"stop", "restart"} {
		for _, serviceState := range []string{status.ServiceStatusStopped, status.ServiceStatusRunning} {
			t.Run(operation+"/"+serviceState, func(t *testing.T) {
				service, store, _, _ := independentDeploymentTestService(t, false)
				service.commandStore, service.dispatcher = store, directoryDispatcher{}
				service.gatewayCoordinator = &gatewayDeploymentCoordinatorFake{gateway: &model.GatewayConfig{ApplicationId: "gateway-1", NetworkName: "traefik"}}
				store.service.Status = serviceState
				input := deploymentdto.ServiceTargetInput{ServiceId: store.service.Id}
				var id string
				var err error
				if operation == "stop" {
					id, err = service.StopApplication(context.Background(), "user-1", "project-1", "app-1", input)
				} else {
					id, err = service.RestartApplication(context.Background(), "user-1", "project-1", "app-1", input)
				}
				if err != nil {
					t.Fatal(err)
				}
				if store.deployment.WorkingDirectory == nil || *store.deployment.WorkingDirectory != "/custom/demo" || store.service.CurrentDeploymentId == nil || *store.service.CurrentDeploymentId != id {
					t.Fatalf("deployment=%+v service=%+v", store.deployment, store.service)
				}
			})
		}
	}
}

func TestCanceledDeploymentKeepsTheDirectoryBoundAtCancellation(t *testing.T) {
	for _, duringPreparation := range []bool{true, false} {
		t.Run(map[bool]string{true: "preparation", false: "command"}[duringPreparation], func(t *testing.T) {
			service, store, _, _ := independentDeploymentTestService(t, false)
			store.service.RuntimeDirectory, store.service.RuntimeTargetRevision = "/old/service", 1
			runtime := &directoryFailureRuntime{workspaceFake: testWorkspace(t.TempDir())}
			cancel := func() { store.deployment.Status = status.WorkStatusCanceled }
			want := "/custom/demo"
			if duringPreparation {
				runtime.onStage, runtime.stageErr, want = cancel, context.Canceled, "/old/service"
			} else {
				runtime.onRun, runtime.runErr = cancel, context.Canceled
			}
			service.runtime, service.logStore = runtime, runtime
			if err := executeIndependentDeployment(service, "deploy"); err != nil {
				t.Fatal(err)
			}
			if store.service.RuntimeDirectory != want || store.deployment.Status != status.WorkStatusCanceled {
				t.Fatalf("directory=%s deployment status=%s", store.service.RuntimeDirectory, store.deployment.Status)
			}
		})
	}
}

func TestDeploymentDirectoryBindingFollowsPreparation(t *testing.T) {
	for _, stageFailure := range []bool{true, false} {
		t.Run(map[bool]string{true: "preparation failure", false: "command failure"}[stageFailure], func(t *testing.T) {
			service, store, _, _ := independentDeploymentTestService(t, false)
			store.service.RuntimeDirectory, store.service.RuntimeTargetRevision = "/old/service", 1
			failure := errors.New("target operation failed")
			runtime := &directoryFailureRuntime{workspaceFake: testWorkspace(t.TempDir()), runErr: failure}
			if stageFailure {
				runtime.stageErr = failure
			}
			service.runtime, service.logStore = runtime, runtime
			if err := executeIndependentDeployment(service, "deploy"); !errors.Is(err, failure) {
				t.Fatalf("error=%v", err)
			}
			want := "/custom/demo"
			if stageFailure {
				want = "/old/service"
			}
			if store.service.RuntimeDirectory != want || runtime.ran == stageFailure {
				t.Fatalf("directory=%s ran=%t", store.service.RuntimeDirectory, runtime.ran)
			}
			if runtime.ran && runtime.location.Directory != "/custom/demo" {
				t.Fatalf("command location=%+v", runtime.location)
			}
		})
	}
}

func TestComposeMapsEachRelativeBindSource(t *testing.T) {
	service, store, _, _ := independentDeploymentTestService(t, true)
	plan, _, err := BuildEffectiveServicePlan(store.application, store.version, store.service, store.components, store.serviceComponents, nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	plan.Gateway = testGatewayEnrichmentPlan("", "", "").Gateway
	result, err := service.RenderComposeDetailed(context.Background(), RenderInput{
		Plan: plan, LogicalSvcDir: "/app/custom", ComposeMountSourceDir: "/host/custom",
		ResolveMountSource: func(_ context.Context, source string) (string, error) {
			source = filepath.ToSlash(source)
			if strings.HasSuffix(source, "/gateway/acme") {
				return "/separate/acme", nil
			}
			return strings.Replace(source, "/app/custom", "/host/custom", 1), nil
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(result.Compose, "/separate/acme:/letsencrypt") || !strings.Contains(result.Compose, "/host/custom/gateway/dynamic:") {
		t.Fatalf("mapped Compose:\n%s", result.Compose)
	}
}

type directoryOwnershipStore struct {
	*runtimeQueryStore
	services []model.Service
}

func (s *directoryOwnershipStore) DirectoryServices(context.Context, string) ([]model.Service, error) {
	return s.services, nil
}

func TestDirectoryOwnershipChecksNestedDockerMountSources(t *testing.T) {
	service := model.Service{Id: "current", Code: "api", ProjectId: "project-1"}
	store := &directoryOwnershipStore{services: []model.Service{{
		Id: "other", Code: "other", DeploymentDirectory: "/other-owned", DirectoryTargetRevision: 1,
	}}}
	usecase := Service{executionStore: store, runtime: testWorkspace(t.TempDir())}
	target := deploymentTestTarget(1)
	if err := usecase.checkDirectoryOwnership(context.Background(), target, service, "/outside/workspace", nil); err != nil {
		t.Fatal(err)
	}
	if err := usecase.checkDirectoryOwnership(context.Background(), target, service, "/other-owned/child", nil); err == nil {
		t.Fatal("overlapping service directories were accepted")
	}
	if err := usecase.checkDirectoryOwnership(context.Background(), target, service, "/outside/workspace", []ResolvedMount{{Relative: true, HostSource: "/other-owned/data"}}); err == nil {
		t.Fatal("nested Docker mapping overlapped another service")
	}
}

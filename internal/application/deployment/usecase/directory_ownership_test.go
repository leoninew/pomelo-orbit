package deploymentsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type directoryMountRuntime struct {
	*directoryFailureRuntime
	mounts     []deploymentport.RunningBindMount
	inspectErr error
	queries    []string
}

func (r *directoryMountRuntime) ComposeMountSourceDir(_ context.Context, _ environmentport.Target, location deploymentport.ServiceLocation) (string, error) {
	return strings.Replace(location.Directory, "/app/", "/host/", 1), nil
}

func (r *directoryMountRuntime) RunningBindMounts(_ context.Context, _ environmentport.Target, serviceCode string) ([]deploymentport.RunningBindMount, error) {
	r.queries = append(r.queries, serviceCode)
	return r.mounts, r.inspectErr
}
func directoryMountTestService(t *testing.T) (ExecutionService, *directoryMountRuntime, model.Service) {
	t.Helper()
	runtime := &directoryMountRuntime{directoryFailureRuntime: &directoryFailureRuntime{workspaceFake: testWorkspace(t.TempDir())}}
	current := model.Service{Id: "b", Code: "b", ProjectId: "project-1", DeploymentDirectory: "/app/b", DirectoryTargetRevision: 1}
	other := model.Service{Id: "a", Code: "a", RuntimeDirectory: "/app/a", RuntimeTargetRevision: 1, Status: status.ServiceStatusStopped}
	store := &directoryOwnershipStore{runtimeQueryStore: &runtimeQueryStore{service: current}, services: []model.Service{current, other}}
	return NewExecutionService(store, nil, nil, runtime, runtime, 0, 0, 0, nil, nil, runtime, nil, nil), runtime, current
}

func TestDirectoryOwnershipRejectsRunningSharedMounts(t *testing.T) {
	for _, source := range []string{"/host/shared", "/host/shared/data", "/host"} {
		t.Run(source, func(t *testing.T) {
			service, runtime, current := directoryMountTestService(t)

			runtime.mounts = []deploymentport.RunningBindMount{{Source: "/different/data"}, {Source: source}}
			err := service.checkDirectoryOwnership(context.Background(), deploymentTestTarget(1), current, "/app/b", []ResolvedMount{{Relative: true, HostSource: "/host/shared"}})
			if err == nil || !strings.Contains(err.Error(), "running service a") {
				t.Fatalf("expected running service conflict, got %v", err)
			}
			if len(runtime.queries) != 1 || runtime.queries[0] != "a" {
				t.Fatalf("queried services=%v", runtime.queries)
			}
		})
	}
}

func TestDirectoryOwnershipAllowsUnusedDataDirectories(t *testing.T) {
	for _, mounts := range [][]deploymentport.RunningBindMount{nil, {{Source: "/other/data"}}} {
		service, runtime, current := directoryMountTestService(t)
		runtime.mounts = mounts
		if err := service.checkDirectoryOwnership(context.Background(), deploymentTestTarget(1), current, "/app/b", []ResolvedMount{{Relative: true, HostSource: "/host/shared"}}); err != nil {
			t.Fatal(err)
		}
	}
}
func TestDirectoryOwnershipUsesTargetMountPaths(t *testing.T) {
	for _, tc := range []struct{ name, platform, desired, actual string }{
		{"local", "", "/host/shared", "/host/shared"},
		{"DooD on Docker Desktop", "", "/run/desktop/mnt/host/d/Shared/Data", "/host_mnt/d/shared/data"},
		{"Linux SSH", "linux", "/host/shared", "/host/shared"},
		{"Windows SSH", "windows", "D:/Shared/Data", `d:\shared\data`},
		{"Windows Docker Desktop", "windows", "D:/Shared/Data", "/run/desktop/mnt/host/d/shared/data"},
		{"Windows host mount", "windows", "D:/Shared/Data", "/host_mnt/d/shared/data"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, runtime, current := directoryMountTestService(t)
			target := deploymentTestTarget(1)
			directory := current.DeploymentDirectory
			if tc.platform != "" {
				target = testSSHTarget(current.ProjectId)
				target.Environment.SSH.Platform = tc.platform
			}
			if tc.platform == "windows" {
				directory = "D:/Apps/b"
				service.executionStore.(*directoryOwnershipStore).services[1].RuntimeDirectory = "D:/Apps/a"
			}

			runtime.mounts = []deploymentport.RunningBindMount{{Source: tc.actual}}
			if err := service.checkDirectoryOwnership(context.Background(), target, current, directory, []ResolvedMount{{Relative: true, HostSource: tc.desired}}); err == nil || !strings.Contains(err.Error(), "running service a") {
				t.Fatal("running mount conflict was accepted")
			}
		})
	}
}

func TestDirectoryOwnershipBlocksWhenMountInspectionFails(t *testing.T) {
	service, runtime, current := directoryMountTestService(t)
	failure := errors.New("inventory unavailable")
	runtime.inspectErr = failure
	if err := service.checkDirectoryOwnership(context.Background(), deploymentTestTarget(1), current, "/app/b", nil); !errors.Is(err, failure) {
		t.Fatalf("inspection failure was lost: %v", err)
	}
}
func TestDirectoryOwnershipAllowsOwnRunningMounts(t *testing.T) {
	service, runtime, current := directoryMountTestService(t)
	service.executionStore = &directoryOwnershipStore{services: []model.Service{current}}
	if err := service.checkDirectoryOwnership(context.Background(), deploymentTestTarget(1), current, "/app/b", nil); err != nil {
		t.Fatal(err)
	}
	if len(runtime.queries) != 0 {
		t.Fatal("checked this service's own containers for a conflict")
	}
}

func TestDeploymentRejectsOccupiedMountBeforePreparation(t *testing.T) {
	service, runtime, current := directoryMountTestService(t)
	current.RuntimeDirectory, current.RuntimeTargetRevision = "/old/b", 1
	service.executionStore.(*directoryOwnershipStore).service = current
	plan := model.EffectiveServicePlan{
		Application: model.Application{Code: "app", Kind: status.ApplicationKindStandard},
		Service:     current,
		Components: []model.EffectiveServiceComponent{{
			Name: "web", Image: "nginx:latest",
			Mounts: []model.VersionComponentMount{{SourceType: "directory", Source: "./data", Target: "/data"}},
		}},
		JoinTraefikNetwork: new(bool),
	}

	runtime.mounts = []deploymentport.RunningBindMount{{Source: "/host/b/data"}}
	if err := service.renderAndDeployWithOptions(context.Background(), deploymentTestTarget(1), plan, "deployment-1", deploymentdto.DeployOptionsJSON{PullPolicy: "missing"}); err == nil || !strings.Contains(err.Error(), "running service a") {
		t.Fatalf("expected occupied data directory conflict, got %v", err)
	}
	if len(runtime.staged) != 0 || runtime.ran || service.executionStore.(*directoryOwnershipStore).service.RuntimeDirectory != "/old/b" {
		t.Fatal("occupied directory caused file preparation or changed the running service")
	}
}

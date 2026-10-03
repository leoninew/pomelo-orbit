package deploymentsvc

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type directoryMountRuntime struct {
	*directoryFailureRuntime
	psOutput, inspectOutput string
	psErr, inspectErr       error
	queries                 []composeCommand
}

func (r *directoryMountRuntime) ComposeMountSourceDir(_ context.Context, _ environmentport.Target, location deploymentport.ServiceLocation) (string, error) {
	return strings.Replace(location.Directory, "/app/", "/host/", 1), nil
}

func (r *directoryMountRuntime) QueryAtEnvironmentRoot(_ context.Context, _ environmentport.Target, name string, args ...string) (string, error) {
	r.queries = append(r.queries, composeCommand{Name: name, Args: args})
	if args[0] == "ps" {
		return r.psOutput, r.psErr
	}
	return r.inspectOutput, r.inspectErr
}

func directoryMountFixture(t *testing.T, running bool, mountType, source string) string {
	t.Helper()
	body, err := json.Marshal(map[string]any{
		"Running": running,
		"Mounts":  []map[string]string{{"Type": mountType, "Source": source}},
	})
	if err != nil {
		t.Fatal(err)
	}
	return string(body)
}

func directoryMountTestService(t *testing.T) (Service, *directoryMountRuntime, model.Service) {
	t.Helper()
	runtime := &directoryMountRuntime{directoryFailureRuntime: &directoryFailureRuntime{workspaceFake: testWorkspace(t.TempDir())}}
	current := model.Service{Id: "b", Code: "b", ProjectId: "project-1", DeploymentDirectory: "/app/b", DirectoryTargetRevision: 1}
	other := model.Service{Id: "a", Code: "a", RuntimeDirectory: "/app/a", RuntimeTargetRevision: 1, Status: status.ServiceStatusStopped}
	store := &directoryOwnershipStore{runtimeQueryStore: &runtimeQueryStore{service: current}, services: []model.Service{current, other}}
	return Service{executionStore: store, runtime: runtime, logStore: runtime}, runtime, current
}

func TestDirectoryOwnershipRejectsRunningSharedMounts(t *testing.T) {
	for _, source := range []string{"/host/shared", "/host/shared/data", "/host"} {
		t.Run(source, func(t *testing.T) {
			service, runtime, current := directoryMountTestService(t)
			runtime.psOutput = "container-one\ncontainer-two\n"
			runtime.inspectOutput = directoryMountFixture(t, true, "bind", "/different/data") + "\n" + directoryMountFixture(t, true, "bind", source)
			err := service.checkDirectoryOwnership(context.Background(), deploymentTestTarget(1), current, "/app/b", []ResolvedMount{{Relative: true, HostSource: "/host/shared"}})
			if err == nil || !strings.Contains(err.Error(), "running service a") {
				t.Fatalf("expected running service conflict, got %v", err)
			}
			if len(runtime.queries) != 2 || runtime.queries[0].Name != "docker" || !strings.Contains(runtime.queries[0].String(), "label=com.docker.compose.project=a") || !strings.Contains(runtime.queries[0].String(), "status=running") || !strings.Contains(runtime.queries[1].String(), "container-one container-two") {
				t.Fatalf("runtime queries=%+v", runtime.queries)
			}
		})
	}
}

func TestDirectoryOwnershipAllowsUnusedDataDirectories(t *testing.T) {
	for _, tc := range []struct {
		name, ids, mountType, source string
		running                      bool
	}{
		{"no running containers", "", "bind", "/host/shared", true},
		{"stopped after listing", "container-one", "bind", "/host/shared", false},
		{"unrelated bind", "container-one", "bind", "/other/data", true},
		{"named volume", "container-one", "volume", "/host/shared", true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			service, runtime, current := directoryMountTestService(t)
			runtime.psOutput = tc.ids
			runtime.inspectOutput = directoryMountFixture(t, tc.running, tc.mountType, tc.source)
			if err := service.checkDirectoryOwnership(context.Background(), deploymentTestTarget(1), current, "/app/b", []ResolvedMount{{Relative: true, HostSource: "/host/shared"}}); err != nil {
				t.Fatal(err)
			}
			if tc.ids == "" && len(runtime.queries) != 1 {
				t.Fatalf("inspected containers despite an empty running list: %+v", runtime.queries)
			}
		})
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
			runtime.psOutput = "container-one"
			runtime.inspectOutput = directoryMountFixture(t, true, "bind", tc.actual)
			if err := service.checkDirectoryOwnership(context.Background(), target, current, directory, []ResolvedMount{{Relative: true, HostSource: tc.desired}}); err == nil || !strings.Contains(err.Error(), "running service a") {
				t.Fatal("running mount conflict was accepted")
			}
		})
	}
}

func TestDirectoryOwnershipBlocksWhenMountInspectionFails(t *testing.T) {
	failure := errors.New("Docker unavailable")
	for _, operation := range []string{"ps", "inspect", "invalid JSON"} {
		t.Run(operation, func(t *testing.T) {
			service, runtime, current := directoryMountTestService(t)
			runtime.psOutput = "container-one"
			switch operation {
			case "ps":
				runtime.psErr = failure
			case "inspect":
				runtime.inspectErr = failure
			default:
				runtime.inspectOutput = "invalid JSON"
			}
			if err := service.checkDirectoryOwnership(context.Background(), deploymentTestTarget(1), current, "/app/b", nil); err == nil || operation != "invalid JSON" && !errors.Is(err, failure) {
				t.Fatalf("inspection failure was lost: %v", err)
			}
		})
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
	runtime.psOutput = "container-one"
	runtime.inspectOutput = directoryMountFixture(t, true, "bind", "/host/b/data")
	if err := service.renderAndDeployWithOptions(context.Background(), deploymentTestTarget(1), plan, "deployment-1", false); err == nil || !strings.Contains(err.Error(), "running service a") {
		t.Fatalf("expected occupied data directory conflict, got %v", err)
	}
	if len(runtime.staged) != 0 || runtime.ran || service.executionStore.(*directoryOwnershipStore).service.RuntimeDirectory != "/old/b" {
		t.Fatal("occupied directory caused file preparation or changed the running service")
	}
}

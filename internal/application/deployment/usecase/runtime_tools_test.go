package deploymentsvc

import (
	"context"
	"strings"
	"testing"
	"testing/synctest"
	"time"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

type runtimeInspectorFake struct {
	deploymentport.RuntimeInspector
	observation deploymentport.ContainerObservation
	err         error
	scopes      []deploymentport.RuntimeScope
	inspections []deploymentport.ContainerInspection
	inspected   []string
	networks    []string
	probes      []string
}

func (f *runtimeInspectorFake) Containers(_ context.Context, scope deploymentport.RuntimeScope) (deploymentport.ContainerObservation, error) {
	f.scopes = append(f.scopes, scope)
	result := f.observation
	result.Containers = append([]deploymentdto.RuntimeContainer(nil), result.Containers...)
	return result, f.err
}

func (f *runtimeInspectorFake) InspectContainer(_ context.Context, _ deploymentport.RuntimeScope, id string) (deploymentport.ContainerInspection, error) {
	f.inspected = append(f.inspected, id)
	index := min(len(f.inspected)-1, len(f.inspections)-1)
	return f.inspections[index], f.err
}

func (f *runtimeInspectorFake) InspectNetwork(_ context.Context, _ deploymentport.RuntimeScope, name string) (any, error) {
	f.networks = append(f.networks, name)
	return "network diagnostics", f.err
}

func (f *runtimeInspectorFake) HTTPProbe(_ context.Context, _ deploymentport.RuntimeScope, component string, _ int, _ string) (string, error) {
	f.probes = append(f.probes, component)
	return "healthy", f.err
}

func newRuntimeInspectionService() (Service, *runtimeInspectorFake, deploymentdto.RuntimeTarget) {
	_, store := newRuntimeQueryService()
	store.version = model.Version{Id: "version-1"}
	store.components = []model.VersionComponent{{Id: "component-1", Name: "web"}}
	store.service.RuntimeDirectory, store.service.RuntimeTargetRevision = "/custom/demo", 1
	inspector := &runtimeInspectorFake{observation: deploymentport.ContainerObservation{Containers: []deploymentdto.RuntimeContainer{{Id: "container-1", Service: "web", State: "running"}}, Raw: "container diagnostics"}, inspections: []deploymentport.ContainerInspection{{Raw: "inspect diagnostics", Networks: []string{"demo-default_backend"}, State: "running"}}}
	service := NewCommandService(store, store, store, store, nil, nil, staticTargetResolver{target: testSSHTarget("project-1")}, &workspaceFake{hasServiceDir: true}, nil, nil, nil, nil, nil, inspector, lifecycleCommandsFake{})
	target := deploymentdto.RuntimeTarget{ProjectId: "project-1", ApplicationId: "app-1", ServiceId: "service-1", ServiceCode: "demo-default", ComposeProject: "demo-default", WorkingDirectory: "/custom/demo"}
	return service, inspector, target
}

func TestRuntimeInspectionRequiresManagedContainerAndNetwork(t *testing.T) {
	service, inspector, target := newRuntimeInspectionService()
	ctx := context.Background()
	if _, err := service.RuntimeContainerInspect(ctx, target, "other-container"); err == nil || len(inspector.inspected) != 0 {
		t.Fatalf("unmanaged container inspected: %v, %v", inspector.inspected, err)
	}
	result, err := service.RuntimeContainerInspect(ctx, target, "container-1")
	if err != nil || result.Data != "inspect diagnostics" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	if _, err := service.RuntimeNetworkInspect(ctx, target, "other-network"); err == nil || len(inspector.networks) != 0 {
		t.Fatalf("unmanaged network inspected: %v, %v", inspector.networks, err)
	}
	network, err := service.RuntimeNetworkInspect(ctx, target, "backend")
	if err != nil || network.Data != "network diagnostics" || len(inspector.networks) != 1 || inspector.networks[0] != "demo-default_backend" {
		t.Fatalf("network=%+v err=%v", network, err)
	}
}

func TestRuntimeStatusAddsVersionIdentityToObservedContainers(t *testing.T) {
	service, inspector, target := newRuntimeInspectionService()
	store := service.service.(*runtimeQueryStore)
	store.service.RuntimeTargetRevision = 7
	ctx := context.Background()
	resolved, err := service.ResolveRuntimeTarget(ctx, "user-1", "project-1", "service-1", false)
	if err != nil || resolved.WorkingDirectory != target.WorkingDirectory {
		t.Fatalf("historical runtime directory=%+v err=%v", resolved, err)
	}
	result, err := service.RuntimeComposePS(ctx, target)
	if err != nil || len(result.Containers) != 1 || result.Containers[0].ComponentId != "component-1" || result.Containers[0].VersionId != "version-1" || result.Raw != "container diagnostics" {
		t.Fatalf("result=%+v err=%v", result, err)
	}
	containers, err := service.ApplicationStatus(ctx, "user-1", "project-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1"})
	if err != nil || len(containers) != 1 || containers[0].ComponentId != "component-1" {
		t.Fatalf("containers=%v err=%v", containers, err)
	}
	for _, scope := range inspector.scopes {
		if scope.Target.Environment.Id != "environment-1" || scope.Location.Directory != "/custom/demo" || scope.Project != "demo-default" {
			t.Fatalf("scope=%+v", scope)
		}
	}
}

func TestRuntimeProbeRequiresDeclaredComponent(t *testing.T) {
	service, inspector, target := newRuntimeInspectionService()
	if _, err := service.RuntimeHTTPProbe(context.Background(), target, "other", 80, "/health"); err == nil || len(inspector.probes) != 0 {
		t.Fatalf("undeclared component probed: %v %v", inspector.probes, err)
	}
	result, err := service.RuntimeHTTPProbe(context.Background(), target, "web", 80, "/health")
	if err != nil || result.Text != "healthy" || len(inspector.probes) != 1 {
		t.Fatalf("result=%+v err=%v", result, err)
	}
}

func TestRuntimeStabilityRejectsIncreasingRestartCount(t *testing.T) {
	synctest.Test(t, func(t *testing.T) {
		service, inspector, target := newRuntimeInspectionService()
		inspector.inspections = []deploymentport.ContainerInspection{{State: "running", RestartCount: 1}, {State: "running", RestartCount: 2}}
		result, err := service.observeRuntimeStability(context.Background(), target, deploymentdto.DeploymentVerificationInput{StabilityWindow: 10 * time.Second, StabilityPoll: 2 * time.Second})
		if err != nil || result["state"] != "failed" {
			t.Fatalf("result=%+v err=%v", result, err)
		}
		issues, _ := result["issues"].([]string)
		if len(issues) != 1 || !strings.Contains(issues[0], "restart count increased") || len(inspector.inspected) != 2 {
			t.Fatalf("issues=%v inspections=%v", issues, inspector.inspected)
		}
	})
}

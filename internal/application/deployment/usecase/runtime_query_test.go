package deploymentsvc

import (
	"context"
	"testing"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
)

type statusQueryRunner struct {
	called bool
}

func (r *statusQueryRunner) Run(context.Context, string, string, ...string) (string, error) {
	r.called = true
	return "", context.Canceled
}

func TestApplicationStatusReturnsNoContainersBeforeFirstDeployment(t *testing.T) {
	service, store, _ := newCommandTestService()
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

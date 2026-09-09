package deploymentsvc

import (
	"context"
	"testing"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestObservedServiceStatus(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		containers []deploymentdto.RuntimeContainer
		want       string
	}{
		{name: "no containers", want: status.ServiceStatusStopped},
		{name: "all running", containers: []deploymentdto.RuntimeContainer{{State: "running"}, {State: "RUNNING"}}, want: status.ServiceStatusRunning},
		{name: "non-running container", containers: []deploymentdto.RuntimeContainer{{State: "running"}, {State: "exited"}}, want: status.ServiceStatusFaulted},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := observedServiceStatus(test.containers); got != test.want {
				t.Fatalf("observed status = %q, want %q", got, test.want)
			}
		})
	}
}

func TestReconcileCanceledServiceUsesObservedRuntimeWithoutChangingDeployment(t *testing.T) {
	tests := []struct {
		name      string
		workspace bool
		output    string
		runErr    error
		want      string
	}{
		{name: "running", workspace: true, output: `[{"State":"running"}]`, want: status.ServiceStatusRunning},
		{name: "stopped", workspace: true, output: "[]", want: status.ServiceStatusStopped},
		{name: "unobservable", workspace: true, runErr: context.Canceled, want: status.ServiceStatusFaulted},
		{name: "workspace absent", want: status.ServiceStatusStopped},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &canceledServiceStore{}
			runtime := testWorkspace(t.TempDir())
			runtime.hasServiceDir = test.workspace
			runtime.queryOutput = test.output
			runtime.queryErr = test.runErr
			service := Service{executionStore: store, runtime: runtime}
			service.reconcileCanceledService(context.Background(), environmentport.Target{}, model.Application{Code: "demo"}, model.Service{Id: "service-1", InstanceKey: "default"})
			if store.status != test.want {
				t.Fatalf("service status = %q, want %q", store.status, test.want)
			}
		})
	}
}

type canceledServiceStore struct {
	deploymentport.ExecutionStore
	status string
}

func (s *canceledServiceStore) UpdateServiceStatus(_ context.Context, _ string, value string) error {
	s.status = value
	return nil
}

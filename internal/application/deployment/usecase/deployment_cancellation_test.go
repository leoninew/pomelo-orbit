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
		name       string
		containers []deploymentdto.RuntimeContainer
		runErr     error
		want       string
	}{
		{name: "running", containers: []deploymentdto.RuntimeContainer{{State: "running"}}, want: status.ServiceStatusRunning},
		{name: "stopped", want: status.ServiceStatusStopped},
		{name: "unobservable", runErr: context.Canceled},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			store := &canceledServiceStore{}
			observer := &runtimeInspectorFake{observation: deploymentport.ContainerObservation{Containers: test.containers}, err: test.runErr}
			service := NewExecutionService(store, nil, nil, nil, nil, 0, 0, 0, nil, nil, nil, observer, nil)
			service.reconcileCanceledService(context.Background(), "project-1", "deployment-1", environmentport.Target{}, model.Service{Id: "service-1", Code: "demo-default"})
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

func (s *canceledServiceStore) ReconcileServiceAfterCancellation(_ context.Context, _, _, _ string, value string) (bool, error) {
	s.status = value
	return true, nil
}

func (s *canceledServiceStore) Service(context.Context, string, string) (model.Service, error) {
	currentId := "deployment-1"
	return model.Service{Id: "service-1", Code: "demo-default", RuntimeDirectory: "/custom/demo", CurrentDeploymentId: &currentId}, nil
}

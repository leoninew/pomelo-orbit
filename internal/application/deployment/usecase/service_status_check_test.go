package deploymentsvc

import (
	"context"
	"errors"
	"reflect"
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type statusCheckProject struct{ repository.ProjectReader }

func (statusCheckProject) Project(context.Context, string) (model.Project, error) {
	return model.Project{Id: "project-1"}, nil
}
func (statusCheckProject) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type statusCheckStore struct {
	services []model.Service
	saved    []serviceStatusObservation
	changed  bool
}

func (s *statusCheckStore) DirectoryServices(context.Context, string) ([]model.Service, error) {
	return append([]model.Service(nil), s.services...), nil
}
func (s *statusCheckStore) UpdateServiceObservedStatus(_ context.Context, baseline model.Service, environmentId string, revision int64, state string) (bool, error) {
	if environmentId != "environment-1" || revision != 1 {
		return false, errors.New("incorrect observation target")
	}
	s.saved = append(s.saved, serviceStatusObservation{service: baseline, status: state})
	return s.changed, nil
}

type statusCheckInventory struct {
	containers map[string][]deploymentport.ContainerState
	err        error
	calls      int
}

func (i *statusCheckInventory) Containers(_ context.Context, target environmentport.Target) (map[string][]deploymentport.ContainerState, error) {
	i.calls++
	if target.Environment.Id != "environment-1" {
		return nil, errors.New("incorrect inventory target")
	}
	return i.containers, i.err
}
func newStatusCheckTestService() (*StatusCheckService, *statusCheckStore, *statusCheckInventory) {
	store := &statusCheckStore{changed: true, services: []model.Service{
		{Id: "service-1", ProjectId: "project-1", Code: "example", Status: status.ServiceStatusStopped, VersionId: "version-1", DeploymentDirectory: "/custom/example"},
		{Id: "service-2", ProjectId: "project-1", Code: "other", Status: status.ServiceStatusRunning, VersionId: "version-1"},
	}}
	inventory := &statusCheckInventory{containers: map[string][]deploymentport.ContainerState{"example": {{State: "running"}}, "other": {{State: "exited"}}}}
	resolver := staticTargetResolver{target: environmentport.Target{Environment: model.Environment{Id: "environment-1", ProjectId: "project-1", TargetRevision: 1}}}
	return NewStatusCheckService(statusCheckProject{}, store, resolver, inventory), store, inventory
}

func TestServiceStatusCheckObservesBatchAndConfirmsEachService(t *testing.T) {
	service, store, inventory := newStatusCheckTestService()
	ctx := context.Background()
	preview, err := service.PreviewServiceStatusCheck(ctx, "user-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 2 || preview.Items[0].ObservedStatus != status.ServiceStatusRunning || preview.Items[1].ObservedStatus != status.ServiceStatusStopped || len(store.saved) != 0 {
		t.Fatalf("preview=%+v writes=%v", preview, store.saved)
	}
	if _, err := service.ConfirmServiceStatusCheck(ctx, "other-user", "project-1", preview.Token, "service-1"); !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("another actor used snapshot: %v", err)
	}
	for _, item := range preview.Items {
		state, err := service.ConfirmServiceStatusCheck(ctx, "user-1", "project-1", preview.Token, item.ServiceId)
		if err != nil || state != item.ObservedStatus {
			t.Fatalf("confirm=%s %v", state, err)
		}
	}
	if inventory.calls != 1 || len(store.saved) != 2 || !reflect.DeepEqual(store.saved[0].service, store.services[0]) || store.saved[0].status != status.ServiceStatusRunning {
		t.Fatalf("calls=%d saved=%+v", inventory.calls, store.saved)
	}
}

func TestServiceStatusCheckFailedQueryLeavesRecordsUntouched(t *testing.T) {
	service, store, inventory := newStatusCheckTestService()
	inventory.err = errors.New("Docker daemon connection refused")
	if _, err := service.PreviewServiceStatusCheck(context.Background(), "user-1", "project-1"); !apperror.IsKind(err, apperror.KindUnavailable) || !errors.Is(err, inventory.err) {
		t.Fatalf("error=%v", err)
	}
	if len(store.saved) != 0 || len(service.checks.snapshots) != 0 {
		t.Fatal("failed query saved records or published a snapshot")
	}
}

func TestServiceStatusCheckReportsConditionalWriteConflict(t *testing.T) {
	service, store, inventory := newStatusCheckTestService()
	ctx := context.Background()
	preview, err := service.PreviewServiceStatusCheck(ctx, "user-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	store.changed = false
	if _, err := service.ConfirmServiceStatusCheck(ctx, "user-1", "project-1", preview.Token, "service-1"); !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("conflict=%v", err)
	}
	if inventory.calls != 1 {
		t.Fatal("confirmation repeated external observation")
	}
}
func TestInventoryServiceStatus(t *testing.T) {
	tests := []struct {
		name       string
		containers []deploymentport.ContainerState
		want       string
	}{
		{name: "no containers", want: status.ServiceStatusStopped},
		{name: "stopped outside Orbit", containers: []deploymentport.ContainerState{{State: "exited"}, {State: "exited"}}, want: status.ServiceStatusStopped},
		{name: "started outside Orbit", containers: []deploymentport.ContainerState{{State: "running"}, {State: "running"}}, want: status.ServiceStatusRunning},
		{name: "partial stop", containers: []deploymentport.ContainerState{{State: "running"}, {State: "exited"}}, want: status.ServiceStatusFaulted},
		{name: "restarting", containers: []deploymentport.ContainerState{{State: "restarting"}}, want: status.ServiceStatusFaulted},
		{name: "unhealthy", containers: []deploymentport.ContainerState{{State: "running", Unhealthy: true}}, want: status.ServiceStatusFaulted},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := inventoryServiceStatus(test.containers); got != test.want {
				t.Fatalf("status = %q, want %q", got, test.want)
			}
		})
	}
}

package deploymentsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	servicerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/service"
)

type statusCheckRuntime struct {
	deploymentport.Runtime
	output string
	err    error
	calls  int
}

func (r *statusCheckRuntime) QueryAtEnvironmentRoot(_ context.Context, target environmentport.Target, name string, args ...string) (string, error) {
	r.calls++
	if target.Environment.Id != "environment-1" || name != "docker" || len(args) != 6 || args[0] != "ps" || args[1] != "--all" || args[5] != "{{json .}}" {
		return "", errors.New("expected one environment-scoped docker ps --all query")
	}
	return r.output, r.err
}

func TestServiceStatusCheckObservesBatchAndConfirmsEachService(t *testing.T) {
	s, database := takeoverTestService(t)
	runtime := &statusCheckRuntime{output: `
{"Labels":"com.docker.compose.project=example,com.docker.compose.oneoff=False","State":"running","Status":"Up 1 minute"}
{"Labels":"com.docker.compose.project=other","State":"exited","Status":"Exited (0) 1 minute ago"}
{"Labels":"com.docker.compose.project=other,com.docker.compose.oneoff=True","State":"running"}
{"Labels":"com.docker.compose.project=example-extra","State":"exited"}
`}
	s.project, s.service, s.runtime, s.statusChecks = takeoverProjectReader{}, servicerepo.NewRepository(database), runtime, newServiceStatusChecks()
	ctx := context.Background()
	preview, err := s.PreviewServiceStatusCheck(ctx, "user-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(preview.Items) != 2 || preview.Items[0].ObservedStatus != status.ServiceStatusRunning || preview.Items[1].ObservedStatus != status.ServiceStatusStopped {
		t.Fatalf("expected exact project matching and one-off exclusion: %+v", preview.Items)
	}
	before, err := s.service.Service(ctx, "project-1", "service-1")
	if err != nil || before.Status != status.ServiceStatusStopped {
		t.Fatalf("preview changed saved status: %+v, %v", before, err)
	}
	if _, err := s.ConfirmServiceStatusCheck(ctx, "other-user", "project-1", preview.Token, "service-1"); !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("another actor could use the snapshot: %v", err)
	}
	for _, item := range preview.Items {
		state, err := s.ConfirmServiceStatusCheck(ctx, "user-1", "project-1", preview.Token, item.ServiceId)
		if err != nil || state != item.ObservedStatus {
			t.Fatalf("confirm %s: %q, %v", item.ServiceId, state, err)
		}
		stored, err := s.service.Service(ctx, "project-1", item.ServiceId)
		if err != nil || stored.Status != item.ObservedStatus || stored.VersionId != "version-1" || stored.DeploymentDirectory == "" {
			t.Fatalf("expected status-only update: %+v, %v", stored, err)
		}
	}
	if runtime.calls != 1 {
		t.Fatalf("Docker queried %d times, want one for the whole round", runtime.calls)
	}
}

func TestServiceStatusCheckFailedQueryLeavesRecordsUntouched(t *testing.T) {
	s, database := takeoverTestService(t)
	runtime := &statusCheckRuntime{output: "Docker daemon connection refused", err: errors.New("exit status 1")}
	s.project, s.service, s.runtime, s.statusChecks = takeoverProjectReader{}, servicerepo.NewRepository(database), runtime, newServiceStatusChecks()
	ctx := context.Background()
	if _, err := s.PreviewServiceStatusCheck(ctx, "user-1", "project-1"); !apperror.IsKind(err, apperror.KindUnavailable) || !strings.Contains(err.Error(), runtime.output) {
		t.Fatalf("expected unavailable error, got %v", err)
	}
	stored, err := s.service.Service(ctx, "project-1", "service-2")
	if err != nil || stored.Status != status.ServiceStatusRunning || len(s.statusChecks.snapshots) != 0 {
		t.Fatalf("failed query changed service or published a snapshot: %+v, %v", stored, err)
	}
}

func TestServiceStatusCheckRejectsChangedBaselineAndActiveDeployment(t *testing.T) {
	tests := []struct {
		name   string
		before []string
		after  []string
	}{
		{name: "changed environment", after: []string{`UPDATE environment SET target_revision = 2 WHERE id = 'environment-1'`}},
		{name: "changed service", after: []string{`UPDATE service SET version_id = 'version-2', updated_at = '2030-01-01 00:00:00' WHERE id = 'service-1'`}},
		{name: "new operation", after: []string{`UPDATE service SET current_deployment_id = 'new-deployment' WHERE id = 'service-1'`}},
		{name: "waiting operation", before: []string{
			`INSERT INTO deployment (id,project_id,service_id,application_name,operation_type,trigger_type,status,is_rollback) VALUES ('deployment-1','project-1','service-1','App','deploy','manual','waiting_to_run',0)`,
			`UPDATE service SET current_deployment_id = 'deployment-1' WHERE id = 'service-1'`,
		}},
		{name: "running operation", before: []string{
			`INSERT INTO deployment (id,project_id,service_id,application_name,operation_type,trigger_type,status,is_rollback) VALUES ('deployment-1','project-1','service-1','App','deploy','manual','running',0)`,
			`UPDATE service SET current_deployment_id = 'deployment-1' WHERE id = 'service-1'`,
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			s, database := takeoverTestService(t)
			for _, statement := range test.before {
				if _, err := database.Exec(statement); err != nil {
					t.Fatal(err)
				}
			}
			runtime := &statusCheckRuntime{output: `{"Labels":"com.docker.compose.project=example","State":"running"}`}
			s.project, s.service, s.runtime, s.statusChecks = takeoverProjectReader{}, servicerepo.NewRepository(database), runtime, newServiceStatusChecks()
			ctx := context.Background()
			preview, err := s.PreviewServiceStatusCheck(ctx, "user-1", "project-1")
			if err != nil {
				t.Fatal(err)
			}
			for _, statement := range test.after {
				if _, err := database.Exec(statement); err != nil {
					t.Fatal(err)
				}
			}
			if _, err := s.ConfirmServiceStatusCheck(ctx, "user-1", "project-1", preview.Token, "service-1"); !apperror.IsKind(err, apperror.KindConflict) {
				t.Fatalf("expected conflict, got %v", err)
			}
			stored, err := s.service.Service(ctx, "project-1", "service-1")
			if err != nil || stored.Status != status.ServiceStatusStopped || runtime.calls != 1 {
				t.Fatalf("stale observation changed status or queried Docker again: %+v, %v", stored, err)
			}
		})
	}
}

func TestInventoryServiceStatus(t *testing.T) {
	tests := []struct {
		name       string
		containers []serviceInventoryContainer
		want       string
	}{
		{name: "no containers", want: status.ServiceStatusStopped},
		{name: "stopped outside Orbit", containers: []serviceInventoryContainer{{State: "exited"}, {State: "exited"}}, want: status.ServiceStatusStopped},
		{name: "started outside Orbit", containers: []serviceInventoryContainer{{State: "running"}, {State: "running"}}, want: status.ServiceStatusRunning},
		{name: "partial stop", containers: []serviceInventoryContainer{{State: "running"}, {State: "exited"}}, want: status.ServiceStatusFaulted},
		{name: "restarting", containers: []serviceInventoryContainer{{State: "restarting"}}, want: status.ServiceStatusFaulted},
		{name: "unhealthy", containers: []serviceInventoryContainer{{State: "running", Status: "Up 1 minute (unhealthy)"}}, want: status.ServiceStatusFaulted},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := inventoryServiceStatus(test.containers); got != test.want {
				t.Fatalf("status = %q, want %q", got, test.want)
			}
		})
	}
}

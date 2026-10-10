package deploymentsvc

import (
	"bytes"
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	deploymentport "github.com/leoninew/pomelo-orbit/internal/application/deployment/port"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/queue/dispatch"
	tasksvc "github.com/leoninew/pomelo-orbit/internal/queue/task"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	deploymentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/deployment"
	servicerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/service"
	taskrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/task"
)

type takeoverProjectReader struct{ repository.ProjectReader }

func (takeoverProjectReader) Project(context.Context, string) (model.Project, error) {
	return model.Project{Id: "project-1"}, nil
}
func (takeoverProjectReader) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

func takeoverTestService(t *testing.T) (Service, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", filepath.Join(t.TempDir(), "operations.db"))
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = database.Close() })
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"INSERT INTO project (id,name,code) VALUES ('project-1','Project','project')",
		"INSERT INTO application (id,project_id,name,code,kind) VALUES ('app-1','project-1','App','app','standard')",
		"INSERT INTO version (id,application_id,label,status) VALUES ('version-1','app-1','v1','draft')",
		"INSERT INTO version (id,application_id,label,status) VALUES ('version-2','app-1','v2','draft')",
		"INSERT INTO environment (id,project_id,code,target_type,workspace_root,target_revision) VALUES ('environment-1','project-1','project','local','/workspace',1)",
		"INSERT INTO service (id,project_id,application_id,code,version_id,status,deployment_directory,directory_target_revision) VALUES ('service-1','project-1','app-1','example','version-1','stopped','/custom/example',1)",
		"INSERT INTO service (id,project_id,application_id,code,version_id,status,deployment_directory,directory_target_revision) VALUES ('service-2','project-1','app-1','other','version-1','running','/custom/other',1)",
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	store := &stores{project: takeoverProjectReader{}, application: applicationrepo.NewRepository(database), service: servicerepo.NewRepository(database), deployment: deploymentrepo.NewRepository(database)}
	target := environmentport.Target{Environment: model.Environment{Id: "environment-1", ProjectId: "project-1", TargetType: model.EnvironmentTargetTypeLocal, TargetRevision: 1}}
	return Service{commandStore: store, executionStore: store, dispatcher: dispatch.NewDeploymentDispatcher(tasksvc.New(taskrepo.NewRepository(database), 1)), transactionRunner: tx.NewTransactionRunner(database), targetResolver: staticTargetResolver{target: target}, executionTimeout: 3 * time.Second, cancelTimeout: 200 * time.Millisecond, pollInterval: 5 * time.Millisecond}, database
}

func admitTakeoverStop(t *testing.T, service Service, serviceId string) string {
	t.Helper()
	id, err := service.StopApplication(context.Background(), "user-1", "project-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: serviceId})
	if err != nil {
		t.Fatal(err)
	}
	return id
}

func awaitTakeoverResult(t *testing.T, done <-chan error) error {
	t.Helper()
	select {
	case err := <-done:
		return err
	case <-time.After(5 * time.Second):
		t.Fatal("operation did not finish within test deadline")
		return nil
	}
}

type enqueueFailureDispatcher struct {
	deploymentport.Dispatcher
	failure error
}

func (d enqueueFailureDispatcher) DispatchStop(ctx context.Context, input deploymentdto.StopDispatchInput) error {
	if err := d.Dispatcher.DispatchStop(ctx, input); err != nil {
		return err
	}
	return d.failure
}

func TestFailedAdmissionRollsBackCurrentOperationAndEnqueuedTask(t *testing.T) {
	service, database := takeoverTestService(t)
	first := admitTakeoverStop(t, service, "service-1")
	failure := errors.New("enqueue transaction failed")
	service.dispatcher = enqueueFailureDispatcher{Dispatcher: service.dispatcher, failure: failure}
	id, err := service.StopApplication(context.Background(), "user-1", "project-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1"})
	if !errors.Is(err, failure) || id != "" {
		t.Fatalf("id=%s error=%v", id, err)
	}
	current, err := service.executionStore.Service(context.Background(), "project-1", "service-1")
	if err != nil {
		t.Fatal(err)
	}
	if current.CurrentDeploymentId == nil || *current.CurrentDeploymentId != first {
		t.Fatalf("current=%+v", current)
	}
	for _, table := range []string{"deployment", "background_task"} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("%s count=%d after rollback", table, count)
		}
	}
}

func TestDeployRestartAndStopReplaceEachOtherRegardlessOfStoredStatus(t *testing.T) {
	service, database := takeoverTestService(t)
	service.gatewayCoordinator = &gatewayDeploymentCoordinatorFake{}
	ctx := context.Background()
	previous := ""
	for index, operation := range []string{"deploy", "restart", "stop", "deploy"} {
		storedStatus := []string{status.ServiceStatusRunning, status.ServiceStatusStopped, status.ServiceStatusFaulted}[index%3]
		if _, err := database.Exec("UPDATE service SET status = ? WHERE id = 'service-1'", storedStatus); err != nil {
			t.Fatal(err)
		}
		var id string
		var err error
		switch operation {
		case "deploy":
			result, deployErr := service.DeployService(ctx, "user-1", "project-1", "service-1", deploymentdto.DeployServiceInput{DeploymentDirectory: "~/custom/example", EnvironmentTargetRevision: 1})
			id, err = result.DeploymentId, deployErr
		case "restart":
			id, err = service.RestartApplication(ctx, "user-1", "project-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1"})
		case "stop":
			id, err = service.StopApplication(ctx, "user-1", "project-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1"})
		}
		if err != nil {
			t.Fatalf("admit %s with status %s: %v", operation, storedStatus, err)
		}
		current, err := service.executionStore.Service(ctx, "project-1", "service-1")
		if err != nil || current.CurrentDeploymentId == nil || *current.CurrentDeploymentId != id {
			t.Fatalf("current operation=%+v %v", current, err)
		}
		if previous != "" {
			if changed, err := service.executionStore.CompleteDeployment(ctx, "project-1", previous, status.WorkStatusRanToCompletion, ""); err != nil || changed {
				t.Fatalf("previous operation completed: %t %v", changed, err)
			}
		}
		if begun, err := service.executionStore.BeginDeployment(ctx, "project-1", id); err != nil || !begun {
			t.Fatalf("begin=%t %v", begun, err)
		}
		previous = id
	}
}

type failingTakeoverCancellation struct {
	deploymentport.ExecutionStore
	failure error
}

type blockedTakeoverCancellation struct {
	deploymentport.ExecutionStore
	release <-chan struct{}
	exited  chan<- struct{}
}

func (s blockedTakeoverCancellation) CancelSupersededDeployments(context.Context, string, string, string) ([]string, error) {
	defer close(s.exited)
	<-s.release
	return nil, nil
}

func TestCancellationTimeoutDoesNotBlockNewOperation(t *testing.T) {
	service, _ := takeoverTestService(t)
	service.cancelTimeout = 30 * time.Millisecond
	release, exited := make(chan struct{}), make(chan struct{})
	t.Cleanup(func() {
		close(release)
		select {
		case <-exited:
		case <-time.After(time.Second):
			t.Error("cancellation helper did not exit")
		}
	})
	service.executionStore = blockedTakeoverCancellation{ExecutionStore: service.executionStore, release: release, exited: exited}
	id := admitTakeoverStop(t, service, "service-1")
	done := make(chan error, 1)
	go func() {
		done <- service.executeDeployment(context.Background(), "project-1", "app-1", id, "stop", func(context.Context, model.Application, model.Deployment) error { return nil })
	}()
	if err := awaitTakeoverResult(t, done); err != nil {
		t.Fatal(err)
	}
	select {
	case <-exited:
		t.Fatal("new operation waited for cancellation helper")
	default:
	}
}

func (s failingTakeoverCancellation) CancelSupersededDeployments(context.Context, string, string, string) ([]string, error) {
	return nil, s.failure
}

type takeoverLog struct {
	sync.Mutex
	buffer bytes.Buffer
}

func (w *takeoverLog) Write(data []byte) (int, error) {
	w.Lock()
	defer w.Unlock()
	return w.buffer.Write(data)
}
func (w *takeoverLog) String() string { w.Lock(); defer w.Unlock(); return w.buffer.String() }

func TestNewOperationCompletesBeforeOldRuntimeExitsDespiteCancellationFailure(t *testing.T) {
	for _, lateFailure := range []bool{false, true} {
		t.Run(map[bool]string{false: "late success", true: "late failure"}[lateFailure], func(t *testing.T) {
			service, _ := takeoverTestService(t)
			logs := &takeoverLog{}
			service.logger = slog.New(slog.NewJSONHandler(logs, nil))
			store := service.executionStore
			first := admitTakeoverStop(t, service, "service-1")
			started, release, exited := make(chan struct{}), make(chan struct{}), make(chan struct{})
			var releaseOnce sync.Once
			t.Cleanup(func() {
				releaseOnce.Do(func() { close(release) })
				select {
				case <-exited:
				case <-time.After(time.Second):
					t.Error("late runtime did not exit")
				}
			})
			oldDone := make(chan error, 1)
			go func() {
				oldDone <- service.executeDeployment(context.Background(), "project-1", "app-1", first, "stop", func(ctx context.Context, _ model.Application, _ model.Deployment) error {
					close(started)
					<-release
					defer close(exited)
					if lateFailure {
						return errors.New("late Docker failure")
					}
					return nil
				})
			}()
			select {
			case <-started:
			case <-time.After(time.Second):
				t.Fatal("old operation did not start")
			}
			second := admitTakeoverStop(t, service, "service-1")
			service.executionStore = failingTakeoverCancellation{ExecutionStore: store, failure: errors.New("old cancellation write failed")}
			ran := false
			if err := service.executeDeployment(context.Background(), "project-1", "app-1", second, "stop", func(context.Context, model.Application, model.Deployment) error { ran = true; return nil }); err != nil {
				t.Fatal(err)
			}
			if !ran {
				t.Fatal("new operation did not execute")
			}
			if err := awaitTakeoverResult(t, oldDone); err != nil {
				t.Fatalf("old cancellation became an error: %v", err)
			}
			select {
			case <-exited:
				t.Fatal("new operation waited for old runtime exit")
			default:
			}
			releaseOnce.Do(func() { close(release) })
			<-exited
			current, err := store.Service(context.Background(), "project-1", "service-1")
			if err != nil {
				t.Fatal(err)
			}
			completed, err := store.Deployment(context.Background(), "project-1", second)
			if err != nil {
				t.Fatal(err)
			}
			if current.CurrentDeploymentId == nil || *current.CurrentDeploymentId != second || current.Status != status.ServiceStatusStopped || completed.Status != status.WorkStatusRanToCompletion {
				t.Fatalf("service=%+v deployment=%+v", current, completed)
			}
			if output := logs.String(); !strings.Contains(output, "old cancellation write failed") || !strings.Contains(output, `"level":"WARN"`) || strings.Contains(output, `"level":"ERROR"`) {
				t.Fatalf("cancellation diagnostics: %s", output)
			}
		})
	}
}

func TestDeploymentExecutionTimeoutReleasesWaitAndPreservesTerminalResult(t *testing.T) {
	service, _ := takeoverTestService(t)
	service.executionTimeout = 30 * time.Millisecond
	id := admitTakeoverStop(t, service, "service-1")
	release, exited := make(chan struct{}), make(chan struct{})
	var releaseOnce sync.Once
	t.Cleanup(func() {
		releaseOnce.Do(func() { close(release) })
		select {
		case <-exited:
		case <-time.After(time.Second):
			t.Error("late runtime did not exit")
		}
	})
	err := service.executeDeployment(context.Background(), "project-1", "app-1", id, "stop", func(ctx context.Context, _ model.Application, _ model.Deployment) error {
		setDeploymentStage(ctx, "compose_stop")
		<-release
		close(exited)
		return nil
	})
	if !errors.Is(err, context.DeadlineExceeded) {
		t.Fatalf("timeout result=%v", err)
	}
	stored, err := service.executionStore.Deployment(context.Background(), "project-1", id)
	if err != nil {
		t.Fatal(err)
	}
	if stored.Status != status.WorkStatusFaulted || stored.ErrorMessage == nil || !strings.Contains(*stored.ErrorMessage, "30ms") || !strings.Contains(*stored.ErrorMessage, "compose_stop") {
		t.Fatalf("timed out deployment=%+v", stored)
	}
	releaseOnce.Do(func() { close(release) })
	<-exited
	stored, err = service.executionStore.Deployment(context.Background(), "project-1", id)
	if err != nil || stored.Status != status.WorkStatusFaulted {
		t.Fatalf("late result changed timeout: %+v %v", stored, err)
	}
}

func TestLifecycleWritesRequireCurrentAndExecutableOperation(t *testing.T) {
	service, database := takeoverTestService(t)
	ctx := context.Background()
	first := admitTakeoverStop(t, service, "service-1")
	if begun, err := service.executionStore.BeginDeployment(ctx, "project-1", first); err != nil || !begun {
		t.Fatalf("begin=%t %v", begun, err)
	}
	second := admitTakeoverStop(t, service, "service-1")
	version := "version-2"
	if changed, err := service.executionStore.UpdateServiceDeploymentResult(ctx, "project-1", "service-1", first, status.ServiceStatusRunning, &version); err != nil || changed {
		t.Fatalf("old result changed Service: %t %v", changed, err)
	}
	if changed, err := service.executionStore.BindServiceRuntimeDirectory(ctx, "project-1", "service-1", first, "/old/runtime", 1); err != nil || changed {
		t.Fatalf("old directory bound: %t %v", changed, err)
	}
	if changed, err := service.executionStore.CompleteDeployment(ctx, "project-1", first, status.WorkStatusRanToCompletion, ""); err != nil || changed {
		t.Fatalf("old task completed: %t %v", changed, err)
	}
	if begun, err := service.executionStore.BeginDeployment(ctx, "project-1", second); err != nil || !begun {
		t.Fatalf("begin=%t %v", begun, err)
	}
	if changed, err := service.executionStore.BindServiceRuntimeDirectory(ctx, "project-1", "service-1", second, "/current/runtime", 1); err != nil || !changed {
		t.Fatalf("current binding=%t %v", changed, err)
	}
	if changed, err := service.executionStore.CompleteDeployment(ctx, "project-1", second, status.WorkStatusFaulted, "timeout"); err != nil || !changed {
		t.Fatalf("complete=%t %v", changed, err)
	}
	if changed, err := service.executionStore.UpdateServiceDeploymentResult(ctx, "project-1", "service-1", second, status.ServiceStatusRunning, &version); err != nil || changed {
		t.Fatalf("terminal task wrote Service: %t %v", changed, err)
	}
	if _, err := service.executionStore.CancelObsoleteDeployment(ctx, "project-1", first); err != nil {
		t.Fatal(err)
	}
	if changed, err := service.executionStore.ReconcileServiceAfterCancellation(ctx, "project-1", "service-1", first, status.ServiceStatusRunning); err != nil || changed {
		t.Fatalf("old observation changed Service: %t %v", changed, err)
	}
	repo := deploymentrepo.NewRepository(database)
	if err := repo.DeleteDeployment(ctx, "project-1", second); err != nil {
		t.Fatal(err)
	}
	if changed, err := service.executionStore.BindServiceRuntimeDirectory(ctx, "project-1", "service-1", first, "/resurrected/runtime", 1); err != nil || changed {
		t.Fatalf("history deletion restored old binding: %t %v", changed, err)
	}
	if changed, err := service.executionStore.CompleteDeployment(ctx, "project-1", first, status.WorkStatusRanToCompletion, ""); err != nil || changed {
		t.Fatalf("history deletion restored old completion: %t %v", changed, err)
	}
	current, err := service.executionStore.Service(ctx, "project-1", "service-1")
	if err != nil {
		t.Fatal(err)
	}
	if current.CurrentDeploymentId == nil || *current.CurrentDeploymentId != second || current.RuntimeDirectory != "/current/runtime" || current.VersionId != "version-1" {
		t.Fatalf("history deletion changed Service: %+v", current)
	}
	if active, err := repo.HasActiveDeployment(ctx, "project-1", "service-1"); err != nil || active {
		t.Fatalf("obsolete operation reported active: %t %v", active, err)
	}
}

func TestLateCancellationCannotCancelNewerServiceOperation(t *testing.T) {
	service, _ := takeoverTestService(t)
	ctx := context.Background()
	first := admitTakeoverStop(t, service, "service-1")
	second := admitTakeoverStop(t, service, "service-1")
	current := admitTakeoverStop(t, service, "service-1")
	other := admitTakeoverStop(t, service, "service-2")
	if ids, err := service.executionStore.CancelSupersededDeployments(ctx, "project-1", "service-1", second); err != nil || len(ids) != 0 {
		t.Fatalf("obsolete owner canceled tasks: %v %v", ids, err)
	}
	if changed, err := service.executionStore.CancelObsoleteDeployment(ctx, "project-1", current); err != nil || changed {
		t.Fatalf("current operation canceled: %t %v", changed, err)
	}
	ids, err := service.executionStore.CancelSupersededDeployments(ctx, "project-1", "service-1", current)
	if err != nil || len(ids) != 2 {
		t.Fatalf("superseded ids=%v %v", ids, err)
	}
	for _, id := range []string{first, second} {
		deployment, err := service.executionStore.Deployment(ctx, "project-1", id)
		if err != nil || deployment.Status != status.WorkStatusCanceled {
			t.Fatalf("old deployment=%+v %v", deployment, err)
		}
	}
	deployment, err := service.executionStore.Deployment(ctx, "project-1", other)
	if err != nil || deployment.Status != status.WorkStatusWaitingToRun {
		t.Fatalf("other Service changed: %+v %v", deployment, err)
	}
}

func TestOldTerminalOrMissingOperationDoesNotBlockNewExecution(t *testing.T) {
	for _, oldState := range []string{status.WorkStatusFaulted, status.WorkStatusRanToCompletion, "missing"} {
		t.Run(oldState, func(t *testing.T) {
			service, database := takeoverTestService(t)
			ctx := context.Background()
			first := admitTakeoverStop(t, service, "service-1")
			repo := deploymentrepo.NewRepository(database)
			if oldState == "missing" {
				if err := repo.DeleteDeployment(ctx, "project-1", first); err != nil {
					t.Fatal(err)
				}
			} else {
				if begun, err := repo.BeginDeployment(ctx, "project-1", first); err != nil || !begun {
					t.Fatalf("begin=%t %v", begun, err)
				}
				if changed, err := repo.CompleteDeployment(ctx, "project-1", first, oldState, "original result"); err != nil || !changed {
					t.Fatalf("complete=%t %v", changed, err)
				}
			}
			current := admitTakeoverStop(t, service, "service-1")
			ran := false
			if err := service.executeDeployment(ctx, "project-1", "app-1", current, "stop", func(context.Context, model.Application, model.Deployment) error { ran = true; return nil }); err != nil || !ran {
				t.Fatalf("execution=%t %v", ran, err)
			}
			if oldState != "missing" {
				old, err := repo.Deployment(ctx, "project-1", first)
				if err != nil || old.Status != oldState || old.ErrorMessage == nil || *old.ErrorMessage != "original result" {
					t.Fatalf("old result=%+v %v", old, err)
				}
			}
		})
	}
}

//go:build !unit

package deploymentsvc

import (
	"context"
	"database/sql"
	"errors"

	"testing"

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

func takeoverTestService(t *testing.T) (Service, *stores, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
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
		"INSERT INTO version (id,application_id,label,status) VALUES ('version-1','app-1','v1','unpublished')",
		"INSERT INTO version (id,application_id,label,status) VALUES ('version-2','app-1','v2','unpublished')",
		"INSERT INTO environment (id,project_id,code,target_type,workspace_root,target_revision) VALUES ('environment-1','project-1','project','local','/workspace',1)",
		"INSERT INTO service (id,project_id,application_id,code,version_id,status,deployment_directory,directory_target_revision) VALUES ('service-1','project-1','app-1','example','version-1','stopped','/custom/example',1)",
		"INSERT INTO service (id,project_id,application_id,code,version_id,status,deployment_directory,directory_target_revision) VALUES ('service-2','project-1','app-1','other','version-1','running','/custom/other',1)",
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	store := &stores{ProjectReader: takeoverProjectReader{}, ApplicationStore: applicationrepo.NewRepository(database), ServiceStore: servicerepo.NewRepository(database), DeploymentStore: deploymentrepo.NewRepository(database)}
	target := environmentport.Target{Environment: model.Environment{Id: "environment-1", ProjectId: "project-1", TargetType: model.EnvironmentTargetTypeLocal, TargetRevision: 1}}
	service := NewCommandService(store, store, store, store, dispatch.NewDeploymentDispatcher(tasksvc.New(taskrepo.NewRepository(database), 1)), nil, staticTargetResolver{target: target}, nil, nil, nil, nil, nil, tx.NewTransactionRunner(database), nil, lifecycleCommandsFake{})
	return service, store, database
}

func admitTakeoverStop(t *testing.T, service Service, serviceId string) string {
	t.Helper()
	id, err := service.StopApplication(context.Background(), "user-1", "project-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: serviceId})
	if err != nil {
		t.Fatal(err)
	}
	return id
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
	service, store, database := takeoverTestService(t)
	first := admitTakeoverStop(t, service, "service-1")
	failure := errors.New("enqueue transaction failed")
	service.dispatcher = enqueueFailureDispatcher{Dispatcher: service.dispatcher, failure: failure}
	id, err := service.StopApplication(context.Background(), "user-1", "project-1", "app-1", deploymentdto.ServiceTargetInput{ServiceId: "service-1"})
	if !errors.Is(err, failure) || id != "" {
		t.Fatalf("id=%s error=%v", id, err)
	}
	current, err := store.Service(context.Background(), "project-1", "service-1")
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
	service, store, database := takeoverTestService(t)
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
		current, err := store.Service(ctx, "project-1", "service-1")
		if err != nil || current.CurrentDeploymentId == nil || *current.CurrentDeploymentId != id {
			t.Fatalf("current operation=%+v %v", current, err)
		}
		if previous != "" {
			if changed, err := store.CompleteDeployment(ctx, "project-1", previous, status.WorkStatusRanToCompletion, ""); err != nil || changed {
				t.Fatalf("previous operation completed: %t %v", changed, err)
			}
		}
		if begun, err := store.BeginDeployment(ctx, "project-1", id); err != nil || !begun {
			t.Fatalf("begin=%t %v", begun, err)
		}
		previous = id
	}
}

func TestLifecycleWritesRequireCurrentAndExecutableOperation(t *testing.T) {
	service, store, database := takeoverTestService(t)
	ctx := context.Background()
	first := admitTakeoverStop(t, service, "service-1")
	if begun, err := store.BeginDeployment(ctx, "project-1", first); err != nil || !begun {
		t.Fatalf("begin=%t %v", begun, err)
	}
	second := admitTakeoverStop(t, service, "service-1")
	version := "version-2"
	if changed, err := store.UpdateServiceDeploymentResult(ctx, "project-1", "service-1", first, status.ServiceStatusRunning, &version); err != nil || changed {
		t.Fatalf("old result changed Service: %t %v", changed, err)
	}
	if changed, err := store.BindServiceRuntimeDirectory(ctx, "project-1", "service-1", first, "/old/runtime", 1); err != nil || changed {
		t.Fatalf("old directory bound: %t %v", changed, err)
	}
	if changed, err := store.CompleteDeployment(ctx, "project-1", first, status.WorkStatusRanToCompletion, ""); err != nil || changed {
		t.Fatalf("old task completed: %t %v", changed, err)
	}
	if begun, err := store.BeginDeployment(ctx, "project-1", second); err != nil || !begun {
		t.Fatalf("begin=%t %v", begun, err)
	}
	if changed, err := store.BindServiceRuntimeDirectory(ctx, "project-1", "service-1", second, "/current/runtime", 1); err != nil || !changed {
		t.Fatalf("current binding=%t %v", changed, err)
	}
	if changed, err := store.CompleteDeployment(ctx, "project-1", second, status.WorkStatusFaulted, "timeout"); err != nil || !changed {
		t.Fatalf("complete=%t %v", changed, err)
	}
	if changed, err := store.UpdateServiceDeploymentResult(ctx, "project-1", "service-1", second, status.ServiceStatusRunning, &version); err != nil || changed {
		t.Fatalf("terminal task wrote Service: %t %v", changed, err)
	}
	if _, err := store.CancelObsoleteDeployment(ctx, "project-1", first); err != nil {
		t.Fatal(err)
	}
	if changed, err := store.ReconcileServiceAfterCancellation(ctx, "project-1", "service-1", first, status.ServiceStatusRunning); err != nil || changed {
		t.Fatalf("old observation changed Service: %t %v", changed, err)
	}
	repo := deploymentrepo.NewRepository(database)
	if err := repo.DeleteDeployment(ctx, "project-1", second); err != nil {
		t.Fatal(err)
	}
	if changed, err := store.BindServiceRuntimeDirectory(ctx, "project-1", "service-1", first, "/resurrected/runtime", 1); err != nil || changed {
		t.Fatalf("history deletion restored old binding: %t %v", changed, err)
	}
	if changed, err := store.CompleteDeployment(ctx, "project-1", first, status.WorkStatusRanToCompletion, ""); err != nil || changed {
		t.Fatalf("history deletion restored old completion: %t %v", changed, err)
	}
	current, err := store.Service(ctx, "project-1", "service-1")
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
	service, store, _ := takeoverTestService(t)
	ctx := context.Background()
	first := admitTakeoverStop(t, service, "service-1")
	second := admitTakeoverStop(t, service, "service-1")
	current := admitTakeoverStop(t, service, "service-1")
	other := admitTakeoverStop(t, service, "service-2")
	if ids, err := store.CancelSupersededDeployments(ctx, "project-1", "service-1", second); err != nil || len(ids) != 0 {
		t.Fatalf("obsolete owner canceled tasks: %v %v", ids, err)
	}
	if changed, err := store.CancelObsoleteDeployment(ctx, "project-1", current); err != nil || changed {
		t.Fatalf("current operation canceled: %t %v", changed, err)
	}
	ids, err := store.CancelSupersededDeployments(ctx, "project-1", "service-1", current)
	if err != nil || len(ids) != 2 {
		t.Fatalf("superseded ids=%v %v", ids, err)
	}
	for _, id := range []string{first, second} {
		deployment, err := store.Deployment(ctx, "project-1", id)
		if err != nil || deployment.Status != status.WorkStatusCanceled {
			t.Fatalf("old deployment=%+v %v", deployment, err)
		}
	}
	deployment, err := store.Deployment(ctx, "project-1", other)
	if err != nil || deployment.Status != status.WorkStatusWaitingToRun {
		t.Fatalf("other Service changed: %+v %v", deployment, err)
	}
}

func TestOldTerminalOrMissingOperationPreservesNewLifecycle(t *testing.T) {
	for _, oldState := range []string{status.WorkStatusFaulted, status.WorkStatusRanToCompletion, "missing"} {
		t.Run(oldState, func(t *testing.T) {
			service, _, database := takeoverTestService(t)
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
			if begun, err := repo.BeginDeployment(ctx, "project-1", current); err != nil || !begun {
				t.Fatalf("begin current=%t %v", begun, err)
			}
			if _, err := repo.CancelSupersededDeployments(ctx, "project-1", "service-1", current); err != nil {
				t.Fatal(err)
			}
			if changed, err := repo.CompleteDeployment(ctx, "project-1", current, status.WorkStatusRanToCompletion, ""); err != nil || !changed {
				t.Fatalf("complete current=%t %v", changed, err)
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

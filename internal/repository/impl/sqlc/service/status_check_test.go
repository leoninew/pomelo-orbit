package servicerepo

import (
	"context"
	"database/sql"
	"testing"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	_ "modernc.org/sqlite"
)

func TestObservedStatusRequiresUnchangedBaselineAndNoActiveDeployment(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`
 INSERT INTO project (id,name,code) VALUES ('project-1','Project','project');
 INSERT INTO application (id,project_id,name,code,kind) VALUES ('app-1','project-1','App','app','standard');
 INSERT INTO version (id,application_id,label,status) VALUES ('version-1','app-1','v1','unpublished'),('version-2','app-1','v2','unpublished');
 INSERT INTO environment (id,project_id,code,target_type,workspace_root,target_revision) VALUES ('environment-1','project-1','project','local','/workspace',1);
 INSERT INTO service (id,project_id,application_id,code,version_id,status,deployment_directory,directory_target_revision) VALUES ('service-1','project-1','app-1','example','version-1','stopped','/custom/example',1);
 `); err != nil {
		t.Fatal(err)
	}
	repo := NewRepository(database)
	tests := []struct {
		name   string
		before []string
		after  []string
	}{
		{name: "unchanged baseline"},
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
			transaction, err := database.Begin()
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = transaction.Rollback() }()
			ctx := tx.WithTx(context.Background(), transaction)
			for _, statement := range test.before {
				if _, err := transaction.Exec(statement); err != nil {
					t.Fatal(err)
				}
			}
			baseline, err := repo.Service(ctx, "project-1", "service-1")
			if err != nil {
				t.Fatal(err)
			}
			for _, statement := range test.after {
				if _, err := transaction.Exec(statement); err != nil {
					t.Fatal(err)
				}
			}
			changed, err := repo.UpdateServiceObservedStatus(ctx, baseline, "environment-1", 1, status.ServiceStatusRunning)
			wantChanged := test.name == "unchanged baseline"
			if err != nil || changed != wantChanged {
				t.Fatalf("changed=%v error=%v", changed, err)
			}
			stored, err := repo.Service(ctx, "project-1", "service-1")
			if err != nil {
				t.Fatal(err)
			}
			wantStatus := status.ServiceStatusStopped
			if wantChanged {
				wantStatus = status.ServiceStatusRunning
			}
			if stored.Status != wantStatus || stored.DeploymentDirectory != baseline.DeploymentDirectory {
				t.Fatalf("unexpected persisted service: %+v", stored)
			}
		})
	}
}

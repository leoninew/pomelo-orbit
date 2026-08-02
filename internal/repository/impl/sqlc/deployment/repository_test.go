package deploymentrepo

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestRepositoryPersistsEffectivePlanHash(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	for _, statement := range []string{
		"INSERT INTO application (id, name, code, kind) VALUES ('app-1', 'Example', 'example', 'standard')",
		"INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'app-1', 'v1', 'draft')",
		"INSERT INTO service (id, application_id, instance_key, version_id, status) VALUES ('service-1', 'app-1', 'default', 'version-1', 'stopped')",
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("seed deployment relation: %v", err)
		}
	}

	hash := "f4d6ed0b5af0c349"
	repository := NewRepository(database)
	if err := repository.CreateDeployment(context.Background(), model.Deployment{
		Id: "deployment-1", ApplicationId: stringPtr("app-1"), ApplicationName: "Example",
		VersionId: stringPtr("version-1"), ServiceId: stringPtr("service-1"),
		EffectivePlanHash: &hash, OperationType: "deploy", TriggerType: "manual",
		CommandText: "docker compose up", Status: status.WorkStatusWaitingToRun,
	}); err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	if err := repository.CompleteDeployment(context.Background(), "deployment-1", status.WorkStatusRanToCompletion, ""); err != nil {
		t.Fatalf("complete deployment: %v", err)
	}

	stored, err := repository.Deployment(context.Background(), "deployment-1")
	if err != nil {
		t.Fatalf("load deployment: %v", err)
	}
	if stored.EffectivePlanHash == nil || *stored.EffectivePlanHash != hash {
		t.Fatalf("stored effective plan hash = %#v, want %q", stored.EffectivePlanHash, hash)
	}
	latest, err := repository.LatestSuccessfulDeploymentPlanHash(context.Background(), "service-1")
	if err != nil {
		t.Fatalf("load latest successful plan hash: %v", err)
	}
	if latest == nil || *latest != hash {
		t.Fatalf("latest successful plan hash = %#v, want %q", latest, hash)
	}
}

func stringPtr(value string) *string {
	return &value
}

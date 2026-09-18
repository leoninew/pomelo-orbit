package deploymentrepo

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
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
	if _, err := database.Exec(`INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project')`); err != nil {
		t.Fatalf("seed Project: %v", err)
	}
	for _, statement := range []string{
		"INSERT INTO application (id, name, code, kind, project_id) VALUES ('app-1', 'Example', 'example', 'standard', 'project-1')",
		"INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'app-1', 'v1', 'draft')",
		"INSERT INTO service (id, project_id, application_id, code, version_id, status) VALUES ('service-1', 'project-1', 'app-1', 'example-default', 'version-1', 'stopped')",
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("seed deployment relation: %v", err)
		}
	}

	hash := "f4d6ed0b5af0c349"
	environmentId := "environment-1"
	targetRevision := int64(3)
	credentialId := "credential-1"
	credentialRevision := int64(2)
	gatewayApplicationId := "gateway-app-1"
	repository := NewRepository(database)
	if err := repository.CreateDeployment(context.Background(), "project-1", model.Deployment{
		Id: "deployment-1", ProjectId: stringPtr("project-1"), ApplicationId: stringPtr("app-1"), ApplicationName: "Example",
		VersionId: stringPtr("version-1"), ServiceId: stringPtr("service-1"),
		EnvironmentId: &environmentId, EnvironmentTargetRevision: &targetRevision,
		SSHCredentialId: &credentialId, SSHCredentialRevision: &credentialRevision,
		GatewayApplicationId: &gatewayApplicationId,
		EffectivePlanHash:    &hash, OperationType: "deploy", TriggerType: "manual",
		CommandText: "docker compose up", Status: status.WorkStatusWaitingToRun,
	}); err != nil {
		t.Fatalf("create deployment: %v", err)
	}
	begun, err := repository.BeginDeployment(context.Background(), "project-1", "deployment-1")
	if err != nil || !begun {
		t.Fatalf("begin deployment: begun=%t err=%v", begun, err)
	}
	if _, err := repository.CompleteDeployment(context.Background(), "project-1", "deployment-1", status.WorkStatusRanToCompletion, ""); err != nil {
		t.Fatalf("complete deployment: %v", err)
	}

	stored, err := repository.Deployment(context.Background(), "project-1", "deployment-1")
	if err != nil {
		t.Fatalf("load deployment: %v", err)
	}
	if stored.EffectivePlanHash == nil || *stored.EffectivePlanHash != hash {
		t.Fatalf("stored effective plan hash = %#v, want %q", stored.EffectivePlanHash, hash)
	}
	if stored.EnvironmentId == nil || *stored.EnvironmentId != environmentId ||
		stored.EnvironmentTargetRevision == nil || *stored.EnvironmentTargetRevision != targetRevision ||
		stored.SSHCredentialId == nil || *stored.SSHCredentialId != credentialId ||
		stored.SSHCredentialRevision == nil || *stored.SSHCredentialRevision != credentialRevision ||
		stored.GatewayApplicationId == nil || *stored.GatewayApplicationId != gatewayApplicationId {
		t.Fatalf("stored deployment target snapshot = %#v", stored)
	}
	latest, err := repository.LatestSuccessfulDeploymentPlanHash(context.Background(), "project-1", "service-1")
	if err != nil {
		t.Fatalf("load latest successful plan hash: %v", err)
	}
	if latest == nil || *latest != hash {
		t.Fatalf("latest successful plan hash = %#v, want %q", latest, hash)
	}
}

func TestRepositoryDeletesDeployment(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	if _, err := database.Exec(`INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project')`); err != nil {
		t.Fatalf("seed Project: %v", err)
	}
	if _, err := database.Exec(`
		INSERT INTO deployment (id, project_id, application_name, operation_type, trigger_type, status, is_rollback)
		VALUES ('deployment-1', 'project-1', 'Example', 'deploy', 'manual', 'ran_to_completion', 0)
	`); err != nil {
		t.Fatalf("seed deployment: %v", err)
	}

	repo := NewRepository(database)
	if err := repo.DeleteDeployment(context.Background(), "project-1", "deployment-1"); err != nil {
		t.Fatalf("DeleteDeployment returned error: %v", err)
	}
	if _, err := repo.Deployment(context.Background(), "project-1", "deployment-1"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("Deployment error = %v, want ErrNotFound", err)
	}
}

func stringPtr(value string) *string {
	return &value
}

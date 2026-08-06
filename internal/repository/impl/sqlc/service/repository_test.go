package servicerepo

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
)

func TestDeleteServicePreservesDeploymentHistory(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `INSERT INTO application (id, name, code, kind) VALUES ('app-1', 'App', 'app', 'application')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'app-1', 'v1', 'unpublished')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO service (id, application_id, version_id, instance_key, status) VALUES ('service-1', 'app-1', 'version-1', 'default', 'stopped')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO deployment (id, application_name, operation_type, trigger_type, status, is_rollback, service_id) VALUES ('deployment-1', 'App', 'deploy', 'manual', 'succeeded', 0, 'service-1')`); err != nil {
		t.Fatal(err)
	}

	if err := NewRepository(database).DeleteService(ctx, "service-1"); err != nil {
		t.Fatal(err)
	}
	var serviceID string
	if err := database.QueryRowContext(ctx, `SELECT service_id FROM deployment WHERE id = 'deployment-1'`).Scan(&serviceID); err != nil {
		t.Fatal(err)
	}
	if serviceID != "service-1" {
		t.Fatalf("expected preserved deployment service reference, got %q", serviceID)
	}
}

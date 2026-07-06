package db

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"gitee.com/leoninew/pomelo-orbit/internal/config"
)

func openMemoryDB(t *testing.T) *sqlx.DB {
	t.Helper()
	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	return database
}

func TestMigrateUpPreparesApplicationData(t *testing.T) {
	database := openMemoryDB(t)
	defer func() { _ = database.Close() }()

	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	assertTaskQueueUsable(t, database)
	assertAdminUserPermissions(t, database)
	assertDeploymentCommandTextAvailable(t, database)
	assertGolangMigrateVersion(t, database, 7, false)
	assertLegacyMigrationHistoryTableAbsent(t, database)
}

func TestMigrateUpIsIdempotent(t *testing.T) {
	database := openMemoryDB(t)
	defer func() { _ = database.Close() }()

	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	assertGolangMigrateVersion(t, database, 7, false)
}

func TestReadMigrationVersion(t *testing.T) {
	database := openMemoryDB(t)
	defer func() { _ = database.Close() }()

	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	version, err := ReadMigrationVersion(database, config.DatabaseDriverSQLite)
	if err != nil {
		t.Fatal(err)
	}
	if version.Version != 7 || version.Dirty {
		t.Fatalf("unexpected migration version: %+v", version)
	}
}

func assertTaskQueueUsable(t *testing.T, database *sqlx.DB) {
	t.Helper()
	_, err := database.Exec(`
		INSERT INTO background_task (id, task_type, payload_json, status, attempts, max_attempts)
		VALUES ('task-1', 'ci.pipeline_run.execute', '{}', 'pending', 0, 3)
	`)
	if err != nil {
		t.Fatal(err)
	}
	var status string
	if err := database.Get(&status, "SELECT status FROM background_task WHERE id = ?", "task-1"); err != nil {
		t.Fatal(err)
	}
	if status != "pending" {
		t.Fatalf("unexpected task status: %s", status)
	}
}

func assertAdminUserPermissions(t *testing.T, database *sqlx.DB) {
	t.Helper()
	var permissionCount int
	err := database.Get(&permissionCount, `
		SELECT COUNT(*)
		FROM user
		JOIN user_role ON user_role.user_id = user.id
		JOIN role_permission ON role_permission.role_id = user_role.role_id
		JOIN permission ON permission.id = role_permission.permission_id
		WHERE user.username = 'admin'
		  AND permission.code IN ('login:read', 'setting:read', 'setting:write')
	`)
	if err != nil {
		t.Fatal(err)
	}
	if permissionCount != 3 {
		t.Fatalf("expected admin settings and login permissions, got %d", permissionCount)
	}
}

func assertDeploymentCommandTextAvailable(t *testing.T, database *sqlx.DB) {
	t.Helper()
	_, err := database.Exec(`
		INSERT INTO deployment (
			id, application_name, operation_type, trigger_type, status, is_rollback, command_text
		) VALUES (
			'deploy-1', 'demo', 'deploy', 'manual', 'succeeded', 0, 'docker compose up -d'
		)
	`)
	if err != nil {
		t.Fatal(err)
	}
	var commandText string
	if err := database.Get(&commandText, "SELECT command_text FROM deployment WHERE id = ?", "deploy-1"); err != nil {
		t.Fatal(err)
	}
	if commandText != "docker compose up -d" {
		t.Fatalf("unexpected deployment command text: %s", commandText)
	}
}

func assertGolangMigrateVersion(t *testing.T, database *sqlx.DB, wantVersion int, wantDirty bool) {
	t.Helper()
	var version int
	var dirty bool
	if err := database.QueryRowx("SELECT version, dirty FROM schema_migrations").Scan(&version, &dirty); err != nil {
		t.Fatal(err)
	}
	if version != wantVersion || dirty != wantDirty {
		t.Fatalf("unexpected schema migration state: version=%d dirty=%t", version, dirty)
	}
}

func assertLegacyMigrationHistoryTableAbsent(t *testing.T, database *sqlx.DB) {
	t.Helper()
	var count int
	if err := database.Get(&count, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", "__migration_history"); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("legacy migration history table should not be created, got %d", count)
	}
}

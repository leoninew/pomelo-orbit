package db

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
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
	assertDomainBaselineSchema(t, database)
	assertPipelineSeedPresent(t, database)
	assertGolangMigrateVersion(t, database, 30, false)
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
	assertGolangMigrateVersion(t, database, 30, false)
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
	if version.Version != 30 || version.Dirty {
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

func assertDomainBaselineSchema(t *testing.T, database *sqlx.DB) {
	t.Helper()
	required := []string{
		"background_task",
		"role", "permission", "role_permission",
		"user", "user_role",
		"login_history", "login_attempt",
		"project", "project_member",
		"credential",
		"pipeline_template", "pipeline_stage", "pipeline_template_stage", "pipeline_snapshot",
		"repository", "repository_webhook",
		"pipeline_run", "pipeline_stage_run", "artifact",
		"application", "version", "version_component", "version_expose",
		"environment",
		"gateway_config",
		"service",
		"deployment",
		"route",
	}
	for _, name := range required {
		var count int
		if err := database.Get(&count, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", name); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected table %s, got count %d", name, count)
		}
	}
	for _, legacy := range []string{"build_stage", "stage_run", "application_config_file", "environment_binding"} {
		var count int
		if err := database.Get(&count, "SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", legacy); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("legacy table %s should not exist", legacy)
		}
	}
}

func assertPipelineSeedPresent(t *testing.T, database *sqlx.DB) {
	t.Helper()
	var stageCount int
	if err := database.Get(&stageCount, "SELECT COUNT(*) FROM pipeline_stage"); err != nil {
		t.Fatal(err)
	}
	if stageCount < 1 {
		t.Fatalf("expected pipeline_stage seed rows, got %d", stageCount)
	}
	var repoCount int
	if err := database.Get(&repoCount, "SELECT COUNT(*) FROM repository"); err != nil {
		t.Fatal(err)
	}
	if repoCount < 1 {
		t.Fatalf("expected repository seed rows, got %d", repoCount)
	}
	var appCount int
	if err := database.Get(&appCount, "SELECT COUNT(*) FROM application"); err != nil {
		t.Fatal(err)
	}
	if appCount != 0 {
		t.Fatalf("expected no application demo seed, got %d", appCount)
	}
	var envCount int
	if err := database.Get(&envCount, "SELECT COUNT(*) FROM environment WHERE code = ?", "local"); err != nil {
		t.Fatal(err)
	}
	if envCount != 1 {
		t.Fatalf("expected default local environment seed, got %d", envCount)
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

package db

import (
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"backend/internal/config"
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

func TestMigratorUpAndStatus(t *testing.T) {
	database := openMemoryDB(t)
	defer func() { _ = database.Close() }()

	migrator := NewMigrator(database, config.DatabaseDriverSQLite)
	if err := migrator.Up(); err != nil {
		t.Fatal(err)
	}
	statuses, err := migrator.Status()
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) != 6 {
		t.Fatalf("expected 6 migrations, got %d", len(statuses))
	}
	for _, status := range statuses {
		if !status.Applied {
			t.Fatalf("expected migration %s applied", status.Filename)
		}
	}
	assertTableExists(t, database, "background_task")
	assertTableExists(t, database, "pipeline_run")
	assertTableExists(t, database, "route")
	assertTableExists(t, database, "role_permission")

	var permissionCount int
	if err := database.Get(&permissionCount, "SELECT COUNT(*) FROM permission WHERE code IN ('login:read', 'setting:read', 'setting:write')"); err != nil {
		t.Fatal(err)
	}
	if permissionCount != 3 {
		t.Fatalf("expected auth permissions, got %d", permissionCount)
	}
}

func assertTableExists(t *testing.T, database *sqlx.DB, name string) {
	t.Helper()
	var tableName string
	if err := database.Get(&tableName, "SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?", name); err != nil {
		t.Fatal(err)
	}
	if tableName != name {
		t.Fatalf("expected table %s, got %q", name, tableName)
	}
}

func TestMigratorDetectsChecksumMismatch(t *testing.T) {
	database := openMemoryDB(t)
	defer func() { _ = database.Close() }()

	migrator := NewMigrator(database, config.DatabaseDriverSQLite)
	if err := migrator.ensureHistoryTable(); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("INSERT INTO __migration_history (filename, checksum, execution_time_ms) VALUES (?, ?, ?)", "sqlite/v0.1.0__background_task.sql", "bad", 0); err != nil {
		t.Fatal(err)
	}
	if err := migrator.Up(); err == nil {
		t.Fatal("expected checksum mismatch")
	}
}

func TestMigratorListsMySQLMigrations(t *testing.T) {
	files, err := migrationFiles(config.DatabaseDriverMySQL)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 6 {
		t.Fatalf("expected 6 mysql migrations, got %d", len(files))
	}
	if files[0] != "mysql/v0.1.0__background_task.sql" {
		t.Fatalf("unexpected first mysql migration: %s", files[0])
	}
}

func TestMigratorRendersMigrationContentWithLiquidTemplate(t *testing.T) {
	got, err := renderMigrationContent("test.sql", "CREATE TABLE {{ table_name | default: 'demo' }} (id TEXT);", map[string]any{"enabled": true})
	if err != nil {
		t.Fatal(err)
	}
	if got != "CREATE TABLE demo (id TEXT);" {
		t.Fatalf("unexpected rendered migration: %q", got)
	}
}

func TestMigratorRejectsInvalidMigrationTemplate(t *testing.T) {
	_, err := renderMigrationContent("test.sql", "CREATE TABLE {{ missing }} (id TEXT);", map[string]any{"table_name": "demo"})
	if err == nil {
		t.Fatal("expected render error")
	}
}

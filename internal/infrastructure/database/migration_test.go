package db

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"

	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

func openMemoryDb(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatalf("open memory db: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })
	return database
}

func assertSQLiteIntegrity(t *testing.T, database *sql.DB) {
	t.Helper()
	violations, err := database.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatalf("run foreign key check: %v", err)
	}
	defer func() {
		if err := violations.Close(); err != nil {
			t.Errorf("close foreign key check: %v", err)
		}
	}()
	if violations.Next() {
		t.Fatal("database has foreign-key violations")
	}
	if err := violations.Err(); err != nil {
		t.Fatalf("read foreign key check: %v", err)
	}
	var integrity string
	if err := database.QueryRow("PRAGMA integrity_check").Scan(&integrity); err != nil {
		t.Fatalf("run integrity check: %v", err)
	}
	if integrity != "ok" {
		t.Fatalf("integrity check = %q, want ok", integrity)
	}
}

func TestMigrateUpSQLite(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}
	version, err := ReadMigrationVersion(database, config.DatabaseDriverSQLite)
	if err != nil {
		t.Fatalf("read version: %v", err)
	}
	if version.Version == 0 {
		t.Fatal("expected non-zero migration version")
	}
}

func TestMigrateUpSQLiteSeedIDsAreULIDs(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for _, table := range []string{
		"user",
		"project",
		"permission",
		"role",
		"repository",
		"pipeline_stage",
		"pipeline_template",
		"pipeline_template_stage",
	} {
		rows, err := database.Query("SELECT id FROM " + table)
		if err != nil {
			t.Fatalf("list %s IDs: %v", table, err)
		}
		for rows.Next() {
			var id string
			if err := rows.Scan(&id); err != nil {
				_ = rows.Close()
				t.Fatalf("scan %s ID: %v", table, err)
			}
			if _, err := ulid.ParseStrict(id); err != nil {
				_ = rows.Close()
				t.Fatalf("%s has non-ULID ID %q: %v", table, id, err)
			}
		}
		if err := rows.Err(); err != nil {
			_ = rows.Close()
			t.Fatalf("iterate %s IDs: %v", table, err)
		}
		if err := rows.Close(); err != nil {
			t.Fatalf("close %s IDs: %v", table, err)
		}
	}
}

func TestRagflowBaselineImportsIntoCurrentSQLite(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate database: %v", err)
	}
	root, err := filepath.Abs(filepath.Join("..", "..", ".."))
	if err != nil {
		t.Fatalf("resolve repository root: %v", err)
	}
	baseline, err := os.ReadFile(filepath.Join(root, "scripts", "ragflow-split", "ragflow-bundled-sqlite.sql"))
	if err != nil {
		t.Fatalf("read RAGFlow baseline: %v", err)
	}
	if _, err := database.Exec(string(baseline)); err != nil {
		t.Fatalf("import RAGFlow baseline: %v", err)
	}
	assertSQLiteIntegrity(t, database)
}

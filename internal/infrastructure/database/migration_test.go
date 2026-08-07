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

func TestMigrateUpSQLiteBuiltInSeedIDsAreULIDs(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for _, table := range []string{
		"user",
		"project",
		"permission",
		"role",
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

func TestMigrateUpSQLiteCreatesPipelineSchema(t *testing.T) {
	database, err := openSQLite(config.SQLiteConfig{Path: filepath.Join(t.TempDir(), "pomelo-orbit.db")})
	if err != nil {
		t.Fatalf("open sqlite database: %v", err)
	}
	t.Cleanup(func() { _ = database.Close() })

	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for _, table := range []string{
		"pipeline",
		"pipeline_stage",
		"pipeline_snapshot",
		"pipeline_run",
		"pipeline_run_version_binding",
		"artifact",
	} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count); err != nil {
			t.Fatalf("check %s table: %v", table, err)
		}
		if count != 1 {
			t.Fatalf("expected %s table", table)
		}
	}
	for _, table := range []string{"pipeline_template", "pipeline_template_stage"} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type = 'table' AND name = ?", table).Scan(&count); err != nil {
			t.Fatalf("check retired %s table: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("retired table %s still exists", table)
		}
	}

	var templates, stages int
	if err := database.QueryRow("SELECT COUNT(*) FROM pipeline WHERE kind = 'template'").Scan(&templates); err != nil {
		t.Fatalf("count template pipelines: %v", err)
	}
	if err := database.QueryRow("SELECT COUNT(*) FROM pipeline_stage").Scan(&stages); err != nil {
		t.Fatalf("count pipeline stages: %v", err)
	}
	if templates != 0 || stages != 0 {
		t.Fatalf("unexpected pipeline demo seed data: templates=%d stages=%d", templates, stages)
	}

	for table, want := range map[string]int{
		"permission":     7,
		"role":           1,
		"user_role":      1,
		"project_member": 1,
		"repository":     0,
	} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
			t.Fatalf("count %s rows: %v", table, err)
		}
		if count != want {
			t.Fatalf("%s rows=%d, want %d", table, count, want)
		}
	}

	for _, column := range []string{"pipeline_id", "pipeline_name", "pipeline_version"} {
		var value string
		if err := database.QueryRow("SELECT " + column + " FROM pipeline_run LIMIT 1").Scan(&value); err != sql.ErrNoRows {
			t.Fatalf("pipeline_run %s column is missing or unexpectedly populated: %v", column, err)
		}
	}
	for _, column := range []string{"artifact_name", "artifact_image_ref", "artifact_local_image_sha256", "artifact_source_commit_sha", "entrypoint_json"} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM pragma_table_info('version_component') WHERE name = ?", column).Scan(&count); err != nil {
			t.Fatalf("check version_component %s column: %v", column, err)
		}
		if count != 1 {
			t.Fatalf("version_component %s column is missing", column)
		}
	}

	rows, err := database.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatalf("run foreign key check: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("close foreign key check: %v", err)
		}
	}()
	if rows.Next() {
		t.Fatal("foreign key check found violations")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate foreign key check: %v", err)
	}
}

func TestOptionalSQLiteSeedExports(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for _, name := range []string{"000041_pipeline-demo.sqlite.sql"} {
		path := filepath.Join("..", "..", "..", "data", "exports", name)
		contents, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("read %s: %v", name, err)
		}
		if _, err := database.Exec(string(contents)); err != nil {
			t.Fatalf("apply %s: %v", name, err)
		}
		if _, err := database.Exec(string(contents)); err != nil {
			t.Fatalf("reapply %s: %v", name, err)
		}
	}

	for table, want := range map[string]int{
		"permission":      7,
		"role":            1,
		"role_permission": 7,
		"user_role":       1,
		"project_member":  1,
		"repository":      2,
		"pipeline":        2,
		"pipeline_stage":  6,
	} {
		var got int
		if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
			t.Fatalf("count %s rows: %v", table, err)
		}
		if got != want {
			t.Fatalf("%s rows=%d, want %d", table, got, want)
		}
	}

	rows, err := database.Query("PRAGMA foreign_key_check")
	if err != nil {
		t.Fatalf("run foreign key check: %v", err)
	}
	defer func() {
		if err := rows.Close(); err != nil {
			t.Errorf("close foreign key check: %v", err)
		}
	}()
	if rows.Next() {
		t.Fatal("foreign key check found violations")
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate foreign key check: %v", err)
	}
}

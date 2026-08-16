package db

import (
	"database/sql"
	"testing"

	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"

	"github.com/leoninew/pomelo-orbit/internal/config"
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
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("repeat migrate up: %v", err)
	}
}

func TestMigrateUpSQLiteSeedsExportedData(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for table, want := range map[string]int{
		"application":                1,
		"gateway_config":             1,
		"user":                       1,
		"project":                    1,
		"permission":                 9,
		"role":                       1,
		"pipeline":                   2,
		"pipeline_stage":             5,
		"pipeline_stage_reference":   6,
		"route":                      1,
		"service":                    1,
		"service_component":          1,
		"version":                    1,
		"version_component":          1,
		"version_component_endpoint": 3,
		"version_component_mount":    3,
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

func TestSQLiteSystemSeedIDsAreULIDs(t *testing.T) {
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

func TestMigrateUpSQLiteAllowsMultipleActivePipelineRunsForRepository(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for _, id := range []string{"run-1", "run-2"} {
		if _, err := database.Exec(`
            INSERT INTO pipeline_run (
                id, repository_id, repository_name, snapshot_id, pipeline_id, pipeline_name,
                pipeline_version, trigger, repository_ref, variables_snapshot, status
            ) VALUES (?, 'repository-1', 'Repository', 'snapshot-1', 'pipeline-1', 'Pipeline', 1, 'manual', 'main', '[]', 'waiting_to_run')
        `, id); err != nil {
			t.Fatalf("insert active pipeline run %s: %v", id, err)
		}
	}
}

func TestMigrateUpSQLiteAllowsMultipleActiveDeploymentsForService(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for _, id := range []string{"deployment-1", "deployment-2"} {
		if _, err := database.Exec(`
            INSERT INTO deployment (
                id, application_name, operation_type, trigger_type, status, is_rollback, service_id
            ) VALUES (?, 'Application', 'deploy', 'manual', 'waiting_to_run', 0, 'service-1')
        `, id); err != nil {
			t.Fatalf("insert active deployment %s: %v", id, err)
		}
	}
}

func TestMigrateUpSQLitePreservesFrozenStageReferenceAfterTemplateDeletion(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	if _, err := database.Exec(`
        INSERT INTO pipeline_stage (id, project_id, kind, name, image, script, description, version)
        VALUES ('stage-template', 'project-1', 'template', 'build', 'golang:1.24', 'go build ./...', '', 1)
    `); err != nil {
		t.Fatalf("insert template stage: %v", err)
	}
	if _, err := database.Exec(`
        INSERT INTO pipeline (id, project_id, kind, name, description, variable_declarations, version)
        VALUES ('template-pipeline', 'project-1', 'template', 'pipeline', '', '[]', 1)
    `); err != nil {
		t.Fatalf("insert template pipeline: %v", err)
	}
	if _, err := database.Exec(`
        INSERT INTO pipeline_stage_reference (
            id, pipeline_id, source_template_stage_id, source_template_stage_name,
            source_template_stage_version, source_template_stage_description,
            name, image, script, description, depends_on, sort_order
        ) VALUES (
			'reference-1', 'template-pipeline', 'stage-template', 'build', 1, '',
            'build', 'golang:1.24', 'go build ./...', '', '[]', 0
        )
    `); err != nil {
		t.Fatalf("insert stage reference: %v", err)
	}
	if _, err := database.Exec(`
        DELETE FROM pipeline_stage WHERE id = 'stage-template'
    `); err != nil {
		t.Fatalf("delete template stage: %v", err)
	}
	var references int
	if err := database.QueryRow("SELECT COUNT(*) FROM pipeline_stage_reference WHERE id = 'reference-1'").Scan(&references); err != nil {
		t.Fatalf("count stage references: %v", err)
	}
	if references != 1 {
		t.Fatal("template stage deletion must not remove frozen references")
	}
}

func TestSQLiteSeedContainsExportedPipelineLibrary(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for table, want := range map[string]int{
		"pipeline":                 2,
		"pipeline_stage":           5,
		"pipeline_stage_reference": 6,
	} {
		var got int
		if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&got); err != nil {
			t.Fatalf("count %s rows: %v", table, err)
		}
		if got != want {
			t.Fatalf("%s rows=%d, want %d", table, got, want)
		}
	}
	for _, check := range []struct {
		query string
		want  int
	}{
		{"SELECT version FROM pipeline_stage WHERE id = '01KRCWNJVA1DM02TJXZ4STJD01'", 2},
		{"SELECT version FROM pipeline WHERE id = '01KZG83K2MXG08EJ6G48SG38B3'", 10},
		{"SELECT source_template_stage_version FROM pipeline_stage_reference WHERE id = '01KZGBG9NT6NCK8AT6H6ENV874'", 2},
	} {
		var got int
		if err := database.QueryRow(check.query).Scan(&got); err != nil {
			t.Fatalf("read refreshed template seed: %v", err)
		}
		if got != check.want {
			t.Fatalf("seeded template value=%d, want %d", got, check.want)
		}
	}
}

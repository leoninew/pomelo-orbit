package db

import (
	"database/sql"
	"io/fs"
	"path"
	"strings"
	"testing"

	"github.com/oklog/ulid/v2"
	_ "modernc.org/sqlite"

	"github.com/leoninew/pomelo-orbit/internal/config"
	migrationfiles "github.com/leoninew/pomelo-orbit/sql"
)

const (
	seededGatewayApplicationID = "01M10RRA8F863EJ2N9TC3Z2EC1"
	seededGatewayBaseVersionID = "01M10RRA8F863EJ2N9TDN9JSBW"
	seededGatewayBaseComponent = "01M10RRA8F863EJ2N9TN8CWTEY"
	seededGatewayServiceID     = "01M10RRA8F863EJ2N9TTG49G6S"
	seededGatewayRouteID       = "01M10RRA8F863EJ2N9TYPF0CV7"
)

func TestDatabaseMigrationFilesAlign(t *testing.T) {
	t.Helper()

	migrationsByDriver := map[string]map[string]map[string]struct{}{
		config.DatabaseDriverSQLite:   migrationDirections(t, config.DatabaseDriverSQLite),
		config.DatabaseDriverMySQL:    migrationDirections(t, config.DatabaseDriverMySQL),
		config.DatabaseDriverPostgres: migrationDirections(t, config.DatabaseDriverPostgres),
	}
	for driver, migrations := range migrationsByDriver {
		for migration, directions := range migrations {
			for otherDriver, otherMigrations := range migrationsByDriver {
				if _, ok := otherMigrations[migration]; !ok {
					t.Errorf("%s is missing %s migration %s", otherDriver, driver, migration)
				}
			}
			assertMigrationDirections(t, driver, migration, directions)
		}
	}
}

func migrationDirections(t *testing.T, driver string) map[string]map[string]struct{} {
	t.Helper()

	entries, err := fs.ReadDir(migrationfiles.Files, path.Join(migrationsRoot, driver))
	if err != nil {
		t.Fatalf("read %s migrations: %v", driver, err)
	}

	migrations := make(map[string]map[string]struct{}, len(entries)/2)
	for _, entry := range entries {
		if entry.IsDir() {
			t.Fatalf("unexpected migration directory %s", entry.Name())
		}
		parts := strings.Split(entry.Name(), ".")
		if len(parts) != 3 || parts[2] != "sql" || !strings.Contains(parts[0], "_") {
			t.Fatalf("invalid migration file %s", entry.Name())
		}
		direction := parts[1]
		if direction != "up" && direction != "down" {
			t.Fatalf("invalid migration direction in %s", entry.Name())
		}
		if migrations[parts[0]] == nil {
			migrations[parts[0]] = make(map[string]struct{}, 2)
		}
		if _, exists := migrations[parts[0]][direction]; exists {
			t.Fatalf("duplicate %s migration %s", direction, parts[0])
		}
		migrations[parts[0]][direction] = struct{}{}
	}
	return migrations
}

func assertMigrationDirections(t *testing.T, driver, migration string, directions map[string]struct{}) {
	t.Helper()
	for _, direction := range []string{"up", "down"} {
		if _, ok := directions[direction]; !ok {
			t.Errorf("%s migration %s is missing %s", driver, migration, direction)
		}
	}
}

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
		"application":                  0,
		"gateway_config":               0,
		"user":                         1,
		"project":                      1,
		"permission":                   9,
		"role":                         1,
		"pipeline":                     2,
		"pipeline_stage":               5,
		"pipeline_stage_reference":     6,
		"route":                        0,
		"service":                      0,
		"service_component":            0,
		"version":                      0,
		"version_component":            0,
		"version_component_endpoint":   0,
		"version_component_mount":      0,
		"gateway_acme_profile_version": 0,
		"environment":                  0,
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

func TestMigrateUpSQLiteDoesNotSeedDeploymentResources(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate up: %v", err)
	}

	for table, id := range map[string]string{
		"application": seededGatewayApplicationID,
		"service":     seededGatewayServiceID,
		"route":       seededGatewayRouteID,
	} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE id = ?", id).Scan(&count); err != nil {
			t.Fatalf("count seeded %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("empty database seeded %s %s", table, id)
		}
	}
	var environments int
	if err := database.QueryRow("SELECT COUNT(*) FROM environment").Scan(&environments); err != nil {
		t.Fatalf("count environments: %v", err)
	}
	if environments != 0 {
		t.Fatalf("environment rows=%d, want 0", environments)
	}
}

func TestMigrateUpSQLiteDropsGatewayConfigComponentName(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateTo(database, config.DatabaseDriverSQLite, 39); err != nil {
		t.Fatalf("migrate to environment schema: %v", err)
	}
	var count int
	if err := database.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('gateway_config') WHERE name = 'traefik_component_name'`).Scan(&count); err != nil {
		t.Fatalf("inspect gateway_config before drop: %v", err)
	}
	if count != 1 {
		t.Fatalf("traefik_component_name columns=%d, want 1 before 000040", count)
	}
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatalf("migrate remaining: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('gateway_config') WHERE name = 'traefik_component_name'`).Scan(&count); err != nil {
		t.Fatalf("inspect gateway_config after drop: %v", err)
	}
	if count != 0 {
		t.Fatal("gateway_config still has traefik_component_name")
	}
}

func TestMigrateToSQLiteReplacesHistoricalGatewaySeed(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateTo(database, config.DatabaseDriverSQLite, 36); err != nil {
		t.Fatalf("migrate to runtime schema: %v", err)
	}

	for _, statement := range []string{
		`INSERT INTO application (id, name, code, kind, project_id) VALUES ('01M01MP0950ECGK2DS1FWYNC0B', 'Legacy Traefik', 'traefik', 'gateway', '01KRRKK0K3T519ZQZES3M4QA9Z')`,
		`INSERT INTO version (id, application_id, label, status, component_summary) VALUES ('01M01MP0950ECGK2DS1J4N4P50', '01M01MP0950ECGK2DS1FWYNC0B', 'traefik:3.6', 'unpublished', 'traefik')`,
		`INSERT INTO gateway_config (application_id, rest_api_url, base_domain, default_entrypoint, tls_mode) VALUES ('01M01MP0950ECGK2DS1FWYNC0B', 'http://localhost:8080', 'lvh.me', 'web', 'none')`,
		`INSERT INTO version_component (id, version_id, name, image, pull_policy) VALUES ('01M01MP096R73Z3MG28P91CSPN', '01M01MP0950ECGK2DS1J4N4P50', 'traefik', 'traefik:3.6', 'missing')`,
		`INSERT INTO service (id, application_id, instance_key, code, version_id, status) VALUES ('01M01RHDXW3ZXC7YKNT54RWM1M', '01M01MP0950ECGK2DS1FWYNC0B', 'default', 'traefik-default', '01M01MP0950ECGK2DS1J4N4P50', 'stopped')`,
		`INSERT INTO service_component (id, service_id, source_version_component_id, component_name, status) VALUES ('01M01RHDXW3ZXC7YKNT8GDNQNB', '01M01RHDXW3ZXC7YKNT54RWM1M', '01M01MP096R73Z3MG28P91CSPN', 'traefik', 'active')`,
		`INSERT INTO route (id, name, protocol, domain, path_prefix, target_url, enabled, https_enabled, cert_type, acme_challenge, project_id) VALUES ('01M01ZNW6CPQCB7P5PN669HWJ9', 'traefik', 'http', 'traefik-dashboard.lvh.me', '/', 'http://traefik-traefik:8080', 1, 0, 'manual', 'http', '01KRRKK0K3T519ZQZES3M4QA9Z')`,
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatalf("seed historical Gateway fixture: %v", err)
		}
	}
	if err := MigrateTo(database, config.DatabaseDriverSQLite, 37); err != nil {
		t.Fatalf("migrate Gateway seed: %v", err)
	}

	for table, id := range map[string]string{
		"application": "01M01MP0950ECGK2DS1FWYNC0B",
		"route":       "01M01ZNW6CPQCB7P5PN669HWJ9",
	} {
		var count int
		if err := database.QueryRow("SELECT COUNT(*) FROM "+table+" WHERE id = ?", id).Scan(&count); err != nil {
			t.Fatalf("count historical %s: %v", table, err)
		}
		if count != 0 {
			t.Fatalf("historical %s seed was not removed", table)
		}
	}
	var versions, bindings int
	if err := database.QueryRow(`SELECT COUNT(*) FROM version WHERE application_id = ?`, seededGatewayApplicationID).Scan(&versions); err != nil {
		t.Fatalf("count replacement Gateway Versions: %v", err)
	}
	if err := database.QueryRow(`SELECT COUNT(*) FROM gateway_acme_profile_version WHERE application_id = ?`, seededGatewayApplicationID).Scan(&bindings); err != nil {
		t.Fatalf("count replacement Gateway bindings: %v", err)
	}
	if versions != 0 || bindings != 0 {
		t.Fatalf("replacement Gateway topology: Versions=%d bindings=%d", versions, bindings)
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
		"application",
		"version",
		"version_component",
		"service",
		"service_component",
		"route",
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
		{"SELECT version FROM pipeline_stage WHERE id = '01KRCWNJVA1DM02TJXZ4STJD06'", 3},
		{"SELECT version FROM pipeline WHERE id = '01KZG83K2MXG08EJ6G48SG38B3'", 11},
		{"SELECT source_template_stage_version FROM pipeline_stage_reference WHERE id = '01KZGBG9NT6NCK8AT6H6ENV874'", 2},
		{"SELECT source_template_stage_version FROM pipeline_stage_reference WHERE id = '01KZGBGDK1G249681EVBDA9035'", 3},
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

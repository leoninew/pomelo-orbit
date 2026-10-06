package db

import (
	"database/sql"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestComponentIdentityMigrationPreservesDefaults(t *testing.T) {
	database := openMemoryDb(t)
	database.SetMaxOpenConns(1)
	if err := MigrateTo(database, config.DatabaseDriverSQLite, 51); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO project (id, name, code) VALUES ('p', 'Project', 'project')`,
		`INSERT INTO application (id, project_id, name, code, kind) VALUES ('a', 'p', 'Orbit', 'orbit', 'standard')`,
		`INSERT INTO version (id, application_id, label, status) VALUES ('v', 'a', '1', 'unpublished')`,
		`INSERT INTO version_component (id, version_id, name, image, pull_policy) VALUES ('vc', 'v', 'orbit', 'orbit:1', 'missing')`,
		`INSERT INTO version_component_mount (component_id, source_type, source, target, position) VALUES ('vc', 'directory', '/srv/data', '/data', 0)`,
		`INSERT INTO service (id, project_id, application_id, version_id, code, status) VALUES ('s', 'p', 'a', 'v', 'orbit-default', 'stopped')`,
		`INSERT INTO service_component (id, service_id, source_version_component_id, component_name) VALUES ('sc', 's', 'vc', 'orbit')`,
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"version_component", "service_component"} {
		var user, groups sql.NullString
		if err := database.QueryRow("SELECT container_user, group_add_json FROM "+table).Scan(&user, &groups); err != nil {
			t.Fatal(err)
		}
		if user.Valid || groups.Valid {
			t.Fatalf("migration injected identity into %s", table)
		}
	}
	var shared int
	if err := database.QueryRow("SELECT shared FROM version_component_mount").Scan(&shared); err != nil {
		t.Fatal(err)
	}
	if shared != 0 {
		t.Fatal("migration released an existing exclusive mount")
	}
	if err := MigrateTo(database, config.DatabaseDriverSQLite, 51); err != nil {
		t.Fatal(err)
	}
}

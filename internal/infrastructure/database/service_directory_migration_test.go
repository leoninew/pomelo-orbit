package db

import (
	"database/sql"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestServiceDirectoryMigrationBackfillsKnownPathsAndGatewayTypes(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateTo(database, config.DatabaseDriverSQLite, 49); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO project (id,name,code) VALUES ('directory-project','Project','directory-project')`,
		`INSERT INTO application (id,project_id,name,code,kind) VALUES ('actual-gateway','directory-project','Gateway','gateway','standard'), ('ordinary-traefik','directory-project','Traefik','traefik','standard')`,
		`INSERT INTO gateway_config (application_id,rest_api_url,rest_api_host_url,internal_domain) VALUES ('actual-gateway','http://traefik:8080','http://127.0.0.1:8080','example.test')`,
		`INSERT INTO version (id,application_id,label,status) VALUES ('directory-version','ordinary-traefik','v1','unpublished')`,
		`INSERT INTO environment (id,project_id,code,target_type,workspace_root,target_revision) VALUES ('directory-env','directory-project','directory-project','local','/srv/orbit',3)`,
		`INSERT INTO service (id,project_id,application_id,code,version_id,status) VALUES ('known','directory-project','ordinary-traefik','known','directory-version','stopped'), ('fresh','directory-project','ordinary-traefik','fresh','directory-version','stopped')`,
		`INSERT INTO deployment (id,project_id,service_id,application_name,operation_type,trigger_type,status,is_rollback,environment_id,environment_target_type,environment_target_revision) VALUES ('history','directory-project','known','Traefik','deploy','manual','ran_to_completion',0,'directory-env','local',3), ('queued','directory-project','known','Traefik','deploy','manual','waiting_to_run',0,'directory-env','local',3), ('stale','directory-project','known','Traefik','deploy','manual','waiting_to_run',0,'directory-env','local',2)`,
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	for id, want := range map[string]string{"actual-gateway": "gateway", "ordinary-traefik": "standard"} {
		var actual string
		if err := database.QueryRow("SELECT kind FROM application WHERE id = ?", id).Scan(&actual); err != nil || actual != want {
			t.Fatalf("%s kind=%s err=%v", id, actual, err)
		}
	}
	for id, runtime := range map[string]string{"known": "/srv/orbit/deployment/known", "fresh": ""} {
		var configured, operation string
		var revision int64
		if err := database.QueryRow("SELECT deployment_directory, runtime_directory, directory_target_revision FROM service WHERE id = ?", id).Scan(&configured, &operation, &revision); err != nil {
			t.Fatal(err)
		}
		if configured != "/srv/orbit/deployment/"+id || operation != runtime || revision != 3 {
			t.Fatalf("%s directory=%s runtime=%s revision=%d", id, configured, operation, revision)
		}
	}
	for id, known := range map[string]bool{"history": false, "queued": true, "stale": false} {
		var directory sql.NullString
		if err := database.QueryRow("SELECT working_directory FROM deployment WHERE id = ?", id).Scan(&directory); err != nil {
			t.Fatal(err)
		}
		if directory.Valid != known || known && directory.String != "/srv/orbit/deployment/known" {
			t.Fatalf("%s path=%v", id, directory)
		}
	}
}

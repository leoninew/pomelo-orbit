package servicerepo

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
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
	if _, err := database.ExecContext(ctx, `INSERT INTO service (id, project_id, application_id, version_id, instance_key, code, status) VALUES ('service-1', 'project-1', 'app-1', 'version-1', 'default', 'app-default', 'stopped')`); err != nil {
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

func TestListServicesByProjectSearchesApplicationNameAndServiceCode(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, statement := range []string{
		`INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project')`,
		`INSERT INTO application (id, name, code, kind, project_id) VALUES ('app-1', 'Application', 'application-code', 'standard', 'project-1')`,
		`INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'app-1', 'version-label', 'unpublished')`,
		`INSERT INTO service (id, project_id, application_id, version_id, instance_key, code, status) VALUES ('service-1', 'project-1', 'app-1', 'version-1', 'instance-one', 'service-one', 'stopped')`,
		`INSERT INTO service (id, project_id, application_id, version_id, instance_key, code, status) VALUES ('service-2', 'project-1', 'app-1', 'version-1', 'instance-two', 'service-two', 'stopped')`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

	repository := NewRepository(database)
	all, err := repository.ListServicesByProject(ctx, "project-1", "", "", "", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 2 || len(all.Items) != 2 {
		t.Fatalf("unfiltered services = %#v", all)
	}

	matched, err := repository.ListServicesByProject(ctx, "project-1", "", "", "Application", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if matched.Total != 2 || len(matched.Items) != 2 {
		t.Fatalf("application name search = %#v", matched)
	}

	matched, err = repository.ListServicesByProject(ctx, "project-1", "", "", "service-two", 1, 20)
	if err != nil {
		t.Fatal(err)
	}
	if matched.Total != 1 || len(matched.Items) != 1 || matched.Items[0].Code != "service-two" {
		t.Fatalf("service code search = %#v", matched)
	}

	for _, search := range []string{"application-code", "instance-one", "version-label"} {
		matched, err = repository.ListServicesByProject(ctx, "project-1", "", "", search, 1, 20)
		if err != nil {
			t.Fatal(err)
		}
		if matched.Total != 0 || len(matched.Items) != 0 {
			t.Fatalf("search %q unexpectedly matched %#v", search, matched)
		}
	}
}

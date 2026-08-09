package applicationrepo

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func TestListApplicationsBindsTypedFilters(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, project := range []struct {
		id   string
		code string
	}{
		{id: "project-1", code: "project-one"},
		{id: "project-2", code: "project-two"},
	} {
		if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES (?, ?, ?)`, project.id, project.code, project.code); err != nil {
			t.Fatal(err)
		}
	}

	projectOne := "project-1"
	projectTwo := "project-2"
	repo := NewRepository(database)
	for _, app := range []model.Application{
		{Id: "application-1", ProjectId: &projectOne, Name: "Application One", Code: "application-one", Kind: "standard"},
		{Id: "application-2", ProjectId: &projectTwo, Name: "Application Two", Code: "application-two", Kind: "gateway"},
	} {
		if err := repo.CreateApplication(ctx, app); err != nil {
			t.Fatal(err)
		}
	}

	filtered, err := repo.ListApplications(ctx, &projectOne, 1, 1, "One", "standard")
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].Id != "application-1" {
		t.Fatalf("unexpected filtered page: %+v", filtered)
	}

	all, err := repo.ListApplications(ctx, nil, 1, 100, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if all.Total != 2 || len(all.Items) != 2 {
		t.Fatalf("unexpected unfiltered page: %+v", all)
	}
}

func TestDeleteApplicationRejectsReferencedVersion(t *testing.T) {
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
	if _, err := database.ExecContext(ctx, `INSERT INTO service (id, application_id, version_id, instance_key, code, status) VALUES ('service-1', 'app-1', 'version-1', 'default', 'app-default', 'stopped')`); err != nil {
		t.Fatal(err)
	}

	err = NewRepository(database).DeleteApplication(ctx, "app-1")
	if !errors.Is(err, repository.ErrReferenced) {
		t.Fatalf("expected referenced version error, got %v", err)
	}
}

func TestDeleteGatewayApplicationDeletesStoppedResources(t *testing.T) {
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
		`INSERT INTO application (id, name, code, kind) VALUES ('gateway-1', 'Gateway', 'gateway', 'gateway')`,
		`INSERT INTO gateway_config (application_id, rest_api_url, base_domain) VALUES ('gateway-1', 'http://127.0.0.1:8080', 'example.test')`,
		`INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'gateway-1', 'managed', 'unpublished')`,
		`INSERT INTO service (id, application_id, version_id, instance_key, code, status) VALUES ('service-1', 'gateway-1', 'version-1', 'default', 'gateway-default', 'stopped')`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}

	if err := NewRepository(database).DeleteGatewayApplication(ctx, "gateway-1"); err != nil {
		t.Fatal(err)
	}
	for _, table := range []string{"application", "gateway_config", "version", "service"} {
		var count int
		if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 0 {
			t.Fatalf("expected %s to be deleted, found %d rows", table, count)
		}
	}
}

func TestDeleteVersionClearsForkReferenceAndPreservesPipelineRunHistory(t *testing.T) {
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
	if _, err := database.ExecContext(ctx, `INSERT INTO version (id, application_id, label, status, created_from_version_id) VALUES ('version-2', 'app-1', 'v2', 'unpublished', 'version-1')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO version (id, application_id, label, status) VALUES ('version-3', 'app-1', 'v3', 'unpublished')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO deployment (id, application_name, operation_type, trigger_type, status, is_rollback, version_id) VALUES ('deployment-1', 'App', 'deploy', 'manual', 'succeeded', 0, 'version-1')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO pipeline_run_version_binding (pipeline_run_id, application_id, application_name, source_version_id, source_version_label, generated_version_id, generated_version_label) VALUES ('run-1', 'app-1', 'App', 'version-1', 'v1', 'version-3', 'v3')`); err != nil {
		t.Fatal(err)
	}

	repo := NewRepository(database)
	refs, err := repo.CountVersionRuntimeRefs(ctx, "version-1")
	if err != nil {
		t.Fatal(err)
	}
	if refs != 0 {
		t.Fatalf("expected fork reference to be deletable, got %d blocking references", refs)
	}
	if err := repo.DeleteVersion(ctx, "version-1"); err != nil {
		t.Fatal(err)
	}

	var createdFrom sql.NullString
	if err := database.QueryRowContext(ctx, `SELECT created_from_version_id FROM version WHERE id = 'version-2'`).Scan(&createdFrom); err != nil {
		t.Fatal(err)
	}
	if createdFrom.Valid {
		t.Fatalf("expected cleared fork reference, got %q", createdFrom.String)
	}
	var deploymentVersionID string
	if err := database.QueryRowContext(ctx, `SELECT version_id FROM deployment WHERE id = 'deployment-1'`).Scan(&deploymentVersionID); err != nil {
		t.Fatal(err)
	}
	if deploymentVersionID != "version-1" {
		t.Fatalf("expected preserved deployment version reference, got %q", deploymentVersionID)
	}
	var sourceVersionID, generatedVersionID string
	if err := database.QueryRowContext(ctx, `SELECT source_version_id, generated_version_id FROM pipeline_run_version_binding WHERE pipeline_run_id = 'run-1'`).Scan(&sourceVersionID, &generatedVersionID); err != nil {
		t.Fatal(err)
	}
	if sourceVersionID != "version-1" || generatedVersionID != "version-3" {
		t.Fatalf("expected preserved pipeline run history, got source=%q generated=%q", sourceVersionID, generatedVersionID)
	}

	refs, err = repo.CountVersionRuntimeRefs(ctx, "version-3")
	if err != nil {
		t.Fatal(err)
	}
	if refs != 0 {
		t.Fatalf("expected generated version history to be non-blocking, got %d blocking references", refs)
	}
	if err := repo.DeleteVersion(ctx, "version-3"); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `SELECT generated_version_id FROM pipeline_run_version_binding WHERE pipeline_run_id = 'run-1'`).Scan(&generatedVersionID); err != nil {
		t.Fatal(err)
	}
	if generatedVersionID != "version-3" {
		t.Fatalf("expected preserved generated version history, got %q", generatedVersionID)
	}
}

func TestDeleteApplicationAllowsForkLineage(t *testing.T) {
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
	if _, err := database.ExecContext(ctx, `INSERT INTO version (id, application_id, label, status, created_from_version_id) VALUES ('version-2', 'app-1', 'v2', 'unpublished', 'version-1')`); err != nil {
		t.Fatal(err)
	}

	if err := NewRepository(database).DeleteApplication(ctx, "app-1"); err != nil {
		t.Fatal(err)
	}
	var remaining int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM application WHERE id = 'app-1'`).Scan(&remaining); err != nil {
		t.Fatal(err)
	}
	if remaining != 0 {
		t.Fatalf("expected application to be deleted, got %d remaining rows", remaining)
	}
}

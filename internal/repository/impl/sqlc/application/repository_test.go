package applicationrepo

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
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

	filtered, err := repo.ListApplications(ctx, projectOne, 1, 1, "One", "standard")
	if err != nil {
		t.Fatal(err)
	}
	if filtered.Total != 1 || len(filtered.Items) != 1 || filtered.Items[0].Id != "application-1" {
		t.Fatalf("unexpected filtered page: %+v", filtered)
	}

	otherProject, err := repo.ListApplications(ctx, projectTwo, 1, 100, "", "")
	if err != nil {
		t.Fatal(err)
	}
	if otherProject.Total != 1 || len(otherProject.Items) != 1 || otherProject.Items[0].Id != "application-2" {
		t.Fatalf("unexpected second project page: %+v", otherProject)
	}
}

func TestDeleteApplicationRejectsServiceReference(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO application (id, name, code, kind, project_id) VALUES ('app-1', 'App', 'app', 'application', 'project-1')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'app-1', 'v1', 'unpublished')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO service (id, project_id, application_id, version_id, code, status) VALUES ('service-1', 'project-1', 'app-1', 'version-1', 'app-default', 'stopped')`); err != nil {
		t.Fatal(err)
	}

	if err := NewRepository(database).DeleteApplication(ctx, "project-1", "app-1"); err == nil {
		t.Fatal("expected Application deletion to reject the Service reference")
	}
	for _, check := range []struct {
		table  string
		column string
		value  string
	}{
		{table: "application", column: "id", value: "app-1"},
		{table: "service", column: "id", value: "service-1"},
		{table: "version", column: "id", value: "version-1"},
	} {
		var count int
		if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM "+check.table+" WHERE "+check.column+" = ?", check.value).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != 1 {
			t.Fatalf("expected %s reference to remain, got %d rows", check.table, count)
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
	if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO application (id, name, code, kind, project_id) VALUES ('app-1', 'App', 'app', 'application', 'project-1')`); err != nil {
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
	if err := repo.DeleteVersion(ctx, "project-1", "version-1"); err != nil {
		t.Fatal(err)
	}

	var createdFrom sql.NullString
	if err := database.QueryRowContext(ctx, `SELECT created_from_version_id FROM version WHERE id = 'version-2'`).Scan(&createdFrom); err != nil {
		t.Fatal(err)
	}
	if createdFrom.Valid {
		t.Fatalf("expected cleared fork reference, got %q", createdFrom.String)
	}
	var deploymentVersionId string
	if err := database.QueryRowContext(ctx, `SELECT version_id FROM deployment WHERE id = 'deployment-1'`).Scan(&deploymentVersionId); err != nil {
		t.Fatal(err)
	}
	if deploymentVersionId != "version-1" {
		t.Fatalf("expected preserved deployment version reference, got %q", deploymentVersionId)
	}
	var sourceVersionId, generatedVersionId string
	if err := database.QueryRowContext(ctx, `SELECT source_version_id, generated_version_id FROM pipeline_run_version_binding WHERE pipeline_run_id = 'run-1'`).Scan(&sourceVersionId, &generatedVersionId); err != nil {
		t.Fatal(err)
	}
	if sourceVersionId != "version-1" || generatedVersionId != "version-3" {
		t.Fatalf("expected preserved pipeline run history, got source=%q generated=%q", sourceVersionId, generatedVersionId)
	}

	if err := repo.DeleteVersion(ctx, "project-1", "version-3"); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `SELECT generated_version_id FROM pipeline_run_version_binding WHERE pipeline_run_id = 'run-1'`).Scan(&generatedVersionId); err != nil {
		t.Fatal(err)
	}
	if generatedVersionId != "version-3" {
		t.Fatalf("expected preserved generated version history, got %q", generatedVersionId)
	}
}

func TestDeleteApplicationAllowsForkLineage(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateTo(database, config.DatabaseDriverSQLite, 30); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO application (id, name, code, kind, project_id) VALUES ('app-1', 'App', 'app', 'application', 'project-1')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'app-1', 'v1', 'unpublished')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO version (id, application_id, label, status, created_from_version_id) VALUES ('version-2', 'app-1', 'v2', 'unpublished', 'version-1')`); err != nil {
		t.Fatal(err)
	}

	if err := NewRepository(database).DeleteApplication(ctx, "project-1", "app-1"); err != nil {
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

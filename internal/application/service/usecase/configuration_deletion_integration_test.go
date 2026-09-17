package servicesvc

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	deploymentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/deployment"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	servicerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/service"
)

func TestRemoveServiceDeletesRunningServiceWithoutRuntimeOperation(t *testing.T) {
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
		`INSERT INTO project_member (project_id, user_id) VALUES ('project-1', 'user-1')`,
		`INSERT INTO application (id, project_id, name, code, kind) VALUES ('application-1', 'project-1', 'Application', 'application', 'standard')`,
		`INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'application-1', 'v1', 'unpublished')`,
		`INSERT INTO service (id, project_id, application_id, version_id, code, status) VALUES ('service-1', 'project-1', 'application-1', 'version-1', 'application', 'running')`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	service := New(
		projectrepo.NewRepository(database),
		applicationrepo.NewRepository(database),
		servicerepo.NewRepository(database),
		deploymentrepo.NewRepository(database),
	)
	if err := service.RemoveService(ctx, "user-1", "project-1", "service-1"); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM service WHERE id = 'service-1'`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("remaining service bindings = %d", count)
	}
}

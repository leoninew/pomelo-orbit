package applicationrepo

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

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
	if _, err := database.ExecContext(ctx, `INSERT INTO service (id, application_id, version_id, instance_key, status) VALUES ('service-1', 'app-1', 'version-1', 'default', 'stopped')`); err != nil {
		t.Fatal(err)
	}

	err = NewRepository(database).DeleteApplication(ctx, "app-1")
	if !errors.Is(err, repository.ErrReferenced) {
		t.Fatalf("expected referenced version error, got %v", err)
	}
}

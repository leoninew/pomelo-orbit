package projectsvc

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	projectdto "github.com/leoninew/pomelo-orbit/internal/application/project/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	environmentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	userrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/user"
)

const projectTestUserId = "01KKX2YNPF6VJ9N7QYCWG61KVK"

func TestProjectServiceCreateUpdateMembersAndDeprecate(t *testing.T) {
	service, database := newProjectIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	created, err := service.Create(ctx, projectTestUserId, testProjectCreateInput(t, "Second Project", "second"))
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Code != "second" || !created.IsActive {
		t.Fatalf("unexpected created project: %+v", created)
	}
	_, err = environmentrepo.NewRepository(database).EnvironmentByProject(ctx, created.Id)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected no environment for new project, got %v", err)
	}
	assertCount(t, database, `SELECT COUNT(*) FROM credential WHERE project_id = ?`, created.Id, 0)
	loaded, err := service.LoadForUser(ctx, created.Id, projectTestUserId)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Update(ctx, loaded, projectdto.SaveInput{Name: "Second Project Updated"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Second Project Updated" || updated.Code != "second" {
		t.Fatalf("unexpected updated project: %+v", updated)
	}
	members, err := service.Members(ctx, updated.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].Username != "admin" {
		t.Fatalf("unexpected project members: %+v", members)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO user (id, username, password_hash, status, oauth_provider, oauth_provider_id, email, auth_source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`, "member-user", "member", "hash", "enabled", "", "", "member@example.test", "password"); err != nil {
		t.Fatal(err)
	}
	members, err = service.AddMember(ctx, updated.Id, "member-user")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 2 {
		t.Fatalf("unexpected members after add: %+v", members)
	}
	members, err = service.RemoveMember(ctx, updated.Id, "member-user")
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 {
		t.Fatalf("unexpected members after remove: %+v", members)
	}
	if err := service.Deprecate(ctx, updated, projectTestUserId); err != nil {
		t.Fatal(err)
	}
	deprecated, err := service.LoadForUser(ctx, updated.Id, projectTestUserId)
	if err != nil {
		t.Fatal(err)
	}
	if deprecated.IsActive {
		t.Fatalf("expected deprecated project, got %+v", deprecated)
	}
}

func TestProjectServiceRejectsDuplicateCodeAndLastActiveDeprecation(t *testing.T) {
	service, database := newProjectIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	if _, err := service.Create(ctx, projectTestUserId, testProjectCreateInput(t, "Duplicate", "default")); err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("expected duplicate code conflict, got %v", err)
	}
	defaultProject, err := service.LoadForUser(ctx, "01KRRKK0K3T519ZQZES3M4QA9Z", projectTestUserId)
	if err != nil {
		t.Fatal(err)
	}
	_, err = environmentrepo.NewRepository(database).EnvironmentByProject(ctx, defaultProject.Id)
	if !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("expected seeded project to have no environment, got %v", err)
	}
	if err := service.Deprecate(ctx, defaultProject, projectTestUserId); err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected last active project deprecation validation error, got %v", err)
	}
}

func newProjectIntegrationService(t *testing.T) (Service, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	projectStore := projectrepo.NewRepository(database)
	return New(projectStore, userrepo.NewRepository(database), environmentrepo.NewRepository(database)), database
}

func testProjectCreateInput(t *testing.T, name string, code string) projectdto.CreateInput {
	t.Helper()
	return projectdto.CreateInput{
		Name: name,
		Code: code,
	}
}

func assertCount(t *testing.T, database *sql.DB, query string, value string, want int) {
	t.Helper()
	var got int
	if err := database.QueryRow(query, value).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("query %q with %q returned %d, want %d", query, value, got, want)
	}
}

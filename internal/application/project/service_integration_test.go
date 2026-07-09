package projectsvc

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	userrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/user"
)

const projectTestUserId = "01KKX2YNPF6VJ9N7QYCWG61KVK"

func TestProjectServiceCreateUpdateMembersAndDeprecate(t *testing.T) {
	service, database := newProjectIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	created, err := service.Create(ctx, projectTestUserId, SaveInput{Name: "Second Project", Code: "second"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Code != "second" || !created.IsActive {
		t.Fatalf("unexpected created project: %+v", created)
	}
	loaded, err := service.LoadForUser(ctx, created.Id, projectTestUserId)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Update(ctx, loaded, SaveInput{Name: "Second Project Updated", Code: "second-updated"})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Second Project Updated" || updated.Code != "second-updated" {
		t.Fatalf("unexpected updated project: %+v", updated)
	}
	members, err := service.Members(ctx, updated.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(members) != 1 || members[0].Username != "admin" {
		t.Fatalf("unexpected project members: %+v", members)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO user (id, username, password_hash, status, oauth_provider, oauth_provider_id, auth_source, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, datetime('now'), datetime('now'))`, "member-user", "member", "hash", "enabled", "", "", "password"); err != nil {
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
	if _, err := service.Create(ctx, projectTestUserId, SaveInput{Name: "Duplicate", Code: "default"}); err == nil || apperror.StatusCode(err) != 409 {
		t.Fatalf("expected duplicate code conflict, got %v", err)
	}
	defaultProject, err := service.LoadForUser(ctx, "01KRRKK0K3T519ZQZES3M4QA9Z", projectTestUserId)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.Deprecate(ctx, defaultProject, projectTestUserId); err == nil || apperror.StatusCode(err) != 400 {
		t.Fatalf("expected last active project deprecation validation error, got %v", err)
	}
}

func newProjectIntegrationService(t *testing.T) (Service, *sqlx.DB) {
	t.Helper()
	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	service := New(projectrepo.NewRepository(database, config.DatabaseDriverSQLite), userrepo.NewRepository(database, config.DatabaseDriverSQLite))
	return service, database
}

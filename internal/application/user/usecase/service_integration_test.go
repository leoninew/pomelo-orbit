package usersvc

import (
	"context"
	"strings"
	"testing"

	"database/sql"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	userdto "github.com/leoninew/pomelo-orbit/internal/application/user/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	rolerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/role"
	userrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/user"
)

func TestUserServiceCreateUpdateStatusAndDelete(t *testing.T) {
	service, database := newUserIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	email := "Operator@example.test"

	created, err := service.Create(ctx, userdto.CreateInput{Username: " operator ", Password: "secret1", Email: email})
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Username != "operator" || created.Status != "enabled" || created.Email != "operator@example.test" {
		t.Fatalf("unexpected created user: %+v", created)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(created.PasswordHash), []byte("secret1")); err != nil {
		t.Fatal(err)
	}
	actor := userdto.Actor{UserId: "administrator", Permissions: []string{"user:write"}}
	updated, err := service.UpdateByActor(ctx, actor, created.Id, userdto.UpdateInput{Username: stringPtr("operator2"), Email: stringPtr("operator2@example.test"), Password: stringPtr("newpass1"), Status: stringPtr("disabled")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.User.Username != "operator2" || updated.User.Email != "operator2@example.test" || updated.User.Status != "disabled" {
		t.Fatalf("unexpected updated user: %+v", updated.User)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.User.PasswordHash), []byte("newpass1")); err != nil {
		t.Fatal(err)
	}
	if err := service.SetStatusByActor(ctx, actor, updated.User.Id, "enabled"); err != nil {
		t.Fatal(err)
	}
	var projectID, roleID string
	if err := database.QueryRowContext(ctx, `SELECT id FROM project WHERE code = 'default'`).Scan(&projectID); err != nil {
		t.Fatal(err)
	}
	if err := database.QueryRowContext(ctx, `SELECT id FROM role WHERE code = 'admin'`).Scan(&roleID); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []struct {
		query string
		args  []any
	}{
		{`INSERT INTO project_member (project_id, user_id) VALUES (?, ?)`, []any{projectID, updated.User.Id}},
		{`INSERT INTO user_role (user_id, role_id) VALUES (?, ?)`, []any{updated.User.Id, roleID}},
		{`INSERT INTO login_history (id, user_id, username, success) VALUES (?, ?, ?, ?)`, []any{"login-history-1", updated.User.Id, updated.User.Username, true}},
	} {
		if _, err := database.ExecContext(ctx, statement.query, statement.args...); err != nil {
			t.Fatal(err)
		}
	}
	if err := service.DeleteByActor(ctx, actor, updated.User.Id); err != nil {
		t.Fatal(err)
	}
	for _, check := range []struct {
		query string
		want  int
	}{
		{`SELECT COUNT(*) FROM user WHERE id = ?`, 0},
		{`SELECT COUNT(*) FROM user_role WHERE user_id = ?`, 0},
		{`SELECT COUNT(*) FROM project_member WHERE user_id = ?`, 0},
		{`SELECT COUNT(*) FROM login_history WHERE user_id = ?`, 1},
	} {
		var count int
		if err := database.QueryRowContext(ctx, check.query, updated.User.Id).Scan(&count); err != nil {
			t.Fatal(err)
		}
		if count != check.want {
			t.Fatalf("%s count=%d, want %d", check.query, count, check.want)
		}
	}
}

func TestUserServiceRejectsDuplicateUsernameAndEmail(t *testing.T) {
	service, database := newUserIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	email := "operator@example.test"
	if _, err := service.Create(ctx, userdto.CreateInput{Username: "operator", Password: "secret1", Email: email}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, userdto.CreateInput{Username: "operator", Password: "secret1", Email: "other@example.test"}); err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("expected duplicate username conflict, got %v", err)
	}
	if _, err := service.Create(ctx, userdto.CreateInput{Username: "operator2", Password: "secret1", Email: email}); err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("expected duplicate email conflict, got %v", err)
	}
	if _, err := service.Create(ctx, userdto.CreateInput{Username: "operator3", Password: "secret1"}); err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected missing email validation error, got %v", err)
	}
}

func TestUserServiceEnforcesPasswordLength(t *testing.T) {
	service, database := newUserIntegrationService(t)
	defer func() { _ = database.Close() }()

	if _, err := service.Create(context.Background(), userdto.CreateInput{Username: "operator", Email: "operator@example.test", Password: strings.Repeat("a", 36)}); err != nil {
		t.Fatalf("expected 36-character password to be accepted, got %v", err)
	}
	if _, err := service.Create(context.Background(), userdto.CreateInput{Username: "operator2", Email: "operator2@example.test", Password: strings.Repeat("a", 37)}); err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected 37-character password to be rejected, got %v", err)
	}
}

func newUserIntegrationService(t *testing.T) (Service, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	return New(
		userrepo.NewRepository(database),
		rolerepo.NewRepository(database),
		projectrepo.NewRepository(database),
	), database
}

func stringPtr(value string) *string {
	return &value
}

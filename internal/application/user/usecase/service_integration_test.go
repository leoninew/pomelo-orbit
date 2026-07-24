package usersvc

import (
	"context"
	"testing"

	"database/sql"

	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	userdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	rolerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/role"
	userrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/user"
)

func TestUserServiceCreateUpdateStatusAndDelete(t *testing.T) {
	service, database := newUserIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	email := "operator@example.test"

	created, err := service.Create(ctx, userdto.CreateInput{Username: " operator ", Password: "secret1", Email: &email})
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Username != "operator" || created.Status != "enabled" || created.Email == nil || *created.Email != email {
		t.Fatalf("unexpected created user: %+v", created)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(created.PasswordHash), []byte("secret1")); err != nil {
		t.Fatal(err)
	}
	actor := userdto.Actor{UserId: "administrator", Permissions: []string{"user:write"}}
	updated, err := service.UpdateByActor(ctx, actor, created.Id, userdto.UpdateInput{Username: stringPtr("operator2"), Password: stringPtr("newpass1"), Status: stringPtr("disabled")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.User.Username != "operator2" || updated.User.Status != "disabled" {
		t.Fatalf("unexpected updated user: %+v", updated.User)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.User.PasswordHash), []byte("newpass1")); err != nil {
		t.Fatal(err)
	}
	if err := service.SetStatusByActor(ctx, actor, updated.User.Id, "enabled"); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteByActor(ctx, actor, updated.User.Id); err != nil {
		t.Fatal(err)
	}
}

func TestUserServiceRejectsDuplicateUsernameAndEmail(t *testing.T) {
	service, database := newUserIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	email := "operator@example.test"
	if _, err := service.Create(ctx, userdto.CreateInput{Username: "operator", Password: "secret1", Email: &email}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(ctx, userdto.CreateInput{Username: "operator", Password: "secret1"}); err == nil || apperror.StatusCode(err) != 409 {
		t.Fatalf("expected duplicate username conflict, got %v", err)
	}
	if _, err := service.Create(ctx, userdto.CreateInput{Username: "operator2", Password: "secret1", Email: &email}); err == nil || apperror.StatusCode(err) != 409 {
		t.Fatalf("expected duplicate email conflict, got %v", err)
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
	), database
}

func stringPtr(value string) *string {
	return &value
}

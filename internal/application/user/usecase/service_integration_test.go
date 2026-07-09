package usersvc

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
	"golang.org/x/crypto/bcrypt"
	_ "modernc.org/sqlite"

	userdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/dto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
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
	updated, err := service.Update(ctx, userdto.UpdateInput{User: created, Username: stringPtr("operator2"), Password: stringPtr("newpass1"), Status: stringPtr("disabled")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Username != "operator2" || updated.Status != "disabled" {
		t.Fatalf("unexpected updated user: %+v", updated)
	}
	if err := bcrypt.CompareHashAndPassword([]byte(updated.PasswordHash), []byte("newpass1")); err != nil {
		t.Fatal(err)
	}
	if err := service.SetStatus(ctx, updated.Id, "enabled"); err != nil {
		t.Fatal(err)
	}
	if err := service.Delete(ctx, updated.Id); err != nil {
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

func newUserIntegrationService(t *testing.T) (Service, *sqlx.DB) {
	t.Helper()
	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	return New(userrepo.NewRepository(database, config.DatabaseDriverSQLite)), database
}

func stringPtr(value string) *string {
	return &value
}

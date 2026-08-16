package authsvc

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"

	authdto "github.com/leoninew/pomelo-orbit/internal/application/auth/dto"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	authrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/auth"
	userrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/user"
)

const authTestSecretKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestLoginChangePasswordAndHistory(t *testing.T) {
	service, database := newAuthIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	csrfToken, err := service.NewCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	token, err := service.Login(ctx, authdto.LoginInput{Username: "admin", Password: "admin", CSRFToken: csrfToken, IP: "127.0.0.1", UserAgent: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected login token")
	}
	user, err := userrepo.NewRepository(database).UserByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ChangePassword(ctx, authdto.ChangePasswordInput{User: user, OldPassword: "admin", NewPassword: "newpass1"}); err != nil {
		t.Fatal(err)
	}
	csrfToken, err = service.NewCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Login(ctx, authdto.LoginInput{Username: "admin", Password: "newpass1", CSRFToken: csrfToken}); err != nil {
		t.Fatal(err)
	}
	history, err := service.ListLoginHistory(ctx, 1, 10, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if history.Total == 0 || len(history.Items) == 0 || history.Items[0].Username != "admin" {
		t.Fatalf("unexpected login history: %+v", history)
	}
}

func TestLoginRejectsInvalidCSRFToken(t *testing.T) {
	service, database := newAuthIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	_, err := service.Login(ctx, authdto.LoginInput{Username: "admin", Password: "admin", CSRFToken: "csrf"})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected invalid csrf validation error, got %v", err)
	}
}

func TestChangePasswordRejectsWrongOldPassword(t *testing.T) {
	service, database := newAuthIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	user, err := userrepo.NewRepository(database).UserByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ChangePassword(ctx, authdto.ChangePasswordInput{User: user, OldPassword: "wrong", NewPassword: "newpass1"}); err == nil || apperror.StatusCode(err) != 401 {
		t.Fatalf("expected wrong old password to be unauthorized, got %v", err)
	}
}

func newAuthIntegrationService(t *testing.T) (Service, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := New(userrepo.NewRepository(database), authrepo.NewRepository(database), jwt.NewTokenService(authTestSecretKey), logger, authTestSecretKey)
	return service, database
}

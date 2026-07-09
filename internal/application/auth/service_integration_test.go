package authsvc

import (
	"context"
	"io"
	"log/slog"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	jwt "gitee.com/leoninew/PomeloOrbit-go/internal/auth/jwt"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	userrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/user"
)

const authTestSecretKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestLoginChangePasswordAndHistory(t *testing.T) {
	service, database := newAuthIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	token, err := service.Login(ctx, LoginInput{Username: "admin", Password: "admin", CSRFToken: "csrf", IP: "127.0.0.1", UserAgent: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected login token")
	}
	user, err := userrepo.NewRepository(database, config.DatabaseDriverSQLite).UserByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ChangePassword(ctx, ChangePasswordInput{User: user, OldPassword: "admin", NewPassword: "newpass1"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Login(ctx, LoginInput{Username: "admin", Password: "newpass1", CSRFToken: "csrf"}); err != nil {
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

func TestChangePasswordRejectsWrongOldPassword(t *testing.T) {
	service, database := newAuthIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	user, err := userrepo.NewRepository(database, config.DatabaseDriverSQLite).UserByUsername(ctx, "admin")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ChangePassword(ctx, ChangePasswordInput{User: user, OldPassword: "wrong", NewPassword: "newpass1"}); err == nil || apperror.StatusCode(err) != 401 {
		t.Fatalf("expected wrong old password to be unauthorized, got %v", err)
	}
}

func newAuthIntegrationService(t *testing.T) (Service, *sqlx.DB) {
	t.Helper()
	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	service := New(userrepo.NewRepository(database, config.DatabaseDriverSQLite), jwt.NewTokenService(authTestSecretKey), logger)
	return service, database
}

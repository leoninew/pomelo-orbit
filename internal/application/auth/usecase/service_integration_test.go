package authsvc

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	authdto "github.com/leoninew/pomelo-orbit/internal/application/auth/dto"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
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
	token, err := service.Login(ctx, authdto.LoginInput{Email: "ADMIN@LVH.ME", Password: "admin@lvh.me", CSRFToken: csrfToken, IP: "127.0.0.1", UserAgent: "test"})
	if err != nil {
		t.Fatal(err)
	}
	if token == "" {
		t.Fatal("expected login token")
	}
	user, err := userrepo.NewRepository(database).UserByEmail(ctx, "admin@lvh.me")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ChangePassword(ctx, authdto.ChangePasswordInput{User: user, OldPassword: "admin@lvh.me", NewPassword: "newpass1"}); err != nil {
		t.Fatal(err)
	}
	csrfToken, err = service.NewCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	if _, err := service.Login(ctx, authdto.LoginInput{Email: "admin@lvh.me", Password: "newpass1", CSRFToken: csrfToken}); err != nil {
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

	_, err := service.Login(ctx, authdto.LoginInput{Email: "admin@lvh.me", Password: "admin@lvh.me", CSRFToken: "csrf"})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected invalid csrf validation error, got %v", err)
	}
}

func TestLoginDoesNotAcceptUsername(t *testing.T) {
	service, database := newAuthIntegrationService(t)
	defer func() { _ = database.Close() }()
	csrfToken, err := service.NewCSRFToken()
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Login(context.Background(), authdto.LoginInput{Email: "admin", Password: "admin@lvh.me", CSRFToken: csrfToken})
	if err == nil || !apperror.IsKind(err, apperror.KindUnauthorized) {
		t.Fatalf("expected username login to be rejected, got %v", err)
	}
}

func TestChangePasswordRejectsWrongOldPassword(t *testing.T) {
	service, database := newAuthIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	user, err := userrepo.NewRepository(database).UserByEmail(ctx, "admin@lvh.me")
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ChangePassword(ctx, authdto.ChangePasswordInput{User: user, OldPassword: "wrong", NewPassword: "newpass1"}); err == nil || apperror.StatusCode(err) != 401 {
		t.Fatalf("expected wrong old password to be unauthorized, got %v", err)
	}
}

func TestMCPAccessTokenLifecycle(t *testing.T) {
	service, database := newAuthIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	users := userrepo.NewRepository(database)
	user, err := users.UserByEmail(ctx, "admin@lvh.me")
	if err != nil {
		t.Fatal(err)
	}

	created, err := service.CreateMCPAccessToken(ctx, user.Id, authdto.MCPAccessTokenCreateInput{Name: "local Codex"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Token == "" || created.Token == created.AccessToken.TokenHash {
		t.Fatalf("created MCP access token must return a distinct raw token and persisted hash: %#v", created)
	}
	var persistedHash string
	if err := database.QueryRowContext(ctx, "SELECT token_hash FROM mcp_access_token WHERE id = ?", created.AccessToken.Id).Scan(&persistedHash); err != nil {
		t.Fatal(err)
	}
	if persistedHash != created.AccessToken.TokenHash || persistedHash == created.Token {
		t.Fatalf("persisted MCP token hash = %q, want stored digest only", persistedHash)
	}
	if _, err := service.AuthenticateMCPAccessToken(ctx, created.Token); err != nil {
		t.Fatalf("AuthenticateMCPAccessToken() error = %v", err)
	}
	if _, err := service.AuthenticateMCPAccessToken(ctx, created.Token+"x"); err == nil || !apperror.IsKind(err, apperror.KindUnauthorized) {
		t.Fatalf("tampered MCP access token error = %v, want unauthorized", err)
	}
	if err := service.RevokeMCPAccessToken(ctx, user.Id, created.AccessToken.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateMCPAccessToken(ctx, created.Token); err == nil || !apperror.IsKind(err, apperror.KindUnauthorized) {
		t.Fatalf("revoked MCP access token error = %v, want unauthorized", err)
	}

	expiredRaw := "orbit_mcp_pat_expired_test_token"
	expiresAt := time.Now().UTC().Add(-time.Second)
	if err := authrepo.NewRepository(database).CreateMCPAccessToken(ctx, model.MCPAccessToken{Id: "expired-token", UserId: user.Id, Name: "expired", TokenHash: hashMCPAccessToken(expiredRaw), ExpiresAt: &expiresAt, CreatedAt: time.Now().UTC()}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateMCPAccessToken(ctx, expiredRaw); err == nil || !apperror.IsKind(err, apperror.KindUnauthorized) {
		t.Fatalf("expired MCP access token error = %v, want unauthorized", err)
	}

	active, err := service.CreateMCPAccessToken(ctx, user.Id, authdto.MCPAccessTokenCreateInput{Name: "disabled user"})
	if err != nil {
		t.Fatal(err)
	}
	if err := users.SetUserStatus(ctx, user.Id, "disabled"); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateMCPAccessToken(ctx, active.Token); err == nil || !apperror.IsKind(err, apperror.KindUnauthorized) {
		t.Fatalf("disabled user MCP access token error = %v, want unauthorized", err)
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

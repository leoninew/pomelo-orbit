//go:build !unit

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
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	authrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/auth"
	userrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/user"
)

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
	if created.Token == nil || *created.Token == "" || *created.Token == created.AccessToken.TokenHash {
		t.Fatal("created MCP access token must return a distinct raw token and persisted hash")
	}
	rawToken := *created.Token
	var persistedHash, persistedCiphertext string
	if err := database.QueryRowContext(ctx, "SELECT token_hash, encrypted_token FROM mcp_access_token WHERE id = ?", created.AccessToken.Id).Scan(&persistedHash, &persistedCiphertext); err != nil {
		t.Fatal(err)
	}
	if persistedHash != created.AccessToken.TokenHash || persistedHash == rawToken || persistedCiphertext == rawToken {
		t.Fatal("MCP token must not be persisted in plaintext")
	}
	plain, err := security.DecryptString(authTestSecretKey, persistedCiphertext)
	if err != nil || plain != rawToken {
		t.Fatal("stored MCP token must be recoverable with the configured Fernet key")
	}
	listed, err := service.ListMCPAccessTokens(ctx, user.Id)
	if err != nil || len(listed) != 1 || listed[0].Token == nil || *listed[0].Token != rawToken {
		t.Fatal("token owner must be able to retrieve the stored token")
	}
	otherUserTokens, err := service.ListMCPAccessTokens(ctx, "another-user")
	if err != nil || len(otherUserTokens) != 0 {
		t.Fatal("token list must be scoped to its owner")
	}
	if err := service.RevokeMCPAccessToken(ctx, "another-user", created.AccessToken.Id); !apperror.IsKind(err, apperror.KindNotFound) {
		t.Fatalf("another user's revoke error = %v, want not found", err)
	}
	if _, err := service.AuthenticateMCPAccessToken(ctx, rawToken); err != nil {
		t.Fatalf("AuthenticateMCPAccessToken() error = %v", err)
	}
	if _, err := service.AuthenticateMCPAccessToken(ctx, rawToken+"x"); err == nil || !apperror.IsKind(err, apperror.KindUnauthorized) {
		t.Fatalf("tampered MCP access token error = %v, want unauthorized", err)
	}
	if err := service.RevokeMCPAccessToken(ctx, user.Id, created.AccessToken.Id); err != nil {
		t.Fatal(err)
	}
	if _, err := service.AuthenticateMCPAccessToken(ctx, rawToken); err == nil || !apperror.IsKind(err, apperror.KindUnauthorized) {
		t.Fatalf("revoked MCP access token error = %v, want unauthorized", err)
	}

	expiredRaw := "orbit_mcp_pat_expired_test_token"
	expiredEncrypted, err := security.EncryptString(authTestSecretKey, expiredRaw)
	if err != nil {
		t.Fatal(err)
	}
	expiresAt := time.Now().UTC().Add(-time.Second)
	if err := authrepo.NewRepository(database).CreateMCPAccessToken(ctx, model.MCPAccessToken{Id: "expired-token", UserId: user.Id, Name: "expired", TokenHash: hashMCPAccessToken(expiredRaw), EncryptedToken: &expiredEncrypted, ExpiresAt: &expiresAt, CreatedAt: time.Now().UTC()}); err != nil {
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
	if _, err := service.AuthenticateMCPAccessToken(ctx, *active.Token); err == nil || !apperror.IsKind(err, apperror.KindUnauthorized) {
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

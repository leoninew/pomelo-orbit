//go:build !unit

package authsvc

import (
	"context"
	"errors"
	"reflect"
	"testing"

	authport "github.com/leoninew/pomelo-orbit/internal/application/auth/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	userrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/user"
)

func TestGoogleLoginPersistsLinkedAndNewAccounts(t *testing.T) {
	auth, database := newAuthIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	users := userrepo.NewRepository(database)
	admin, err := users.UserByEmail(ctx, "admin@lvh.me")
	if err != nil {
		t.Fatal(err)
	}
	roles, err := users.UserRoles(ctx, admin.Id)
	if err != nil {
		t.Fatal(err)
	}
	provider := &googleProviderFacts{identity: authport.GoogleIdentity{Subject: "admin-google-subject", Email: "ADMIN@LVH.ME", EmailVerified: true, Name: "Admin"}}
	google := NewGoogle(auth, provider, tx.NewTransactionRunner(database))
	result, err := completeGoogle(t, google, provider)
	if err != nil {
		t.Fatal(err)
	}
	linked, err := users.UserByOAuthAccount(ctx, "google", provider.identity.Subject)
	if err != nil {
		t.Fatal(err)
	}
	if linked.Id != admin.Id || linked.PasswordHash != admin.PasswordHash || linked.LastLoginAt == nil {
		t.Fatalf("linked account=%+v", linked)
	}
	linkedRoles, err := users.UserRoles(ctx, linked.Id)
	if err != nil || !reflect.DeepEqual(roles, linkedRoles) {
		t.Fatalf("roles=%v error=%v", linkedRoles, err)
	}
	claims, err := auth.tokens.Verify(result.Token)
	if err != nil || claims.Sub != admin.Id {
		t.Fatalf("claims=%+v error=%v", claims, err)
	}
	if _, err := users.UserByOAuthAccount(ctx, "other", provider.identity.Subject); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("provider lookup error=%v", err)
	}

	provider.identity = authport.GoogleIdentity{Subject: "new-google-subject", Email: "new-user@example.test", EmailVerified: true, Name: "Google User"}
	if _, err := completeGoogle(t, google, provider); err != nil {
		t.Fatal(err)
	}
	created, err := users.UserByOAuthAccount(ctx, "google", provider.identity.Subject)
	if err != nil {
		t.Fatal(err)
	}
	if created.PasswordHash != "" || created.LastLoginAt == nil || created.Status != "enabled" {
		t.Fatalf("new account=%+v", created)
	}
	newRoles, err := users.UserRoles(ctx, created.Id)
	if err != nil || len(newRoles) != 0 {
		t.Fatalf("new user roles=%v error=%v", newRoles, err)
	}
	provider.identity.Email = "changed-google-email@example.test"
	if _, err := completeGoogle(t, google, provider); err != nil {
		t.Fatal(err)
	}
	var count int
	if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM user WHERE oauth_provider_id = ?", provider.identity.Subject).Scan(&count); err != nil || count != 1 {
		t.Fatalf("Google subject must reuse one account: count=%d error=%v", count, err)
	}
	if err := users.SetUserStatus(ctx, created.Id, "disabled"); err != nil {
		t.Fatal(err)
	}
	if result, err := completeGoogle(t, google, provider); !apperror.IsKind(err, apperror.KindForbidden) || result.Token != "" {
		t.Fatalf("disabled login result=%+v error=%v", result, err)
	}
}

func TestGoogleLoginRollsBackAccountChangesWhenHistoryFails(t *testing.T) {
	for _, email := range []string{"admin@lvh.me", "new-user@example.test"} {
		t.Run(email, func(t *testing.T) {
			auth, database := newAuthIntegrationService(t)
			defer func() { _ = database.Close() }()
			ctx := context.Background()
			if _, err := database.ExecContext(ctx, `CREATE TRIGGER reject_google_history BEFORE INSERT ON login_history BEGIN SELECT RAISE(ABORT, 'history unavailable'); END`); err != nil {
				t.Fatal(err)
			}
			provider := &googleProviderFacts{identity: authport.GoogleIdentity{Subject: "rollback-subject", Email: email, EmailVerified: true}}
			google := NewGoogle(auth, provider, tx.NewTransactionRunner(database))
			result, err := completeGoogle(t, google, provider)
			if err == nil || result.Token != "" {
				t.Fatalf("result=%+v error=%v", result, err)
			}
			var count int
			if err := database.QueryRowContext(ctx, "SELECT COUNT(*) FROM user WHERE oauth_provider_id = ?", provider.identity.Subject).Scan(&count); err != nil || count != 0 {
				t.Fatalf("uncommitted identity persisted: count=%d error=%v", count, err)
			}
			user, err := userrepo.NewRepository(database).UserByEmail(ctx, email)
			if email == "admin@lvh.me" {
				if err != nil || user.OAuthProvider != "" || user.LastLoginAt != nil {
					t.Fatalf("existing account changed after rollback: user=%+v error=%v", user, err)
				}
			} else if !errors.Is(err, repository.ErrNotFound) {
				t.Fatalf("new account survived rollback: error=%v", err)
			}
		})
	}
}

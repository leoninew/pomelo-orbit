package authsvc

import (
	"context"
	"testing"
	"time"

	authdto "github.com/leoninew/pomelo-orbit/internal/application/auth/dto"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestMCPGrantIsLoopbackBoundSingleUseAndIssuesBearerToken(t *testing.T) {
	user := model.User{Id: "user-1", Username: "operator", Status: "enabled"}
	service := New(mcpGrantUserStore{user: user}, nil, jwt.NewTokenService("abcdefghijklmnopqrstuvwxyz123456"), nil, "abcdefghijklmnopqrstuvwxyz123456")
	grant, err := service.IssueMCPGrant(context.Background(), user, authdto.MCPGrantInput{CallbackURL: "http://127.0.0.1:48123/mcp/callback", State: "abcdefghijklmnopqrstuvwxyz123456"})
	if err != nil {
		t.Fatalf("IssueMCPGrant() error = %v", err)
	}
	if grant.Code == "" || !grant.ExpiresAt.After(time.Now()) {
		t.Fatalf("grant = %#v", grant)
	}
	token, err := service.ExchangeMCPGrant(context.Background(), grant.Code)
	if err != nil {
		t.Fatalf("ExchangeMCPGrant() error = %v", err)
	}
	claims, err := jwt.NewTokenService("abcdefghijklmnopqrstuvwxyz123456").Verify(token)
	if err != nil || claims.Sub != user.Id {
		t.Fatalf("token claims = %#v, error = %v", claims, err)
	}
	if _, err := service.ExchangeMCPGrant(context.Background(), grant.Code); err == nil {
		t.Fatal("ExchangeMCPGrant() replay error = nil")
	}
}

func TestMCPGrantRejectsNonLoopbackCallback(t *testing.T) {
	service := New(mcpGrantUserStore{}, nil, jwt.NewTokenService("abcdefghijklmnopqrstuvwxyz123456"), nil, "abcdefghijklmnopqrstuvwxyz123456")
	_, err := service.IssueMCPGrant(context.Background(), model.User{Id: "user-1", Status: "enabled"}, authdto.MCPGrantInput{CallbackURL: "https://example.com/mcp/callback", State: "abcdefghijklmnopqrstuvwxyz123456"})
	if err == nil {
		t.Fatal("IssueMCPGrant() error = nil")
	}
}

func TestMCPGrantExpiresBeforeExchange(t *testing.T) {
	service := New(mcpGrantUserStore{user: model.User{Id: "user-1", Username: "operator", Status: "enabled"}}, nil, jwt.NewTokenService("abcdefghijklmnopqrstuvwxyz123456"), nil, "abcdefghijklmnopqrstuvwxyz123456")
	service.grants.put("expired-code", mcpGrantRecord{userID: "user-1", expiresAt: time.Now().UTC().Add(-time.Second)})
	if _, err := service.ExchangeMCPGrant(context.Background(), "expired-code"); err == nil {
		t.Fatal("ExchangeMCPGrant() expired code error = nil")
	}
}

type mcpGrantUserStore struct {
	repository.UserStore
	user model.User
}

func (s mcpGrantUserStore) UserById(context.Context, string) (model.User, error) {
	return s.user, nil
}

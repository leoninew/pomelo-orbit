package authsvc

import (
	"context"
	"testing"

	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type accessTokenFacts struct {
	repository.AuthStore
	items           []model.MCPAccessToken
	requestedUserId string
}

func (f *accessTokenFacts) ListMCPAccessTokens(_ context.Context, userId string) ([]model.MCPAccessToken, error) {
	f.requestedUserId = userId
	return f.items, nil
}

func TestListMCPAccessTokensDecryptsPresentValues(t *testing.T) {
	const plain = "orbit_mcp_pat_test"
	encrypted, err := security.EncryptString(authTestSecretKey, plain)
	if err != nil {
		t.Fatal(err)
	}
	facts := &accessTokenFacts{items: []model.MCPAccessToken{
		{Id: "available", EncryptedToken: &encrypted},
		{Id: "empty"},
	}}
	service := New(nil, facts, jwt.TokenService{}, nil, authTestSecretKey)
	items, err := service.ListMCPAccessTokens(context.Background(), "owner")
	if err != nil {
		t.Fatal(err)
	}
	if facts.requestedUserId != "owner" || len(items) != 2 || items[0].Token == nil || *items[0].Token != plain || items[1].Token != nil {
		t.Fatal("expected owner-scoped decrypted and absent token values")
	}
	if *facts.items[0].EncryptedToken != encrypted {
		t.Fatal("listing must not replace persisted ciphertext with plaintext")
	}
}

func TestListMCPAccessTokensRejectsUndecryptableValues(t *testing.T) {
	ciphertext, err := security.EncryptString(authTestSecretKey, "orbit_mcp_pat_test")
	if err != nil {
		t.Fatal(err)
	}
	for _, test := range []struct{ name, key, value string }{
		{"invalid ciphertext", authTestSecretKey, "not-a-fernet-token"},
		{"wrong key", "AQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQEBAQE=", ciphertext},
	} {
		t.Run(test.name, func(t *testing.T) {
			facts := &accessTokenFacts{items: []model.MCPAccessToken{{EncryptedToken: &test.value}}}
			service := New(nil, facts, jwt.TokenService{}, nil, test.key)
			items, err := service.ListMCPAccessTokens(context.Background(), "owner")
			if !apperror.IsKind(err, apperror.KindInternal) || items != nil {
				t.Fatal("decryption failure must fail the request, not return an empty token")
			}
		})
	}
}

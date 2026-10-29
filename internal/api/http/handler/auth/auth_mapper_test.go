package authhandler

import (
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	authdto "github.com/leoninew/pomelo-orbit/internal/application/auth/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestMCPAccessTokenResponseDoesNotExposePersistenceSecrets(t *testing.T) {
	ciphertext := "ciphertext-must-not-leak"
	plain := "plain-token"
	response := mcpAccessTokenResponse(authdto.MCPAccessTokenDetail{AccessToken: model.MCPAccessToken{
		Id:             "token-1",
		UserId:         "user-1",
		Name:           "local Codex",
		TokenHash:      "must-not-leak",
		EncryptedToken: &ciphertext,
		CreatedAt:      time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC),
	}, Token: &plain})
	encoded, err := protojson.Marshal(&response)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "must-not-leak") {
		t.Fatalf("MCP access token response leaks persistence fields: %s", encoded)
	}
	if response.Token == nil || *response.Token != plain {
		t.Fatal("token response must expose only the explicitly decrypted value")
	}
}

func TestMCPAccessTokenResponsePreservesAbsentValue(t *testing.T) {
	response := mcpAccessTokenResponse(authdto.MCPAccessTokenDetail{AccessToken: model.MCPAccessToken{Id: "empty"}})
	if response.Token != nil {
		t.Fatal("absent token must remain absent")
	}
}

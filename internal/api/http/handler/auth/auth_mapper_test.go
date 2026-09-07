package authhandler

import (
	"strings"
	"testing"
	"time"

	"google.golang.org/protobuf/encoding/protojson"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestMCPAccessTokenResponseDoesNotExposeHash(t *testing.T) {
	response := mcpAccessTokenResponse(model.MCPAccessToken{
		Id:        "token-1",
		UserId:    "user-1",
		Name:      "local Codex",
		TokenHash: "must-not-leak",
		CreatedAt: time.Date(2026, 9, 7, 0, 0, 0, 0, time.UTC),
	})
	encoded, err := protojson.Marshal(&response)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(encoded), "must-not-leak") {
		t.Fatalf("MCP access token response leaks persistence fields: %s", encoded)
	}
}

package repository

import (
	"context"

	"github.com/leoninew/pomelo-orbit/internal/model"
)

// AuthStore persists authentication audit records.
type AuthStore interface {
	SaveLoginHistory(ctx context.Context, history model.LoginHistory) error
	ListLoginHistory(ctx context.Context, page int, perPage int, search string) (Page[model.LoginHistory], error)
	CreateMCPAccessToken(ctx context.Context, token model.MCPAccessToken) error
	ListMCPAccessTokens(ctx context.Context, userId string) ([]model.MCPAccessToken, error)
	MCPAccessTokenByHash(ctx context.Context, tokenHash string) (model.MCPAccessToken, error)
	MCPAccessTokenForUser(ctx context.Context, userId string, tokenId string) (model.MCPAccessToken, error)
	DeleteMCPAccessToken(ctx context.Context, tokenId string) error
}

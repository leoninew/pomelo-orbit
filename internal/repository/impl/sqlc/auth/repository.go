package authrepo

import (
	"context"
	"database/sql"
	"fmt"

	authsqlc "github.com/leoninew/pomelo-orbit/internal/gen/sqlc/auth"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/dbmodel"
	"github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlcommon"
)

var _ repository.AuthStore = Repository{}

type Repository struct {
	db *sql.DB
}

func NewRepository(db *sql.DB) Repository {
	return Repository{db: db}
}

func (r Repository) q(ctx context.Context) *authsqlc.Queries {
	return dbmodel.Queries(ctx, r.db, func(dbtx tx.DbTX) *authsqlc.Queries {
		return authsqlc.New(dbtx)
	})
}

func (r Repository) SaveLoginHistory(ctx context.Context, history model.LoginHistory) error {
	err := r.q(ctx).SaveLoginHistory(ctx, authsqlc.SaveLoginHistoryParams{
		ID:        history.Id,
		UserID:    history.UserId,
		Username:  history.Username,
		IpAddress: dbmodel.NullString(history.IpAddress),
		UserAgent: dbmodel.NullString(history.UserAgent),
		LoginAt:   history.LoginAt,
		Success:   dbmodel.BoolInt(history.Success),
	})
	if err != nil {
		return fmt.Errorf("save login history %s: %w", history.Id, err)
	}
	return nil
}

func (r Repository) ListLoginHistory(ctx context.Context, page int, perPage int, search string) (repository.Page[model.LoginHistory], error) {
	page, perPage = repository.NormalizePage(page, perPage)
	raw, pattern := dbmodel.LowerSearchPattern(search)
	searchPattern := sql.NullString{String: pattern, Valid: raw != ""}
	q := r.q(ctx)
	total, err := q.CountLoginHistory(ctx, authsqlc.CountLoginHistoryParams{SearchPattern: searchPattern})
	if err != nil {
		return repository.Page[model.LoginHistory]{}, fmt.Errorf("count login history: %w", err)
	}
	rows, err := q.ListLoginHistory(ctx, authsqlc.ListLoginHistoryParams{
		SearchPattern: searchPattern,
		Limit:         int32(perPage),
		Offset:        int32((page - 1) * perPage),
	})
	if err != nil {
		return repository.Page[model.LoginHistory]{}, fmt.Errorf("list login history: %w", err)
	}
	items := make([]model.LoginHistory, 0, len(rows))
	for _, row := range rows {
		items = append(items, model.LoginHistory{
			Id:        row.ID,
			UserId:    row.UserID,
			Username:  row.Username,
			IpAddress: dbmodel.StringPtr(row.IpAddress),
			UserAgent: dbmodel.StringPtr(row.UserAgent),
			LoginAt:   row.LoginAt,
			Success:   dbmodel.IntBool(row.Success),
		})
	}
	return repository.Page[model.LoginHistory]{Items: items, Total: int(total), Page: page, PerPage: perPage}, nil
}

func (r Repository) CreateMCPAccessToken(ctx context.Context, token model.MCPAccessToken) error {
	err := r.q(ctx).CreateMCPAccessToken(ctx, authsqlc.CreateMCPAccessTokenParams{
		ID:        token.Id,
		UserID:    token.UserId,
		Name:      token.Name,
		TokenHash: token.TokenHash,
		ExpiresAt: dbmodel.NullTime(token.ExpiresAt),
		CreatedAt: token.CreatedAt,
	})
	if err != nil {
		return fmt.Errorf("create MCP access token %s: %w", token.Id, err)
	}
	return nil
}

func (r Repository) ListMCPAccessTokens(ctx context.Context, userId string) ([]model.MCPAccessToken, error) {
	rows, err := r.q(ctx).ListMCPAccessTokens(ctx, userId)
	if err != nil {
		return nil, fmt.Errorf("list MCP access tokens for user %s: %w", userId, err)
	}
	items := make([]model.MCPAccessToken, 0, len(rows))
	for _, row := range rows {
		items = append(items, mcpAccessTokenFrom(row))
	}
	return items, nil
}

func (r Repository) MCPAccessTokenByHash(ctx context.Context, tokenHash string) (model.MCPAccessToken, error) {
	row, err := r.q(ctx).MCPAccessTokenByHash(ctx, tokenHash)
	if err != nil {
		return model.MCPAccessToken{}, fmt.Errorf("load MCP access token by hash: %w", sqlcommon.TranslateError(err))
	}
	return mcpAccessTokenFrom(row), nil
}

func (r Repository) MCPAccessTokenForUser(ctx context.Context, userId string, tokenId string) (model.MCPAccessToken, error) {
	row, err := r.q(ctx).MCPAccessTokenForUser(ctx, authsqlc.MCPAccessTokenForUserParams{ID: tokenId, UserID: userId})
	if err != nil {
		return model.MCPAccessToken{}, fmt.Errorf("load MCP access token %s for user %s: %w", tokenId, userId, sqlcommon.TranslateError(err))
	}
	return mcpAccessTokenFrom(row), nil
}

func (r Repository) DeleteMCPAccessToken(ctx context.Context, tokenId string) error {
	if err := r.q(ctx).DeleteMCPAccessToken(ctx, tokenId); err != nil {
		return fmt.Errorf("delete MCP access token %s: %w", tokenId, err)
	}
	return nil
}

func mcpAccessTokenFrom(row authsqlc.MCPAccessToken) model.MCPAccessToken {
	return model.MCPAccessToken{
		Id:        row.ID,
		UserId:    row.UserID,
		Name:      row.Name,
		TokenHash: row.TokenHash,
		ExpiresAt: dbmodel.TimePtr(row.ExpiresAt),
		CreatedAt: row.CreatedAt,
	}
}

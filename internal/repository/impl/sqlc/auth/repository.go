package authrepo

import (
	"context"
	"database/sql"
	"fmt"

	authsqlc "gitee.com/leoninew/PomeloOrbit-go/internal/gen/sqlc/auth"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/dbmodel"
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
	total, err := q.CountLoginHistory(ctx, searchPattern)
	if err != nil {
		return repository.Page[model.LoginHistory]{}, fmt.Errorf("count login history: %w", err)
	}
	rows, err := q.ListLoginHistory(ctx, authsqlc.ListLoginHistoryParams{
		SearchPattern: searchPattern,
		Limit:         int64(perPage),
		Offset:        int64((page - 1) * perPage),
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

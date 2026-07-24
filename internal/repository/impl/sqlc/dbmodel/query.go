package dbmodel

import (
	"context"
	"database/sql"
	"strings"

	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
)

// Queries picks the request/message transaction when present, otherwise the root DB.
func Queries[T any](ctx context.Context, db *sql.DB, newFn func(tx.DBTX) T) T {
	if dbtx, err := tx.DBTXFrom(ctx); err == nil {
		return newFn(dbtx)
	}
	return newFn(db)
}

func SearchPattern(search string) (raw string, pattern string) {
	raw = strings.TrimSpace(search)
	if raw == "" {
		return "", ""
	}
	return raw, "%" + raw + "%"
}

func LowerSearchPattern(search string) (raw string, pattern string) {
	raw = strings.TrimSpace(search)
	if raw == "" {
		return "", ""
	}
	return raw, "%" + strings.ToLower(raw) + "%"
}

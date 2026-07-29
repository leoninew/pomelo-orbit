package tx

import (
	"context"
	"database/sql"
	"fmt"
)

// DbTX is the common interface implemented by *sql.DB and *sql.Tx.
type DbTX interface {
	ExecContext(context.Context, string, ...any) (sql.Result, error)
	PrepareContext(context.Context, string) (*sql.Stmt, error)
	QueryContext(context.Context, string, ...any) (*sql.Rows, error)
	QueryRowContext(context.Context, string, ...any) *sql.Row
}

type dbKey struct{}
type txKey struct{}

// WithDb attaches the root *sql.DB to ctx.
func WithDb(ctx context.Context, db *sql.DB) context.Context {
	return context.WithValue(ctx, dbKey{}, db)
}

// WithTx attaches an active *sql.Tx to ctx (request/message UoW).
func WithTx(ctx context.Context, tx *sql.Tx) context.Context {
	return context.WithValue(ctx, txKey{}, tx)
}

// DbFrom returns the root *sql.DB from ctx.
func DbFrom(ctx context.Context) (*sql.DB, error) {
	db, ok := ctx.Value(dbKey{}).(*sql.DB)
	if !ok || db == nil {
		return nil, fmt.Errorf("database missing from context")
	}
	return db, nil
}

// TxFrom returns the active transaction if present.
func TxFrom(ctx context.Context) (*sql.Tx, bool) {
	tx, ok := ctx.Value(txKey{}).(*sql.Tx)
	return tx, ok && tx != nil
}

// DbTXFrom returns the active *sql.Tx when present, otherwise *sql.DB.
func DbTXFrom(ctx context.Context) (DbTX, error) {
	if tx, ok := TxFrom(ctx); ok {
		return tx, nil
	}
	return DbFrom(ctx)
}

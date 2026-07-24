package tx

import (
	"context"
	"database/sql"
	"fmt"
)

// RunInTx executes fn inside a short independent transaction.
// It does not nest into an outer request/message UoW.
func RunInTx(ctx context.Context, db *sql.DB, fn func(ctx context.Context) error) error {
	if db == nil {
		return fmt.Errorf("database is nil")
	}
	sqlTx, err := db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	committed := false
	defer func() {
		if !committed {
			_ = sqlTx.Rollback()
		}
	}()
	if err := fn(WithTx(WithDB(ctx, db), sqlTx)); err != nil {
		return err
	}
	if err := sqlTx.Commit(); err != nil {
		return fmt.Errorf("commit transaction: %w", err)
	}
	committed = true
	return nil
}

// RunMessageUoW runs a short multi-write segment for worker handlers.
// Do not wrap Docker, long file IO, or external HTTP — use WithDB for those paths
// and call RunMessageUoW / RunInTx only around the DB phase.
func RunMessageUoW(ctx context.Context, db *sql.DB, fn func(ctx context.Context) error) error {
	return RunInTx(ctx, db, fn)
}

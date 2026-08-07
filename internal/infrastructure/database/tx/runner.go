package tx

import (
	"context"
	"database/sql"
)

// TransactionRunner adapts the database UoW to an application transaction port.
type TransactionRunner struct {
	db *sql.DB
}

func NewTransactionRunner(db *sql.DB) TransactionRunner {
	return TransactionRunner{db: db}
}

func (r TransactionRunner) RunInTransaction(ctx context.Context, fn func(context.Context) error) error {
	return RunInTx(ctx, r.db, fn)
}

package tx

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestRunInTxReusesContextTransaction(t *testing.T) {
	db := openRunTestDatabase(t)
	if _, err := db.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	outerTx, err := db.BeginTx(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = outerTx.Rollback() }()

	ctx, cancel := context.WithTimeout(WithTx(context.Background(), outerTx), time.Second)
	defer cancel()
	if err := RunInTx(ctx, db, func(txCtx context.Context) error {
		activeTx, ok := TxFrom(txCtx)
		if !ok || activeTx != outerTx {
			t.Fatal("expected RunInTx to reuse the outer transaction")
		}
		_, err := activeTx.ExecContext(txCtx, `INSERT INTO probe(v) VALUES ('reused')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}
	if err := outerTx.Commit(); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM probe`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one committed row, got %d", count)
	}
}

func TestRunInTxStartsTransactionWithoutContextTransaction(t *testing.T) {
	db := openRunTestDatabase(t)
	if _, err := db.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	if err := RunInTx(context.Background(), db, func(txCtx context.Context) error {
		activeTx, ok := TxFrom(txCtx)
		if !ok {
			t.Fatal("expected RunInTx to create a transaction")
		}
		_, err := activeTx.ExecContext(txCtx, `INSERT INTO probe(v) VALUES ('independent')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM probe`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one committed row, got %d", count)
	}
}

func TestTransactionRunnerRunsInTransaction(t *testing.T) {
	db := openRunTestDatabase(t)
	if _, err := db.Exec(`CREATE TABLE probe (id INTEGER PRIMARY KEY, v TEXT)`); err != nil {
		t.Fatal(err)
	}

	runner := NewTransactionRunner(db)
	if err := runner.RunInTransaction(context.Background(), func(txCtx context.Context) error {
		transaction, ok := TxFrom(txCtx)
		if !ok {
			t.Fatal("expected transaction runner to attach a transaction")
		}
		_, err := transaction.ExecContext(txCtx, `INSERT INTO probe(v) VALUES ('runner')`)
		return err
	}); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM probe`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected one committed row, got %d", count)
	}
}

func openRunTestDatabase(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })
	return db
}

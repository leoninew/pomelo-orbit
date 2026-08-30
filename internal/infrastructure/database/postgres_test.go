package db

import (
	"context"
	"database/sql/driver"
	"testing"
)

func TestRebindPostgresPlaceholders(t *testing.T) {
	backtick := string(rune(96))
	query := "SELECT ?, 'it''s ?', \"identifier?\", " + backtick + "item?" + backtick + ", " + backtick + "it" + backtick + backtick + "em?" + backtick + ", -- ?\n /* ? */ ?"
	want := "SELECT $1, 'it''s ?', \"identifier?\", \"item?\", \"it\"\"em?\", -- ?\n /* ? */ $2"

	if got := rebindPostgresPlaceholders(query); got != want {
		t.Fatalf("rebind query = %q, want %q", got, want)
	}
}

func TestPostgresPlaceholderConnectionRebindsExecutionAndPrepare(t *testing.T) {
	connection := &postgresTestConnection{}
	wrapped := &postgresPlaceholderConn{Conn: connection}

	if _, err := wrapped.ExecContext(context.Background(), "UPDATE item SET name = ? WHERE id = ?", nil); err != nil {
		t.Fatalf("exec context: %v", err)
	}
	if connection.execStatement != "UPDATE item SET name = $1 WHERE id = $2" {
		t.Fatalf("exec statement = %q", connection.execStatement)
	}

	if _, err := wrapped.QueryContext(context.Background(), "SELECT '?' AS value, "+string(rune(96))+"id?"+string(rune(96))+" AS name, ? AS id", nil); err != nil {
		t.Fatalf("query context: %v", err)
	}
	if connection.queryStatement != "SELECT '?' AS value, \"id?\" AS name, $1 AS id" {
		t.Fatalf("query statement = %q", connection.queryStatement)
	}

	if _, err := wrapped.Prepare("SELECT ?"); err != nil {
		t.Fatalf("prepare: %v", err)
	}
	if connection.prepareStatement != "SELECT $1" {
		t.Fatalf("prepare statement = %q", connection.prepareStatement)
	}
}

type postgresTestConnection struct {
	execStatement    string
	queryStatement   string
	prepareStatement string
}

func (c *postgresTestConnection) Prepare(query string) (driver.Stmt, error) {
	c.prepareStatement = query
	return nil, nil
}

func (*postgresTestConnection) Close() error {
	return nil
}

func (*postgresTestConnection) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (c *postgresTestConnection) ExecContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Result, error) {
	c.execStatement = query
	return driver.RowsAffected(1), nil
}

func (c *postgresTestConnection) QueryContext(_ context.Context, query string, _ []driver.NamedValue) (driver.Rows, error) {
	c.queryStatement = query
	return nil, nil
}

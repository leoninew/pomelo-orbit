package db

import (
	"context"
	"database/sql/driver"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestOpenSQLiteConfiguresPragmas(t *testing.T) {
	database, err := openSQLite(config.SQLiteConfig{Path: filepath.Join(t.TempDir(), "pomelo.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	var busyTimeout int
	if err := database.QueryRow("PRAGMA busy_timeout").Scan(&busyTimeout); err != nil {
		t.Fatal(err)
	}
	if busyTimeout != 5000 {
		t.Fatalf("expected busy_timeout 5000, got %d", busyTimeout)
	}

	var journalMode string
	if err := database.QueryRow("PRAGMA journal_mode").Scan(&journalMode); err != nil {
		t.Fatal(err)
	}
	if strings.ToLower(journalMode) != "wal" {
		t.Fatalf("expected journal_mode wal, got %q", journalMode)
	}

	var foreignKeys int
	if err := database.QueryRow("PRAGMA foreign_keys").Scan(&foreignKeys); err != nil {
		t.Fatal(err)
	}
	if foreignKeys != 1 {
		t.Fatalf("expected foreign_keys on, got %d", foreignKeys)
	}
}

func TestMySQLModeConnectorEnablesANSIQuotes(t *testing.T) {
	connection := &mysqlModeTestConnection{}
	connector := mysqlModeConnector{Connector: mysqlModeTestConnector{connection: connection}}

	if _, err := connector.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(connection.statement, "ANSI_QUOTES") {
		t.Fatalf("expected ANSI_QUOTES session setup, got %q", connection.statement)
	}
}

type mysqlModeTestConnector struct {
	connection driver.Conn
}

func (c mysqlModeTestConnector) Connect(context.Context) (driver.Conn, error) {
	return c.connection, nil
}

func (mysqlModeTestConnector) Driver() driver.Driver {
	return nil
}

type mysqlModeTestConnection struct {
	statement string
}

func (c *mysqlModeTestConnection) Prepare(string) (driver.Stmt, error) {
	return nil, driver.ErrSkip
}

func (c *mysqlModeTestConnection) Close() error {
	return nil
}

func (c *mysqlModeTestConnection) Begin() (driver.Tx, error) {
	return nil, driver.ErrSkip
}

func (c *mysqlModeTestConnection) ExecContext(_ context.Context, statement string, _ []driver.NamedValue) (driver.Result, error) {
	c.statement = statement
	return driver.RowsAffected(0), nil
}

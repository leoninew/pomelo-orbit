package db

import (
	"context"
	"database/sql/driver"
	"path/filepath"
	"strings"
	"testing"

	gomysql "github.com/go-sql-driver/mysql"
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

func TestMySQLModeConnectorConfiguresSession(t *testing.T) {
	connection := &mysqlModeTestConnection{}
	connector := mysqlModeConnector{Connector: mysqlModeTestConnector{connection: connection}}

	if _, err := connector.Connect(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(connection.statements) != 2 || !strings.Contains(connection.statements[0], "ANSI_QUOTES") || !strings.Contains(connection.statements[1], "READ COMMITTED") {
		t.Fatalf("unexpected MySQL session setup: %q", connection.statements)
	}
}

func TestMySQLDsnCountsMatchedRows(t *testing.T) {
	dsn, err := mysqlDsn("user:pass@tcp(localhost:3306)/orbit")
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := gomysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ClientFoundRows {
		t.Fatal("expected matched-row semantics for pipeline locking")
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
	statements []string
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
	c.statements = append(c.statements, statement)
	return driver.RowsAffected(0), nil
}

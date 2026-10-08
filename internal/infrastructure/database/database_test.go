package db

import (
	"context"
	"database/sql/driver"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestOpenSQLiteConfiguresPragmas(t *testing.T) {
	database, err := Open(config.DatabaseConfig{Url: "sqlite:///" + filepath.ToSlash(filepath.Join(t.TempDir(), "pomelo.db"))})
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
	parsed, err := url.Parse("mysql://user:p%40ss%3Aword@localhost/orbit?parseTime=true&loc=Asia%2FShanghai&timeout=5s")
	if err != nil {
		t.Fatal(err)
	}
	dsn, err := mysqlDsn(parsed)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := gomysql.ParseDSN(dsn)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.ClientFoundRows || !cfg.MultiStatements {
		t.Fatal("expected matched-row semantics and multi-statement migrations")
	}
	if cfg.User != "user" || cfg.Passwd != "p@ss:word" || cfg.Addr != "localhost:3306" || cfg.DBName != "orbit" {
		t.Fatal("MySQL URL connection fields were not preserved")
	}
	if !cfg.ParseTime || cfg.Loc.String() != "Asia/Shanghai" || cfg.Timeout.String() != "5s" {
		t.Fatal("MySQL URL options were not preserved")
	}
}

func TestOpenSQLiteUrlPaths(t *testing.T) {
	t.Chdir(t.TempDir())
	absolutePath := filepath.Join(t.TempDir(), "orbit.db")
	for _, tc := range []struct {
		name string
		url  string
		path string
	}{
		{name: "relative", url: "sqlite:///nested/orbit.db", path: "nested/orbit.db"},
		{name: "absolute", url: "sqlite:///" + filepath.ToSlash(absolutePath), path: absolutePath},
		{name: "escaped path", url: "sqlite:///nested/orbit%20%23test.db", path: "nested/orbit #test.db"},
		{name: "memory", url: "sqlite:///:memory:", path: ""},
	} {
		t.Run(tc.name, func(t *testing.T) {
			database, err := Open(config.DatabaseConfig{Url: tc.url})
			if err != nil {
				t.Fatal(err)
			}
			defer func() { _ = database.Close() }()
			if _, err := database.Exec("CREATE TABLE url_test (id INTEGER PRIMARY KEY)"); err != nil {
				t.Fatal(err)
			}
			if tc.path != "" {
				if _, err := os.Stat(tc.path); err != nil {
					t.Fatalf("SQLite database missing at URL path: %v", err)
				}
			}
		})
	}
}

func TestOpenSQLiteUrlPreservesQuery(t *testing.T) {
	databaseUrl := "sqlite:///" + filepath.ToSlash(filepath.Join(t.TempDir(), "orbit%20%23test.db"))
	database, err := Open(config.DatabaseConfig{Url: databaseUrl})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("CREATE TABLE url_test (id INTEGER PRIMARY KEY)"); err != nil {
		t.Fatal(err)
	}
	if err := database.Close(); err != nil {
		t.Fatal(err)
	}
	readOnly, err := Open(config.DatabaseConfig{Url: databaseUrl + "?mode=ro"})
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = readOnly.Close() }()
	if _, err := readOnly.Exec("INSERT INTO url_test (id) VALUES (1)"); err == nil {
		t.Fatal("SQLite URL mode=ro did not prevent writes")
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

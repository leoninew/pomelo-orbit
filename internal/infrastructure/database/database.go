package db

import (
	"context"
	"database/sql"
	"database/sql/driver"
	"fmt"
	"os"
	"path/filepath"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/lib/pq"
	_ "modernc.org/sqlite"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func Open(cfg config.DatabaseConfig) (*sql.DB, error) {
	switch cfg.Driver {
	case config.DatabaseDriverSQLite:
		return openSQLite(cfg.SQLite)
	case config.DatabaseDriverMySQL:
		return openMySQL(cfg.MySQL)
	case config.DatabaseDriverPostgres:
		return openPostgres(cfg.Postgres)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
}

func openSQLite(cfg config.SQLiteConfig) (*sql.DB, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}
	database, err := sql.Open("sqlite", cfg.Path)
	if err != nil {
		return nil, fmt.Errorf("open sqlite database: %w", err)
	}
	database.SetMaxOpenConns(1)
	if err := configureSQLite(database); err != nil {
		_ = database.Close()
		return nil, err
	}
	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping sqlite database: %w", err)
	}
	return database, nil
}

func configureSQLite(database *sql.DB) error {
	pragmas := []string{
		"PRAGMA busy_timeout = 5000",
		"PRAGMA journal_mode = WAL",
		"PRAGMA foreign_keys = ON",
	}
	for _, pragma := range pragmas {
		if _, err := database.Exec(pragma); err != nil {
			return fmt.Errorf("configure sqlite %s: %w", pragma, err)
		}
	}
	return nil
}

func openMySQL(cfg config.MySQLConfig) (*sql.DB, error) {
	dsn, err := mysqlDsn(cfg.Dsn)
	if err != nil {
		return nil, err
	}
	mysqlConfig, err := gomysql.ParseDSN(dsn)
	if err != nil {
		return nil, fmt.Errorf("parse mysql dsn: %w", err)
	}
	connector, err := gomysql.NewConnector(mysqlConfig)
	if err != nil {
		return nil, fmt.Errorf("create mysql connector: %w", err)
	}
	database := sql.OpenDB(mysqlModeConnector{Connector: connector})
	database.SetMaxOpenConns(20)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(30 * time.Minute)
	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping mysql database: %w", err)
	}
	return database, nil
}

func openPostgres(cfg config.PostgresConfig) (*sql.DB, error) {
	connector, err := pq.NewConnector(cfg.Dsn)
	if err != nil {
		return nil, fmt.Errorf("parse postgres dsn: %w", err)
	}
	database := sql.OpenDB(postgresPlaceholderConnector{Connector: connector})
	database.SetMaxOpenConns(20)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(30 * time.Minute)
	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping postgres database: %w", err)
	}
	return database, nil
}

type mysqlModeConnector struct {
	driver.Connector
}

func (c mysqlModeConnector) Connect(ctx context.Context) (driver.Conn, error) {
	connection, err := c.Connector.Connect(ctx)
	if err != nil {
		return nil, err
	}

	executor, ok := connection.(driver.ExecerContext)
	if !ok {
		_ = connection.Close()
		return nil, fmt.Errorf("mysql driver connection does not support session configuration")
	}
	if _, err := executor.ExecContext(ctx, "SET SESSION sql_mode = IF(FIND_IN_SET('ANSI_QUOTES', @@SESSION.sql_mode), @@SESSION.sql_mode, CONCAT_WS(',', @@SESSION.sql_mode, 'ANSI_QUOTES'))", nil); err != nil {
		_ = connection.Close()
		return nil, fmt.Errorf("enable mysql ANSI_QUOTES mode: %w", err)
	}
	return connection, nil
}

func mysqlDsn(dsn string) (string, error) {
	cfg, err := gomysql.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("parse mysql dsn: %w", err)
	}
	cfg.MultiStatements = true
	return cfg.FormatDSN(), nil
}

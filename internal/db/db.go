package db

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	gomysql "github.com/go-sql-driver/mysql"
	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	"gitee.com/leoninew/pomelo-orbit/internal/config"
)

func Open(cfg config.DatabaseConfig) (*sqlx.DB, error) {
	switch cfg.Driver {
	case config.DatabaseDriverSQLite:
		return openSQLite(cfg.SQLite)
	case config.DatabaseDriverMySQL:
		return openMySQL(cfg.MySQL)
	default:
		return nil, fmt.Errorf("unsupported database driver: %s", cfg.Driver)
	}
}

func openSQLite(cfg config.SQLiteConfig) (*sqlx.DB, error) {
	if err := os.MkdirAll(filepath.Dir(cfg.Path), 0o755); err != nil {
		return nil, fmt.Errorf("create sqlite directory: %w", err)
	}
	database, err := sqlx.Open("sqlite", cfg.Path)
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

func configureSQLite(database *sqlx.DB) error {
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

func openMySQL(cfg config.MySQLConfig) (*sqlx.DB, error) {
	dsn, err := mysqlDSN(cfg.DSN)
	if err != nil {
		return nil, err
	}
	database, err := sqlx.Open("mysql", dsn)
	if err != nil {
		return nil, fmt.Errorf("open mysql database: %w", err)
	}
	database.SetMaxOpenConns(20)
	database.SetMaxIdleConns(5)
	database.SetConnMaxLifetime(30 * time.Minute)
	if err := database.Ping(); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping mysql database: %w", err)
	}
	return database, nil
}

func mysqlDSN(dsn string) (string, error) {
	cfg, err := gomysql.ParseDSN(dsn)
	if err != nil {
		return "", fmt.Errorf("parse mysql dsn: %w", err)
	}
	cfg.MultiStatements = true
	return cfg.FormatDSN(), nil
}

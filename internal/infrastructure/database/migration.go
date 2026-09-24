package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"log/slog"
	"path"
	"strconv"
	"strings"

	gomigrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	migratepostgres "github.com/golang-migrate/migrate/v4/database/postgres"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"github.com/leoninew/pomelo-orbit/internal/config"
	migrationfiles "github.com/leoninew/pomelo-orbit/sql"
)

const migrationsRoot = "migration"

type MigrationVersion struct {
	Version uint
	Dirty   bool
}

func MigrateUp(sqlDb *sql.DB, driver string) error {
	return MigrateUpWithLogger(sqlDb, driver, nil)
}

func MigrateUpWithLogger(sqlDb *sql.DB, driver string, logger *slog.Logger) error {
	runner, err := newMigrationRunner(sqlDb, driver, logger)
	if err != nil {
		return err
	}
	defer func() { _, _ = runner.Close() }()

	if err := runner.Up(); err != nil && !errors.Is(err, gomigrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func MigrateTo(sqlDb *sql.DB, driver string, version uint) error {
	runner, err := newMigrationRunner(sqlDb, driver, nil)
	if err != nil {
		return err
	}
	defer func() { _, _ = runner.Close() }()

	if err := runner.Migrate(version); err != nil && !errors.Is(err, gomigrate.ErrNoChange) {
		return fmt.Errorf("migrate to version %d: %w", version, err)
	}
	return nil
}

func ReadMigrationVersion(sqlDb *sql.DB, driver string) (MigrationVersion, error) {
	databaseDriver, err := migrationDatabaseDriver(sqlDb, driver)
	if err != nil {
		return MigrationVersion{}, err
	}
	defer func() { _ = databaseDriver.Close() }()

	version, dirty, err := databaseDriver.Version()
	if err != nil {
		return MigrationVersion{}, fmt.Errorf("read migration version: %w", err)
	}
	if version < 0 {
		return MigrationVersion{}, nil
	}
	return MigrationVersion{Version: uint(version), Dirty: dirty}, nil
}

func newMigrationRunner(sqlDb *sql.DB, driver string, logger *slog.Logger) (*gomigrate.Migrate, error) {
	migrationSource, err := iofs.New(migrationfiles.Files, path.Join(migrationsRoot, driver))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, fmt.Errorf("unsupported migration source driver: %s", driver)
		}
		return nil, fmt.Errorf("open %s migrations: %w", driver, err)
	}

	databaseDriver, err := migrationDatabaseDriver(sqlDb, driver)
	if err != nil {
		_ = migrationSource.Close()
		return nil, err
	}

	runner, err := gomigrate.NewWithInstance(driver, migrationSource, driver, databaseDriver)
	if err != nil {
		_ = migrationSource.Close()
		_ = databaseDriver.Close()
		return nil, fmt.Errorf("create migration runner: %w", err)
	}
	if logger != nil {
		runner.Log = migrationLogger{logger: logger}
	}
	return runner, nil
}

type migrationLogger struct {
	logger *slog.Logger
}

func (l migrationLogger) Printf(format string, args ...interface{}) {
	message := strings.TrimSpace(fmt.Sprintf(format, args...))
	version, direction, identifier, duration, ok := parseMigrationLog(message)
	if !ok {
		l.logger.Debug("schema migration event", "message", message)
		return
	}
	l.logger.Info("schema migration applied",
		"version", version,
		"direction", direction,
		"identifier", identifier,
		"duration", duration,
	)
}

func (migrationLogger) Verbose() bool {
	return false
}

func parseMigrationLog(message string) (uint64, string, string, string, bool) {
	closeParenthesis := strings.LastIndex(message, " (")
	if closeParenthesis <= 0 || !strings.HasSuffix(message, ")") {
		return 0, "", "", "", false
	}

	prefix := message[:closeParenthesis]
	duration := message[closeParenthesis+2 : len(message)-1]
	parts := strings.SplitN(prefix, " ", 2)
	if len(parts) != 2 {
		return 0, "", "", "", false
	}
	versionAndDirection := strings.SplitN(parts[0], "/", 2)
	if len(versionAndDirection) != 2 {
		return 0, "", "", "", false
	}
	version, err := strconv.ParseUint(versionAndDirection[0], 10, 64)
	if err != nil {
		return 0, "", "", "", false
	}

	direction := ""
	switch versionAndDirection[1] {
	case "u":
		direction = "up"
	case "d":
		direction = "down"
	default:
		return 0, "", "", "", false
	}
	return version, direction, parts[1], duration, true
}

func migrationDatabaseDriver(sqlDb *sql.DB, driver string) (database.Driver, error) {
	switch driver {
	case config.DatabaseDriverSQLite:
		// SQLite table-rebuild migrations temporarily disable foreign keys.
		// PRAGMA foreign_keys is a connection setting and cannot change inside
		// the transaction wrapper used by golang-migrate.
		driver, err := migratesqlite.WithInstance(sqlDb, &migratesqlite.Config{NoTxWrap: true})
		if err != nil {
			return nil, err
		}
		return noCloseDatabaseDriver{Driver: driver}, nil
	case config.DatabaseDriverMySQL:
		connection, err := sqlDb.Conn(context.Background())
		if err != nil {
			return nil, err
		}
		driver, err := migratemysql.WithConnection(context.Background(), connection, &migratemysql.Config{})
		if err != nil {
			_ = connection.Close()
			return nil, err
		}
		return driver, nil
	case config.DatabaseDriverPostgres:
		connection, err := sqlDb.Conn(context.Background())
		if err != nil {
			return nil, err
		}
		driver, err := migratepostgres.WithConnection(context.Background(), connection, &migratepostgres.Config{})
		if err != nil {
			_ = connection.Close()
			return nil, err
		}
		return driver, nil
	default:
		return nil, fmt.Errorf("unsupported migration database driver: %s", driver)
	}
}

type noCloseDatabaseDriver struct {
	database.Driver
}

func (d noCloseDatabaseDriver) Close() error {
	return nil
}

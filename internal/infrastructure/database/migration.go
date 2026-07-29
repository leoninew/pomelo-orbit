package db

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"io/fs"
	"path"

	gomigrate "github.com/golang-migrate/migrate/v4"
	"github.com/golang-migrate/migrate/v4/database"
	migratemysql "github.com/golang-migrate/migrate/v4/database/mysql"
	migratesqlite "github.com/golang-migrate/migrate/v4/database/sqlite"
	"github.com/golang-migrate/migrate/v4/source/iofs"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	migrationfiles "gitee.com/leoninew/PomeloOrbit-go/sql"
)

const migrationsRoot = "migration"

type MigrationVersion struct {
	Version uint
	Dirty   bool
}

func MigrateUp(sqlDb *sql.DB, driver string) error {
	runner, err := newMigrationRunner(sqlDb, driver)
	if err != nil {
		return err
	}
	defer func() { _, _ = runner.Close() }()

	if err := runner.Up(); err != nil && !errors.Is(err, gomigrate.ErrNoChange) {
		return fmt.Errorf("run migrations: %w", err)
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

func newMigrationRunner(sqlDb *sql.DB, driver string) (*gomigrate.Migrate, error) {
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
	return runner, nil
}

func migrationDatabaseDriver(sqlDb *sql.DB, driver string) (database.Driver, error) {
	switch driver {
	case config.DatabaseDriverSQLite:
		driver, err := migratesqlite.WithInstance(sqlDb, &migratesqlite.Config{})
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

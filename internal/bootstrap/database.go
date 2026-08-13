package bootstrap

import (
	"database/sql"
	"fmt"
	"log/slog"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
)

func OpenDatabase(cfg config.Config) (*sql.DB, error) {
	return db.Open(cfg.Database)
}

func RunMigrations(database *sql.DB, driver string, logger *slog.Logger) error {
	before, err := MigrationVersion(database, driver)
	if err != nil {
		return fmt.Errorf("read schema migration version before migration: %w", err)
	}
	logger.Info("run schema migrations", "driver", driver, "from_version", before.Version, "dirty", before.Dirty)
	if err := db.MigrateUp(database, driver); err != nil {
		return err
	}
	after, err := MigrationVersion(database, driver)
	if err != nil {
		return fmt.Errorf("read schema migration version after migration: %w", err)
	}
	logger.Info("schema migrations complete", "driver", driver,
		"from_version", before.Version,
		"to_version", after.Version,
		"applied", after.Version != before.Version,
		"dirty", after.Dirty,
	)
	return db.MigrateData(database, driver, logger)
}

func MigrationVersion(database *sql.DB, driver string) (db.MigrationVersion, error) {
	return db.ReadMigrationVersion(database, driver)
}

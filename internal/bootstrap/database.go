package bootstrap

import (
	"database/sql"
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
)

func OpenDatabase(cfg config.Config) (*sql.DB, error) {
	return db.Open(cfg.Database)
}

func RunMigrations(database *sql.DB, driver string, logger *slog.Logger) error {
	logger.Info("run schema migrations", "driver", driver)
	if err := db.MigrateUp(database, driver); err != nil {
		return err
	}
	logger.Info("schema migrations complete", "driver", driver)
	return db.MigrateData(database, driver, logger)
}

func MigrationVersion(database *sql.DB, driver string) (db.MigrationVersion, error) {
	return db.ReadMigrationVersion(database, driver)
}

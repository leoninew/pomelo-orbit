package bootstrap

import (
	"database/sql"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
)

func OpenDatabase(cfg config.Config) (*sql.DB, error) {
	return db.Open(cfg.Database)
}

func RunMigrations(database *sql.DB, driver string) error {
	return db.MigrateUp(database, driver)
}

func MigrationVersion(database *sql.DB, driver string) (db.MigrationVersion, error) {
	return db.ReadMigrationVersion(database, driver)
}

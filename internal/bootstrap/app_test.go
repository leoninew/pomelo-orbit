package bootstrap

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestMigrateAppliesSchemaAndSeedData(t *testing.T) {
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: config.DatabaseDriverSQLite, SQLite: config.SQLiteConfig{Path: filepath.Join(t.TempDir(), "pomelo-orbit.db")}},
	}
	app := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := app.Migrate(); err != nil {
		t.Fatal(err)
	}
	database, err := OpenDatabase(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	var adminCount int
	if err := database.QueryRow("SELECT COUNT(*) FROM user WHERE username = 'admin'").Scan(&adminCount); err != nil {
		t.Fatal(err)
	}
	if adminCount != 1 {
		t.Fatalf("admin count=%d, want 1", adminCount)
	}
	version, err := app.MigrationVersion()
	if err != nil {
		t.Fatal(err)
	}
	if version.Version == 0 || version.Dirty {
		t.Fatalf("unexpected migration version: %+v", version)
	}
}

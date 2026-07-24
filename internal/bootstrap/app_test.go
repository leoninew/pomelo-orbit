package bootstrap

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

func TestMigrateAndMigrationVersion(t *testing.T) {
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: config.DatabaseDriverSQLite, SQLite: config.SQLiteConfig{Path: filepath.Join(t.TempDir(), "pomelo-orbit.db")}},
		Worker:   config.WorkerConfig{MaxAttempts: 3},
	}
	app := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := app.Migrate(); err != nil {
		t.Fatal(err)
	}
	version, err := app.MigrationVersion()
	if err != nil {
		t.Fatal(err)
	}
	if version.Version != 30 || version.Dirty {
		t.Fatalf("unexpected migration version: %+v", version)
	}
}

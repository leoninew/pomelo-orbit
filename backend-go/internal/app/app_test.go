package app

import (
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	"backend/internal/config"
)

func TestMigrateAndMigrationStatus(t *testing.T) {
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: config.DatabaseDriverSQLite, SQLite: config.SQLiteConfig{Path: filepath.Join(t.TempDir(), "backend-go.db")}},
		Worker:   config.WorkerConfig{MaxAttempts: 3},
	}
	app := New(cfg, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err := app.Migrate(); err != nil {
		t.Fatal(err)
	}
	statuses, err := app.MigrationStatus()
	if err != nil {
		t.Fatal(err)
	}
	if len(statuses) == 0 {
		t.Fatal("expected migration statuses")
	}
	for _, status := range statuses {
		if !status.Applied {
			t.Fatalf("expected migration %s applied", status.Filename)
		}
	}
}

package bootstrap

import (
	"bytes"
	"io/fs"
	"log/slog"
	"path"
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
	migrationfiles "github.com/leoninew/pomelo-orbit/sql"
)

func TestRunMigrationsLogsEachMigrationWithoutSQL(t *testing.T) {
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: config.DatabaseDriverSQLite, SQLite: config.SQLiteConfig{Path: path.Join(t.TempDir(), "pomelo-orbit.db")}},
	}
	database, err := OpenDatabase(cfg)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	var logs bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logs, nil))
	if err := RunMigrations(database, config.DatabaseDriverSQLite, logger); err != nil {
		t.Fatal(err)
	}

	entries, err := fs.ReadDir(migrationfiles.Files, "migration/sqlite")
	if err != nil {
		t.Fatal(err)
	}
	expected := 0
	for _, entry := range entries {
		if strings.HasSuffix(entry.Name(), ".up.sql") {
			expected++
		}
	}
	output := logs.String()
	if applied := strings.Count(output, "msg=\"schema migration applied\""); applied != expected {
		t.Fatalf("migration log count = %d, want %d; logs:\n%s", applied, expected, output)
	}
	if strings.Contains(output, "CREATE TABLE") || strings.Contains(output, "INSERT INTO") {
		t.Fatalf("migration SQL leaked into logs:\n%s", output)
	}
}

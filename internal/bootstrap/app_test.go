package bootstrap

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestMigrateAndMigrationVersion(t *testing.T) {
	cfg := config.Config{
		Database: config.DatabaseConfig{Driver: config.DatabaseDriverSQLite, SQLite: config.SQLiteConfig{Path: filepath.Join(t.TempDir(), "pomelo-orbit.db")}},
	}
	var logs bytes.Buffer
	app := New(cfg, slog.New(slog.NewTextHandler(&logs, nil)))
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
	if _, err := database.Exec("SELECT source_is_host_path FROM service_component_mount LIMIT 0"); err != nil {
		t.Fatalf("service component mount source path mode column: %v", err)
	}
	for _, want := range []string{
		`msg="run schema migrations" driver=sqlite from_version=0 dirty=false`,
		`msg="schema migrations complete" driver=sqlite from_version=0 to_version=32 applied=true dirty=false`,
		`msg="execute data migration" path=migration/data/sqlite/000029_seed_system.up.sql`,
	} {
		if !strings.Contains(logs.String(), want) {
			t.Fatalf("expected log entry %q in:\n%s", want, logs.String())
		}
	}
	if strings.Contains(logs.String(), "000029_seed_system.down.sql") {
		t.Fatalf("down migration must be filtered from startup logs:\n%s", logs.String())
	}
	version, err := app.MigrationVersion()
	if err != nil {
		t.Fatal(err)
	}
	if version.Version != 32 || version.Dirty {
		t.Fatalf("unexpected migration version: %+v", version)
	}
}

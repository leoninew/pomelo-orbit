package bootstrap

import (
	"bytes"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
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
	for _, want := range []string{
		`msg="run schema migrations" driver=sqlite`,
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
	if version.Version != 30 || version.Dirty {
		t.Fatalf("unexpected migration version: %+v", version)
	}
}

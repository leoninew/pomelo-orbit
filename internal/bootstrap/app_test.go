package bootstrap

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
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
	if err := database.QueryRow("SELECT COUNT(*) FROM user WHERE username = 'admin' AND email = 'admin@lvh.me'").Scan(&adminCount); err != nil {
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

func TestValidateContainerWorkspaceMountsIncludesConfigurationKey(t *testing.T) {
	cfg := config.Config{Workspace: config.WorkspaceConfig{Pipeline: "/app/data/pipeline"}, Logging: config.LoggingConfig{DeploymentRoot: "/app/data/deployment-logs"}}
	var resolved []string
	err := validateContainerWorkspaceMounts(context.Background(), cfg, true, func(_ context.Context, path string) (string, error) {
		resolved = append(resolved, path)
		if path == cfg.Logging.DeploymentRoot {
			return "", errors.New("not mounted")
		}
		return "/srv/orbit/ci", nil
	})
	if err == nil || !strings.Contains(err.Error(), "logging.deployment_root must be bind mounted when Orbit runs in a container") {
		t.Fatalf("unexpected validation error: %v", err)
	}
	if len(resolved) != 2 || resolved[0] != cfg.Workspace.Pipeline || resolved[1] != cfg.Logging.DeploymentRoot {
		t.Fatalf("resolved paths = %#v", resolved)
	}
}

func TestValidateContainerWorkspaceMountsSkipsNativeOrbit(t *testing.T) {
	cfg := config.Config{Workspace: config.WorkspaceConfig{Pipeline: "relative-ci"}, Logging: config.LoggingConfig{DeploymentRoot: "relative-cd"}}
	resolverCalls := 0
	err := validateContainerWorkspaceMounts(context.Background(), cfg, false, func(context.Context, string) (string, error) {
		resolverCalls++
		return "", errors.New("resolver must not run")
	})
	if err != nil {
		t.Fatalf("native Orbit validation returned error: %v", err)
	}
	if resolverCalls != 0 {
		t.Fatalf("native Orbit invoked Docker workspace resolver %d times", resolverCalls)
	}
}

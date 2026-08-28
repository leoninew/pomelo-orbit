package app

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
)

const (
	gatewayApplicationID = "01M10RRA8F863EJ2N9TC3Z2EC1"
)

func TestSQLiteMigrationE2E(t *testing.T) {
	cfg := loadE2EConfig(t, []byte(fmt.Sprintf(`database:
  driver: sqlite
  sqlite:
    path: %q
`, filepath.Join(t.TempDir(), "pomelo-orbit.db"))))
	if cfg.Database.Driver != config.DatabaseDriverSQLite {
		t.Fatalf("expected sqlite config, got %s", cfg.Database.Driver)
	}

	runMigrationE2E(t, cfg)
}

func TestMySQLMigrationE2E(t *testing.T) {
	configPath := os.Getenv("BACKEND_GO_E2E_CONFIG")
	if configPath == "" {
		t.Skip("set BACKEND_GO_E2E_CONFIG to a MySQL config file to run MySQL e2e")
	}
	configPath, err := filepath.Abs(configPath)
	if err != nil {
		t.Fatal(err)
	}
	envConfig, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	cfg := loadE2EConfig(t, envConfig)
	if cfg.Database.Driver != config.DatabaseDriverMySQL {
		t.Fatalf("expected mysql config, got %s", cfg.Database.Driver)
	}

	runMigrationE2E(t, cfg)
}

func loadE2EConfig(t *testing.T, environmentConfig []byte) config.Config {
	t.Helper()

	defaultConfig, err := os.ReadFile(filepath.Join(repositoryRoot(t), config.DefaultConfigFile))
	if err != nil {
		t.Fatal(err)
	}
	configDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(configDir, filepath.Dir(config.DefaultConfigFile)), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(configDir, config.DefaultConfigFile), defaultConfig, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POMELO_ORBIT_APP__ENV", "e2e")
	t.Setenv("POMELO_ORBIT_JWT__SECRET_KEY", "e2e-test-secret-key-must-be-at-least-32-bytes")
	if err := os.WriteFile(filepath.Join(configDir, config.EnvConfigFile("e2e")), environmentConfig, 0o644); err != nil {
		t.Fatal(err)
	}
	originalDir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(configDir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(originalDir); err != nil {
			t.Errorf("restore working directory: %v", err)
		}
	})

	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	return cfg
}

func runMigrationE2E(t *testing.T, cfg config.Config) {
	t.Helper()
	database, err := db.Open(cfg.Database)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()

	if err := db.MigrateUp(database, cfg.Database.Driver); err != nil {
		t.Fatal(err)
	}
	if err := db.MigrateUp(database, cfg.Database.Driver); err != nil {
		t.Fatal(err)
	}
	version, err := db.ReadMigrationVersion(database, cfg.Database.Driver)
	if err != nil {
		t.Fatal(err)
	}
	if version.Dirty {
		t.Fatalf("migration is dirty: %+v", version)
	}

	var gatewayApplicationCount int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM application WHERE id = ? AND code = ?",
		gatewayApplicationID,
		"traefik",
	).Scan(&gatewayApplicationCount); err != nil {
		t.Fatal(err)
	}
	if gatewayApplicationCount != 1 {
		t.Fatalf("expected current gateway application, got %d", gatewayApplicationCount)
	}

	var profileVersionCount int
	if err := database.QueryRow(
		"SELECT COUNT(*) FROM gateway_acme_profile_version WHERE application_id = ?",
		gatewayApplicationID,
	).Scan(&profileVersionCount); err != nil {
		t.Fatal(err)
	}
	if profileVersionCount != 4 {
		t.Fatalf("expected four gateway profile versions, got %d", profileVersionCount)
	}
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("locate e2e test file")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", "..", ".."))
}

package config

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestLoadDefaultConfigFile(t *testing.T) {
	setupDefaultConfig(t)

	cfg, err := Load("")
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Database.Driver != DatabaseDriverSQLite {
		t.Fatalf("unexpected database driver: %s", cfg.Database.Driver)
	}
	if cfg.App.Name != "Pomelo Orbit Backend Go" {
		t.Fatalf("unexpected app name: %s", cfg.App.Name)
	}
	if cfg.Server.Port != 8080 {
		t.Fatalf("unexpected server port: %d", cfg.Server.Port)
	}
	if cfg.Logging.File != "logs/backend-go.log" {
		t.Fatalf("unexpected logging file: %s", cfg.Logging.File)
	}
	if cfg.Logging.MaxSizeMB != 100 {
		t.Fatalf("unexpected logging max size: %d", cfg.Logging.MaxSizeMB)
	}
	if cfg.Logging.MaxBackups != 7 {
		t.Fatalf("unexpected logging max backups: %d", cfg.Logging.MaxBackups)
	}
	if !cfg.Turnstile.Enabled {
		t.Fatal("expected turnstile enabled")
	}
	if cfg.Turnstile.SiteKey != "1x00000000000000000000AA" {
		t.Fatalf("unexpected turnstile site key: %s", cfg.Turnstile.SiteKey)
	}
	if cfg.Turnstile.SecretKey != "1x0000000000000000000000000000000AA" {
		t.Fatalf("unexpected turnstile secret key: %s", cfg.Turnstile.SecretKey)
	}
	if cfg.Turnstile.VerifyURL != "https://challenges.cloudflare.com/turnstile/v0/siteverify" {
		t.Fatalf("unexpected turnstile verify url: %s", cfg.Turnstile.VerifyURL)
	}
	if cfg.Worker.PollInterval != time.Second {
		t.Fatalf("unexpected poll interval: %s", cfg.Worker.PollInterval)
	}
}

func TestLoadConfigMergesCustomConfig(t *testing.T) {
	setupDefaultConfig(t)
	path := writeConfig(t, `logging:
  level: "DEBUG"
  max_size_mb: 50
  max_backups: 3
database:
  sqlite:
    path: "data/test.db"
orbit:
  root: "../.."
worker:
  id: "worker-1"
  max_attempts: 4
  concurrency: 3
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Logging.Level != "DEBUG" {
		t.Fatalf("unexpected logging level: %s", cfg.Logging.Level)
	}
	if cfg.Logging.MaxSizeMB != 50 {
		t.Fatalf("unexpected logging max size: %d", cfg.Logging.MaxSizeMB)
	}
	if cfg.Logging.MaxBackups != 3 {
		t.Fatalf("unexpected logging max backups: %d", cfg.Logging.MaxBackups)
	}
	if cfg.Worker.Id != "worker-1" {
		t.Fatalf("unexpected worker id: %s", cfg.Worker.Id)
	}
	if cfg.Worker.Concurrency != 3 {
		t.Fatalf("unexpected concurrency: %d", cfg.Worker.Concurrency)
	}
	if cfg.Worker.PollInterval != time.Second {
		t.Fatalf("unexpected poll interval from defaults: %s", cfg.Worker.PollInterval)
	}
	if cfg.Worker.LeaseDuration != 5*time.Minute {
		t.Fatalf("unexpected lease duration from defaults: %s", cfg.Worker.LeaseDuration)
	}
}

func TestLoadConfigEnvOverrides(t *testing.T) {
	setupDefaultConfig(t)
	path := writeConfig(t, `server:
  host: "127.0.0.1"
  port: 9000
database:
  sqlite:
    path: "data/test.db"
worker:
  concurrency: 1
`)
	t.Setenv("POMELO_ORBIT_BACKEND__SERVER__HOST", "0.0.0.0")
	t.Setenv("POMELO_ORBIT_BACKEND__SERVER__PORT", "8088")
	t.Setenv("POMELO_ORBIT_BACKEND__DATABASE__SQLITE__PATH", "/data/pomelo-orbit.db")
	t.Setenv("POMELO_ORBIT_BACKEND__LOGGING__MAX_SIZE_MB", "25")
	t.Setenv("POMELO_ORBIT_BACKEND__LOGGING__MAX_BACKUPS", "4")
	t.Setenv("POMELO_ORBIT_BACKEND__TURNSTILE__ENABLED", "false")
	t.Setenv("POMELO_ORBIT_BACKEND__TURNSTILE__SITE_KEY", "site-from-env")
	t.Setenv("POMELO_ORBIT_BACKEND__TURNSTILE__SECRET_KEY", "secret-from-env")
	t.Setenv("POMELO_ORBIT_BACKEND__TURNSTILE__VERIFY_URL", "https://turnstile.example.test")
	t.Setenv("POMELO_ORBIT_BACKEND__WORKER__CONCURRENCY", "4")
	t.Setenv("POMELO_ORBIT_BACKEND__WORKER__POLL_INTERVAL", "2s")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("unexpected server host: %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 8088 {
		t.Fatalf("unexpected server port: %d", cfg.Server.Port)
	}
	if cfg.Database.SQLite.Path != "/data/pomelo-orbit.db" {
		t.Fatalf("unexpected sqlite path: %s", cfg.Database.SQLite.Path)
	}
	if cfg.Logging.MaxSizeMB != 25 {
		t.Fatalf("unexpected logging max size: %d", cfg.Logging.MaxSizeMB)
	}
	if cfg.Logging.MaxBackups != 4 {
		t.Fatalf("unexpected logging max backups: %d", cfg.Logging.MaxBackups)
	}
	if cfg.Turnstile.Enabled {
		t.Fatal("expected turnstile disabled")
	}
	if cfg.Turnstile.SiteKey != "site-from-env" {
		t.Fatalf("unexpected turnstile site key: %s", cfg.Turnstile.SiteKey)
	}
	if cfg.Turnstile.SecretKey != "secret-from-env" {
		t.Fatalf("unexpected turnstile secret key: %s", cfg.Turnstile.SecretKey)
	}
	if cfg.Turnstile.VerifyURL != "https://turnstile.example.test" {
		t.Fatalf("unexpected turnstile verify url: %s", cfg.Turnstile.VerifyURL)
	}
	if cfg.Worker.Concurrency != 4 {
		t.Fatalf("unexpected worker concurrency: %d", cfg.Worker.Concurrency)
	}
	if cfg.Worker.PollInterval != 2*time.Second {
		t.Fatalf("unexpected poll interval: %s", cfg.Worker.PollInterval)
	}
}

func TestLoadConfigJWTEnvOverride(t *testing.T) {
	setupDefaultConfig(t)
	path := writeConfig(t, `database:
  sqlite:
    path: "data/test.db"
jwt:
  secret_key: "from-yaml"
`)
	t.Setenv("POMELO_ORBIT_BACKEND__JWT__SECRET_KEY", "from-env")

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.JWT.SecretKey != "from-env" {
		t.Fatalf("unexpected jwt secret: %s", cfg.JWT.SecretKey)
	}
}

func TestLoadConfigMySQL(t *testing.T) {
	setupDefaultConfig(t)
	path := writeConfig(t, `database:
  driver: mysql
  mysql:
    dsn: "user:pass@tcp(127.0.0.1:3306)/pomelo_orbit?parseTime=true"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Database.Driver != DatabaseDriverMySQL {
		t.Fatalf("unexpected database driver: %s", cfg.Database.Driver)
	}
	if cfg.Database.MySQL.DSN == "" {
		t.Fatal("expected mysql dsn")
	}
}

func TestLoadConfigDerivesDatabasePath(t *testing.T) {
	setupDefaultConfig(t)
	path := writeConfig(t, `database:
  sqlite:
    path: ""
orbit:
  root: "/opt/pomelo-orbit"
`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	want := filepath.Join("/opt/pomelo-orbit", "data", "db", "pomelo-orbit.db")
	if cfg.Database.SQLite.Path != want {
		t.Fatalf("unexpected sqlite path: %s", cfg.Database.SQLite.Path)
	}
}

func TestLoadConfigValidatesLogging(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{name: "missing file", content: `logging:
  file: ""
`},
		{name: "missing max size", content: `logging:
  max_size_mb: 0
`},
		{name: "missing max backups", content: `logging:
  max_backups: 0
`},
	}
	for _, tc := range cases {
		setupDefaultConfig(t)
		path := writeConfig(t, tc.content)
		if _, err := Load(path); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}

func TestLoadConfigValidatesTurnstile(t *testing.T) {
	cases := []struct {
		name    string
		content string
		wantErr bool
	}{
		{name: "missing site key", content: `turnstile:
  site_key: ""
`, wantErr: true},
		{name: "missing secret key", content: `turnstile:
  secret_key: ""
`, wantErr: true},
		{name: "missing verify url", content: `turnstile:
  verify_url: ""
`, wantErr: true},
		{name: "disabled without keys", content: `turnstile:
  enabled: false
  site_key: ""
  secret_key: ""
  verify_url: ""
`, wantErr: false},
	}
	for _, tc := range cases {
		setupDefaultConfig(t)
		path := writeConfig(t, tc.content)
		_, err := Load(path)
		if tc.wantErr && err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
		if !tc.wantErr && err != nil {
			t.Fatalf("unexpected error for %s: %v", tc.name, err)
		}
	}
}

func setupDefaultConfig(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(cwd); err != nil {
			t.Fatal(err)
		}
	})
	if err := os.WriteFile(defaultConfigFile, []byte(defaultConfigContent), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

const defaultConfigContent = `app:
  name: Pomelo Orbit Backend Go
  version: 0.1.0
  debug: false
server:
  host: localhost
  port: 8080
logging:
  level: info
  file: logs/backend-go.log
  max_size_mb: 100
  max_backups: 7
database:
  driver: sqlite
  sqlite:
    path: data/db/pomelo-orbit.db
  mysql:
    dsn: ""
orbit:
  root: .
jwt:
  secret_key: ""
traefik:
  domain_suffix: lvh.me
turnstile:
  enabled: true
  site_key: "1x00000000000000000000AA"
  secret_key: "1x0000000000000000000000000000000AA"
  verify_url: "https://challenges.cloudflare.com/turnstile/v0/siteverify"
cert:
  letsencrypt:
    enabled: false
    email: ""
    challenge: http
    dns_provider: ""
worker:
  id: ""
  poll_interval: 1s
  lease_duration: 5m
  max_attempts: 3
  concurrency: 1
`

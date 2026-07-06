package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	testJWTSecret           = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	standardBase64JWTSecret = "J69d31L/5Dg4yhJXhp+CacFovpi8Ikhr34zSsHmE3x4="
)

func TestLoadDefaultConfigFile(t *testing.T) {
	setupDefaultConfig(t)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Database.Driver != DatabaseDriverSQLite {
		t.Fatalf("unexpected database driver: %s", cfg.Database.Driver)
	}
	if cfg.App.Name != "Pomelo Orbit Backend Go" {
		t.Fatalf("unexpected app name: %s", cfg.App.Name)
	}
	if cfg.Server.Host != "127.0.0.1" {
		t.Fatalf("unexpected server host: %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 9021 {
		t.Fatalf("unexpected server port: %d", cfg.Server.Port)
	}
	if len(cfg.Server.CORSAllowedOrigins) != 0 {
		t.Fatalf("unexpected cors allowed origins: %v", cfg.Server.CORSAllowedOrigins)
	}
	if len(cfg.Server.ApiPathPrefixes) != 1 || cfg.Server.ApiPathPrefixes[0] != "/api" {
		t.Fatalf("unexpected api path prefixes: %v", cfg.Server.ApiPathPrefixes)
	}
	if cfg.Server.PublicURL != "" {
		t.Fatalf("unexpected server public url: %s", cfg.Server.PublicURL)
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
	if cfg.Logging.HTTPBodyEnabled {
		t.Fatal("expected http body logging disabled")
	}
	if cfg.Logging.HTTPBodyMaxBytes != 4096 {
		t.Fatalf("unexpected http body max bytes: %d", cfg.Logging.HTTPBodyMaxBytes)
	}
	if cfg.JWT.SecretKey != testJWTSecret {
		t.Fatalf("unexpected jwt secret key: %s", cfg.JWT.SecretKey)
	}
	if cfg.Traefik.APIURL != "http://traefik:8080" {
		t.Fatalf("unexpected traefik api url: %s", cfg.Traefik.APIURL)
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
	if cfg.EnvFilePath != filepath.Join(currentDir(t), ".env") {
		t.Fatalf("unexpected env file path: %s", cfg.EnvFilePath)
	}
}

func TestLoadConfigMergesEnvConfig(t *testing.T) {
	setupDefaultConfig(t)
	t.Setenv("POMELO_ORBIT_APP__ENV", "develop")
	writeEnvConfig(t, "develop", `logging:
  level: "DEBUG"
  max_size_mb: 50
  max_backups: 3
  http_body_enabled: true
  http_body_max_bytes: 2048
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

	cfg, err := Load()
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
	if !cfg.Logging.HTTPBodyEnabled {
		t.Fatal("expected http body logging enabled")
	}
	if cfg.Logging.HTTPBodyMaxBytes != 2048 {
		t.Fatalf("unexpected http body max bytes: %d", cfg.Logging.HTTPBodyMaxBytes)
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

func TestLoadConfigRejectsInvalidEnvConfig(t *testing.T) {
	setupDefaultConfig(t)
	t.Setenv("POMELO_ORBIT_APP__ENV", "develop")
	writeEnvConfig(t, "develop", `database:
  sqlite:
    path: [invalid]
`)

	if _, err := Load(); err == nil {
		t.Fatal("expected error for invalid env config")
	}
}

func TestLoadConfigEnvOverrides(t *testing.T) {
	setupDefaultConfig(t)
	t.Setenv("POMELO_ORBIT_APP__ENV", "develop")
	writeEnvConfig(t, "develop", `server:
  host: "127.0.0.1"
  port: 9000
database:
  sqlite:
    path: "data/test.db"
worker:
  concurrency: 1
`)
	t.Setenv("POMELO_ORBIT_SERVER__HOST", "0.0.0.0")
	t.Setenv("POMELO_ORBIT_SERVER__PORT", "8088")
	t.Setenv("POMELO_ORBIT_SERVER__CORS_ALLOWED_ORIGINS", "https://orbit.preflite.cn/,https://preview.preflite.cn,https://orbit.preflite.cn")
	t.Setenv("POMELO_ORBIT_SERVER__API_PATH_PREFIXES", "/api/,/graphql,/api")
	t.Setenv("POMELO_ORBIT_SERVER__PUBLIC_URL", "https://orbit-api.preflite.cn/")
	t.Setenv("POMELO_ORBIT_DATABASE__SQLITE__PATH", "/data/pomelo-repository.db")
	t.Setenv("POMELO_ORBIT_LOGGING__MAX_SIZE_MB", "25")
	t.Setenv("POMELO_ORBIT_LOGGING__MAX_BACKUPS", "4")
	t.Setenv("POMELO_ORBIT_LOGGING__HTTP_BODY_ENABLED", "true")
	t.Setenv("POMELO_ORBIT_LOGGING__HTTP_BODY_MAX_BYTES", "8192")
	t.Setenv("POMELO_ORBIT_TURNSTILE__ENABLED", "false")
	t.Setenv("POMELO_ORBIT_TURNSTILE__SITE_KEY", "site-from-env")
	t.Setenv("POMELO_ORBIT_TURNSTILE__SECRET_KEY", "secret-from-env")
	t.Setenv("POMELO_ORBIT_TURNSTILE__VERIFY_URL", "https://turnstile.example.test")
	t.Setenv("POMELO_ORBIT_TRAEFIK__API_URL", "http://traefik.example.test:8080")
	t.Setenv("POMELO_ORBIT_ORBIT__ROOT", "/srv/pomelo-orbit")
	t.Setenv("POMELO_ORBIT_WORKER__CONCURRENCY", "4")
	t.Setenv("POMELO_ORBIT_WORKER__POLL_INTERVAL", "2s")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Server.Host != "0.0.0.0" {
		t.Fatalf("unexpected server host: %s", cfg.Server.Host)
	}
	if cfg.Server.Port != 8088 {
		t.Fatalf("unexpected server port: %d", cfg.Server.Port)
	}
	if len(cfg.Server.CORSAllowedOrigins) != 2 || cfg.Server.CORSAllowedOrigins[0] != "https://orbit.preflite.cn" || cfg.Server.CORSAllowedOrigins[1] != "https://preview.preflite.cn" {
		t.Fatalf("unexpected cors allowed origins: %v", cfg.Server.CORSAllowedOrigins)
	}
	if len(cfg.Server.ApiPathPrefixes) != 2 || cfg.Server.ApiPathPrefixes[0] != "/api" || cfg.Server.ApiPathPrefixes[1] != "/graphql" {
		t.Fatalf("unexpected api path prefixes: %v", cfg.Server.ApiPathPrefixes)
	}
	if cfg.Server.PublicURL != "https://orbit-api.preflite.cn" {
		t.Fatalf("unexpected server public url: %s", cfg.Server.PublicURL)
	}
	if cfg.Database.SQLite.Path != "/data/pomelo-repository.db" {
		t.Fatalf("unexpected sqlite path: %s", cfg.Database.SQLite.Path)
	}
	if cfg.Logging.MaxSizeMB != 25 {
		t.Fatalf("unexpected logging max size: %d", cfg.Logging.MaxSizeMB)
	}
	if cfg.Logging.MaxBackups != 4 {
		t.Fatalf("unexpected logging max backups: %d", cfg.Logging.MaxBackups)
	}
	if !cfg.Logging.HTTPBodyEnabled {
		t.Fatal("expected http body logging enabled")
	}
	if cfg.Logging.HTTPBodyMaxBytes != 8192 {
		t.Fatalf("unexpected http body max bytes: %d", cfg.Logging.HTTPBodyMaxBytes)
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
	if cfg.Traefik.APIURL != "http://traefik.example.test:8080" {
		t.Fatalf("unexpected traefik api url: %s", cfg.Traefik.APIURL)
	}
	if cfg.Orbit.Root != "/srv/pomelo-orbit" {
		t.Fatalf("unexpected orbit root: %s", cfg.Orbit.Root)
	}
	if cfg.Worker.Concurrency != 4 {
		t.Fatalf("unexpected worker concurrency: %d", cfg.Worker.Concurrency)
	}
	if cfg.Worker.PollInterval != 2*time.Second {
		t.Fatalf("unexpected poll interval: %s", cfg.Worker.PollInterval)
	}
}

func TestLoadConfigRejectsInvalidApiPathPrefixes(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{name: "empty", content: `server:
  api_path_prefixes: []
`, want: "server.api_path_prefixes must not be empty"},
		{name: "root", content: `server:
  api_path_prefixes:
    - /
`, want: "server.api_path_prefixes must not contain root path"},
		{name: "missing slash", content: `server:
  api_path_prefixes:
    - api
`, want: "server.api_path_prefixes must start with /: api"},
	}
	for _, tc := range cases {
		setupDefaultConfig(t)
		writeEnvConfig(t, "develop", tc.content)

		_, err := Load()
		if err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
		if !strings.Contains(err.Error(), tc.want) {
			t.Fatalf("unexpected error for %s: %v", tc.name, err)
		}
	}
}

func TestLoadConfigRejectsInvalidPublicURL(t *testing.T) {
	setupDefaultConfig(t)
	t.Setenv("POMELO_ORBIT_SERVER__PUBLIC_URL", "ftp://orbit-api.preflite.cn")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid public url")
	}
	if !strings.Contains(err.Error(), "server.public_url must use http or https scheme") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigRejectsInvalidCORSOrigin(t *testing.T) {
	setupDefaultConfig(t)
	t.Setenv("POMELO_ORBIT_SERVER__CORS_ALLOWED_ORIGINS", "https://orbit.preflite.cn/path")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid cors origin")
	}
	if !strings.Contains(err.Error(), "server.cors_allowed_origins must not include path, query, or fragment") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestLoadConfigEnvFileOverridesConfig(t *testing.T) {
	setupDefaultConfig(t)
	preserveEnv(t, "POMELO_ORBIT_SERVER__PORT", "POMELO_ORBIT_TURNSTILE__SITE_KEY", "POMELO_ORBIT_TURNSTILE__SECRET_KEY")
	writeDotEnv(t, `POMELO_ORBIT_SERVER__PORT=8081
POMELO_ORBIT_TURNSTILE__SITE_KEY=site-from-dotenv
POMELO_ORBIT_TURNSTILE__SECRET_KEY=secret-from-dotenv
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Server.Port != 8081 {
		t.Fatalf("unexpected server port: %d", cfg.Server.Port)
	}
	if cfg.Turnstile.SiteKey != "site-from-dotenv" {
		t.Fatalf("unexpected turnstile site key: %s", cfg.Turnstile.SiteKey)
	}
	if cfg.Turnstile.SecretKey != "secret-from-dotenv" {
		t.Fatalf("unexpected turnstile secret key: %s", cfg.Turnstile.SecretKey)
	}
}

func TestLoadConfigEnvFileOverridesEnvConfig(t *testing.T) {
	setupDefaultConfig(t)
	preserveEnv(t, "POMELO_ORBIT_SERVER__PORT", "POMELO_ORBIT_TURNSTILE__SITE_KEY")
	t.Setenv("POMELO_ORBIT_APP__ENV", "develop")
	writeEnvConfig(t, "develop", `server:
  port: 7000
turnstile:
  site_key: "site-from-env-config"
`)
	writeEnvDotEnv(t, "develop", `POMELO_ORBIT_SERVER__PORT=8082
POMELO_ORBIT_TURNSTILE__SITE_KEY=site-from-dotenv
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Server.Port != 8082 {
		t.Fatalf("unexpected server port: %d", cfg.Server.Port)
	}
	if cfg.Turnstile.SiteKey != "site-from-dotenv" {
		t.Fatalf("unexpected turnstile site key: %s", cfg.Turnstile.SiteKey)
	}
}

func TestLoadConfigOSEnvOverridesEnvFile(t *testing.T) {
	setupDefaultConfig(t)
	preserveEnv(t, "POMELO_ORBIT_SERVER__PORT", "POMELO_ORBIT_TURNSTILE__SITE_KEY")
	writeDotEnv(t, `POMELO_ORBIT_SERVER__PORT=8083
POMELO_ORBIT_TURNSTILE__SITE_KEY=site-from-dotenv
`)
	t.Setenv("POMELO_ORBIT_SERVER__PORT", "9090")
	t.Setenv("POMELO_ORBIT_TURNSTILE__SITE_KEY", "site-from-os-env")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Server.Port != 9090 {
		t.Fatalf("unexpected server port: %d", cfg.Server.Port)
	}
	if cfg.Turnstile.SiteKey != "site-from-os-env" {
		t.Fatalf("unexpected turnstile site key: %s", cfg.Turnstile.SiteKey)
	}
}

func TestLoadConfigRejectsInvalidEnvFile(t *testing.T) {
	setupDefaultConfig(t)
	writeDotEnv(t, "POMELO_ORBIT_SERVER__PORT='unterminated")

	_, err := Load()
	if err == nil {
		t.Fatal("expected error for invalid env file")
	}
	if !strings.Contains(err.Error(), "read env file") {
		t.Fatalf("expected env file error, got: %v", err)
	}
}

func TestLoadConfigJWTEnvOverride(t *testing.T) {
	setupDefaultConfig(t)
	writeEnvConfig(t, "develop", `database:
  sqlite:
    path: "data/test.db"
jwt:
  secret_key: "from-yaml"
`)
	t.Setenv("POMELO_ORBIT_JWT__SECRET_KEY", standardBase64JWTSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.JWT.SecretKey != standardBase64JWTSecret {
		t.Fatalf("unexpected jwt secret: %s", cfg.JWT.SecretKey)
	}
}

func TestLoadConfigBaseIgnoresEnvOverrides(t *testing.T) {
	setupDefaultConfig(t)
	t.Setenv("POMELO_ORBIT_APP__ENV", "develop")
	writeEnvConfig(t, "develop", `server:
  host: "127.0.0.1"
  port: 9000
database:
  sqlite:
    path: "data/test.db"
`)
	t.Setenv("POMELO_ORBIT_SERVER__PORT", "8088")
	t.Setenv("POMELO_ORBIT_LOGGING__LEVEL", "DEBUG")
	t.Setenv("POMELO_ORBIT_DATABASE__SQLITE__PATH", "/overridden/db.sqlite")

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Server.Port != 8088 {
		t.Fatalf("expected full config port 8088, got %d", cfg.Server.Port)
	}
	if cfg.Logging.Level != "DEBUG" {
		t.Fatalf("expected full config logging level DEBUG, got %s", cfg.Logging.Level)
	}
	if cfg.Base == nil {
		t.Fatal("expected base config to be populated")
	}
	if cfg.Base.Server.Port != 9000 {
		t.Fatalf("expected base port 9000 from env config, got %d", cfg.Base.Server.Port)
	}
	if cfg.Base.Logging.Level != "info" {
		t.Fatalf("expected base logging level info from defaults, got %s", cfg.Base.Logging.Level)
	}
	if cfg.Base.Database.SQLite.Path != "data/test.db" {
		t.Fatalf("expected base sqlite path from env config, got %s", cfg.Base.Database.SQLite.Path)
	}
}

func TestLoadConfigValidatesJWTSecretKey(t *testing.T) {
	cases := []struct {
		name    string
		content string
	}{
		{name: "missing secret key", content: `jwt:
  secret_key: ""
`},
		{name: "short secret key", content: `jwt:
  secret_key: "too-short"
`},
	}
	for _, tc := range cases {
		setupDefaultConfig(t)
		writeEnvConfig(t, "develop", tc.content)
		if _, err := Load(); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}

func TestLoadConfigAcceptsStandardBase64JWTSecretKey(t *testing.T) {
	setupDefaultConfig(t)
	writeEnvConfig(t, "develop", fmt.Sprintf(`jwt:
  secret_key: %q
`, standardBase64JWTSecret))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.JWT.SecretKey != standardBase64JWTSecret {
		t.Fatalf("unexpected jwt secret: %s", cfg.JWT.SecretKey)
	}
}

func TestLoadConfigMySQL(t *testing.T) {
	setupDefaultConfig(t)
	writeEnvConfig(t, "develop", `database:
  driver: mysql
  mysql:
    dsn: "user:pass@tcp(127.0.0.1:3306)/pomelo_orbit?parseTime=true"
`)

	cfg, err := Load()
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

func TestLoadConfigRequiresSQLitePath(t *testing.T) {
	setupDefaultConfig(t)
	writeEnvConfig(t, "develop", `database:
  sqlite:
    path: ""
`)

	if _, err := Load(); err == nil {
		t.Fatal("expected error for empty sqlite path")
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
		{name: "missing http body max bytes", content: `logging:
  http_body_max_bytes: 0
`},
	}
	for _, tc := range cases {
		setupDefaultConfig(t)
		writeEnvConfig(t, "develop", tc.content)
		if _, err := Load(); err == nil {
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
		writeEnvConfig(t, "develop", tc.content)
		_, err := Load()
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
	if err := os.MkdirAll(filepath.Dir(DefaultConfigFile), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(DefaultConfigFile, []byte(defaultConfigContent), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeEnvConfig(t *testing.T, envName string, content string) {
	t.Helper()
	t.Setenv("POMELO_ORBIT_APP__ENV", envName)
	path := EnvConfigFile(envName)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeDotEnv(t *testing.T, content string) {
	t.Helper()
	if err := os.WriteFile(".env", []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func writeEnvDotEnv(t *testing.T, envName string, content string) {
	t.Helper()
	if err := os.WriteFile(fmt.Sprintf(".env.%s", envName), []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func currentDir(t *testing.T) string {
	t.Helper()
	cwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	return cwd
}

func preserveEnv(t *testing.T, keys ...string) {
	t.Helper()
	originals := make(map[string]string, len(keys))
	present := make(map[string]bool, len(keys))
	for _, key := range keys {
		value, ok := os.LookupEnv(key)
		originals[key] = value
		present[key] = ok
		if err := os.Unsetenv(key); err != nil {
			t.Fatal(err)
		}
	}
	t.Cleanup(func() {
		for _, key := range keys {
			if present[key] {
				if err := os.Setenv(key, originals[key]); err != nil {
					t.Fatal(err)
				}
				continue
			}
			if err := os.Unsetenv(key); err != nil {
				t.Fatal(err)
			}
		}
	})
}

const defaultConfigContent = `app:
  name: Pomelo Orbit Backend Go
  version: 0.1.0
  debug: false
server:
  host: 127.0.0.1
  port: 9021
  cors_allowed_origins: []
  api_path_prefixes:
    - /api
  public_url: ""
logging:
  level: info
  file: logs/backend-go.log
  max_size_mb: 100
  max_backups: 7
  http_body_enabled: false
  http_body_max_bytes: 4096
database:
  driver: sqlite
  sqlite:
    path: data/db/pomelo-repository.db
  mysql:
    dsn: ""
orbit:
  root: .
jwt:
  secret_key: "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
traefik:
  api_url: http://traefik:8080
  domain_suffix: lvh.me
  dynamic_route_dir: data/cd/traefik/data/dynamic
  cert_dir: data/cd/traefik/data/certs
  container_name: traefik
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
settings:
  secret_keys:
    - database__mysql__dsn
    - jwt__secret_key
    - turnstile__secret_key
worker:
  id: ""
  poll_interval: 1s
  lease_duration: 5m
  max_attempts: 3
  concurrency: 1
`

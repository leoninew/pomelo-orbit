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
	testJwtSecret           = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
	standardBase64JwtSecret = "J69d31L/5Dg4yhJXhp+CacFovpi8Ikhr34zSsHmE3x4="
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
	if len(cfg.Server.CorsAllowedOrigins) != 0 {
		t.Fatalf("unexpected cors allowed origins: %v", cfg.Server.CorsAllowedOrigins)
	}
	if len(cfg.Server.ApiPathPrefixes) != 1 || cfg.Server.ApiPathPrefixes[0] != "/api" {
		t.Fatalf("unexpected api path prefixes: %v", cfg.Server.ApiPathPrefixes)
	}
	if cfg.Server.PublicUrl != "" {
		t.Fatalf("unexpected server public url: %s", cfg.Server.PublicUrl)
	}
	if cfg.Logging.File != "logs/pomelo-orbit.log" {
		t.Fatalf("unexpected logging file: %s", cfg.Logging.File)
	}
	if cfg.Logging.MaxSizeMB != 100 {
		t.Fatalf("unexpected logging max size: %d", cfg.Logging.MaxSizeMB)
	}
	if cfg.Logging.MaxBackups != 7 {
		t.Fatalf("unexpected logging max backups: %d", cfg.Logging.MaxBackups)
	}
	if cfg.Logging.HTTP.Enabled {
		t.Fatal("expected http access logging disabled")
	}
	if cfg.Logging.HTTP.RequestBodyLimit != 4096 {
		t.Fatalf("unexpected http request body limit: %d", cfg.Logging.HTTP.RequestBodyLimit)
	}
	if cfg.Logging.HTTP.ResponseBodyLimit != 4096 {
		t.Fatalf("unexpected http response body limit: %d", cfg.Logging.HTTP.ResponseBodyLimit)
	}
	if !cfg.Logging.HTTP.SkipAssetEnabled {
		t.Fatal("expected successful asset log skipping enabled")
	}
	if cfg.Jwt.SecretKey != testJwtSecret {
		t.Fatalf("unexpected jwt secret key: %s", cfg.Jwt.SecretKey)
	}
	if cfg.Traefik.CertDir != "data/deployment/traefik/data/certs" {
		t.Fatalf("unexpected traefik cert dir: %s", cfg.Traefik.CertDir)
	}
	if cfg.Traefik.Image != "traefik:3.6" {
		t.Fatalf("unexpected traefik image: %q", cfg.Traefik.Image)
	}
	if cfg.Traefik.RestApiUrl != "http://localhost:8080" || cfg.Traefik.BaseDomain != "lvh.me" {
		t.Fatalf("unexpected traefik endpoints: rest=%q domain=%q", cfg.Traefik.RestApiUrl, cfg.Traefik.BaseDomain)
	}
	if cfg.Traefik.RestReadyTimeout != 20*time.Second {
		t.Fatalf("unexpected traefik rest ready timeout: %s", cfg.Traefik.RestReadyTimeout)
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
	if cfg.Turnstile.VerifyUrl != "https://challenges.cloudflare.com/turnstile/v0/siteverify" {
		t.Fatalf("unexpected turnstile verify url: %s", cfg.Turnstile.VerifyUrl)
	}
	if cfg.Worker.PollInterval != time.Second {
		t.Fatalf("unexpected poll interval: %s", cfg.Worker.PollInterval)
	}
	if cfg.PipelineRun.ExecutionTimeout != time.Hour {
		t.Fatalf("unexpected pipeline execution timeout: %s", cfg.PipelineRun.ExecutionTimeout)
	}
	if cfg.Worker.LeaseDuration != time.Hour+5*time.Minute {
		t.Fatalf("unexpected worker lease duration: %s", cfg.Worker.LeaseDuration)
	}
	if cfg.Worker.MaxAttempts != 1 {
		t.Fatalf("unexpected worker max attempts: %d", cfg.Worker.MaxAttempts)
	}
	if cfg.EnvFilePath != filepath.Join(currentDir(t), ".env") {
		t.Fatalf("unexpected env file path: %s", cfg.EnvFilePath)
	}
}

func TestValidateMCPClientRejectsAPIURLPath(t *testing.T) {
	cfg := Config{
		MCP: MCPConfig{
			APIUrl:      "http://127.0.0.1:9021/api",
			WebUrl:      "http://localhost:9020",
			AuthTimeout: time.Minute,
		},
	}

	err := cfg.ValidateMCPClient()
	if err == nil || !strings.Contains(err.Error(), "mcp.api_url must not include path") {
		t.Fatalf("ValidateMCPClient() error = %v", err)
	}
}

func TestLoadConfigMergesEnvConfig(t *testing.T) {
	setupDefaultConfig(t)
	t.Setenv("POMELO_ORBIT_APP__ENV", "develop")
	writeEnvConfig(t, "develop", `logging:
  level: "DEBUG"
  max_size_mb: 50
  max_backups: 3
  http:
    enabled: true
    request_body_limit: 2048
    response_body_limit: 1024
    skip_asset_enabled: false
database:
  sqlite:
    path: "data/test.db"
orbit:
  root: "../.."
worker:
  id: "worker-1"
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
	if !cfg.Logging.HTTP.Enabled {
		t.Fatal("expected http access logging enabled")
	}
	if cfg.Logging.HTTP.RequestBodyLimit != 2048 {
		t.Fatalf("unexpected http request body limit: %d", cfg.Logging.HTTP.RequestBodyLimit)
	}
	if cfg.Logging.HTTP.ResponseBodyLimit != 1024 {
		t.Fatalf("unexpected http response body limit: %d", cfg.Logging.HTTP.ResponseBodyLimit)
	}
	if cfg.Logging.HTTP.SkipAssetEnabled {
		t.Fatal("expected successful asset log skipping disabled")
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
	if cfg.Worker.LeaseDuration != time.Hour+5*time.Minute {
		t.Fatalf("unexpected lease duration from defaults: %s", cfg.Worker.LeaseDuration)
	}
	if cfg.PipelineRun.ExecutionTimeout != time.Hour {
		t.Fatalf("unexpected pipeline execution timeout from defaults: %s", cfg.PipelineRun.ExecutionTimeout)
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
	t.Setenv("POMELO_ORBIT_LOGGING__HTTP__ENABLED", "true")
	t.Setenv("POMELO_ORBIT_LOGGING__HTTP__REQUEST_BODY_LIMIT", "8192")
	t.Setenv("POMELO_ORBIT_LOGGING__HTTP__RESPONSE_BODY_LIMIT", "2048")
	t.Setenv("POMELO_ORBIT_LOGGING__HTTP__SKIP_ASSET_ENABLED", "false")
	t.Setenv("POMELO_ORBIT_TURNSTILE__ENABLED", "false")
	t.Setenv("POMELO_ORBIT_TURNSTILE__SITE_KEY", "site-from-env")
	t.Setenv("POMELO_ORBIT_TURNSTILE__SECRET_KEY", "secret-from-env")
	t.Setenv("POMELO_ORBIT_TURNSTILE__VERIFY_URL", "https://turnstile.example.test")
	t.Setenv("POMELO_ORBIT_TRAEFIK__CERT_DIR", "data/custom/certs")
	t.Setenv("POMELO_ORBIT_TRAEFIK__IMAGE", "traefik:v3.9")
	t.Setenv("POMELO_ORBIT_TRAEFIK__REST_API_URL", "http://127.0.0.1:9080")
	t.Setenv("POMELO_ORBIT_TRAEFIK__REST_READY_TIMEOUT", "45s")
	t.Setenv("POMELO_ORBIT_ORBIT__ROOT", "/srv/pomelo-orbit")
	t.Setenv("POMELO_ORBIT_WORKER__CONCURRENCY", "4")
	t.Setenv("POMELO_ORBIT_WORKER__POLL_INTERVAL", "2s")
	t.Setenv("POMELO_ORBIT_WORKER__LEASE_DURATION", "3h")
	t.Setenv("POMELO_ORBIT_PIPELINE_RUN__EXECUTION_TIMEOUT", "2h")

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
	if len(cfg.Server.CorsAllowedOrigins) != 2 || cfg.Server.CorsAllowedOrigins[0] != "https://orbit.preflite.cn" || cfg.Server.CorsAllowedOrigins[1] != "https://preview.preflite.cn" {
		t.Fatalf("unexpected cors allowed origins: %v", cfg.Server.CorsAllowedOrigins)
	}
	if len(cfg.Server.ApiPathPrefixes) != 2 || cfg.Server.ApiPathPrefixes[0] != "/api" || cfg.Server.ApiPathPrefixes[1] != "/graphql" {
		t.Fatalf("unexpected api path prefixes: %v", cfg.Server.ApiPathPrefixes)
	}
	if cfg.Server.PublicUrl != "https://orbit-api.preflite.cn" {
		t.Fatalf("unexpected server public url: %s", cfg.Server.PublicUrl)
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
	if !cfg.Logging.HTTP.Enabled {
		t.Fatal("expected http access logging enabled")
	}
	if cfg.Logging.HTTP.RequestBodyLimit != 8192 {
		t.Fatalf("unexpected http request body limit: %d", cfg.Logging.HTTP.RequestBodyLimit)
	}
	if cfg.Logging.HTTP.ResponseBodyLimit != 2048 {
		t.Fatalf("unexpected http response body limit: %d", cfg.Logging.HTTP.ResponseBodyLimit)
	}
	if cfg.Logging.HTTP.SkipAssetEnabled {
		t.Fatal("expected successful asset log skipping disabled")
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
	if cfg.Turnstile.VerifyUrl != "https://turnstile.example.test" {
		t.Fatalf("unexpected turnstile verify url: %s", cfg.Turnstile.VerifyUrl)
	}
	if cfg.Traefik.CertDir != "data/custom/certs" {
		t.Fatalf("unexpected traefik cert dir: %s", cfg.Traefik.CertDir)
	}
	if cfg.Traefik.Image != "traefik:v3.9" || cfg.Traefik.RestApiUrl != "http://127.0.0.1:9080" || cfg.Traefik.RestReadyTimeout != 45*time.Second {
		t.Fatalf("unexpected Traefik env overrides: %#v", cfg.Traefik)
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
	if cfg.Worker.LeaseDuration != 3*time.Hour {
		t.Fatalf("unexpected worker lease duration: %s", cfg.Worker.LeaseDuration)
	}
	if cfg.PipelineRun.ExecutionTimeout != 2*time.Hour {
		t.Fatalf("unexpected pipeline execution timeout: %s", cfg.PipelineRun.ExecutionTimeout)
	}
}

func TestLoadConfigRejectsInvalidPipelineExecutionTimeout(t *testing.T) {
	cases := []struct {
		name    string
		content string
		want    string
	}{
		{
			name: "non-positive timeout",
			content: `pipeline_run:
  execution_timeout: 0s
`,
			want: "pipeline_run.execution_timeout must be positive",
		},
		{
			name: "lease does not exceed timeout",
			content: `pipeline_run:
  execution_timeout: 1h
worker:
  lease_duration: 1h
`,
			want: "worker.lease_duration must exceed pipeline_run.execution_timeout",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			setupDefaultConfig(t)
			writeEnvConfig(t, "develop", tc.content)

			_, err := Load()
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("expected %q, got %v", tc.want, err)
			}
		})
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

func TestLoadConfigRejectsInvalidPublicUrl(t *testing.T) {
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

func TestLoadConfigRejectsInvalidCorsOrigin(t *testing.T) {
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

func TestLoadConfigJwtEnvOverride(t *testing.T) {
	setupDefaultConfig(t)
	writeEnvConfig(t, "develop", `database:
  sqlite:
    path: "data/test.db"
jwt:
  secret_key: "from-yaml"
`)
	t.Setenv("POMELO_ORBIT_JWT__SECRET_KEY", standardBase64JwtSecret)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Jwt.SecretKey != standardBase64JwtSecret {
		t.Fatalf("unexpected jwt secret: %s", cfg.Jwt.SecretKey)
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

func TestLoadConfigValidatesJwtSecretKey(t *testing.T) {
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

func TestLoadConfigAcceptsStandardBase64JwtSecretKey(t *testing.T) {
	setupDefaultConfig(t)
	writeEnvConfig(t, "develop", fmt.Sprintf(`jwt:
  secret_key: %q
`, standardBase64JwtSecret))

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Jwt.SecretKey != standardBase64JwtSecret {
		t.Fatalf("unexpected jwt secret: %s", cfg.Jwt.SecretKey)
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
	if cfg.Database.MySQL.Dsn == "" {
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
	}
	for _, tc := range cases {
		setupDefaultConfig(t)
		writeEnvConfig(t, "develop", tc.content)
		if _, err := Load(); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}

func TestLoadConfigNormalizesHTTPBodyLimits(t *testing.T) {
	setupDefaultConfig(t)
	writeEnvConfig(t, "develop", `logging:
  http:
    request_body_limit: -1
    response_body_limit: 0
`)

	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Logging.HTTP.RequestBodyLimit != 0 || cfg.Logging.HTTP.ResponseBodyLimit != 0 {
		t.Fatalf("expected disabled HTTP body logging limits, got %+v", cfg.Logging.HTTP)
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
  file: logs/pomelo-orbit.log
  max_size_mb: 100
  max_backups: 7
  http:
    enabled: false
    request_body_limit: 4096
    response_body_limit: 4096
    skip_asset_enabled: true
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
  cert_dir: data/deployment/traefik/data/certs
  image: traefik:3.6
  rest_api_url: http://localhost:8080
  base_domain: lvh.me
  rest_ready_timeout: 20s
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
pipeline_run:
  execution_timeout: 1h
worker:
  id: ""
  poll_interval: 1s
  lease_duration: 1h5m
  max_attempts: 1
  concurrency: 1
`

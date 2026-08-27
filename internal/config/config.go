package config

import (
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	DefaultConfigFile     = "configs/config.yaml"
	DefaultLLMTimeout     = 60 * time.Second
	jwtSecretKeyMinLength = 32
)

func EnvConfigFile(env string) string {
	return fmt.Sprintf("configs/config.%s.yaml", env)
}

type Config struct {
	App         AppConfig         `mapstructure:"app" yaml:"app"`
	Server      ServerConfig      `mapstructure:"server" yaml:"server"`
	Logging     LoggingConfig     `mapstructure:"logging" yaml:"logging"`
	Database    DatabaseConfig    `mapstructure:"database" yaml:"database"`
	Workspace   WorkspaceConfig   `mapstructure:"workspace" yaml:"workspace"`
	PipelineRun PipelineRunConfig `mapstructure:"pipeline_run" yaml:"pipeline_run"`
	Worker      WorkerConfig      `mapstructure:"worker" yaml:"worker"`
	Orbit       OrbitConfig       `mapstructure:"orbit" yaml:"orbit"`
	Jwt         JwtConfig         `mapstructure:"jwt" yaml:"jwt"`
	Traefik     TraefikConfig     `mapstructure:"traefik" yaml:"traefik"`
	Turnstile   TurnstileConfig   `mapstructure:"turnstile" yaml:"turnstile"`
	Settings    SettingsConfig    `mapstructure:"settings" yaml:"settings"`
	LLM         LLMConfig         `mapstructure:"llm" yaml:"llm"`
	MCP         MCPConfig         `mapstructure:"mcp" yaml:"mcp"`
	EnvFilePath string            `mapstructure:"-" yaml:"-"`
	Base        *Config           `mapstructure:"-" yaml:"-"`
}

type AppConfig struct {
	Name    string `mapstructure:"name" yaml:"name"`
	Version string `mapstructure:"version" yaml:"version"`
	Env     string `mapstructure:"env" yaml:"env"`
	Debug   bool   `mapstructure:"debug" yaml:"debug"`
}

type ServerConfig struct {
	Host               string   `mapstructure:"host" yaml:"host"`
	Port               int      `mapstructure:"port" yaml:"port"`
	CorsAllowedOrigins []string `mapstructure:"cors_allowed_origins" yaml:"cors_allowed_origins"`
	ApiPathPrefixes    []string `mapstructure:"api_path_prefixes" yaml:"api_path_prefixes"`
	PublicUrl          string   `mapstructure:"public_url" yaml:"public_url"`
}

type LoggingConfig struct {
	Level      string        `mapstructure:"level" yaml:"level"`
	File       string        `mapstructure:"file" yaml:"file"`
	MaxSizeMB  int           `mapstructure:"max_size_mb" yaml:"max_size_mb"`
	MaxBackups int           `mapstructure:"max_backups" yaml:"max_backups"`
	HTTP       LogHTTPConfig `mapstructure:"http" yaml:"http"`
}

type LogHTTPConfig struct {
	Enabled           bool `mapstructure:"enabled" yaml:"enabled"`
	RequestBodyLimit  int  `mapstructure:"request_body_limit" yaml:"request_body_limit"`
	ResponseBodyLimit int  `mapstructure:"response_body_limit" yaml:"response_body_limit"`
	SkipAssetEnabled  bool `mapstructure:"skip_asset_enabled" yaml:"skip_asset_enabled"`
}

const (
	DatabaseDriverSQLite = "sqlite"
	DatabaseDriverMySQL  = "mysql"
)

type DatabaseConfig struct {
	Driver string       `mapstructure:"driver" yaml:"driver"`
	SQLite SQLiteConfig `mapstructure:"sqlite" yaml:"sqlite"`
	MySQL  MySQLConfig  `mapstructure:"mysql" yaml:"mysql"`
}

type SQLiteConfig struct {
	Path string `mapstructure:"path" yaml:"path"`
}

type MySQLConfig struct {
	Dsn string `mapstructure:"dsn" yaml:"dsn"`
}

// WorkspaceConfig contains Orbit-managed runtime workspace roots. Relative
// values resolve against orbit.root; absolute values may be outside orbit.root.
// Both are normalized once to absolute Orbit-visible paths during config loading.
type WorkspaceConfig struct {
	Pipeline   string `mapstructure:"pipeline" yaml:"pipeline"`
	Deployment string `mapstructure:"deployment" yaml:"deployment"`
}

type WorkerConfig struct {
	Id            string        `mapstructure:"id" yaml:"id"`
	PollInterval  time.Duration `mapstructure:"poll_interval" yaml:"poll_interval"`
	LeaseDuration time.Duration `mapstructure:"lease_duration" yaml:"lease_duration"`
	MaxAttempts   int           `mapstructure:"max_attempts" yaml:"max_attempts"`
	Concurrency   int           `mapstructure:"concurrency" yaml:"concurrency"`
}

type PipelineRunConfig struct {
	ExecutionTimeout time.Duration `mapstructure:"execution_timeout" yaml:"execution_timeout"`
}

type OrbitConfig struct {
	Root string `mapstructure:"root" yaml:"root"`
}

type JwtConfig struct {
	SecretKey string `mapstructure:"secret_key" yaml:"secret_key"`
}

// TraefikConfig is process-level gateway infrastructure only: paths, image pin,
// environment endpoints, and deploy readiness timing. Product identity (code,
// component name, network name) and display defaults (name, entrypoint, tls)
// are code constants — not operator configuration.
// Fields are normalized and validated during config Load; callers consume them as-is.
// Per-gateway runtime fields after create live in GatewayConfig.
type TraefikConfig struct {
	Image            string        `mapstructure:"image" yaml:"image"`
	RestApiUrl       string        `mapstructure:"rest_api_url" yaml:"rest_api_url"`
	BaseDomain       string        `mapstructure:"base_domain" yaml:"base_domain"`
	RestReadyTimeout time.Duration `mapstructure:"rest_ready_timeout" yaml:"rest_ready_timeout"`
}

type TurnstileConfig struct {
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
	SiteKey   string `mapstructure:"site_key" yaml:"site_key"`
	SecretKey string `mapstructure:"secret_key" yaml:"secret_key"`
	VerifyUrl string `mapstructure:"verify_url" yaml:"verify_url"`
}

type SettingsConfig struct {
	SecretKeys []string `mapstructure:"secret_keys" yaml:"secret_keys"`
}

type LLMConfig struct {
	BaseUrl           string        `mapstructure:"base_url" yaml:"base_url"`
	ApiKey            string        `mapstructure:"api_key" yaml:"api_key"`
	Model             string        `mapstructure:"model" yaml:"model"`
	Timeout           time.Duration `mapstructure:"timeout" yaml:"timeout"`
	MaxToolCallRounds int           `mapstructure:"max_tool_call_rounds" yaml:"max_tool_call_rounds"`
}

// MCPConfig configures the local stdio MCP client handoff. APIUrl is optional
// because a local server URL can be derived from Server; WebUrl is the browser
// origin that serves the authenticated Orbit UI.
type MCPConfig struct {
	APIUrl      string        `mapstructure:"api_url" yaml:"api_url"`
	WebUrl      string        `mapstructure:"web_url" yaml:"web_url"`
	AuthTimeout time.Duration `mapstructure:"auth_timeout" yaml:"auth_timeout"`
}

func Load() (Config, error) {
	envName := strings.TrimSpace(os.Getenv("POMELO_ORBIT_APP__ENV"))
	envPath, err := envFilePath(envName)
	if err != nil {
		return Config{}, err
	}
	if err := loadEnvFile(envPath); err != nil {
		return Config{}, err
	}

	loader := newLoader()
	loader.SetConfigFile(DefaultConfigFile)
	if err := loader.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read base config: %w", err)
	}

	if envName != "" {
		loader.SetConfigFile(EnvConfigFile(envName))
		if err := loader.MergeInConfig(); err != nil && !isOptionalConfigMissing(err) {
			return Config{}, fmt.Errorf("read env config: %w", err)
		}
	}

	var cfg Config
	if err := loader.Unmarshal(&cfg, configDecodeHook()); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	normalizeServerRuntimeOriginConfig(&cfg.Server)
	normalizeLogHTTPConfig(&cfg.Logging.HTTP)
	if err := normalizeWorkspaceConfig(&cfg.Workspace, cfg.OrbitRoot()); err != nil {
		return Config{}, err
	}
	normalizeTraefikConfig(&cfg.Traefik)
	if cfg.Worker.Id == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return Config{}, fmt.Errorf("get hostname: %w", err)
		}
		cfg.Worker.Id = fmt.Sprintf("%s-%d", hostname, os.Getpid())
	}

	cfg.EnvFilePath = envPath
	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}

	base, err := loadBaseConfig(envName)
	if err != nil {
		return Config{}, err
	}
	cfg.Base = &base
	return cfg, nil
}

func loadBaseConfig(envName string) (Config, error) {
	loader := viper.New()
	loader.SetConfigType("yaml")
	loader.SetConfigFile(DefaultConfigFile)
	if err := loader.ReadInConfig(); err != nil {
		return Config{}, fmt.Errorf("read default config: %w", err)
	}
	if envName != "" {
		loader.SetConfigFile(EnvConfigFile(envName))
		if err := loader.MergeInConfig(); err != nil && !isOptionalConfigMissing(err) {
			return Config{}, fmt.Errorf("read env config: %w", err)
		}
	}
	var base Config
	if err := loader.Unmarshal(&base, configDecodeHook()); err != nil {
		return Config{}, fmt.Errorf("parse default config: %w", err)
	}
	normalizeServerRuntimeOriginConfig(&base.Server)
	normalizeLogHTTPConfig(&base.Logging.HTTP)
	if err := normalizeWorkspaceConfig(&base.Workspace, base.OrbitRoot()); err != nil {
		return Config{}, err
	}
	return base, nil
}

func isOptionalConfigMissing(err error) bool {
	var notFound viper.ConfigFileNotFoundError
	return errors.As(err, &notFound) || errors.Is(err, os.ErrNotExist)
}

func envFilePath(envName string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	if envName == "" {
		return filepath.Join(cwd, ".env"), nil
	}
	return filepath.Join(cwd, fmt.Sprintf(".env.%s", envName)), nil
}

func loadEnvFile(path string) error {
	if _, err := os.Stat(path); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("stat env file: %w", err)
	}
	if err := godotenv.Load(path); err != nil {
		return fmt.Errorf("read env file: %w", err)
	}
	return nil
}

func newLoader() *viper.Viper {
	loader := viper.New()
	loader.SetConfigType("yaml")
	bindEnv(loader)
	return loader
}

func configDecodeHook() viper.DecoderConfigOption {
	return viper.DecodeHook(mapstructure.ComposeDecodeHookFunc(
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	))
}

func bindEnv(loader *viper.Viper) {
	keys := []string{
		"app.name",
		"app.version",
		"app.env",
		"app.debug",
		"server.host",
		"server.port",
		"server.cors_allowed_origins",
		"server.api_path_prefixes",
		"server.public_url",
		"logging.level",
		"logging.file",
		"logging.max_size_mb",
		"logging.max_backups",
		"logging.http.enabled",
		"logging.http.request_body_limit",
		"logging.http.response_body_limit",
		"logging.http.skip_asset_enabled",
		"database.driver",
		"database.sqlite.path",
		"database.mysql.dsn",
		"workspace.pipeline",
		"workspace.deployment",
		"pipeline_run.execution_timeout",
		"orbit.root",
		"jwt.secret_key",
		"traefik.image",
		"traefik.rest_api_url",
		"traefik.base_domain",
		"traefik.rest_ready_timeout",
		"turnstile.enabled",
		"turnstile.site_key",
		"turnstile.secret_key",
		"turnstile.verify_url",
		"worker.id",
		"worker.poll_interval",
		"worker.lease_duration",
		"worker.max_attempts",
		"worker.concurrency",
		"settings.secret_keys",
		"llm.base_url",
		"llm.api_key",
		"llm.model",
		"llm.timeout",
		"llm.max_tool_call_rounds",
		"mcp.api_url",
		"mcp.web_url",
		"mcp.auth_timeout",
	}
	for _, key := range keys {
		envName := "POMELO_ORBIT_" + strings.ToUpper(strings.ReplaceAll(key, ".", "__"))
		_ = loader.BindEnv(key, envName)
	}
}

// MCPAPIUrl returns the HTTP server used only for browser-grant exchange. The
// stdio tools themselves still execute locally through application use cases.
func (c Config) MCPAPIUrl() string {
	if value := strings.TrimRight(strings.TrimSpace(c.MCP.APIUrl), "/"); value != "" {
		return value
	}
	if value := strings.TrimRight(strings.TrimSpace(c.Server.PublicUrl), "/"); value != "" {
		return value
	}
	host := strings.TrimSpace(c.Server.Host)
	if host == "" || host == "0.0.0.0" || host == "::" {
		host = "127.0.0.1"
	}
	return "http://" + net.JoinHostPort(host, strconv.Itoa(c.Server.Port))
}

// ValidateMCPClient is intentionally separate from Validate: browser handoff
// settings are required only for the `server mcp` command, not for HTTP/worker.
func (c Config) ValidateMCPClient() error {
	if err := validateHTTPUrl("mcp.api_url", c.MCPAPIUrl(), true); err != nil {
		return err
	}
	if strings.TrimSpace(c.MCP.WebUrl) == "" {
		return errors.New("mcp.web_url is required for the mcp command")
	}
	if err := validateHTTPUrl("mcp.web_url", c.MCP.WebUrl, false); err != nil {
		return err
	}
	if c.MCP.AuthTimeout <= 0 {
		return errors.New("mcp.auth_timeout must be positive")
	}
	return nil
}

func (c Config) Validate() error {
	if err := validateServerRuntimeOriginConfig(c.Server); err != nil {
		return err
	}
	switch c.Database.Driver {
	case DatabaseDriverSQLite:
		if c.Database.SQLite.Path == "" {
			return errors.New("database.sqlite.path is required")
		}
	case DatabaseDriverMySQL:
		if c.Database.MySQL.Dsn == "" {
			return errors.New("database.mysql.dsn is required")
		}
	default:
		return fmt.Errorf("database.driver must be %s or %s", DatabaseDriverSQLite, DatabaseDriverMySQL)
	}
	if strings.TrimSpace(c.Logging.File) == "" {
		return errors.New("logging.file is required")
	}
	if c.Logging.MaxSizeMB <= 0 {
		return errors.New("logging.max_size_mb must be positive")
	}
	if c.Logging.MaxBackups <= 0 {
		return errors.New("logging.max_backups must be positive")
	}
	if err := validateJwtSecretKey(c.Jwt.SecretKey); err != nil {
		return err
	}
	if c.Turnstile.Enabled {
		if strings.TrimSpace(c.Turnstile.SiteKey) == "" {
			return errors.New("turnstile.site_key is required when turnstile is enabled")
		}
		if strings.TrimSpace(c.Turnstile.SecretKey) == "" {
			return errors.New("turnstile.secret_key is required when turnstile is enabled")
		}
		if strings.TrimSpace(c.Turnstile.VerifyUrl) == "" {
			return errors.New("turnstile.verify_url is required when turnstile is enabled")
		}
	}
	if c.Worker.PollInterval <= 0 {
		return errors.New("worker.poll_interval must be positive")
	}
	if c.Worker.LeaseDuration <= 0 {
		return errors.New("worker.lease_duration must be positive")
	}
	if c.PipelineRun.ExecutionTimeout <= 0 {
		return errors.New("pipeline_run.execution_timeout must be positive")
	}
	if err := validateWorkspaceConfig(c.Workspace); err != nil {
		return err
	}
	if c.Worker.LeaseDuration <= c.PipelineRun.ExecutionTimeout {
		return errors.New("worker.lease_duration must exceed pipeline_run.execution_timeout")
	}
	if c.Worker.MaxAttempts < 1 {
		return errors.New("worker.max_attempts must be at least 1")
	}
	if c.Worker.Concurrency < 1 {
		return errors.New("worker.concurrency must be at least 1")
	}
	if err := validateLLMConfig(c.LLM); err != nil {
		return err
	}
	if err := validateTraefikConfig(c.Traefik); err != nil {
		return err
	}
	return nil
}

// normalizeTraefikConfig trims and canonicalizes load-time values so runtime
// code can read TraefikConfig fields without further config-stage work.
func normalizeTraefikConfig(cfg *TraefikConfig) {
	cfg.Image = strings.TrimSpace(cfg.Image)
	cfg.RestApiUrl = strings.TrimRight(strings.TrimSpace(cfg.RestApiUrl), "/")
	cfg.BaseDomain = strings.ToLower(strings.TrimSpace(cfg.BaseDomain))
}

func validateTraefikConfig(cfg TraefikConfig) error {
	if cfg.Image == "" {
		return errors.New("traefik.image is required")
	}
	if cfg.RestApiUrl == "" {
		return errors.New("traefik.rest_api_url is required")
	}
	if err := validateHTTPUrl("traefik.rest_api_url", cfg.RestApiUrl, false); err != nil {
		return err
	}
	if cfg.BaseDomain == "" {
		return errors.New("traefik.base_domain is required")
	}
	if strings.Contains(cfg.BaseDomain, "://") || strings.Contains(cfg.BaseDomain, "/") || strings.Contains(cfg.BaseDomain, " ") {
		return errors.New("traefik.base_domain must be a bare domain (e.g. lvh.me)")
	}
	if cfg.RestReadyTimeout <= 0 {
		return errors.New("traefik.rest_ready_timeout must be positive")
	}
	return nil
}

func normalizeWorkspaceConfig(cfg *WorkspaceConfig, orbitRoot string) error {
	pipeline, err := normalizeWorkspacePath(orbitRoot, cfg.Pipeline)
	if err != nil {
		return fmt.Errorf("workspace.pipeline: %w", err)
	}
	deployment, err := normalizeWorkspacePath(orbitRoot, cfg.Deployment)
	if err != nil {
		return fmt.Errorf("workspace.deployment: %w", err)
	}
	cfg.Pipeline = pipeline
	cfg.Deployment = deployment
	return nil
}

func normalizeWorkspacePath(orbitRoot string, value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return "", errors.New("is required")
	}
	if !filepath.IsAbs(value) {
		value = filepath.Join(orbitRoot, value)
	}
	path, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve absolute path: %w", err)
	}
	return filepath.Clean(path), nil
}

func validateWorkspaceConfig(cfg WorkspaceConfig) error {
	if cfg.Pipeline == "" {
		return errors.New("workspace.pipeline is required")
	}
	if cfg.Deployment == "" {
		return errors.New("workspace.deployment is required")
	}
	if workspacePathsOverlap(cfg.Pipeline, cfg.Deployment) {
		return errors.New("workspace.pipeline and workspace.deployment must not overlap")
	}
	return nil
}

func workspacePathsOverlap(first string, second string) bool {
	return pathContains(first, second) || pathContains(second, first)
}

func pathContains(parent string, child string) bool {
	relative, err := filepath.Rel(parent, child)
	if err != nil {
		return false
	}
	return relative == "." || (relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative))
}

func validateLLMConfig(cfg LLMConfig) error {
	if cfg.MaxToolCallRounds < 1 {
		return errors.New("llm.max_tool_call_rounds must be at least 1")
	}
	configured := strings.TrimSpace(cfg.BaseUrl) != "" || strings.TrimSpace(cfg.ApiKey) != "" || strings.TrimSpace(cfg.Model) != ""
	if !configured {
		return nil
	}
	if strings.TrimSpace(cfg.BaseUrl) == "" || strings.TrimSpace(cfg.ApiKey) == "" || strings.TrimSpace(cfg.Model) == "" {
		return errors.New("llm.base_url, llm.api_key, and llm.model must be configured together")
	}
	if err := validateHTTPUrl("llm.base_url", cfg.BaseUrl, false); err != nil {
		return err
	}
	if cfg.Timeout <= 0 {
		return errors.New("llm.timeout must be positive")
	}
	return nil
}

func normalizeLogHTTPConfig(cfg *LogHTTPConfig) {
	if cfg.RequestBodyLimit <= 0 {
		cfg.RequestBodyLimit = 0
	}
	if cfg.ResponseBodyLimit <= 0 {
		cfg.ResponseBodyLimit = 0
	}
}

func normalizeServerRuntimeOriginConfig(cfg *ServerConfig) {
	cfg.PublicUrl = strings.TrimRight(strings.TrimSpace(cfg.PublicUrl), "/")
	cfg.CorsAllowedOrigins = normalizeHTTPOrigins(cfg.CorsAllowedOrigins)
	cfg.ApiPathPrefixes = normalizeApiPathPrefixes(cfg.ApiPathPrefixes)
}

func normalizeHTTPOrigins(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimRight(strings.TrimSpace(value), "/")
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func normalizeApiPathPrefixes(values []string) []string {
	out := make([]string, 0, len(values))
	seen := map[string]struct{}{}
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "/" {
			value = strings.TrimRight(value, "/")
		}
		if value == "" {
			continue
		}
		if _, ok := seen[value]; ok {
			continue
		}
		seen[value] = struct{}{}
		out = append(out, value)
	}
	return out
}

func validateServerRuntimeOriginConfig(cfg ServerConfig) error {
	if err := validateApiPathPrefixes(cfg.ApiPathPrefixes); err != nil {
		return err
	}
	if cfg.PublicUrl != "" {
		if err := validateHTTPUrl("server.public_url", cfg.PublicUrl, false); err != nil {
			return err
		}
	}
	for _, origin := range cfg.CorsAllowedOrigins {
		if err := validateHTTPUrl("server.cors_allowed_origins", origin, true); err != nil {
			return err
		}
	}
	return nil
}

func validateApiPathPrefixes(prefixes []string) error {
	if len(prefixes) == 0 {
		return errors.New("server.api_path_prefixes must not be empty")
	}
	for _, prefix := range prefixes {
		if prefix == "/" {
			return errors.New("server.api_path_prefixes must not contain root path")
		}
		if !strings.HasPrefix(prefix, "/") {
			return fmt.Errorf("server.api_path_prefixes must start with /: %s", prefix)
		}
	}
	return nil
}

func validateHTTPUrl(key string, value string, originOnly bool) error {
	parsed, err := url.Parse(strings.TrimSpace(value))
	if err != nil || parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("%s must be an absolute http or https URL", key)
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("%s must use http or https scheme", key)
	}
	if parsed.RawQuery != "" || parsed.Fragment != "" {
		return fmt.Errorf("%s must not include query or fragment", key)
	}
	if originOnly && parsed.Path != "" {
		return fmt.Errorf("%s must not include path, query, or fragment", key)
	}
	return nil
}

func validateJwtSecretKey(secretKey string) error {
	secretKey = strings.TrimSpace(secretKey)
	if secretKey == "" {
		return errors.New("jwt.secret_key is required")
	}
	if len(secretKey) < jwtSecretKeyMinLength {
		return fmt.Errorf("jwt.secret_key must be at least %d characters", jwtSecretKeyMinLength)
	}
	return nil
}

func (c Config) OrbitRoot() string {
	return filepath.Clean(c.Orbit.Root)
}

func (c Config) SQLitePath() string {
	return filepath.Clean(c.Database.SQLite.Path)
}

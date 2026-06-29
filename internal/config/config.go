package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	mapstructure "github.com/go-viper/mapstructure/v2"
	"github.com/joho/godotenv"
	"github.com/spf13/viper"
)

const (
	DefaultConfigFile     = "configs/config.yaml"
	jwtSecretKeyMinLength = 32
)

func EnvConfigFile(env string) string {
	return fmt.Sprintf("configs/config.%s.yaml", env)
}

type Config struct {
	App         AppConfig       `mapstructure:"app" yaml:"app"`
	Server      ServerConfig    `mapstructure:"server" yaml:"server"`
	Logging     LoggingConfig   `mapstructure:"logging" yaml:"logging"`
	Database    DatabaseConfig  `mapstructure:"database" yaml:"database"`
	Worker      WorkerConfig    `mapstructure:"worker" yaml:"worker"`
	Orbit       OrbitConfig     `mapstructure:"orbit" yaml:"orbit"`
	JWT         JWTConfig       `mapstructure:"jwt" yaml:"jwt"`
	Traefik     TraefikConfig   `mapstructure:"traefik" yaml:"traefik"`
	Turnstile   TurnstileConfig `mapstructure:"turnstile" yaml:"turnstile"`
	Cert        CertConfig      `mapstructure:"cert" yaml:"cert"`
	Settings    SettingsConfig  `mapstructure:"settings" yaml:"settings"`
	EnvFilePath string          `mapstructure:"-" yaml:"-"`
}

type AppConfig struct {
	Name    string `mapstructure:"name" yaml:"name"`
	Version string `mapstructure:"version" yaml:"version"`
	Env     string `mapstructure:"env" yaml:"env"`
	Debug   bool   `mapstructure:"debug" yaml:"debug"`
}

type ServerConfig struct {
	Host string `mapstructure:"host" yaml:"host"`
	Port int    `mapstructure:"port" yaml:"port"`
}

type LoggingConfig struct {
	Level            string `mapstructure:"level" yaml:"level"`
	File             string `mapstructure:"file" yaml:"file"`
	MaxSizeMB        int    `mapstructure:"max_size_mb" yaml:"max_size_mb"`
	MaxBackups       int    `mapstructure:"max_backups" yaml:"max_backups"`
	HTTPBodyEnabled  bool   `mapstructure:"http_body_enabled" yaml:"http_body_enabled"`
	HTTPBodyMaxBytes int    `mapstructure:"http_body_max_bytes" yaml:"http_body_max_bytes"`
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
	DSN string `mapstructure:"dsn" yaml:"dsn"`
}

type WorkerConfig struct {
	Id            string        `mapstructure:"id" yaml:"id"`
	PollInterval  time.Duration `mapstructure:"poll_interval" yaml:"poll_interval"`
	LeaseDuration time.Duration `mapstructure:"lease_duration" yaml:"lease_duration"`
	MaxAttempts   int           `mapstructure:"max_attempts" yaml:"max_attempts"`
	Concurrency   int           `mapstructure:"concurrency" yaml:"concurrency"`
}

type OrbitConfig struct {
	Root string `mapstructure:"root" yaml:"root"`
}

type JWTConfig struct {
	SecretKey string `mapstructure:"secret_key" yaml:"secret_key"`
}

type TraefikConfig struct {
	APIURL          string `mapstructure:"api_url" yaml:"api_url"`
	DomainSuffix    string `mapstructure:"domain_suffix" yaml:"domain_suffix"`
	DynamicRouteDir string `mapstructure:"dynamic_route_dir" yaml:"dynamic_route_dir"`
	CertDir         string `mapstructure:"cert_dir" yaml:"cert_dir"`
	ContainerName   string `mapstructure:"container_name" yaml:"container_name"`
}

type TurnstileConfig struct {
	Enabled   bool   `mapstructure:"enabled" yaml:"enabled"`
	SiteKey   string `mapstructure:"site_key" yaml:"site_key"`
	SecretKey string `mapstructure:"secret_key" yaml:"secret_key"`
	VerifyURL string `mapstructure:"verify_url" yaml:"verify_url"`
}

type CertConfig struct {
	LetsEncrypt LetsEncryptConfig `mapstructure:"letsencrypt" yaml:"letsencrypt"`
}

type LetsEncryptConfig struct {
	Enabled     bool   `mapstructure:"enabled" yaml:"enabled"`
	Email       string `mapstructure:"email" yaml:"email"`
	Challenge   string `mapstructure:"challenge" yaml:"challenge"`
	DNSProvider string `mapstructure:"dns_provider" yaml:"dns_provider"`
}

type SettingsConfig struct {
	SecretKeys []string `mapstructure:"secret_keys" yaml:"secret_keys"`
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
	if err := loader.Unmarshal(&cfg, viper.DecodeHook(mapstructure.StringToTimeDurationHookFunc())); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

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
	return cfg, nil
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

func bindEnv(loader *viper.Viper) {
	keys := []string{
		"app.name",
		"app.version",
		"app.env",
		"app.debug",
		"server.host",
		"server.port",
		"logging.level",
		"logging.file",
		"logging.max_size_mb",
		"logging.max_backups",
		"logging.http_body_enabled",
		"logging.http_body_max_bytes",
		"database.driver",
		"database.sqlite.path",
		"database.mysql.dsn",
		"orbit.root",
		"jwt.secret_key",
		"traefik.api_url",
		"traefik.domain_suffix",
		"traefik.dynamic_route_dir",
		"traefik.cert_dir",
		"traefik.container_name",
		"turnstile.enabled",
		"turnstile.site_key",
		"turnstile.secret_key",
		"turnstile.verify_url",
		"cert.letsencrypt.enabled",
		"cert.letsencrypt.email",
		"cert.letsencrypt.challenge",
		"cert.letsencrypt.dns_provider",
		"worker.id",
		"worker.poll_interval",
		"worker.lease_duration",
		"worker.max_attempts",
		"worker.concurrency",
		"settings.secret_keys",
	}
	for _, key := range keys {
		envName := "POMELO_ORBIT_" + strings.ToUpper(strings.ReplaceAll(key, ".", "__"))
		_ = loader.BindEnv(key, envName)
	}
}

func (c Config) Validate() error {
	switch c.Database.Driver {
	case DatabaseDriverSQLite:
		if c.Database.SQLite.Path == "" {
			return errors.New("database.sqlite.path is required")
		}
	case DatabaseDriverMySQL:
		if c.Database.MySQL.DSN == "" {
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
	if c.Logging.HTTPBodyMaxBytes <= 0 {
		return errors.New("logging.http_body_max_bytes must be positive")
	}
	if err := validateJWTSecretKey(c.JWT.SecretKey); err != nil {
		return err
	}
	if c.Turnstile.Enabled {
		if strings.TrimSpace(c.Turnstile.SiteKey) == "" {
			return errors.New("turnstile.site_key is required when turnstile is enabled")
		}
		if strings.TrimSpace(c.Turnstile.SecretKey) == "" {
			return errors.New("turnstile.secret_key is required when turnstile is enabled")
		}
		if strings.TrimSpace(c.Turnstile.VerifyURL) == "" {
			return errors.New("turnstile.verify_url is required when turnstile is enabled")
		}
	}
	if c.Worker.PollInterval <= 0 {
		return errors.New("worker.poll_interval must be positive")
	}
	if c.Worker.LeaseDuration <= 0 {
		return errors.New("worker.lease_duration must be positive")
	}
	if c.Worker.MaxAttempts < 1 {
		return errors.New("worker.max_attempts must be at least 1")
	}
	if c.Worker.Concurrency < 1 {
		return errors.New("worker.concurrency must be at least 1")
	}
	return nil
}

func validateJWTSecretKey(secretKey string) error {
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

func (c Config) DataRoot() string {
	return filepath.Join(c.OrbitRoot(), "data")
}

func (c Config) SQLitePath() string {
	return filepath.Clean(c.Database.SQLite.Path)
}

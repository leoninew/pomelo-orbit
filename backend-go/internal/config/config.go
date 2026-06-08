package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

const defaultConfigFile = "config.defaults.yaml"

type Config struct {
	App      AppConfig      `yaml:"app"`
	Server   ServerConfig   `yaml:"server"`
	Logging  LoggingConfig  `yaml:"logging"`
	Database DatabaseConfig `yaml:"database"`
	Worker   WorkerConfig   `yaml:"worker"`
	Orbit    OrbitConfig    `yaml:"orbit"`
	JWT      JWTConfig      `yaml:"jwt"`
	Traefik  TraefikConfig  `yaml:"traefik"`
	Cert     CertConfig     `yaml:"cert"`
}

type AppConfig struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
	Debug   bool   `yaml:"debug"`
}

type ServerConfig struct {
	Host string `yaml:"host"`
	Port int    `yaml:"port"`
}

type LoggingConfig struct {
	Level string `yaml:"level"`
	File  string `yaml:"file"`
}

type DatabaseConfig struct {
	SQLite SQLiteConfig `yaml:"sqlite"`
}

type SQLiteConfig struct {
	Path string `yaml:"path"`
}

type WorkerConfig struct {
	Id            string        `yaml:"id"`
	PollInterval  time.Duration `yaml:"poll_interval"`
	LeaseDuration time.Duration `yaml:"lease_duration"`
	MaxAttempts   int           `yaml:"max_attempts"`
	Concurrency   int           `yaml:"concurrency"`
}

type OrbitConfig struct {
	Root string `yaml:"root"`
}

type JWTConfig struct {
	SecretKey string `yaml:"secret_key"`
}

type TraefikConfig struct {
	DomainSuffix string `yaml:"domain_suffix"`
}

type CertConfig struct {
	LetsEncrypt LetsEncryptConfig `yaml:"letsencrypt"`
}

type LetsEncryptConfig struct {
	Enabled     bool   `yaml:"enabled"`
	Email       string `yaml:"email"`
	Challenge   string `yaml:"challenge"`
	DNSProvider string `yaml:"dns_provider"`
}

func Load(path string) (Config, error) {
	if path == "" {
		path = defaultConfigFile
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return Config{}, fmt.Errorf("parse config: %w", err)
	}

	applyEnv(&cfg)
	if cfg.Worker.Id == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return Config{}, fmt.Errorf("get hostname: %w", err)
		}
		cfg.Worker.Id = fmt.Sprintf("%s-%d", hostname, os.Getpid())
	}

	if cfg.Worker.PollInterval == 0 {
		cfg.Worker.PollInterval = time.Second
	}
	if cfg.Worker.LeaseDuration == 0 {
		cfg.Worker.LeaseDuration = 5 * time.Minute
	}
	if cfg.Worker.MaxAttempts == 0 {
		cfg.Worker.MaxAttempts = 3
	}
	if cfg.Worker.Concurrency == 0 {
		cfg.Worker.Concurrency = 1
	}

	if cfg.Orbit.Root == "" {
		cfg.Orbit.Root = "."
	}
	if cfg.Database.SQLite.Path == "" {
		cfg.Database.SQLite.Path = filepath.Join(cfg.Orbit.Root, "data", "db", "pomelo-orbit.db")
	}

	if err := cfg.Validate(); err != nil {
		return Config{}, err
	}
	return cfg, nil
}

func (c Config) Validate() error {
	if c.Database.SQLite.Path == "" {
		return errors.New("database.sqlite.path is required")
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

func (c Config) OrbitRoot() string {
	return filepath.Clean(c.Orbit.Root)
}

func (c Config) DataRoot() string {
	return filepath.Join(c.OrbitRoot(), "data")
}

func (c Config) SQLitePath() string {
	return filepath.Clean(c.Database.SQLite.Path)
}

func applyEnv(cfg *Config) {
	if v := os.Getenv("POMELO_ORBIT_BACKEND__LOGGING__LEVEL"); v != "" {
		cfg.Logging.Level = v
	}
	if v := os.Getenv("POMELO_ORBIT_BACKEND__DATABASE__SQLITE__PATH"); v != "" {
		cfg.Database.SQLite.Path = v
	} else if v := os.Getenv("POMELO_ORBIT_DATABASE__SQLITE__PATH"); v != "" {
		cfg.Database.SQLite.Path = v
	}
	if v := os.Getenv("POMELO_ORBIT_JWT__SECRET_KEY"); v != "" {
		cfg.JWT.SecretKey = v
	}
	if v := os.Getenv("POMELO_ORBIT_BACKEND__ORBIT__ROOT"); v != "" {
		cfg.Orbit.Root = v
	}
	if v := os.Getenv("POMELO_ORBIT_BACKEND__WORKER__ID"); v != "" {
		cfg.Worker.Id = v
	}
}

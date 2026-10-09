package config

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"strings"

	"github.com/joho/godotenv"
	"github.com/spf13/viper"

	"github.com/leoninew/pomelo-orbit/internal/common/envfile"
)

const OverrideFile = "overwrite.env"

var profilePattern = regexp.MustCompile(`^[A-Za-z0-9_-]*$`)

type Resolved struct {
	Config          Config
	Baseline        map[string]any
	BaselineSources map[string]string
	Sources         map[string]string
}

type Runtime struct {
	cwd         string
	environment map[string]string
	initial     Config
	effective   Config
	sources     map[string]string
}

func loadRuntime() (Config, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return Config{}, fmt.Errorf("get working directory: %w", err)
	}
	runtime := &Runtime{cwd: cwd, environment: make(map[string]string)}
	for _, entry := range os.Environ() {
		key, value, ok := strings.Cut(entry, "=")
		if ok {
			runtime.environment[key] = value
		}
	}
	overrides, err := envfile.NewStore(runtime.OverridePath()).Load(context.Background())
	if err != nil {
		return Config{}, fmt.Errorf("read overwrite.env: %w", err)
	}
	resolved, err := runtime.Resolve(overrides)
	if err != nil {
		return Config{}, err
	}
	runtime.initial = cloneConfig(resolved.Config)
	cfg := cloneConfig(resolved.Config)
	if cfg.Worker.Id == "" {
		hostname, err := os.Hostname()
		if err != nil {
			return Config{}, fmt.Errorf("get hostname: %w", err)
		}
		cfg.Worker.Id = fmt.Sprintf("%s-%d", hostname, os.Getpid())
		resolved.Sources["worker__id"] = "derived"
	}
	runtime.effective = cloneConfig(cfg)
	runtime.sources = resolved.Sources
	cfg.Runtime = runtime
	return cfg, nil
}

func (r *Runtime) OverridePath() string {
	return filepath.Join(r.cwd, OverrideFile)
}

func (r *Runtime) Effective() Config {
	return cloneConfig(r.effective)
}

func (r *Runtime) Initial() Config {
	return cloneConfig(r.initial)
}

func (r *Runtime) Source(key string) string {
	return r.sources[key]
}

func (r *Runtime) Resolve(overrides map[string]string) (Resolved, error) {
	fields := Fields()
	known := make(map[string]bool, len(fields))
	for _, field := range fields {
		known[field.EnvName] = true
	}
	for key := range overrides {
		if !known[key] {
			return Resolved{}, fmt.Errorf("unknown configuration key %s", key)
		}
	}
	profile := strings.TrimSpace(r.environment[EnvPrefix+"APP__ENV"])
	if value, ok := overrides[EnvPrefix+"APP__ENV"]; ok {
		profile = strings.TrimSpace(value)
	}
	if !profilePattern.MatchString(profile) {
		return Resolved{}, errors.New("app.env must be a profile name without path separators")
	}
	loader := viper.New()
	loader.SetConfigType("yaml")
	loader.SetConfigFile(filepath.Join(r.cwd, DefaultConfigFile))
	if err := loader.ReadInConfig(); err != nil {
		return Resolved{}, fmt.Errorf("read base config: %w", err)
	}
	if profile != "" {
		loader.SetConfigFile(filepath.Join(r.cwd, EnvConfigFile(profile)))
		if err := loader.MergeInConfig(); err != nil && !isOptionalConfigMissing(err) {
			return Resolved{}, fmt.Errorf("read env config: %w", err)
		}
	}
	dotenvPath := filepath.Join(r.cwd, ".env")
	if profile != "" {
		dotenvPath += "." + profile
	}
	dotenv, err := godotenv.Read(dotenvPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return Resolved{}, errors.New("read env file: invalid or unreadable startup dotenv")
	}
	resolved := Resolved{Baseline: make(map[string]any), BaselineSources: make(map[string]string), Sources: make(map[string]string)}
	var parseErrors []error
	for _, field := range fields {
		value := loader.Get(field.Path)
		if value == nil {
			value = field.Value(Config{})
		}
		raw, baseErr := field.Encode(value)
		source := "builtin"
		if value, ok := dotenv[field.EnvName]; ok {
			raw, baseErr, source = value, nil, "dotenv"
		}
		if value, ok := r.environment[field.EnvName]; ok {
			raw, baseErr, source = value, nil, "env"
		}
		baseline, err := field.ParseEnv(raw)
		if baseErr == nil && err == nil {
			if duration, ok := baseline.(interface{ String() string }); ok && field.Type == "duration" {
				baseline = duration.String()
			}
			resolved.Baseline[field.Key] = baseline
		}
		resolved.BaselineSources[field.Key] = source
		if override, ok := overrides[field.EnvName]; ok {
			raw, baseErr, source = override, nil, "override"
		}
		typed, err := field.ParseEnv(raw)
		if baseErr != nil || err != nil {
			parseErrors = append(parseErrors, fmt.Errorf("%s requires %s", field.Path, field.Type))
			continue
		}
		loader.Set(field.Path, typed)
		resolved.Sources[field.Key] = source
	}
	if len(parseErrors) > 0 {
		return resolved, errors.Join(parseErrors...)
	}
	if err := loader.Unmarshal(&resolved.Config, configDecodeHook()); err != nil {
		return resolved, errors.New("parse configuration failed")
	}
	if err := r.normalize(&resolved.Config); err != nil {
		return resolved, err
	}
	return resolved, resolved.Config.Validate()
}

func (r *Runtime) normalize(cfg *Config) error {
	normalizeServerRuntimeOriginConfig(&cfg.Server)
	normalizeLogHTTPConfig(&cfg.Logging.HTTP)
	orbitRoot := cfg.OrbitRoot()
	if !filepath.IsAbs(orbitRoot) {
		orbitRoot = filepath.Join(r.cwd, orbitRoot)
	}
	if err := normalizeLoggingConfig(&cfg.Logging, orbitRoot); err != nil {
		return err
	}
	if err := normalizeWorkspaceConfig(&cfg.Workspace, orbitRoot); err != nil {
		return err
	}
	return normalizeProjectInitializationConfig(&cfg.ProjectInitialization)
}

func cloneConfig(cfg Config) Config {
	clone := cfg
	clone.Runtime = nil
	for _, field := range configFields() {
		value := reflect.ValueOf(cfg).FieldByIndex(field.index)
		if value.Kind() == reflect.Slice {
			copy := reflect.MakeSlice(value.Type(), value.Len(), value.Len())
			reflect.Copy(copy, value)
			reflect.ValueOf(&clone).Elem().FieldByIndex(field.index).Set(copy)
		}
	}
	return clone
}

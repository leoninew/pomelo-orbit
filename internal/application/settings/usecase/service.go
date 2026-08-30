package settingssvc

import (
	"context"
	"fmt"
	"sort"
	"strconv"
	"strings"

	settingsdto "github.com/leoninew/pomelo-orbit/internal/application/settings/dto"
	settingsport "github.com/leoninew/pomelo-orbit/internal/application/settings/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
)

const envPrefix = "POMELO_ORBIT_"

type Service struct {
	cfg      config.Config
	envStore settingsport.EnvStore
}

func New(cfg config.Config, envStore settingsport.EnvStore) Service {
	return Service{cfg: cfg, envStore: envStore}
}

func (s Service) Config(ctx context.Context) (settingsdto.SystemConfig, error) {
	envMap, err := s.envStore.Load(ctx)
	if err != nil {
		return settingsdto.SystemConfig{}, err
	}
	definitions := settingDefinitions(s.baseConfig())
	definitionByKey := make(map[string]settingsdto.Definition, len(definitions))
	orderedKeys := make([]string, 0, len(definitions))
	for _, definition := range definitions {
		definitionByKey[definition.Key] = definition
		orderedKeys = append(orderedKeys, definition.Key)
	}
	seen := map[string]struct{}{}
	for _, key := range orderedKeys {
		seen[key] = struct{}{}
	}
	var extraKeys []string
	for envKey := range envMap {
		key, ok := settingKeyFromEnv(envKey)
		if ok {
			if _, exists := seen[key]; !exists {
				extraKeys = append(extraKeys, key)
			}
		}
	}
	sort.Strings(extraKeys)
	orderedKeys = append(orderedKeys, extraKeys...)

	items := make([]settingsdto.ConfigItem, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		definition, hasDefinition := definitionByKey[key]
		envKey := settingEnvKey(key)
		rawValue, overridden := envMap[envKey]
		defaultValue := any("")
		description := ""
		secret := false
		if hasDefinition {
			defaultValue = definition.Default
			description = settingDescription(definition.Description)
			secret = definition.Secret
		}
		value := defaultValue
		if overridden {
			value = parseSettingValue(rawValue, defaultValue)
		}
		items = append(items, settingsdto.ConfigItem{Key: key, Value: value, Default: defaultValue, IsOverridden: overridden, Secret: secret, Description: description})
	}
	return settingsdto.SystemConfig{Items: items}, nil
}

func (s Service) Update(ctx context.Context, key string, value any) (settingsdto.SystemConfig, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return settingsdto.SystemConfig{}, apperror.New(apperror.KindValidation, "key is required")
	}
	if err := s.envStore.Set(ctx, map[string]string{settingEnvKey(key): settingValueString(value)}); err != nil {
		return settingsdto.SystemConfig{}, err
	}
	return s.Config(ctx)
}

func (s Service) Reset(ctx context.Context, keys []string) (settingsdto.SystemConfig, error) {
	envKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			envKey := settingEnvKey(key)
			envKeys = append(envKeys, envKey)
		}
	}
	if err := s.envStore.Delete(ctx, envKeys); err != nil {
		return settingsdto.SystemConfig{}, err
	}
	return s.Config(ctx)
}

func (s Service) baseConfig() config.Config {
	if s.cfg.Base != nil {
		return *s.cfg.Base
	}
	return s.cfg
}

func settingDefinitions(cfg config.Config) []settingsdto.Definition {
	secretKeys := map[string]struct{}{}
	for _, key := range cfg.Settings.SecretKeys {
		key = strings.TrimSpace(key)
		if key != "" {
			secretKeys[key] = struct{}{}
		}
	}
	markSecret := func(definition settingsdto.Definition) settingsdto.Definition {
		_, definition.Secret = secretKeys[definition.Key]
		return definition
	}
	definitions := []settingsdto.Definition{
		{Key: "server__host", Default: cfg.Server.Host, Description: "HTTP server bind host"},
		{Key: "server__port", Default: cfg.Server.Port, Description: "HTTP server bind port"},
		{Key: "logging__level", Default: cfg.Logging.Level, Description: "Application log level"},
		{Key: "logging__file", Default: cfg.Logging.File, Description: "Backend log file path"},
		{Key: "logging__max_size_mb", Default: cfg.Logging.MaxSizeMB, Description: "Maximum size of one log file in MB"},
		{Key: "logging__max_backups", Default: cfg.Logging.MaxBackups, Description: "Maximum number of rotated log files"},
		{Key: "logging__http__enabled", Default: cfg.Logging.HTTP.Enabled, Description: "Enable HTTP access logging"},
		{Key: "logging__http__request_body_limit", Default: cfg.Logging.HTTP.RequestBodyLimit, Description: "Maximum HTTP request body bytes to log; 0 disables body logging"},
		{Key: "logging__http__response_body_limit", Default: cfg.Logging.HTTP.ResponseBodyLimit, Description: "Maximum HTTP response body bytes to log; 0 disables body logging"},
		{Key: "logging__http__skip_asset_enabled", Default: cfg.Logging.HTTP.SkipAssetEnabled, Description: "Skip successful HTTP access logs for /assets"},
		{Key: "database__driver", Default: cfg.Database.Driver, Description: "Database driver"},
		{Key: "database__sqlite__path", Default: cfg.Database.SQLite.Path, Description: "SQLite database file path"},
		{Key: "database__mysql__dsn", Default: cfg.Database.MySQL.Dsn, Description: "MySQL DSN"},
		{Key: "database__postgres__dsn", Default: cfg.Database.Postgres.Dsn, Description: "PostgreSQL DSN"},
		{Key: "jwt__secret_key", Default: cfg.Jwt.SecretKey, Description: "JWT signing secret"},
		{Key: "traefik__image", Default: cfg.Traefik.Image, Description: "Managed gateway container image pin"},
		{Key: "traefik__rest_api_url", Default: cfg.Traefik.RestApiUrl, Description: "Default Traefik REST control-plane URL for new gateways"},
		{Key: "traefik__base_domain", Default: cfg.Traefik.BaseDomain, Description: "Default gateway base domain for new gateways"},
		{Key: "traefik__rest_ready_timeout", Default: cfg.Traefik.RestReadyTimeout.String(), Description: "How long deploy waits for Traefik REST API readiness"},
		{Key: "turnstile__enabled", Default: cfg.Turnstile.Enabled, Description: "Enable Cloudflare Turnstile verification"},
		{Key: "turnstile__site_key", Default: cfg.Turnstile.SiteKey, Description: "Cloudflare Turnstile site key"},
		{Key: "turnstile__secret_key", Default: cfg.Turnstile.SecretKey, Description: "Cloudflare Turnstile secret key"},
		{Key: "turnstile__verify_url", Default: cfg.Turnstile.VerifyUrl, Description: "Cloudflare Turnstile siteverify URL"},
		{Key: "settings__secret_keys", Default: cfg.Settings.SecretKeys, Description: "Settings fields marked as secret"},
		{Key: "pipeline_run__execution_timeout", Default: cfg.PipelineRun.ExecutionTimeout.String(), Description: "Maximum pipeline execution duration"},
		{Key: "worker__id", Default: cfg.Worker.Id, Description: "Background worker ID"},
		{Key: "worker__poll_interval", Default: cfg.Worker.PollInterval.String(), Description: "Background worker poll interval"},
		{Key: "worker__lease_duration", Default: cfg.Worker.LeaseDuration.String(), Description: "Background task lease duration"},
		{Key: "worker__max_attempts", Default: cfg.Worker.MaxAttempts, Description: "Attempts frozen into each new background task"},
		{Key: "worker__concurrency", Default: cfg.Worker.Concurrency, Description: "Background worker concurrency"},
		{Key: "llm__max_tool_call_rounds", Default: cfg.LLM.MaxToolCallRounds, Description: "Maximum LLM-to-MCP tool-call rounds per deployment dialogue turn"},
	}
	for index := range definitions {
		definitions[index] = markSecret(definitions[index])
	}
	return definitions
}

func settingDescription(description string) string {
	if description == "" {
		return "Requires pomelo-orbit restart to take effect"
	}
	return description + " (requires pomelo-orbit restart to take effect)"
}

func settingEnvKey(key string) string {
	return envPrefix + strings.ToUpper(key)
}

func settingKeyFromEnv(envKey string) (string, bool) {
	if !strings.HasPrefix(envKey, envPrefix) {
		return "", false
	}
	return strings.ToLower(strings.TrimPrefix(envKey, envPrefix)), true
}

func settingValueString(value any) string {
	switch typed := value.(type) {
	case bool:
		return strconv.FormatBool(typed)
	case float64:
		if typed == float64(int64(typed)) {
			return strconv.FormatInt(int64(typed), 10)
		}
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case string:
		return typed
	case []string:
		return strings.Join(typed, ",")
	default:
		return fmt.Sprint(typed)
	}
}

func parseSettingValue(raw string, defaultValue any) any {
	switch defaultValue.(type) {
	case bool:
		return strings.EqualFold(raw, "true")
	case int:
		if parsed, err := strconv.Atoi(raw); err == nil {
			return parsed
		}
	case []string:
		parts := strings.Split(raw, ",")
		values := make([]string, 0, len(parts))
		for _, part := range parts {
			part = strings.TrimSpace(part)
			if part != "" {
				values = append(values, part)
			}
		}
		return values
	}
	if strings.EqualFold(raw, "true") || strings.EqualFold(raw, "false") {
		return strings.EqualFold(raw, "true")
	}
	return raw
}

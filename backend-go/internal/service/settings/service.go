package settingssvc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"backend/internal/apperror"
	"backend/internal/config"
)

const envPrefix = "POMELO_ORBIT_BACKEND__"

type ConfigItem struct {
	Key          string `json:"key"`
	Value        any    `json:"value"`
	Default      any    `json:"default"`
	IsOverridden bool   `json:"is_overridden"`
	Description  string `json:"description,omitempty"`
}

type SystemConfig struct {
	Items []ConfigItem `json:"items"`
}

type Definition struct {
	Key         string
	Default     any
	Description string
	Secret      bool
}

type Service struct {
	cfg config.Config
}

func New(cfg config.Config) Service {
	return Service{cfg: cfg}
}

func (s Service) Config(context.Context) (SystemConfig, error) {
	envMap, err := readEnvFile(s.envPath())
	if err != nil {
		return SystemConfig{}, err
	}
	definitions := settingDefinitions(s.cfg)
	definitionByKey := make(map[string]Definition, len(definitions))
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

	items := make([]ConfigItem, 0, len(orderedKeys))
	for _, key := range orderedKeys {
		definition, hasDefinition := definitionByKey[key]
		envKey := settingEnvKey(key)
		rawValue, overridden := envMap[envKey]
		defaultValue := any("")
		description := ""
		secret := false
		if hasDefinition {
			defaultValue = definition.Default
			description = definition.Description
			secret = definition.Secret
		}
		value := defaultValue
		if overridden {
			value = parseSettingValue(rawValue, defaultValue)
		}
		if secret && valueString(value) != "" {
			value = "****"
		}
		items = append(items, ConfigItem{Key: key, Value: value, Default: defaultValue, IsOverridden: overridden, Description: description})
	}
	return SystemConfig{Items: items}, nil
}

func (s Service) Update(ctx context.Context, key string, value any) (SystemConfig, error) {
	key = strings.TrimSpace(key)
	if key == "" {
		return SystemConfig{}, apperror.New(apperror.KindValidation, "key is required")
	}
	if err := writeEnvValues(s.envPath(), map[string]string{settingEnvKey(key): settingValueString(value)}); err != nil {
		return SystemConfig{}, err
	}
	return s.Config(ctx)
}

func (s Service) Reset(ctx context.Context, keys []string) (SystemConfig, error) {
	envKeys := make([]string, 0, len(keys))
	for _, key := range keys {
		key = strings.TrimSpace(key)
		if key != "" {
			envKeys = append(envKeys, settingEnvKey(key))
		}
	}
	if err := deleteEnvValues(s.envPath(), envKeys); err != nil {
		return SystemConfig{}, err
	}
	return s.Config(ctx)
}

func (s Service) envPath() string {
	return filepath.Join(s.cfg.OrbitRoot(), ".env")
}

func settingDefinitions(cfg config.Config) []Definition {
	return []Definition{
		{Key: "server__host", Default: cfg.Server.Host, Description: "HTTP server bind host"},
		{Key: "server__port", Default: cfg.Server.Port, Description: "HTTP server bind port"},
		{Key: "logging__level", Default: cfg.Logging.Level, Description: "Application log level"},
		{Key: "logging__file", Default: cfg.Logging.File, Description: "Backend log file path"},
		{Key: "logging__max_size_mb", Default: cfg.Logging.MaxSizeMB, Description: "Maximum size of one log file in MB"},
		{Key: "logging__max_backups", Default: cfg.Logging.MaxBackups, Description: "Maximum number of rotated log files"},
		{Key: "database__driver", Default: cfg.Database.Driver, Description: "Database driver"},
		{Key: "database__sqlite__path", Default: cfg.Database.SQLite.Path, Description: "SQLite database file path"},
		{Key: "database__mysql__dsn", Default: cfg.Database.MySQL.DSN, Description: "MySQL DSN", Secret: true},
		{Key: "jwt__secret_key", Default: cfg.JWT.SecretKey, Description: "JWT signing secret", Secret: true},
		{Key: "traefik__domain_suffix", Default: cfg.Traefik.DomainSuffix, Description: "Default Traefik domain suffix"},
		{Key: "turnstile__enabled", Default: cfg.Turnstile.Enabled, Description: "Enable Cloudflare Turnstile verification"},
		{Key: "turnstile__site_key", Default: cfg.Turnstile.SiteKey, Description: "Cloudflare Turnstile site key"},
		{Key: "turnstile__secret_key", Default: cfg.Turnstile.SecretKey, Description: "Cloudflare Turnstile secret key", Secret: true},
		{Key: "turnstile__verify_url", Default: cfg.Turnstile.VerifyURL, Description: "Cloudflare Turnstile siteverify URL"},
		{Key: "cert__letsencrypt__enabled", Default: cfg.Cert.LetsEncrypt.Enabled, Description: "Enable Let's Encrypt certificates"},
		{Key: "cert__letsencrypt__email", Default: cfg.Cert.LetsEncrypt.Email, Description: "Let's Encrypt account email"},
		{Key: "cert__letsencrypt__challenge", Default: cfg.Cert.LetsEncrypt.Challenge, Description: "Let's Encrypt challenge type"},
		{Key: "cert__letsencrypt__dns_provider", Default: cfg.Cert.LetsEncrypt.DNSProvider, Description: "Let's Encrypt DNS challenge provider"},
		{Key: "worker__id", Default: cfg.Worker.Id, Description: "Background worker ID"},
		{Key: "worker__poll_interval", Default: cfg.Worker.PollInterval.String(), Description: "Background worker poll interval"},
		{Key: "worker__lease_duration", Default: cfg.Worker.LeaseDuration.String(), Description: "Background task lease duration"},
		{Key: "worker__max_attempts", Default: cfg.Worker.MaxAttempts, Description: "Default task max attempts"},
		{Key: "worker__concurrency", Default: cfg.Worker.Concurrency, Description: "Background worker concurrency"},
	}
}

func readEnvFile(path string) (map[string]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return map[string]string{}, nil
		}
		return nil, err
	}
	result := map[string]string{}
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		result[key] = strings.Trim(value, `"'`)
	}
	return result, nil
}

func writeEnvValues(path string, values map[string]string) error {
	envMap, err := readEnvFile(path)
	if err != nil {
		return err
	}
	for key, value := range values {
		envMap[key] = value
	}
	return writeEnvFile(path, envMap)
}

func deleteEnvValues(path string, keys []string) error {
	envMap, err := readEnvFile(path)
	if err != nil {
		return err
	}
	for _, key := range keys {
		delete(envMap, key)
	}
	return writeEnvFile(path, envMap)
}

func writeEnvFile(path string, envMap map[string]string) error {
	keys := make([]string, 0, len(envMap))
	for key := range envMap {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	lines := make([]string, 0, len(keys))
	for _, key := range keys {
		lines = append(lines, fmt.Sprintf("%s=%s", key, envMap[key]))
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	content := strings.Join(lines, "\n")
	if content != "" {
		content += "\n"
	}
	return os.WriteFile(path, []byte(content), 0o644)
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
	}
	if strings.EqualFold(raw, "true") || strings.EqualFold(raw, "false") {
		return strings.EqualFold(raw, "true")
	}
	return raw
}

func valueString(value any) string {
	if value == nil {
		return ""
	}
	return fmt.Sprint(value)
}

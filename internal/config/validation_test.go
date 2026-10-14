package config

import (
	"strings"
	"testing"
	"time"
)

func TestValidateLLMRejectsNonPositiveToolCallRounds(t *testing.T) {
	if err := validateLLMConfig(LLMConfig{MaxToolCallRounds: 0}); err == nil || !strings.Contains(err.Error(), "llm.max_tool_call_rounds") {
		t.Fatalf("expected tool round limit validation, got %v", err)
	}
}

func TestValidateRouteRejectsNonPositiveStageTimeouts(t *testing.T) {
	for _, test := range []struct {
		field string
		clear func(*RouteConfig)
	}{
		{"gateway_lock_timeout", func(cfg *RouteConfig) { cfg.GatewayLockTimeout = 0 }},
		{"state_load_timeout", func(cfg *RouteConfig) { cfg.StateLoadTimeout = 0 }},
		{"file_publication_timeout", func(cfg *RouteConfig) { cfg.FilePublicationTimeout = 0 }},
		{"configuration_match_timeout", func(cfg *RouteConfig) { cfg.ConfigurationMatchTimeout = 0 }},
	} {
		t.Run(test.field, func(t *testing.T) {
			cfg := RouteConfig{GatewayLockTimeout: time.Second, StateLoadTimeout: time.Second, FilePublicationTimeout: time.Second, ApiRequestTimeout: time.Second, ReloadTimeout: time.Second, ConfigurationMatchTimeout: time.Second, RecoveryTimeout: time.Second}
			test.clear(&cfg)
			if err := validateRouteConfig(cfg); err == nil || !strings.Contains(err.Error(), "route."+test.field+" must be positive") {
				t.Fatalf("expected stage timeout validation, got %v", err)
			}
		})
	}
}

func TestValidateApiPathPrefixes(t *testing.T) {
	for _, test := range []struct {
		name     string
		prefixes []string
		want     string
	}{
		{"empty", nil, "must not be empty"},
		{"root", []string{"/"}, "must not contain root path"},
		{"missing slash", []string{"api"}, "must start with /"},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateApiPathPrefixes(test.prefixes); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q, got %v", test.want, err)
			}
		})
	}
}

func TestValidateServerRejectsInvalidPublicUrl(t *testing.T) {
	cfg := ServerConfig{ApiPathPrefixes: []string{"/api"}, PublicUrl: "ftp://orbit.example.test"}
	if err := validateServerRuntimeOriginConfig(cfg); err == nil || !strings.Contains(err.Error(), "server.public_url must use http or https scheme") {
		t.Fatalf("expected public URL validation, got %v", err)
	}
}

func TestValidateServerRejectsInvalidCorsOrigin(t *testing.T) {
	cfg := ServerConfig{ApiPathPrefixes: []string{"/api"}, CorsAllowedOrigins: []string{"https://orbit.example.test/path"}}
	if err := validateServerRuntimeOriginConfig(cfg); err == nil || !strings.Contains(err.Error(), "server.cors_allowed_origins must not include path") {
		t.Fatalf("expected CORS origin validation, got %v", err)
	}
}

func TestValidateJwtSecretKeyRejectsInvalidFernetKeys(t *testing.T) {
	for _, test := range []struct{ name, value string }{
		{"invalid encoding", strings.Repeat("!", 44)},
		{"wrong decoded length", strings.Repeat("A", 40)},
		{"standard Base64", "J69d31L/5Dg4yhJXhp+CacFovpi8Ikhr34zSsHmE3x4="},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := validateJwtSecretKey(test.value); err == nil {
				t.Fatal("accepted an invalid Fernet key")
			}
		})
	}
}

package config

import (
	"os"
	"strings"
	"testing"
)

func TestGoogleConfigurationLoadsThroughRuntime(t *testing.T) {
	defaults, err := os.ReadFile("../../configs/config.yaml")
	if err != nil {
		t.Fatal(err)
	}
	setupDefaultConfig(t)
	if err := os.WriteFile(DefaultConfigFile, defaults, 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POMELO_ORBIT_JWT__SECRET_KEY", testJwtSecret)
	t.Setenv("POMELO_ORBIT_GOOGLE__CLIENT_ID", "google-client")
	t.Setenv("POMELO_ORBIT_GOOGLE__CLIENT_SECRET", "secret-value")
	t.Setenv("POMELO_ORBIT_GOOGLE__REDIRECT_URI", "http://localhost:9020/google/callback")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Google.ClientId != "google-client" || cfg.Google.ClientSecret != "secret-value" || cfg.Google.RedirectUri != "http://localhost:9020/google/callback" {
		t.Fatal("Google configuration was not loaded")
	}
	for _, key := range cfg.Settings.SecretKeys {
		if key == "google__client_secret" {
			return
		}
	}
	t.Fatal("Google secret must be masked by system settings")
}

func TestGoogleConfigurationValidation(t *testing.T) {
	for _, test := range []struct {
		name  string
		cfg   GoogleConfig
		valid bool
	}{
		{"disabled", GoogleConfig{}, true},
		{"partial", GoogleConfig{ClientId: "client"}, false},
		{"production", GoogleConfig{"client", "secret-value", "https://orbit.example/google/callback"}, true},
		{"local", GoogleConfig{"client", "secret-value", "http://localhost:9020/google/callback"}, true},
		{"insecure", GoogleConfig{"client", "secret-value", "http://orbit.example/google/callback"}, false},
		{"wrong path", GoogleConfig{"client", "secret-value", "https://orbit.example/api/auth/google/callback"}, false},
		{"query", GoogleConfig{"client", "secret-value", "https://orbit.example/google/callback?redirect=//example.com"}, false},
	} {
		t.Run(test.name, func(t *testing.T) {
			err := validateGoogleConfig(test.cfg)
			if (err == nil) != test.valid {
				t.Fatalf("valid=%v error=%v", test.valid, err)
			}
			if err != nil && strings.Contains(err.Error(), "secret-value") {
				t.Fatal("configuration error exposed the secret")
			}
		})
	}
}

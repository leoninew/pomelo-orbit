package config

import (
	"os"
	"reflect"
	"testing"
)

func TestOverwritePrecedenceResetAndEnvironmentIsolation(t *testing.T) {
	setupDefaultConfig(t)
	t.Setenv("POMELO_ORBIT_SERVER__PORT", "8080")
	writeDotEnv(t, "POMELO_ORBIT_SERVER__PORT=7070\n")
	if err := os.WriteFile(OverrideFile, []byte("POMELO_ORBIT_SERVER__PORT=9090\nPOMELO_ORBIT_APP__DEBUG=false\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 9090 || os.Getenv("POMELO_ORBIT_SERVER__PORT") != "8080" {
		t.Fatal("overwrite did not take precedence without changing ENV")
	}
	if cfg.Runtime.Source("server__port") != "override" {
		t.Fatal("incorrect source")
	}
	if err := os.Remove(OverrideFile); err != nil {
		t.Fatal(err)
	}
	restarted, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if restarted.Server.Port != 8080 || cfg.Server.Port != 9090 {
		t.Fatal("reset or running snapshot changed incorrectly")
	}
}

func TestOverwriteSelectsProfileWithoutMovingFile(t *testing.T) {
	setupDefaultConfig(t)
	writeEnvConfig(t, "development", "server:\n  port: 7000\n")
	writeEnvConfig(t, "release", "server:\n  port: 8000\n")
	t.Setenv("POMELO_ORBIT_APP__ENV", "development")
	if err := os.WriteFile(OverrideFile, []byte("POMELO_ORBIT_APP__ENV=release\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Server.Port != 8000 || cfg.App.Env != "release" {
		t.Fatalf("profile = %s, port = %d", cfg.App.Env, cfg.Server.Port)
	}
	path := cfg.Runtime.OverridePath()
	next, err := cfg.Runtime.Resolve(map[string]string{"POMELO_ORBIT_WORKSPACE__ROOT": "different-data", "POMELO_ORBIT_ORBIT__ROOT": "different-root"})
	if err != nil {
		t.Fatal(err)
	}
	if next.Config.Workspace.Root == cfg.Workspace.Root || path != cfg.Runtime.OverridePath() {
		t.Fatal("override location moved with application config")
	}
}

func TestFieldDiscoveryIncludesNewNestedFieldsWithoutRegistration(t *testing.T) {
	type nested struct {
		Added string `mapstructure:"added"`
	}
	type fixture struct {
		Group    nested `mapstructure:"group"`
		Metadata string `mapstructure:"-"`
	}
	fields := discoverFields(reflect.TypeFor[fixture](), nil, nil)
	if len(fields) != 1 || fields[0].EnvName != "POMELO_ORBIT_GROUP__ADDED" || fields[0].Key != "group__added" {
		t.Fatalf("fields = %#v", fields)
	}
	setupDefaultConfig(t)
	t.Setenv("POMELO_ORBIT_PROJECT_INITIALIZATION__GATEWAY__REST_API_HOST_URL", "http://localhost:9999")
	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.ProjectInitialization.Gateway.RestApiHostUrl != "http://localhost:9999" {
		t.Fatal("previously unbound field was omitted")
	}
}

func TestStringListEncodingPreservesCommaAndEmptyList(t *testing.T) {
	for _, field := range Fields() {
		if field.Type != "string_list" {
			continue
		}
		for _, values := range [][]string{{}, {""}, {"a,b", "c"}} {
			raw, err := field.Encode(values)
			if err != nil {
				t.Fatal(err)
			}
			parsed, err := field.ParseEnv(raw)
			if err != nil || !reflect.DeepEqual(parsed, values) {
				t.Fatalf("list = %#v, %v", parsed, err)
			}
		}
		return
	}
	t.Fatal("no string list field discovered")
}

package settingssvc

import (
	"context"
	"encoding/base64"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	settingsdto "github.com/leoninew/pomelo-orbit/internal/application/settings/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/envfile"
)

func settingsFixture(t *testing.T) (config.Config, Service) {
	t.Helper()
	content, err := os.ReadFile(filepath.Join("..", "..", "..", "..", "configs", "config.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range os.Environ() {
		key, value, _ := strings.Cut(entry, "=")
		if strings.HasPrefix(key, config.EnvPrefix) {
			t.Setenv(key, value)
			if err := os.Unsetenv(key); err != nil {
				t.Fatal(err)
			}
		}
	}
	t.Chdir(t.TempDir())
	if err := os.MkdirAll("configs", 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(config.DefaultConfigFile, content, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(".env", []byte("POMELO_ORBIT_LOGGING__LEVEL=error\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("POMELO_ORBIT_JWT__SECRET_KEY", "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	t.Setenv("POMELO_ORBIT_LOGGING__LEVEL", "warn")
	t.Setenv("POMELO_ORBIT_WORKER__ID", "")
	cfg, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	return cfg, New(cfg.Runtime, envfile.NewStore(cfg.Runtime.OverridePath()))
}

func TestActualValuesSaveRefreshRestartAndReset(t *testing.T) {
	cfg, service := settingsFixture(t)
	ctx := context.Background()
	initial, err := service.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	item := findItem(t, initial, "logging__level")
	if item.Value != "warn" || item.Default != "warn" || item.ValueSource != "env" || initial.PendingRestart {
		t.Fatalf("initial = %+v", item)
	}
	updated, err := service.Update(ctx, initial.Revision, []settingsdto.Update{{Key: item.Key, Value: "debug"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	item = findItem(t, updated, item.Key)
	if item.Value != "warn" || item.NextValue != "debug" || !item.IsOverridden || !item.PendingRestart {
		t.Fatalf("saved = %+v", item)
	}
	refreshed, err := New(cfg.Runtime, envfile.NewStore(cfg.Runtime.OverridePath())).Config(ctx)
	if err != nil || !refreshed.PendingRestart {
		t.Fatalf("refresh lost pending state: %v", err)
	}
	restarted, err := config.Load()
	if err != nil {
		t.Fatal(err)
	}
	service = New(restarted.Runtime, envfile.NewStore(restarted.Runtime.OverridePath()))
	current, err := service.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	item = findItem(t, current, "logging__level")
	if item.Value != "debug" || current.PendingRestart {
		t.Fatalf("restarted = %+v", item)
	}
	reset, err := service.Reset(ctx, current.Revision, []string{item.Key})
	if err != nil {
		t.Fatal(err)
	}
	item = findItem(t, reset, item.Key)
	if item.Value != "debug" || item.NextValue != "warn" || item.IsOverridden || !item.PendingRestart {
		t.Fatalf("reset = %+v", item)
	}
	baselineFile, err := os.ReadFile(".env")
	if err != nil || string(baselineFile) != "POMELO_ORBIT_LOGGING__LEVEL=error\n" {
		t.Fatal("startup dotenv changed")
	}
	if os.Getenv("POMELO_ORBIT_LOGGING__LEVEL") != "warn" {
		t.Fatal("process ENV changed")
	}
}

func TestBatchValidationAndResetAreAtomic(t *testing.T) {
	cfg, service := settingsFixture(t)
	ctx := context.Background()
	initial, err := service.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Update(ctx, initial.Revision, []settingsdto.Update{{Key: "llm__base_url", Value: "https://provider.example/v1"}}, nil)
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected complete configuration validation: %v", err)
	}
	if _, err := os.Stat(cfg.Runtime.OverridePath()); !errors.Is(err, os.ErrNotExist) {
		t.Fatal("invalid candidate was written")
	}
	updated, err := service.Update(ctx, initial.Revision, []settingsdto.Update{
		{Key: "llm__base_url", Value: "https://provider.example/v1"},
		{Key: "llm__api_key", Value: "literal$TOKEN\\with\"quotes"},
		{Key: "llm__model", Value: "test-model"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(cfg.Runtime.OverridePath())
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Reset(ctx, updated.Revision, []string{"llm__model"})
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("partial reset = %v", err)
	}
	after, err := os.ReadFile(cfg.Runtime.OverridePath())
	if err != nil || !reflect.DeepEqual(before, after) {
		t.Fatal("rejected reset changed file")
	}
	reset, err := service.Reset(ctx, updated.Revision, []string{"llm__base_url", "llm__api_key", "llm__model"})
	if err != nil || reset.PendingRestart {
		t.Fatalf("batch reset = %+v, %v", reset, err)
	}
}

func TestAllConfigurationTypesAndSecretsAreEditable(t *testing.T) {
	_, service := settingsFixture(t)
	ctx := context.Background()
	initial, err := service.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	secret := base64.URLEncoding.EncodeToString(make([]byte, 32))
	updated, err := service.Update(ctx, initial.Revision, []settingsdto.Update{
		{Key: "database__url", Value: "postgres://user:password@localhost/orbit"},
		{Key: "jwt__secret_key", Value: secret},
		{Key: "mcp__access_token", Value: "literal$token"},
		{Key: "app__debug", Value: true},
		{Key: "server__port", Value: float64(8081)},
		{Key: "server__cors_allowed_origins", Value: []any{"https://example.com"}},
		{Key: "llm__timeout", Value: "2m"},
	}, nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"database__url", "jwt__secret_key", "mcp__access_token"} {
		item := findItem(t, updated, key)
		if !item.IsOverridden || !item.Secret {
			t.Fatalf("secret not editable or marked: %+v", item)
		}
	}
	_, err = service.Update(ctx, updated.Revision, []settingsdto.Update{{Key: "server__port", Value: 1.5}}, nil)
	if !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("fractional integer = %v", err)
	}
}

func TestStaleRevisionDoesNotOverwriteAnotherSave(t *testing.T) {
	_, service := settingsFixture(t)
	ctx := context.Background()
	initial, err := service.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Update(ctx, initial.Revision, []settingsdto.Update{{Key: "logging__level", Value: "debug"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	_, err = service.Update(ctx, initial.Revision, []settingsdto.Update{{Key: "app__debug", Value: true}}, nil)
	if !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("stale revision = %v", err)
	}
	current, err := service.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if findItem(t, current, "app__debug").IsOverridden {
		t.Fatal("stale update was persisted")
	}
}

func TestSameValueOverrideDoesNotRequireRestart(t *testing.T) {
	_, service := settingsFixture(t)
	ctx := context.Background()
	initial, err := service.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	updated, err := service.Update(ctx, initial.Revision, []settingsdto.Update{{Key: "logging__level", Value: "warn"}}, nil)
	if err != nil || updated.PendingRestart {
		t.Fatalf("same value = %+v, %v", updated, err)
	}
	if !findItem(t, updated, "logging__level").IsOverridden {
		t.Fatal("explicit same-value override missing")
	}
}

func findItem(t *testing.T, response settingsdto.SystemConfig, key string) settingsdto.ConfigItem {
	t.Helper()
	for _, item := range response.Items {
		if item.Key == key {
			return item
		}
	}
	t.Fatalf("missing configuration item %s", key)
	return settingsdto.ConfigItem{}
}

package settingssvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	settingsdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/dto"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/envfile"
)

func TestConfigUpdateAndResetPersistEnvOverrides(t *testing.T) {
	envFilePath := filepath.Join(t.TempDir(), ".env")
	cfg := config.Config{EnvFilePath: envFilePath}
	cfg.Logging.Level = "info"
	service := New(cfg, envfile.NewStore(envFilePath))
	ctx := context.Background()

	initial, err := service.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	loggingLevel, ok := findConfigItem(initial.Items, "logging__level")
	if !ok || loggingLevel.Value != "info" || loggingLevel.IsOverridden {
		t.Fatalf("unexpected initial logging level: %+v", loggingLevel)
	}

	updated, err := service.Update(ctx, "logging__level", "debug")
	if err != nil {
		t.Fatal(err)
	}
	loggingLevel, ok = findConfigItem(updated.Items, "logging__level")
	if !ok || loggingLevel.Value != "debug" || !loggingLevel.IsOverridden {
		t.Fatalf("unexpected updated logging level: %+v", loggingLevel)
	}
	content, err := os.ReadFile(envFilePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "POMELO_ORBIT_LOGGING__LEVEL=debug\n" {
		t.Fatalf("unexpected env file content: %s", string(content))
	}

	reset, err := service.Reset(ctx, []string{"logging__level"})
	if err != nil {
		t.Fatal(err)
	}
	loggingLevel, ok = findConfigItem(reset.Items, "logging__level")
	if !ok || loggingLevel.Value != "info" || loggingLevel.IsOverridden {
		t.Fatalf("unexpected reset logging level: %+v", loggingLevel)
	}
}

func findConfigItem(items []settingsdto.ConfigItem, key string) (settingsdto.ConfigItem, bool) {
	for _, item := range items {
		if item.Key == key {
			return item, true
		}
	}
	return settingsdto.ConfigItem{}, false
}

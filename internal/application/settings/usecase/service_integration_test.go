package settingssvc

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	settingsdto "github.com/leoninew/pomelo-orbit/internal/application/settings/dto"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/envfile"
)

func TestConfigUpdateAndResetPersistEnvOverrides(t *testing.T) {
	envFilePath := filepath.Join(t.TempDir(), ".env")
	cfg := config.Config{EnvFilePath: envFilePath}
	cfg.Logging.Level = "info"
	service := New(Definitions(cfg), envfile.NewStore(envFilePath))
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

func TestConfigIncludesDeploymentDialogueToolCallRounds(t *testing.T) {
	envFilePath := filepath.Join(t.TempDir(), ".env")
	cfg := config.Config{EnvFilePath: envFilePath, LLM: config.LLMConfig{MaxToolCallRounds: 32}}
	service := New(Definitions(cfg), envfile.NewStore(envFilePath))
	ctx := context.Background()

	initial, err := service.Config(ctx)
	if err != nil {
		t.Fatal(err)
	}
	item, ok := findConfigItem(initial.Items, "llm__max_tool_call_rounds")
	if !ok || item.Value != 32 || item.IsOverridden {
		t.Fatalf("unexpected deployment dialogue tool-call rounds: %+v", item)
	}

	updated, err := service.Update(ctx, "llm__max_tool_call_rounds", 48)
	if err != nil {
		t.Fatal(err)
	}
	item, ok = findConfigItem(updated.Items, "llm__max_tool_call_rounds")
	if !ok || item.Value != 48 || !item.IsOverridden {
		t.Fatalf("unexpected updated deployment dialogue tool-call rounds: %+v", item)
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

package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	content := []byte(`logging:
  level: "DEBUG"
database:
  sqlite:
    path: "data/test.db"
orbit:
  root: "../.."
worker:
  id: "worker-1"
  poll_interval: "1s"
  lease_duration: "5m"
  max_attempts: 4
  concurrency: 3
`)
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load returned error: %v", err)
	}
	if cfg.Logging.Level != "DEBUG" {
		t.Fatalf("unexpected logging level: %s", cfg.Logging.Level)
	}
	if cfg.Worker.Id != "worker-1" {
		t.Fatalf("unexpected worker id: %s", cfg.Worker.Id)
	}
	if cfg.Worker.Concurrency != 3 {
		t.Fatalf("unexpected concurrency: %d", cfg.Worker.Concurrency)
	}
}

package logging

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

func TestNewReturnsLogger(t *testing.T) {
	restoreDefaultLogger(t)
	cases := []struct {
		level       string
		enabledInfo bool
	}{
		{level: "DEBUG", enabledInfo: true},
		{level: "INFO", enabledInfo: true},
		{level: "WARN", enabledInfo: false},
		{level: "WARNING", enabledInfo: false},
		{level: " warn ", enabledInfo: false},
		{level: "ERROR", enabledInfo: false},
		{level: "unknown", enabledInfo: true},
	}
	for _, tc := range cases {
		logger, closeLogger, err := New(testConfig(t, tc.level))
		if err != nil {
			t.Fatalf("New returned error for level %s: %v", tc.level, err)
		}
		closeLoggerFn := closeLogger
		t.Cleanup(func() {
			if err := closeLoggerFn(); err != nil {
				t.Fatal(err)
			}
		})
		if logger == nil {
			t.Fatalf("expected logger for level %s", tc.level)
		}
		if logger.Enabled(t.Context(), slog.LevelInfo) != tc.enabledInfo {
			t.Fatalf("unexpected info enabled for level %s", tc.level)
		}
	}
}

func TestNewWritesLogFile(t *testing.T) {
	restoreDefaultLogger(t)
	cfg := testConfig(t, "INFO")
	logger, closeLogger, err := New(cfg)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	logger.Info("file log created", "component", "test")
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(cfg.File)
	if err != nil {
		t.Fatal(err)
	}
	logContent := string(content)
	if !strings.Contains(logContent, "file log created") || !strings.Contains(logContent, "component=test") {
		t.Fatalf("unexpected log content: %s", logContent)
	}
}

func TestNewCreatesLogDirectory(t *testing.T) {
	restoreDefaultLogger(t)
	cfg := config.LoggingConfig{Level: "INFO", File: filepath.Join(t.TempDir(), "nested", "logs", "backend-go.log"), MaxSizeMB: 100, MaxBackups: 7}
	_, closeLogger, err := New(cfg)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(cfg.File)); err != nil {
		t.Fatal(err)
	}
}

func TestNewSetsDefaultLogger(t *testing.T) {
	restoreDefaultLogger(t)
	cfg := testConfig(t, "INFO")
	logger, closeLogger, err := New(cfg)
	if err != nil {
		t.Fatalf("New returned error: %v", err)
	}
	slog.Info("default logger writes file", "component", "default")
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}
	if slog.Default() != logger {
		t.Fatal("expected New to install default logger")
	}

	content, err := os.ReadFile(cfg.File)
	if err != nil {
		t.Fatal(err)
	}
	logContent := string(content)
	if !strings.Contains(logContent, "default logger writes file") || !strings.Contains(logContent, "component=default") {
		t.Fatalf("unexpected log content: %s", logContent)
	}
}

func TestNewValidatesConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.LoggingConfig
	}{
		{name: "missing file", cfg: config.LoggingConfig{Level: "INFO", MaxSizeMB: 100, MaxBackups: 7}},
		{name: "missing max size", cfg: config.LoggingConfig{Level: "INFO", File: filepath.Join(t.TempDir(), "backend-go.log"), MaxBackups: 7}},
		{name: "missing max backups", cfg: config.LoggingConfig{Level: "INFO", File: filepath.Join(t.TempDir(), "backend-go.log"), MaxSizeMB: 100}},
	}
	for _, tc := range cases {
		if _, _, err := New(tc.cfg); err == nil {
			t.Fatalf("expected error for %s", tc.name)
		}
	}
}

func restoreDefaultLogger(t *testing.T) {
	t.Helper()
	oldDefault := slog.Default()
	t.Cleanup(func() {
		slog.SetDefault(oldDefault)
	})
}

func testConfig(t *testing.T, level string) config.LoggingConfig {
	t.Helper()
	return config.LoggingConfig{Level: level, File: filepath.Join(t.TempDir(), "backend-go.log"), MaxSizeMB: 100, MaxBackups: 7}
}

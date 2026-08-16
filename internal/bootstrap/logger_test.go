package bootstrap

import (
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestNewLoggerConfiguresLogLevel(t *testing.T) {
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
		logger, closeLogger, err := NewLogger(testLoggerConfig(t, tc.level))
		if err != nil {
			t.Fatalf("NewLogger returned error for level %s: %v", tc.level, err)
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

func TestNewLoggerWritesLogFile(t *testing.T) {
	restoreDefaultLogger(t)
	cfg := testLoggerConfig(t, "INFO")
	logger, closeLogger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger returned error: %v", err)
	}
	logger.Info("file log created", "component", "test")
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}

	content, err := os.ReadFile(cfg.Logging.File)
	if err != nil {
		t.Fatal(err)
	}
	logContent := string(content)
	if !strings.Contains(logContent, "file log created") || !strings.Contains(logContent, "component=test") {
		t.Fatalf("unexpected log content: %s", logContent)
	}
}

func TestNewLoggerCreatesLogDirectory(t *testing.T) {
	restoreDefaultLogger(t)
	cfg := config.Config{Logging: config.LoggingConfig{Level: "INFO", File: filepath.Join(t.TempDir(), "nested", "logs", "pomelo-orbit.log"), MaxSizeMB: 100, MaxBackups: 7}}
	_, closeLogger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger returned error: %v", err)
	}
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Dir(cfg.Logging.File)); err != nil {
		t.Fatal(err)
	}
}

func TestNewLoggerSetsDefaultLogger(t *testing.T) {
	restoreDefaultLogger(t)
	cfg := testLoggerConfig(t, "INFO")
	logger, closeLogger, err := NewLogger(cfg)
	if err != nil {
		t.Fatalf("NewLogger returned error: %v", err)
	}
	slog.Info("default logger writes file", "component", "default")
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}
	if slog.Default() != logger {
		t.Fatal("expected NewLogger to install default logger")
	}

	content, err := os.ReadFile(cfg.Logging.File)
	if err != nil {
		t.Fatal(err)
	}
	logContent := string(content)
	if !strings.Contains(logContent, "default logger writes file") || !strings.Contains(logContent, "component=default") {
		t.Fatalf("unexpected log content: %s", logContent)
	}
}

func TestNewLoggerValidatesConfig(t *testing.T) {
	cases := []struct {
		name string
		cfg  config.Config
	}{
		{name: "missing file", cfg: config.Config{Logging: config.LoggingConfig{Level: "INFO", MaxSizeMB: 100, MaxBackups: 7}}},
		{name: "missing max size", cfg: config.Config{Logging: config.LoggingConfig{Level: "INFO", File: filepath.Join(t.TempDir(), "pomelo-orbit.log"), MaxBackups: 7}}},
		{name: "missing max backups", cfg: config.Config{Logging: config.LoggingConfig{Level: "INFO", File: filepath.Join(t.TempDir(), "pomelo-orbit.log"), MaxSizeMB: 100}}},
	}
	for _, tc := range cases {
		if _, _, err := NewLogger(tc.cfg); err == nil {
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

func testLoggerConfig(t *testing.T, level string) config.Config {
	t.Helper()
	return config.Config{Logging: config.LoggingConfig{Level: level, File: filepath.Join(t.TempDir(), "pomelo-orbit.log"), MaxSizeMB: 100, MaxBackups: 7}}
}

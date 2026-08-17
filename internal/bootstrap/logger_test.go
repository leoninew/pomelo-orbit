package bootstrap

import (
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

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

func TestNewLoggerRollsBySize(t *testing.T) {
	restoreDefaultLogger(t)
	cfg := config.Config{Logging: config.LoggingConfig{Level: "INFO", File: filepath.Join(t.TempDir(), "pomelo-orbit.log"), MaxSizeMB: 1, MaxBackups: 7}}
	logger, closeLogger, err := newLoggerWithNow(cfg, io.Discard, time.Now)
	if err != nil {
		t.Fatalf("newLoggerWithNow() error = %v", err)
	}

	payload := strings.Repeat("x", 600*1024)
	logger.Info("first entry", "payload", payload)
	logger.Info("second entry", "payload", payload)
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}

	assertRotatedLog(t, cfg.Logging.File, "second entry", "first entry")
}

func TestNewLoggerRollsAtDayBoundary(t *testing.T) {
	restoreDefaultLogger(t)
	cfg := config.Config{Logging: config.LoggingConfig{Level: "INFO", File: filepath.Join(t.TempDir(), "pomelo-orbit.log"), MaxSizeMB: 10, MaxBackups: 7}}
	current := time.Date(2026, 8, 16, 23, 59, 0, 0, time.Local)
	logger, closeLogger, err := newLoggerWithNow(cfg, io.Discard, func() time.Time { return current })
	if err != nil {
		t.Fatalf("newLoggerWithNow() error = %v", err)
	}

	logger.Info("day one")
	current = current.Add(2 * time.Minute)
	logger.Info("day two")
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}

	assertRotatedLog(t, cfg.Logging.File, "day two", "day one")
}

func TestNewLoggerRollsAtDayBoundaryAfterRestart(t *testing.T) {
	restoreDefaultLogger(t)
	cfg := config.Config{Logging: config.LoggingConfig{Level: "INFO", File: filepath.Join(t.TempDir(), "pomelo-orbit.log"), MaxSizeMB: 10, MaxBackups: 7}}
	dayOne := time.Date(2026, 8, 16, 23, 59, 0, 0, time.Local)
	logger, closeLogger, err := newLoggerWithNow(cfg, io.Discard, func() time.Time { return dayOne })
	if err != nil {
		t.Fatalf("newLoggerWithNow() error = %v", err)
	}
	logger.Info("day one")
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}
	if err := os.Chtimes(cfg.Logging.File, dayOne, dayOne); err != nil {
		t.Fatal(err)
	}

	dayTwo := dayOne.Add(2 * time.Minute)
	logger, closeLogger, err = newLoggerWithNow(cfg, io.Discard, func() time.Time { return dayTwo })
	if err != nil {
		t.Fatalf("newLoggerWithNow() error = %v", err)
	}
	logger.Info("day two")
	if err := closeLogger(); err != nil {
		t.Fatal(err)
	}

	assertRotatedLog(t, cfg.Logging.File, "day two", "day one")
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

func assertRotatedLog(t *testing.T, activePath string, activeMessage string, rotatedMessage string) {
	t.Helper()
	ext := filepath.Ext(activePath)
	base := strings.TrimSuffix(filepath.Base(activePath), ext)
	rotated, err := filepath.Glob(filepath.Join(filepath.Dir(activePath), base+"-*"+ext))
	if err != nil {
		t.Fatal(err)
	}
	if len(rotated) == 0 {
		t.Fatalf("expected rotated log alongside %s", activePath)
	}
	active, err := os.ReadFile(activePath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(active), activeMessage) {
		t.Fatalf("active log does not contain %q", activeMessage)
	}
	for _, path := range rotated {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(content), rotatedMessage) {
			return
		}
	}
	t.Fatalf("rotated logs do not contain %q", rotatedMessage)
}

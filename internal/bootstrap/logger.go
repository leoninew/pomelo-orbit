package bootstrap

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func NewLogger(cfg config.Config) (*slog.Logger, func() error, error) {
	return newLogger(cfg, os.Stdout)
}

// NewMCPLogger keeps the stdio transport protocol clean. MCP messages own
// stdout, so diagnostics are emitted to stderr and the normal rolling log.
func NewMCPLogger(cfg config.Config) (*slog.Logger, func() error, error) {
	return newLogger(cfg, os.Stderr)
}

func newLogger(cfg config.Config, console io.Writer) (*slog.Logger, func() error, error) {
	return newLoggerWithNow(cfg, console, time.Now)
}

func newLoggerWithNow(cfg config.Config, console io.Writer, now func() time.Time) (*slog.Logger, func() error, error) {
	loggingCfg := cfg.Logging
	if strings.TrimSpace(loggingCfg.File) == "" {
		return nil, nil, fmt.Errorf("logging.file is required")
	}
	if loggingCfg.MaxSizeMB <= 0 {
		return nil, nil, fmt.Errorf("logging.max_size_mb must be positive")
	}
	if loggingCfg.MaxBackups <= 0 {
		return nil, nil, fmt.Errorf("logging.max_backups must be positive")
	}

	var slogLevel slog.Level
	switch strings.ToUpper(strings.TrimSpace(loggingCfg.Level)) {
	case "DEBUG":
		slogLevel = slog.LevelDebug
	case "WARN", "WARNING":
		slogLevel = slog.LevelWarn
	case "ERROR":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	if err := os.MkdirAll(filepath.Dir(loggingCfg.File), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create log directory: %w", err)
	}

	rollingWriter := newDailyRollingWriter(loggingCfg, now)
	writer := io.MultiWriter(console, rollingWriter)
	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slogLevel})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger, rollingWriter.Close, nil
}

// dailyRollingWriter combines lumberjack's size-based rotation with a forced
// rotation when the first log entry of a new local day is written.
type dailyRollingWriter struct {
	writer     *lumberjack.Logger
	now        func() time.Time
	day        string
	hasWritten bool
	mu         sync.Mutex
}

func newDailyRollingWriter(cfg config.LoggingConfig, now func() time.Time) *dailyRollingWriter {
	currentDay := logDay(now())
	hasWritten := false
	if info, err := os.Stat(cfg.File); err == nil && info.Size() > 0 {
		// A process may start after midnight while the active log belongs to the
		// prior day. Its modification date is the only persisted day marker.
		currentDay = logDay(info.ModTime())
		hasWritten = true
	}

	return &dailyRollingWriter{
		writer: &lumberjack.Logger{
			Filename:   cfg.File,
			MaxSize:    cfg.MaxSizeMB,
			MaxBackups: cfg.MaxBackups,
		},
		now:        now,
		day:        currentDay,
		hasWritten: hasWritten,
	}
}

func (w *dailyRollingWriter) Write(p []byte) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	day := logDay(w.now())
	if w.hasWritten && day != w.day {
		if err := w.writer.Rotate(); err != nil {
			return 0, fmt.Errorf("rotate log for new day: %w", err)
		}
		w.day = day
	}

	n, err := w.writer.Write(p)
	if n > 0 {
		w.hasWritten = true
	}
	return n, err
}

func (w *dailyRollingWriter) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()
	return w.writer.Close()
}

func logDay(value time.Time) string {
	return value.Format("2006-01-02")
}

package logging

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"gitee.com/leoninew/pomelo-orbit/internal/config"
)

func New(cfg config.LoggingConfig) (*slog.Logger, func() error, error) {
	if strings.TrimSpace(cfg.File) == "" {
		return nil, nil, fmt.Errorf("logging.file is required")
	}
	if cfg.MaxSizeMB <= 0 {
		return nil, nil, fmt.Errorf("logging.max_size_mb must be positive")
	}
	if cfg.MaxBackups <= 0 {
		return nil, nil, fmt.Errorf("logging.max_backups must be positive")
	}

	var slogLevel slog.Level
	switch strings.ToUpper(strings.TrimSpace(cfg.Level)) {
	case "DEBUG":
		slogLevel = slog.LevelDebug
	case "WARN", "WARNING":
		slogLevel = slog.LevelWarn
	case "ERROR":
		slogLevel = slog.LevelError
	default:
		slogLevel = slog.LevelInfo
	}

	if err := os.MkdirAll(filepath.Dir(cfg.File), 0o755); err != nil {
		return nil, nil, fmt.Errorf("create log directory: %w", err)
	}

	rollingWriter := &lumberjack.Logger{
		Filename:   cfg.File,
		MaxSize:    cfg.MaxSizeMB,
		MaxBackups: cfg.MaxBackups,
	}
	writer := io.MultiWriter(os.Stdout, rollingWriter)
	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slogLevel})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger, rollingWriter.Close, nil
}

package bootstrap

import (
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/natefinch/lumberjack.v2"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

func NewLogger(cfg config.Config) (*slog.Logger, func() error, error) {
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

	rollingWriter := &lumberjack.Logger{
		Filename:   loggingCfg.File,
		MaxSize:    loggingCfg.MaxSizeMB,
		MaxBackups: loggingCfg.MaxBackups,
	}
	writer := io.MultiWriter(os.Stdout, rollingWriter)
	handler := slog.NewTextHandler(writer, &slog.HandlerOptions{Level: slogLevel})
	logger := slog.New(handler)
	slog.SetDefault(logger)
	return logger, rollingWriter.Close, nil
}

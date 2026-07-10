package cdsvc

import (
	"io"
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/cdworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

type LogWriter interface {
	Writer(logPath string) (io.WriteCloser, error)
}

type Service struct {
	executionStore repository.DeploymentExecutionStore
	cfg            config.Config
	workspace      *cdworkspace.Workspace
	logStore       LogWriter
	logger         *slog.Logger
	runner         CommandRunner
}

func NewExecutionService(store repository.DeploymentExecutionStore, cfg config.Config, logger *slog.Logger, runner CommandRunner, logStore LogWriter) Service {
	return Service{executionStore: store, cfg: cfg, workspace: cdworkspace.New(cfg.DataRoot()), logStore: logStore, logger: logger, runner: runner}
}

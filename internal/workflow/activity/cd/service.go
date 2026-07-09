package cdsvc

import (
	"context"
	"io"
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/cdworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

type DeploymentExecutionStore interface {
	Application(ctx context.Context, id string) (model.Application, error)
	Deployment(ctx context.Context, id string) (model.Deployment, error)
	ConfigFiles(ctx context.Context, applicationId string) ([]model.ApplicationConfigFile, error)
	ServiceConfigs(ctx context.Context, applicationId string) ([]model.ApplicationServiceConfig, error)
	Routes(ctx context.Context, applicationId string) ([]model.ApplicationRoute, error)
	MarkApplicationStatus(ctx context.Context, id string, status string) error
	MarkDeploymentRunning(ctx context.Context, id string) error
	CompleteDeployment(ctx context.Context, id string, status string, message string) error
}

type LogWriter interface {
	Writer(logPath string) (io.WriteCloser, error)
}

type Service struct {
	executionStore DeploymentExecutionStore
	cfg            config.Config
	workspace      *cdworkspace.Workspace
	logStore       LogWriter
	logger         *slog.Logger
	runner         CommandRunner
}

func NewExecutionService(store DeploymentExecutionStore, cfg config.Config, logger *slog.Logger, runner CommandRunner, logStore LogWriter) Service {
	return Service{executionStore: store, cfg: cfg, workspace: cdworkspace.New(cfg.DataRoot()), logStore: logStore, logger: logger, runner: runner}
}

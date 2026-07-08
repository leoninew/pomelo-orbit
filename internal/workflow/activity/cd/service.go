package cdsvc

import (
	"context"
	"log/slog"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/logger/logstore"
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

type Service struct {
	executionStore DeploymentExecutionStore
	cfg            config.Config
	workspace      *Workspace
	logStore       logstore.LogStore
	logger         *slog.Logger
	runner         CommandRunner
}

func NewExecutionService(store DeploymentExecutionStore, cfg config.Config, logger *slog.Logger, runner CommandRunner, logStore logstore.LogStore) Service {
	return Service{executionStore: store, cfg: cfg, workspace: NewWorkspace(cfg.DataRoot()), logStore: logStore, logger: logger, runner: runner}
}

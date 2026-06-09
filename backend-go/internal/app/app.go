package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"backend/internal/cd"
	"backend/internal/ci"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/httpserver"
	"backend/internal/orbit"
	"backend/internal/status"
	"backend/internal/task"
)

type App struct {
	cfg    config.Config
	logger *slog.Logger
}

func New(cfg config.Config, logger *slog.Logger) App {
	return App{cfg: cfg, logger: logger}
}

func (a App) Migrate() error {
	database, err := db.Open(a.cfg.Database)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	if err := db.NewMigrator(database, a.cfg.Database.Driver).Up(); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func (a App) RunWorker(ctx context.Context) error {
	database, err := db.Open(a.cfg.Database)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	if err := db.NewMigrator(database, a.cfg.Database.Driver).Up(); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	repository := task.NewRepository(database, a.cfg.Database.Driver)
	store := orbit.NewStore(database, a.cfg.Database.Driver)
	router := task.NewRouter()
	registerHandlers(router, store, a.cfg, a.logger)

	worker := task.NewWorker(repository, router, a.logger, task.WorkerConfig{
		WorkerId:      a.cfg.Worker.Id,
		PollInterval:  a.cfg.Worker.PollInterval,
		LeaseDuration: a.cfg.Worker.LeaseDuration,
		Concurrency:   a.cfg.Worker.Concurrency,
	})
	return worker.Run(ctx)
}

func (a App) MigrationStatus() ([]db.MigrationStatus, error) {
	database, err := db.Open(a.cfg.Database)
	if err != nil {
		return nil, err
	}
	defer func() { _ = database.Close() }()
	return db.NewMigrator(database, a.cfg.Database.Driver).Status()
}

func (a App) Serve(ctx context.Context) error {
	database, err := db.Open(a.cfg.Database)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	if err := db.NewMigrator(database, a.cfg.Database.Driver).Up(); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}

	repository := task.NewRepository(database, a.cfg.Database.Driver)
	store := orbit.NewStore(database, a.cfg.Database.Driver)
	server := httpserver.New(a.cfg, a.logger, store, repository, a.cfg.Worker.MaxAttempts)
	httpServer := &http.Server{Addr: server.Addr(), Handler: server.Handler()}

	go func() {
		<-ctx.Done()
		if err := httpServer.Shutdown(context.Background()); err != nil {
			a.logger.Error("http server shutdown failed", "error", err)
		}
	}()

	a.logger.Info("http server started", "addr", server.Addr())
	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		return fmt.Errorf("serve http: %w", err)
	}
	return ctx.Err()
}

func registerHandlers(router *task.Router, store orbit.Store, cfg config.Config, logger *slog.Logger) {
	router.Register(status.TaskTypeCIPipelineRunExecute, ci.NewHandler(store, cfg, logger))
	router.Register(status.TaskTypeCDApplicationDeploy, cd.NewDeployHandler(store, cfg, logger))
	router.Register(status.TaskTypeCDApplicationRestart, cd.NewRestartHandler(store, cfg, logger))
}

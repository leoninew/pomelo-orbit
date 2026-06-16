package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"

	"backend/internal/bootstrap"
	"backend/internal/config"
	"backend/internal/db"
)

type App struct {
	cfg    config.Config
	logger *slog.Logger
}

func New(cfg config.Config, logger *slog.Logger) App {
	return App{cfg: cfg, logger: logger}
}

func (a App) Migrate() error {
	database, err := bootstrap.OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	return bootstrap.RunMigrations(database, a.cfg.Database.Driver)
}

func (a App) RunWorker(ctx context.Context) error {
	database, err := bootstrap.OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	if err := bootstrap.RunMigrations(database, a.cfg.Database.Driver); err != nil {
		return err
	}

	taskRepo := bootstrap.NewTaskRepository(database, a.cfg.Database.Driver)
	store := bootstrap.NewRepositoryStore(database, a.cfg.Database.Driver)
	router := bootstrap.NewTaskRouter(store, a.cfg, a.logger)
	worker := bootstrap.NewWorker(a.cfg, a.logger, taskRepo, router)
	return worker.Run(ctx)
}

func (a App) MigrationStatus() ([]db.MigrationStatus, error) {
	database, err := bootstrap.OpenDatabase(a.cfg)
	if err != nil {
		return nil, err
	}
	defer func() { _ = database.Close() }()
	return bootstrap.MigrationStatus(database, a.cfg.Database.Driver)
}

func (a App) Serve(ctx context.Context) error {
	database, err := bootstrap.OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	if err := bootstrap.RunMigrations(database, a.cfg.Database.Driver); err != nil {
		return err
	}

	taskRepo := bootstrap.NewTaskRepository(database, a.cfg.Database.Driver)
	store := bootstrap.NewRepositoryStore(database, a.cfg.Database.Driver)
	server := bootstrap.NewHTTPServer(a.cfg, a.logger, store, taskRepo)
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

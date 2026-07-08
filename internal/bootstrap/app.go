package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"sync"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	database "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
)

type App struct {
	cfg    config.Config
	logger *slog.Logger
}

func New(cfg config.Config, logger *slog.Logger) App {
	return App{cfg: cfg, logger: logger}
}

func (a App) Migrate() error {
	database, err := OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	return RunMigrations(database, a.cfg.Database.Driver)
}

func (a App) RunWorker(ctx context.Context) error {
	database, err := OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	if err := RunMigrations(database, a.cfg.Database.Driver); err != nil {
		return err
	}

	taskRepo := NewTaskRepository(database, a.cfg.Database.Driver)
	store := NewRepositoryStore(database, a.cfg.Database.Driver)
	router := NewTaskRouter(store, a.cfg, a.logger)
	worker := NewWorker(a.cfg, a.logger, taskRepo, router)
	return worker.Run(ctx)
}

func (a App) MigrationVersion() (database.MigrationVersion, error) {
	dbConn, err := OpenDatabase(a.cfg)
	if err != nil {
		return database.MigrationVersion{}, err
	}
	defer func() { _ = dbConn.Close() }()
	return MigrationVersion(dbConn, a.cfg.Database.Driver)
}

func (a App) Serve(ctx context.Context) error {
	database, err := OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	if err := RunMigrations(database, a.cfg.Database.Driver); err != nil {
		return err
	}

	taskRepo := NewTaskRepository(database, a.cfg.Database.Driver)
	store := NewRepositoryStore(database, a.cfg.Database.Driver)
	server := NewHTTPServer(a.cfg, a.logger, store, taskRepo)
	httpServer := &http.Server{Addr: server.Addr(), Handler: server.Handler()}
	router := NewTaskRouter(store, a.cfg, a.logger)
	backgroundWorker := NewWorker(a.cfg, a.logger, taskRepo, router)

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	var shutdownOnce sync.Once
	shutdownHTTP := func() {
		shutdownOnce.Do(func() {
			if err := httpServer.Shutdown(context.Background()); err != nil {
				a.logger.Error("http server shutdown failed", "error", err)
			}
		})
	}
	go func() {
		<-runCtx.Done()
		shutdownHTTP()
	}()

	errCh := make(chan error, 2)
	go func() {
		a.logger.Info("http server started", "addr", server.Addr())
		if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- fmt.Errorf("serve http: %w", err)
			return
		}
		errCh <- nil
	}()
	go func() {
		if err := backgroundWorker.Run(runCtx); err != nil && !errors.Is(err, context.Canceled) {
			errCh <- fmt.Errorf("run background worker: %w", err)
			return
		}
		errCh <- nil
	}()

	var firstErr error
	for completed := 0; completed < 2; completed++ {
		if err := <-errCh; err != nil && firstErr == nil {
			firstErr = err
			cancel()
			shutdownHTTP()
		}
	}
	if firstErr != nil {
		return firstErr
	}
	return ctx.Err()
}

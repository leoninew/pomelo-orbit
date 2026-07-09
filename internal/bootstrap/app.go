package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

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

	return runHTTPServerAndWorker(ctx, a.logger, server.Addr(), httpServer, backgroundWorker)
}

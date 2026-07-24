package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	database "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/queue/worker"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
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

	taskRepo := taskrepo.NewRepository(database)
	router := NewTaskRouter(database, a.cfg, a.logger)
	backgroundWorker := worker.New(database, taskRepo, router, a.logger, worker.Config{
		WorkerId:      a.cfg.Worker.Id,
		PollInterval:  a.cfg.Worker.PollInterval,
		LeaseDuration: a.cfg.Worker.LeaseDuration,
		Concurrency:   a.cfg.Worker.Concurrency,
	})
	return backgroundWorker.Run(ctx)
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

	taskRepo := taskrepo.NewRepository(database)
	server := NewHTTPServer(a.cfg, a.logger, database, taskRepo)
	httpServer := &http.Server{Addr: server.Addr(), Handler: server.Handler()}
	router := NewTaskRouter(database, a.cfg, a.logger)
	backgroundWorker := worker.New(database, taskRepo, router, a.logger, worker.Config{
		WorkerId:      a.cfg.Worker.Id,
		PollInterval:  a.cfg.Worker.PollInterval,
		LeaseDuration: a.cfg.Worker.LeaseDuration,
		Concurrency:   a.cfg.Worker.Concurrency,
	})

	return runHTTPServerAndWorker(ctx, a.logger, server.Addr(), httpServer, backgroundWorker)
}

package bootstrap

import (
	"fmt"
	"log/slog"

	"github.com/jmoiron/sqlx"

	"backend/internal/cd"
	"backend/internal/ci"
	"backend/internal/config"
	"backend/internal/db"
	"backend/internal/repository"
	cirepo "backend/internal/repository/ci"
	taskrepo "backend/internal/repository/task"
	"backend/internal/status"
	transporthttp "backend/internal/transport/http"
	"backend/internal/worker"
)

func OpenDatabase(cfg config.Config) (*sqlx.DB, error) {
	return db.Open(cfg.Database)
}

func RunMigrations(database *sqlx.DB, driver string) error {
	if err := db.NewMigrator(database, driver).Up(); err != nil {
		return fmt.Errorf("run migrations: %w", err)
	}
	return nil
}

func MigrationStatus(database *sqlx.DB, driver string) ([]db.MigrationStatus, error) {
	return db.NewMigrator(database, driver).Status()
}

func NewRepositoryStore(database *sqlx.DB, driver string) repository.Store {
	return repository.NewStore(database, driver)
}

func NewTaskRepository(database *sqlx.DB, driver string) taskrepo.Repository {
	return taskrepo.NewRepository(database, driver)
}

func NewHTTPServer(cfg config.Config, logger *slog.Logger, store repository.Store, taskRepo taskrepo.Repository) transporthttp.Server {
	return transporthttp.New(cfg, logger, store, taskRepo, cfg.Worker.MaxAttempts)
}

func NewTaskRouter(store repository.Store, cfg config.Config, logger *slog.Logger) *worker.Router {
	router := worker.NewRouter()
	ciRepository := cirepo.NewRepository(store.DB(), store.Driver())
	router.Register(status.TaskTypeCIPipelineRunExecute, ci.NewHandler(ciRepository, cfg, logger))
	router.Register(status.TaskTypeCDApplicationDeploy, cd.NewDeployHandler(store, cfg, logger))
	router.Register(status.TaskTypeCDApplicationRestart, cd.NewRestartHandler(store, cfg, logger))
	return router
}

func NewWorker(cfg config.Config, logger *slog.Logger, taskRepo taskrepo.Repository, router worker.Handler) *worker.Worker {
	return worker.New(taskRepo, router, logger, worker.Config{
		WorkerId:      cfg.Worker.Id,
		PollInterval:  cfg.Worker.PollInterval,
		LeaseDuration: cfg.Worker.LeaseDuration,
		Concurrency:   cfg.Worker.Concurrency,
	})
}

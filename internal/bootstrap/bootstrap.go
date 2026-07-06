package bootstrap

import (
	"log/slog"

	"github.com/jmoiron/sqlx"

	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/db"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/logstore"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
	cdrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/cd"
	cirepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/ci"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/task"
	cdsvc "gitee.com/leoninew/PomeloOrbit-go/internal/service/cd"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/service/ci"
	"gitee.com/leoninew/PomeloOrbit-go/internal/status"
	transporthttp "gitee.com/leoninew/PomeloOrbit-go/internal/transport/http"
	"gitee.com/leoninew/PomeloOrbit-go/internal/worker"
	cdworker "gitee.com/leoninew/PomeloOrbit-go/internal/worker/handler/cd"
	ciworker "gitee.com/leoninew/PomeloOrbit-go/internal/worker/handler/ci"
)

func OpenDatabase(cfg config.Config) (*sqlx.DB, error) {
	return db.Open(cfg.Database)
}

func RunMigrations(database *sqlx.DB, driver string) error {
	return db.MigrateUp(database, driver)
}

func MigrationVersion(database *sqlx.DB, driver string) (db.MigrationVersion, error) {
	return db.ReadMigrationVersion(database, driver)
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
	cdRepository := cdrepo.NewRepository(store.DB(), store.Driver())
	logStore := logstore.LogStore{}
	ciService := cisvc.NewExecutionService(ciRepository, cfg.DataRoot(), cfg.JWT.SecretKey, logger, cisvc.DockerRunner{}, logStore)
	cdService := cdsvc.NewExecutionService(cdRepository, cfg, logger, cdsvc.ShellRunner{}, logStore)
	router.Register(status.TaskTypeCIPipelineRunExecute, ciworker.NewHandler(ciService))
	router.Register(status.TaskTypeCDApplicationDeploy, cdworker.NewDeployHandler(cdService))
	router.Register(status.TaskTypeCDApplicationRestart, cdworker.NewRestartHandler(cdService))
	router.Register(status.TaskTypeCDApplicationStop, cdworker.NewStopHandler(cdService))
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

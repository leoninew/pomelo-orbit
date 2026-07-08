package bootstrap

import (
	"log/slog"

	"github.com/jmoiron/sqlx"

	transporthttp "gitee.com/leoninew/PomeloOrbit-go/internal/api/http"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/logger/logstore"
	store "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc"
	cdrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/cd"
	cirepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/ci"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
	"gitee.com/leoninew/PomeloOrbit-go/internal/worker"
	cdworker "gitee.com/leoninew/PomeloOrbit-go/internal/worker/handler/cd"
	ciworker "gitee.com/leoninew/PomeloOrbit-go/internal/worker/handler/ci"
	cdactivity "gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/cd"
	ciactivity "gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/ci"
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

func NewRepositoryStore(database *sqlx.DB, driver string) store.Store {
	return store.NewStore(database, driver)
}

func NewTaskRepository(database *sqlx.DB, driver string) taskrepo.Repository {
	return taskrepo.NewRepository(database, driver)
}

func NewHTTPServer(cfg config.Config, logger *slog.Logger, store store.Store, taskRepo taskrepo.Repository) transporthttp.Server {
	return transporthttp.New(cfg, logger, store, taskRepo, cfg.Worker.MaxAttempts)
}

func NewTaskRouter(store store.Store, cfg config.Config, logger *slog.Logger) *worker.Router {
	router := worker.NewRouter()
	ciRepository := cirepo.NewRepository(store.DB(), store.Driver())
	cdRepository := cdrepo.NewRepository(store.DB(), store.Driver())
	logStore := logstore.LogStore{}
	ciService := ciactivity.NewExecutionService(ciRepository, cfg.DataRoot(), cfg.JWT.SecretKey, logger, ciactivity.DockerRunner{}, logStore)
	cdService := cdactivity.NewExecutionService(cdRepository, cfg, logger, cdactivity.ShellRunner{}, logStore)
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

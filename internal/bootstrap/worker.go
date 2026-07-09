package bootstrap

import (
	"log/slog"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
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

func NewTaskRouter(repositoryStore store.Store, cfg config.Config, logger *slog.Logger) *worker.Router {
	router := worker.NewRouter()
	ciRepository := cirepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	cdRepository := cdrepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	logStore := executionlog.Store{}
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

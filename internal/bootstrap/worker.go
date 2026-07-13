package bootstrap

import (
	"log/slog"

	"github.com/jmoiron/sqlx"

	cdsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/usecase"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/usecase"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	cdrunner "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/runner/cd"
	dockerci "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/runner/dockerci"
	runtimepath "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/cdworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/ciworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	"gitee.com/leoninew/PomeloOrbit-go/internal/queue/worker"
	cdworker "gitee.com/leoninew/PomeloOrbit-go/internal/queue/worker/handler/cd"
	ciworker "gitee.com/leoninew/PomeloOrbit-go/internal/queue/worker/handler/ci"
	cdrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlx/cd"
	cirepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlx/ci"
)

func NewTaskRouter(database *sqlx.DB, cfg config.Config, logger *slog.Logger) *worker.Router {
	router := worker.NewRouter()
	ciRepository := cirepo.NewRepository(database, cfg.Database.Driver)
	cdRepository := cdrepo.NewRepository(database, cfg.Database.Driver)
	logStore := executionlog.Store{}
	ciWorkspace := ciworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	cdWorkspace := cdworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	ciService := cisvc.NewExecutionService(ciRepository, ciWorkspace, cfg.JWT.SecretKey, logger, dockerci.DockerRunner{}, logStore)
	cdService := cdsvc.NewExecutionService(cdRepository, cfg, logger, cdWorkspace, cdrunner.ShellRunner{}, logStore)
	router.Register(status.TaskTypeCIPipelineRunExecute, ciworker.NewHandler(ciService))
	router.Register(status.TaskTypeCDApplicationDeploy, cdworker.NewDeployHandler(cdService))
	router.Register(status.TaskTypeCDApplicationRestart, cdworker.NewRestartHandler(cdService))
	router.Register(status.TaskTypeCDApplicationStop, cdworker.NewStopHandler(cdService))
	return router
}

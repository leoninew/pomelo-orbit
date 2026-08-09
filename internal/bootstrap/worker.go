package bootstrap

import (
	"database/sql"
	"log/slog"

	applicationsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/usecase"
	deploymentsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/usecase"
	gatewaysvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/usecase"
	pipelinerunsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/usecase"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	databasetx "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	deploymentrunner "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/runner/deployment"
	pipelinerunner "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/runner/pipeline"
	runtimepath "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/deploymentworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/pipelineworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/repositorysource"
	"gitee.com/leoninew/PomeloOrbit-go/internal/queue/worker"
	deploymentworker "gitee.com/leoninew/PomeloOrbit-go/internal/queue/worker/handler/deployment"
	pipelinerunworker "gitee.com/leoninew/PomeloOrbit-go/internal/queue/worker/handler/pipeline_run"
)

func NewTaskRouter(database *sql.DB, cfg config.Config, logger *slog.Logger) *worker.Router {
	stores := newDomainStores(database)
	logStore := executionlog.Store{}
	pipelineWorkspace := pipelineworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	localSource := repositorysource.New(runtimepath.ResolvePhysicalPath)
	deploymentWorkspace := deploymentworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	gatewayService := gatewaysvc.New(stores.project, stores.application, stores.gateway, stores.service, stores.deployment)
	applicationService := applicationsvc.New(stores.project, stores.application)
	transactionRunner := databasetx.NewTransactionRunner(database)

	pipelineRunService := pipelinerunsvc.NewExecutionService(
		stores.project,
		stores.credential,
		stores.repository,
		stores.pipeline,
		stores.pipelineRun,
		stores.application,
		applicationService,
		transactionRunner,
		pipelineWorkspace,
		cfg.Jwt.SecretKey,
		logger,
		cfg.PipelineRun.ExecutionTimeout,
		cfg.Worker.PollInterval,
		pipelinerunner.DockerRunner{},
		logStore,
		localSource,
	)
	deploymentService := deploymentsvc.NewExecutionService(
		stores.project,
		stores.application,
		stores.service,
		stores.deployment,
		gatewayService,
		logger,
		deploymentWorkspace,
		deploymentrunner.ShellRunner{},
		logStore,
		cfg.Worker.PollInterval,
	)

	router := worker.NewRouter()
	router.Register(status.TaskTypePipelineRunExecute, pipelinerunworker.NewHandler(pipelineRunService))
	router.Register(status.TaskTypeDeploymentDeploy, deploymentworker.NewDeployHandler(deploymentService))
	router.Register(status.TaskTypeDeploymentRestart, deploymentworker.NewRestartHandler(deploymentService))
	router.Register(status.TaskTypeDeploymentStop, deploymentworker.NewStopHandler(deploymentService))
	return router
}

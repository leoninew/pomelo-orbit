package bootstrap

import (
	"database/sql"
	"log/slog"

	applicationsvc "github.com/leoninew/pomelo-orbit/internal/application/application/usecase"
	deploymentsvc "github.com/leoninew/pomelo-orbit/internal/application/deployment/usecase"
	gatewaysvc "github.com/leoninew/pomelo-orbit/internal/application/gateway/usecase"
	pipelinerunsvc "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/usecase"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/config"
	databasetx "github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	deploymentrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/deployment"
	pipelinerunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/pipeline"
	runtimepath "github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/deploymentworkspace"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/executionlog"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/pipelineworkspace"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/repositorysource"
	"github.com/leoninew/pomelo-orbit/internal/queue/worker"
	deploymentworker "github.com/leoninew/pomelo-orbit/internal/queue/worker/handler/deployment"
	pipelinerunworker "github.com/leoninew/pomelo-orbit/internal/queue/worker/handler/pipeline_run"
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

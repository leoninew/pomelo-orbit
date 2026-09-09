package bootstrap

import (
	"database/sql"
	"log/slog"

	applicationsvc "github.com/leoninew/pomelo-orbit/internal/application/application/usecase"
	credentialsvc "github.com/leoninew/pomelo-orbit/internal/application/credential/usecase"
	deploymentsvc "github.com/leoninew/pomelo-orbit/internal/application/deployment/usecase"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	gatewaysvc "github.com/leoninew/pomelo-orbit/internal/application/gateway/usecase"
	pipelinerunsvc "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/usecase"
	routesvc "github.com/leoninew/pomelo-orbit/internal/application/route/usecase"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/config"
	databasetx "github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/external/traefik"
	localrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/local"
	pipelinerunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/pipeline"
	sshrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/ssh"
	targetrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/target"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/executionlog"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/pipelineworkspace"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/repositorysource"
	"github.com/leoninew/pomelo-orbit/internal/queue/worker"
	deploymentworker "github.com/leoninew/pomelo-orbit/internal/queue/worker/handler/deployment"
	pipelinerunworker "github.com/leoninew/pomelo-orbit/internal/queue/worker/handler/pipeline_run"
)

func NewTaskRouter(database *sql.DB, cfg config.Config, logger *slog.Logger) *worker.Router {
	stores := newDomainStores(database)
	pipelineLogStore := executionlog.Store{}
	deploymentLogStore := executionlog.NewDeploymentStore(cfg.Workspace.Deployment)
	dockerPathResolver := dockerDaemonPathResolver()
	pipelineWorkspace := pipelineworkspace.NewWithResolver(cfg.Workspace.Pipeline, dockerPathResolver)
	localSource := repositorysource.New(dockerPathResolver)
	transactionRunner := databasetx.NewTransactionRunner(database)
	credentialService := credentialsvc.New(stores.project, stores.credential, cfg.Jwt.SecretKey)
	targetResolver := environmentsvc.NewTargetResolver(stores.environment, credentialService)
	localRuntime := localrunner.NewRuntime(cfg.Workspace.Deployment, dockerPathResolver)
	runtime := targetrunner.New(localRuntime, sshrunner.NewRuntime())
	gatewayService := gatewaysvc.New(stores.project, stores.environment, stores.application, stores.gateway, stores.service, stores.route, stores.deployment, cfg, dockerPathResolver, transactionRunner)
	routeManager := traefik.NewRouteManager(targetResolver, runtime)
	routeService := routesvc.New(
		stores.project,
		stores.application,
		stores.service,
		stores.route,
		stores.gateway,
		routeManager,
		traefik.MkcertGenerator{},
		routeManager,
		transactionRunner,
	)
	applicationService := applicationsvc.New(stores.project, stores.application)
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
		pipelineLogStore,
		localSource,
	)
	deploymentService := deploymentsvc.NewExecutionService(
		stores.project,
		stores.application,
		stores.service,
		stores.deployment,
		gatewayService,
		logger,
		targetResolver,
		runtime,
		deploymentLogStore,
		cfg.Worker.PollInterval,
		routeService,
	)

	router := worker.NewRouter()
	router.Register(status.TaskTypePipelineRunExecute, pipelinerunworker.NewHandler(pipelineRunService))
	router.Register(status.TaskTypeDeploymentDeploy, deploymentworker.NewDeployHandler(deploymentService))
	router.Register(status.TaskTypeDeploymentRestart, deploymentworker.NewRestartHandler(deploymentService))
	router.Register(status.TaskTypeDeploymentStop, deploymentworker.NewStopHandler(deploymentService))
	return router
}

package bootstrap

import (
	"database/sql"
	"log/slog"

	transporthttp "gitee.com/leoninew/PomeloOrbit-go/internal/api/http"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/routes"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	applicationsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/usecase"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
	credentialsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/credential/usecase"
	deploymentsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/usecase"
	gatewaysvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/gateway/usecase"
	pipelinesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/usecase"
	pipelinerunsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/usecase"
	projectsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/project/usecase"
	repositorysvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/repository/usecase"
	rolesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/role/usecase"
	routesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/route/usecase"
	servicesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/usecase"
	settingssvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/usecase"
	usersvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/usecase"
	jwt "gitee.com/leoninew/PomeloOrbit-go/internal/auth/jwt"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/external/traefik"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/external/turnstile"
	deploymentrunner "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/runner/deployment"
	pipelinerunner "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/runner/pipeline"
	runtimepath "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/deploymentworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/envfile"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/pipelineworkspace"
	queuedispatch "gitee.com/leoninew/PomeloOrbit-go/internal/queue/dispatch"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
)

func NewHTTPServer(cfg config.Config, logger *slog.Logger, database *sql.DB, taskRepo taskrepo.Repository) transporthttp.Server {
	return transporthttp.New(cfg, logger, newHTTPServerDependencies(cfg, logger, database, taskRepo))
}

func newHTTPServerDependencies(cfg config.Config, logger *slog.Logger, database *sql.DB, taskRepo taskrepo.Repository) routes.Dependencies {
	stores := newDomainStores(database)
	tokenService := jwt.NewTokenService(cfg.JWT.SecretKey)
	taskService := tasksvc.New(taskRepo, cfg.Worker.MaxAttempts)
	pipelineRunDispatcher := queuedispatch.NewPipelineRunDispatcher(taskService)
	deploymentDispatcher := queuedispatch.NewDeploymentDispatcher(taskService)
	authService := authsvc.New(stores.user, stores.auth, tokenService, logger)
	logStore := executionlog.Store{}
	pipelineWorkspace := pipelineworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	deploymentWorkspace := deploymentworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	routeManager := traefik.NewRouteManager(cfg)
	gatewayService := gatewaysvc.New(stores.project, stores.application, stores.gateway, stores.service, deploymentWorkspace)
	return routes.Dependencies{
		Database:          database,
		Authenticator:     security.New(logger, authService),
		AuthService:       authService,
		RoleService:       rolesvc.New(stores.role),
		UserService:       usersvc.New(stores.user, stores.role),
		ProjectService:    projectsvc.New(stores.project, stores.user),
		SettingsService:   settingssvc.New(cfg, envfile.NewStore(cfg.EnvFilePath)),
		CredentialService: credentialsvc.New(stores.project, stores.credential, cfg.JWT.SecretKey),
		RepositoryService: repositorysvc.New(
			stores.project,
			stores.credential,
			stores.repository,
			stores.pipeline,
			stores.pipelineRun,
			pipelineRunDispatcher,
			logger,
		),
		PipelineService: pipelinesvc.New(stores.project, stores.pipeline, logger),
		PipelineRunService: pipelinerunsvc.New(
			stores.project,
			stores.credential,
			stores.repository,
			stores.pipeline,
			stores.pipelineRun,
			pipelineRunDispatcher,
			pipelineWorkspace,
			cfg.JWT.SecretKey,
			logger,
			pipelinerunner.DockerRunner{},
			logStore,
		),
		RouteService: routesvc.New(
			stores.project,
			stores.route,
			stores.gateway,
			cfg,
			routeManager,
			traefik.MkcertGenerator{},
			routeManager,
		),
		ApplicationService: applicationsvc.NewWithCredential(
			stores.project,
			stores.application,
			stores.credential,
			cfg.JWT.SecretKey,
		),
		ServiceService: servicesvc.New(
			stores.project,
			stores.application,
			stores.credential,
			cfg.JWT.SecretKey,
			stores.service,
		),
		DeploymentService: deploymentsvc.NewCommandService(
			stores.project,
			stores.application,
			stores.service,
			stores.deployment,
			stores.gateway,
			deploymentDispatcher,
			logger,
			deploymentWorkspace,
			deploymentrunner.CommandQueryRunner{},
			logStore,
			gatewayService,
		),
		GatewayService:    gatewayService,
		TaskService:       taskService,
		TurnstileVerifier: turnstile.NewVerifier(cfg.Turnstile),
	}
}

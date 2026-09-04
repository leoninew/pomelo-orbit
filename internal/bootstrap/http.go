package bootstrap

import (
	"database/sql"
	"log/slog"

	transporthttp "github.com/leoninew/pomelo-orbit/internal/api/http"
	"github.com/leoninew/pomelo-orbit/internal/api/http/routes"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	deliverymcp "github.com/leoninew/pomelo-orbit/internal/api/mcp/delivery"
	applicationsvc "github.com/leoninew/pomelo-orbit/internal/application/application/usecase"
	authsvc "github.com/leoninew/pomelo-orbit/internal/application/auth/usecase"
	credentialsvc "github.com/leoninew/pomelo-orbit/internal/application/credential/usecase"
	deploymentsvc "github.com/leoninew/pomelo-orbit/internal/application/deployment/usecase"
	dialoguesvc "github.com/leoninew/pomelo-orbit/internal/application/dialogue/usecase"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	gatewaysvc "github.com/leoninew/pomelo-orbit/internal/application/gateway/usecase"
	pipelinesvc "github.com/leoninew/pomelo-orbit/internal/application/pipeline/usecase"
	pipelinerunsvc "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/usecase"
	projectsvc "github.com/leoninew/pomelo-orbit/internal/application/project/usecase"
	repositorysvc "github.com/leoninew/pomelo-orbit/internal/application/repository/usecase"
	rolesvc "github.com/leoninew/pomelo-orbit/internal/application/role/usecase"
	routesvc "github.com/leoninew/pomelo-orbit/internal/application/route/usecase"
	servicesvc "github.com/leoninew/pomelo-orbit/internal/application/service/usecase"
	settingssvc "github.com/leoninew/pomelo-orbit/internal/application/settings/usecase"
	usersvc "github.com/leoninew/pomelo-orbit/internal/application/user/usecase"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	"github.com/leoninew/pomelo-orbit/internal/config"
	databasetx "github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	llmclient "github.com/leoninew/pomelo-orbit/internal/infrastructure/external/openai"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/external/traefik"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/external/turnstile"
	deliverymcpclient "github.com/leoninew/pomelo-orbit/internal/infrastructure/mcp/delivery"

	pipelinerunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/pipeline"
	sshrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/ssh"

	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/envfile"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/executionlog"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/pipelineworkspace"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/repositorysource"
	queuedispatch "github.com/leoninew/pomelo-orbit/internal/queue/dispatch"
	tasksvc "github.com/leoninew/pomelo-orbit/internal/queue/task"
	taskrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewHTTPServer(cfg config.Config, logger *slog.Logger, database *sql.DB, taskRepo taskrepo.Repository) transporthttp.Server {
	deps := newHTTPServerDependencies(cfg, logger, database, taskRepo)
	return transporthttp.New(cfg, logger, deps)
}

type applicationServices struct {
	AuthService        authsvc.Service
	RoleService        rolesvc.Service
	UserService        usersvc.Service
	ProjectService     projectsvc.Service
	SettingsService    settingssvc.Service
	CredentialService  credentialsvc.Service
	EnvironmentService environmentsvc.Service
	RepositoryService  repositorysvc.Service
	PipelineService    pipelinesvc.Service
	PipelineRunService pipelinerunsvc.Service
	RouteService       routesvc.Service
	ApplicationService applicationsvc.Service
	ServiceService     servicesvc.Service
	DeploymentService  deploymentsvc.Service
	DialogueService    dialoguesvc.Service
	GatewayService     gatewaysvc.Service
	TaskService        tasksvc.Service
}

func newDeliveryMCPServer(actorUserId string, services applicationServices) (*mcp.Server, error) {
	deps := newDeliveryMCPDependencies(services)
	deps.ActorUserId = actorUserId
	return deliverymcp.NewServer(deps)
}

func newDeliveryMCPDependencies(services applicationServices) deliverymcp.Dependencies {
	return deliverymcp.Dependencies{
		Project:     services.ProjectService,
		Application: services.ApplicationService,
		Service:     services.ServiceService,
		Deployment:  services.DeploymentService,
		Environment: services.EnvironmentService,
		Gateway:     services.GatewayService,
		Route:       services.RouteService,
	}
}

func newApplicationServices(cfg config.Config, logger *slog.Logger, database *sql.DB, taskRepo taskrepo.Repository) applicationServices {
	stores := newDomainStores(database)
	tokenService := jwt.NewTokenService(cfg.Jwt.SecretKey)
	taskService := tasksvc.New(taskRepo, cfg.Worker.MaxAttempts)
	pipelineRunDispatcher := queuedispatch.NewPipelineRunDispatcher(taskService)
	deploymentDispatcher := queuedispatch.NewDeploymentDispatcher(taskService)
	authService := authsvc.New(stores.user, stores.auth, tokenService, logger, cfg.Jwt.SecretKey)
	transactionRunner := databasetx.NewTransactionRunner(database)
	credentialService := credentialsvc.New(stores.project, stores.credential, cfg.Jwt.SecretKey)
	environmentService := environmentsvc.New(stores.environment, stores.project, credentialService, credentialService, sshrunner.NewEnvironmentProber())
	projectService := projectsvc.New(stores.project, stores.user, stores.environment, credentialService, environmentService)
	pipelineLogStore := executionlog.Store{}
	deploymentLogStore := executionlog.NewDeploymentStore(cfg.Workspace.Deployment)
	dockerPathResolver := dockerDaemonPathResolver()
	pipelineWorkspace := pipelineworkspace.NewWithResolver(cfg.Workspace.Pipeline, dockerPathResolver)
	localSource := repositorysource.New(dockerPathResolver)
	targetResolver := environmentsvc.NewTargetResolver(stores.environment, credentialService)
	remoteRuntime := sshrunner.NewRuntime()
	routeManager := traefik.NewRouteManager(targetResolver, remoteRuntime)
	gatewayCore := gatewaysvc.New(stores.project, stores.environment, stores.application, stores.gateway, stores.service, stores.route, stores.deployment, cfg, dockerPathResolver, transactionRunner)
	deploymentService := deploymentsvc.NewCommandService(
		stores.project,
		stores.application,
		stores.service,
		stores.deployment,
		stores.gateway,
		deploymentDispatcher,
		logger,
		targetResolver,
		remoteRuntime,
		deploymentLogStore,
		gatewayCore,
	)
	gatewayService := gatewayCore
	applicationService := applicationsvc.New(stores.project, stores.application, stores.service)

	services := applicationServices{
		AuthService:        authService,
		RoleService:        rolesvc.New(stores.role),
		UserService:        usersvc.New(stores.user, stores.role, stores.project),
		ProjectService:     projectService,
		SettingsService:    settingssvc.New(cfg, envfile.NewStore(cfg.EnvFilePath)),
		CredentialService:  credentialService,
		EnvironmentService: environmentService,
		RepositoryService: repositorysvc.New(
			stores.project,
			stores.credential,
			stores.repository,
			localSource,
			logger,
		),
		PipelineService: pipelinesvc.New(stores.project, stores.pipeline, stores.application, stores.repository, logger),
		PipelineRunService: pipelinerunsvc.New(
			stores.project,
			stores.credential,
			stores.repository,
			stores.pipeline,
			stores.pipelineRun,
			stores.application,
			applicationService,
			transactionRunner,
			pipelineRunDispatcher,
			pipelineWorkspace,
			cfg.Jwt.SecretKey,
			logger,
			pipelinerunner.DockerRunner{},
			pipelineLogStore,
			localSource,
		),
		RouteService: routesvc.New(
			stores.project,
			stores.application,
			stores.service,
			stores.route,
			stores.gateway,
			routeManager,
			traefik.MkcertGenerator{},
			routeManager,
			transactionRunner,
		),
		ApplicationService: applicationService,
		ServiceService: servicesvc.New(
			stores.project,
			stores.application,
			stores.service,
			stores.deployment,
		),
		DeploymentService: deploymentService,
		GatewayService:    gatewayService,
		TaskService:       taskService,
	}
	services.DialogueService = dialoguesvc.New(
		stores.project,
		stores.dialogue,
		transactionRunner,
		cfg.LLM.MaxToolCallRounds,
		llmclient.New(cfg.LLM),
		deliverymcpclient.NewFactory(func(actorUserId string) (*mcp.Server, error) {
			return newDeliveryMCPServer(actorUserId, services)
		}),
	)
	return services
}

func newHTTPServerDependencies(cfg config.Config, logger *slog.Logger, database *sql.DB, taskRepo taskrepo.Repository) routes.Dependencies {
	services := newApplicationServices(cfg, logger, database, taskRepo)
	return routes.Dependencies{
		Database:           database,
		Authenticator:      security.New(logger, services.AuthService),
		AuthService:        services.AuthService,
		RoleService:        services.RoleService,
		UserService:        services.UserService,
		ProjectService:     services.ProjectService,
		SettingsService:    services.SettingsService,
		CredentialService:  services.CredentialService,
		EnvironmentService: services.EnvironmentService,
		RepositoryService:  services.RepositoryService,
		PipelineService:    services.PipelineService,
		PipelineRunService: services.PipelineRunService,
		RouteService:       services.RouteService,
		ApplicationService: services.ApplicationService,
		ServiceService:     services.ServiceService,
		DeploymentService:  services.DeploymentService,
		DialogueService:    services.DialogueService,
		GatewayService:     services.GatewayService,
		TaskService:        services.TaskService,
		TurnstileVerifier:  turnstile.NewVerifier(cfg.Turnstile),
	}
}

package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	transporthttp "gitee.com/leoninew/PomeloOrbit-go/internal/api/http"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/routes"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	deliverymcp "gitee.com/leoninew/PomeloOrbit-go/internal/api/mcp/delivery"
	applicationsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/usecase"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
	credentialsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/credential/usecase"
	deploymentsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/usecase"
	dialoguesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/dialogue/usecase"
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
	databasetx "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	llmclient "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/external/openai"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/external/traefik"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/external/turnstile"
	deliverymcpclient "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/mcp/delivery"
	deploymentrunner "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/runner/deployment"
	pipelinerunner "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/runner/pipeline"
	runtimepath "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/deploymentworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/envfile"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/pipelineworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/repositorysource"
	queuedispatch "gitee.com/leoninew/PomeloOrbit-go/internal/queue/dispatch"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func NewHTTPServer(cfg config.Config, logger *slog.Logger, database *sql.DB, taskRepo taskrepo.Repository) transporthttp.Server {
	deps := newHTTPServerDependencies(cfg, logger, database, taskRepo)
	return transporthttp.New(cfg, logger, deps, newDeliveryMCPHTTPHandler(deps))
}

func newDeliveryMCPHTTPHandler(deps routes.Dependencies) http.Handler {
	return deliverymcp.NewAuthenticatedStreamableHTTPHandler(
		func(ctx context.Context, authorization string) (*mcp.Server, error) {
			token := strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
			if token == "" {
				return nil, deliverymcp.UnauthenticatedError()
			}
			authenticated, err := deps.AuthService.Authenticate(ctx, token)
			if err != nil {
				return nil, err
			}
			return newDeliveryMCPServer(authenticated.User.Id, deps)
		},
	)
}

func newDeliveryMCPServer(actorUserId string, deps routes.Dependencies) (*mcp.Server, error) {
	return deliverymcp.NewServer(deliverymcp.Dependencies{
		ActorUserId: actorUserId,
		Project:     deps.ProjectService,
		Application: deps.ApplicationService,
		Service:     deps.ServiceService,
		Deployment:  deps.DeploymentService,
		Gateway:     deps.GatewayService,
	})
}

func newHTTPServerDependencies(cfg config.Config, logger *slog.Logger, database *sql.DB, taskRepo taskrepo.Repository) routes.Dependencies {
	stores := newDomainStores(database)
	tokenService := jwt.NewTokenService(cfg.Jwt.SecretKey)
	taskService := tasksvc.New(taskRepo, cfg.Worker.MaxAttempts)
	pipelineRunDispatcher := queuedispatch.NewPipelineRunDispatcher(taskService)
	deploymentDispatcher := queuedispatch.NewDeploymentDispatcher(taskService)
	authService := authsvc.New(stores.user, stores.auth, tokenService, logger)
	transactionRunner := databasetx.NewTransactionRunner(database)
	logStore := executionlog.Store{}
	pipelineWorkspace := pipelineworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	localSource := repositorysource.New(runtimepath.ResolvePhysicalPath)
	deploymentWorkspace := deploymentworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	routeManager := traefik.NewRouteManager(cfg)
	gatewayCore := gatewaysvc.New(stores.project, stores.application, stores.gateway, stores.service, stores.deployment, deploymentWorkspace)
	deploymentService := deploymentsvc.NewCommandService(
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
		gatewayCore,
	)
	gatewayService := gatewayCore.WithDeployer(deploymentService)
	applicationService := applicationsvc.New(stores.project, stores.application, stores.service)
	dialogueService := dialoguesvc.New(llmclient.New(cfg.LLM), deliverymcpclient.NewFactory(deliveryMCPEndpoint(cfg)))
	return routes.Dependencies{
		Database:          database,
		Authenticator:     security.New(logger, authService),
		AuthService:       authService,
		RoleService:       rolesvc.New(stores.role),
		UserService:       usersvc.New(stores.user, stores.role),
		ProjectService:    projectsvc.New(stores.project, stores.user),
		SettingsService:   settingssvc.New(cfg, envfile.NewStore(cfg.EnvFilePath)),
		CredentialService: credentialsvc.New(stores.project, stores.credential, cfg.Jwt.SecretKey),
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
			logStore,
			localSource,
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
		ApplicationService: applicationService,
		ServiceService: servicesvc.New(
			stores.project,
			stores.application,
			stores.service,
			stores.deployment,
		),
		DeploymentService: deploymentService,
		DialogueService:   dialogueService,
		GatewayService:    gatewayService,
		TaskService:       taskService,
		TurnstileVerifier: turnstile.NewVerifier(cfg.Turnstile),
	}
}

func deliveryMCPEndpoint(cfg config.Config) string {
	if publicUrl := strings.TrimRight(strings.TrimSpace(cfg.Server.PublicUrl), "/"); publicUrl != "" {
		return publicUrl + "/mcp"
	}
	return fmt.Sprintf("http://127.0.0.1:%d/mcp", cfg.Server.Port)
}

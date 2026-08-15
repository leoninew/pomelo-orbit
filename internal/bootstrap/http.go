package bootstrap

import (
	"context"
	"database/sql"
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	transporthttp "github.com/leoninew/pomelo-orbit/internal/api/http"
	"github.com/leoninew/pomelo-orbit/internal/api/http/routes"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
	deliverymcp "github.com/leoninew/pomelo-orbit/internal/api/mcp/delivery"
	applicationsvc "github.com/leoninew/pomelo-orbit/internal/application/application/usecase"
	authsvc "github.com/leoninew/pomelo-orbit/internal/application/auth/usecase"
	credentialsvc "github.com/leoninew/pomelo-orbit/internal/application/credential/usecase"
	deploymentsvc "github.com/leoninew/pomelo-orbit/internal/application/deployment/usecase"
	dialoguesvc "github.com/leoninew/pomelo-orbit/internal/application/dialogue/usecase"
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
	deploymentrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/deployment"
	pipelinerunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/pipeline"
	runtimepath "github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local/deploymentworkspace"
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
	authService := authsvc.New(stores.user, stores.auth, tokenService, logger, cfg.Jwt.SecretKey)
	transactionRunner := databasetx.NewTransactionRunner(database)
	logStore := executionlog.Store{}
	pipelineWorkspace := pipelineworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	localSource := repositorysource.New(runtimepath.ResolvePhysicalPath)
	deploymentWorkspace := deploymentworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	routeManager := traefik.NewRouteManager(cfg)
	gatewayCore := gatewaysvc.New(stores.project, stores.application, stores.gateway, stores.service, stores.deployment, cfg, transactionRunner)
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
	gatewayService := gatewayCore
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
			stores.application,
			stores.service,
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

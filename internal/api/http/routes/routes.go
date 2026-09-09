package routes

import (
	"log/slog"
	"net/http"

	transportresponse "github.com/leoninew/pomelo-orbit/internal/api/http/response"

	"github.com/gin-gonic/gin"
	authhandler "github.com/leoninew/pomelo-orbit/internal/api/http/handler/auth"
	transportmiddleware "github.com/leoninew/pomelo-orbit/internal/api/http/middleware"
	"github.com/leoninew/pomelo-orbit/internal/api/http/security"
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
	"github.com/leoninew/pomelo-orbit/internal/config"
	commonv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/common"
	tasksvc "github.com/leoninew/pomelo-orbit/internal/queue/task"
)

type Dependencies struct {
	MutatingUnitOfWork gin.HandlerFunc
	Authenticator      security.Authenticator
	AuthService        authsvc.Service
	RoleService        rolesvc.Service
	UserService        usersvc.Service
	ProjectService     projectsvc.Service
	SettingsService    settingssvc.Service
	CredentialService  credentialsvc.Service
	RepositoryService  repositorysvc.Service
	PipelineService    pipelinesvc.Service
	PipelineRunService pipelinerunsvc.Service
	RouteService       routesvc.Service
	ApplicationService applicationsvc.Service
	ServiceService     servicesvc.Service
	DeploymentService  deploymentsvc.Service
	DialogueService    dialoguesvc.Service
	EnvironmentService environmentsvc.Service
	GatewayService     gatewaysvc.Service
	TaskService        tasksvc.Service
	TurnstileVerifier  authhandler.TurnstileVerifier
}

type Router struct {
	cfg      config.Config
	logger   *slog.Logger
	deps     Dependencies
	fallback func(*gin.Engine)
}

func New(cfg config.Config, logger *slog.Logger, deps Dependencies, fallback func(*gin.Engine)) Router {
	return Router{cfg: cfg, logger: logger, deps: deps, fallback: fallback}
}

func (r Router) Handler() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	engine := gin.New()
	engine.HandleMethodNotAllowed = true
	engine.Use(transportmiddleware.RequestId())
	engine.Use(transportmiddleware.RealIP())
	engine.Use(transportmiddleware.LogRequest(r.logger, transportmiddleware.LogRequestConfig{
		Enabled:           r.cfg.Logging.HTTP.Enabled,
		RequestBodyLimit:  r.cfg.Logging.HTTP.RequestBodyLimit,
		ResponseBodyLimit: r.cfg.Logging.HTTP.ResponseBodyLimit,
		SkipAssetEnabled:  r.cfg.Logging.HTTP.SkipAssetEnabled,
	}))
	engine.Use(transportmiddleware.Recovery(r.logger))
	engine.Use(transportmiddleware.Cors(r.cfg.Server.CorsAllowedOrigins, r.cfg.Server.ApiPathPrefixes))
	engine.GET("/api/health", func(c *gin.Context) {
		transportresponse.ProtoJSON(c, http.StatusOK, &commonv1.HealthResp{Status: "ok"})
	})
	// Dialogue requests can run external LLM and MCP calls. The MCP server owns
	// transactions for its tool writes, so this route must not hold a request UoW.
	r.registerDialogue(engine)
	if r.deps.MutatingUnitOfWork != nil {
		// Request-scoped UoW for mutating API routes (health is registered above).
		engine.Use(r.deps.MutatingUnitOfWork)
	}
	r.registerAuth(engine)
	r.registerUser(engine)
	r.registerRole(engine)
	r.registerSettings(engine)
	r.registerProject(engine)
	r.registerEnvironment(engine)
	r.registerCredential(engine)
	r.registerRepository(engine)
	r.registerPipeline(engine)
	r.registerPipelineRun(engine)
	r.registerApplication(engine)
	r.registerService(engine)
	r.registerDeployment(engine)
	r.registerGateway(engine)
	r.registerRoute(engine)
	r.registerTask(engine)
	r.fallback(engine)
	return engine
}

package routes

import (
	"database/sql"
	"log/slog"
	"net/http"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"

	authhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/auth"
	transportmiddleware "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/middleware"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/security"
	applicationsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/usecase"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
	credentialsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/credential/usecase"
	deploymentsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/usecase"
	environmentsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/environment/usecase"
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
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	commonv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/common"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database/tx"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	Database           *sql.DB
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
	EnvironmentService environmentsvc.Service
	RouteService       routesvc.Service
	ApplicationService applicationsvc.Service
	ServiceService     servicesvc.Service
	DeploymentService  deploymentsvc.Service
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
	engine.Use(transportmiddleware.RequestID())
	engine.Use(transportmiddleware.RealIP())
	engine.Use(transportmiddleware.LogRequest(r.logger, transportmiddleware.LogRequestConfig{BodyEnabled: r.cfg.Logging.HTTPBodyEnabled, BodyMaxBytes: r.cfg.Logging.HTTPBodyMaxBytes, SkipAssets200Enabled: r.cfg.Logging.HTTPSkipAssets200Enabled}))
	engine.Use(transportmiddleware.Recovery(r.logger))
	engine.Use(transportmiddleware.CORS(r.cfg.Server.CORSAllowedOrigins, r.cfg.Server.ApiPathPrefixes))
	engine.GET("/api/health", func(c *gin.Context) {
		transportresponse.ProtoJSON(c, http.StatusOK, &commonv1.HealthResp{Status: "ok"})
	})
	if r.deps.Database != nil {
		// Request-scoped UoW for mutating API routes (health is registered above).
		engine.Use(tx.Middleware(r.deps.Database))
	}
	r.registerAuth(engine)
	r.registerUser(engine)
	r.registerRole(engine)
	r.registerSettings(engine)
	r.registerProject(engine)
	r.registerCredential(engine)
	r.registerRepository(engine)
	r.registerPipeline(engine)
	r.registerPipelineRun(engine)
	r.registerApplication(engine)
	r.registerService(engine)
	r.registerEnvironment(engine)
	r.registerDeployment(engine)
	r.registerGateway(engine)
	r.registerRoute(engine)
	r.registerTask(engine)
	r.fallback(engine)
	return engine
}

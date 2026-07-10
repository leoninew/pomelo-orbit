package routes

import (
	"log/slog"
	"net/http"

	transportresponse "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/response"

	authhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/auth"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	transportmiddleware "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/middleware"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
	cdsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/usecase"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/usecase"
	projectsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/project/usecase"
	rolesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/role/usecase"
	settingssvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/usecase"
	usersvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/usecase"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	"github.com/gin-gonic/gin"
)

type Dependencies struct {
	Authenticator     authz.Authenticator
	AuthService       authsvc.Service
	RoleService       rolesvc.Service
	UserService       usersvc.Service
	ProjectService    projectsvc.Service
	SettingsService   settingssvc.Service
	CIService         cisvc.Service
	CDService         cdsvc.Service
	TaskService       tasksvc.Service
	TurnstileVerifier authhandler.TurnstileVerifier
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
		transportresponse.ProtoJSON(c, http.StatusOK, &pomeloorbit.HealthResp{Status: "ok"})
	})
	r.registerAuth(engine)
	r.registerUser(engine)
	r.registerRole(engine)
	r.registerSettings(engine)
	r.registerProject(engine)
	r.registerCI(engine)
	r.registerCD(engine)
	r.registerTask(engine)
	r.fallback(engine)
	return engine
}

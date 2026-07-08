package transporthttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"

	authhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/auth"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	cdhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/cd"
	cihandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/ci"
	projecthandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/project"
	rolehandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/role"
	settingshandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/settings"
	taskhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/task"
	userhandler "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/user"
	transportmiddleware "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/middleware"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth"
	cdsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci"
	projectsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/project"
	rolesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/role"
	settingssvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings"
	usersvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/user"
	jwt "gitee.com/leoninew/PomeloOrbit-go/internal/auth/jwt"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/logger/logstore"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/traefik"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	store "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc"
	cdrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/cd"
	cirepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/ci"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	rolerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/role"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
	userrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/user"
)

type Server struct {
	appCfg            config.Config
	cfg               config.ServerConfig
	logger            *slog.Logger
	store             store.Store
	logStore          logstore.LogStore
	ciRepository      cirepo.Repository
	cdRepository      cdrepo.Repository
	authService       authsvc.Service
	roleService       rolesvc.Service
	userService       usersvc.Service
	projectService    projectsvc.Service
	settingsService   settingssvc.Service
	ciService         cisvc.Service
	cdService         cdsvc.Service
	tokenService      jwt.TokenService
	taskService       tasksvc.Service
	userRepository    userrepo.Repository
	roleRepository    rolerepo.Repository
	turnstileVerifier turnstileVerifier
}

func New(cfg config.Config, logger *slog.Logger, store store.Store, tasks taskrepo.Repository, defaultMaxAttempts int) Server {
	tokenService := jwt.NewTokenService(cfg.JWT.SecretKey)
	userRepository := userrepo.NewRepository(store.DB(), store.Driver())
	roleRepository := rolerepo.NewRepository(store.DB(), store.Driver())
	projectRepository := projectrepo.NewRepository(store.DB(), store.Driver())
	taskService := tasksvc.New(tasks, defaultMaxAttempts)
	ciRepository := cirepo.NewRepository(store.DB(), store.Driver())
	cdRepository := cdrepo.NewRepository(store.DB(), store.Driver())
	logStore := logstore.LogStore{}
	return Server{appCfg: cfg, cfg: cfg.Server, logger: logger, store: store, logStore: logStore, ciRepository: ciRepository, cdRepository: cdRepository, authService: authsvc.New(userRepository, tokenService, logger), roleService: rolesvc.New(roleRepository), userService: usersvc.New(userRepository), projectService: projectsvc.New(projectRepository, userRepository), settingsService: settingssvc.New(cfg), ciService: cisvc.New(ciRepository, taskService, cfg.DataRoot(), cfg.JWT.SecretKey, logger, logStore), cdService: newCDService(cdRepository, taskService, cfg, logger, logStore), tokenService: tokenService, taskService: taskService, userRepository: userRepository, roleRepository: roleRepository, turnstileVerifier: newTurnstileVerifier(cfg.Turnstile)}
}

func (s Server) Handler() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	r.Use(transportmiddleware.RequestID())
	r.Use(transportmiddleware.RealIP())
	r.Use(transportmiddleware.LogRequest(s.logger, transportmiddleware.LogRequestConfig{BodyEnabled: s.appCfg.Logging.HTTPBodyEnabled, BodyMaxBytes: s.appCfg.Logging.HTTPBodyMaxBytes, SkipAssets200Enabled: s.appCfg.Logging.HTTPSkipAssets200Enabled}))
	r.Use(transportmiddleware.Recovery(s.logger))
	r.Use(transportmiddleware.CORS(s.appCfg.Server.CORSAllowedOrigins, s.appCfg.Server.ApiPathPrefixes))

	r.GET("/api/health", func(c *gin.Context) {
		c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.HealthResp{Status: "ok"}})
	})
	authenticator := authz.New(s.logger, s.userRepository, s.tokenService)
	authhandler.New(s.logger, s.appCfg.Turnstile, s.authService, authenticator, s.turnstileVerifier, s.userRepository).Register(r)
	userhandler.New(s.logger, s.userService, authenticator, s.userRepository, s.roleRepository).Register(r)
	rolehandler.New(s.logger, s.roleService, authenticator, s.roleRepository).Register(r)
	settingshandler.New(s.logger, s.settingsService, authenticator).Register(r)
	projecthandler.New(s.logger, s.projectService, authenticator).Register(r)
	ciService := cisvc.New(s.ciRepository, s.taskService, s.appCfg.DataRoot(), s.appCfg.JWT.SecretKey, s.logger, s.logStore)
	ciHandler := cihandler.New(s.logger, ciService, authenticator)
	ciHandler.RegisterRepositoryRoutes(r)
	ciHandler.RegisterTemplateRoutes(r)
	ciHandler.RegisterBuildStageRoutes(r)
	ciHandler.RegisterPipelineRunRoutes(r)
	ciHandler.RegisterSnapshotRoutes(r)
	ciHandler.RegisterArtifactRoutes(r)
	ciHandler.RegisterCredentialRoutes(r)
	cdService := newCDService(s.cdRepository, s.taskService, s.appCfg, s.logger, s.logStore)
	cdHandler := cdhandler.New(s.logger, cdService, authenticator)
	cdHandler.RegisterApplicationRoutes(r)
	cdHandler.RegisterDeploymentRoutes(r)
	cdHandler.RegisterApplicationExtraRoutes(r)
	cdHandler.RegisterRouteRoutes(r)
	cdHandler.RegisterTraefikRouteRoutes(r)
	taskhandler.New(s.logger, s.taskService).Register(r)
	r.NoRoute(func(c *gin.Context) {
		if isAPIPath(c.Request.URL.Path, s.appCfg.Server.ApiPathPrefixes) {
			c.JSON(http.StatusNotFound, gin.H{"detail": "Not Found"})
			return
		}
		if s.serveStatic(c, "static") {
			return
		}
		c.JSON(http.StatusNotFound, gin.H{"detail": "Not Found"})
	})
	return r
}

func (s Server) serveStatic(c *gin.Context, staticDir string) bool {
	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodHead {
		return false
	}

	indexPath := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return false
	}

	requestPath := path.Clean("/" + c.Request.URL.Path)
	if requestPath != "/" {
		filePath := filepath.Join(staticDir, filepath.FromSlash(strings.TrimPrefix(requestPath, "/")))
		info, err := os.Stat(filePath)
		if err == nil && !info.IsDir() {
			if filepath.Clean(filePath) == filepath.Clean(indexPath) {
				return s.serveIndexHTML(c, indexPath)
			}
			http.ServeFile(c.Writer, c.Request, filePath)
			return true
		}
	}

	return s.serveIndexHTML(c, indexPath)
}

func (s Server) serveIndexHTML(c *gin.Context, indexPath string) bool {
	content, err := os.ReadFile(indexPath)
	if err != nil {
		return false
	}
	c.Header("Content-Type", "text/html; charset=utf-8")
	c.Status(http.StatusOK)
	if c.Request.Method == http.MethodHead {
		return true
	}
	_, _ = c.Writer.Write(injectRuntimeConfig(content, s.appCfg.Server.PublicURL))
	return true
}

func injectRuntimeConfig(content []byte, publicURL string) []byte {
	configValue := map[string]string{}
	publicURL = strings.TrimRight(strings.TrimSpace(publicURL), "/")
	if publicURL != "" {
		configValue["publicUrl"] = publicURL
	}
	configJSON, err := json.Marshal(configValue)
	if err != nil {
		panic(fmt.Sprintf("marshal runtime config: %v", err))
	}
	script := []byte("<script>window.__CONFIG__ = " + string(configJSON) + ";</script>")
	placeholder := []byte("<!-- __RUNTIME_CONFIG__ -->")
	if bytes.Contains(content, placeholder) {
		return bytes.Replace(content, placeholder, script, 1)
	}
	headEnd := []byte("</head>")
	if bytes.Contains(content, headEnd) {
		return bytes.Replace(content, headEnd, append(script, headEnd...), 1)
	}
	return content
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
}

func newCDService(store cdsvc.Store, tasks cdsvc.TaskService, cfg config.Config, logger *slog.Logger, logStore logstore.LogStore) cdsvc.Service {
	routeManager := traefik.NewRouteManager(cfg)
	return cdsvc.New(store, tasks, cfg, logger, logStore, routeManager, traefik.MkcertGenerator{}, routeManager)
}

func isAPIPath(requestPath string, prefixes []string) bool {
	return hasAPIPathPrefix(requestPath, prefixes)
}

func hasAPIPathPrefix(requestPath string, prefixes []string) bool {
	for _, prefix := range prefixes {
		if requestPath == prefix || strings.HasPrefix(requestPath, prefix+"/") {
			return true
		}
	}
	return false
}

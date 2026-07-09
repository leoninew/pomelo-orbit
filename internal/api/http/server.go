package transporthttp

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	transportcodec "gitee.com/leoninew/PomeloOrbit-go/internal/api/http/codec"

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
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	pomeloorbit "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
)

type ServerDependencies struct {
	Authenticator     authz.Authenticator
	AuthService       authsvc.Service
	RoleService       rolesvc.Service
	UserService       usersvc.Service
	ProjectService    projectsvc.Service
	SettingsService   settingssvc.Service
	CIService         cisvc.Service
	CDService         cdsvc.Service
	TaskService       tasksvc.Service
	AuthHandlerStore  authhandler.Store
	UserStore         userhandler.Store
	RoleStore         rolehandler.Store
	UserRoleStore     userhandler.RoleStore
	TurnstileVerifier authhandler.TurnstileVerifier
}

type Server struct {
	appCfg config.Config
	cfg    config.ServerConfig
	logger *slog.Logger
	deps   ServerDependencies
}

func New(cfg config.Config, logger *slog.Logger, deps ServerDependencies) Server {
	return Server{appCfg: cfg, cfg: cfg.Server, logger: logger, deps: deps}
}

func (s Server) Handler() http.Handler {
	gin.SetMode(gin.ReleaseMode)
	r := gin.New()
	s.registerMiddleware(r)
	s.registerHealthRoutes(r)
	s.registerAuthRoutes(r)
	s.registerUserRoutes(r)
	s.registerRoleRoutes(r)
	s.registerSettingsRoutes(r)
	s.registerProjectRoutes(r)
	s.registerCIRoutes(r)
	s.registerCDRoutes(r)
	s.registerTaskRoutes(r)
	s.registerFallbackRoutes(r)
	return r
}

func (s Server) registerMiddleware(r *gin.Engine) {
	r.Use(transportmiddleware.RequestID())
	r.Use(transportmiddleware.RealIP())
	r.Use(transportmiddleware.LogRequest(s.logger, transportmiddleware.LogRequestConfig{BodyEnabled: s.appCfg.Logging.HTTPBodyEnabled, BodyMaxBytes: s.appCfg.Logging.HTTPBodyMaxBytes, SkipAssets200Enabled: s.appCfg.Logging.HTTPSkipAssets200Enabled}))
	r.Use(transportmiddleware.Recovery(s.logger))
	r.Use(transportmiddleware.CORS(s.appCfg.Server.CORSAllowedOrigins, s.appCfg.Server.ApiPathPrefixes))
}

func (s Server) registerHealthRoutes(r *gin.Engine) {
	r.GET("/api/health", func(c *gin.Context) {
		c.Render(http.StatusOK, transportcodec.ProtoJSON{Message: &pomeloorbit.HealthResp{Status: "ok"}})
	})
}

func (s Server) registerAuthRoutes(r *gin.Engine) {
	authhandler.New(s.logger, s.appCfg.Turnstile, s.deps.AuthService, s.deps.Authenticator, s.deps.TurnstileVerifier, s.deps.AuthHandlerStore).Register(r)
}

func (s Server) registerUserRoutes(r *gin.Engine) {
	userhandler.New(s.logger, s.deps.UserService, s.deps.Authenticator, s.deps.UserStore, s.deps.UserRoleStore).Register(r)
}

func (s Server) registerRoleRoutes(r *gin.Engine) {
	rolehandler.New(s.logger, s.deps.RoleService, s.deps.Authenticator, s.deps.RoleStore).Register(r)
}

func (s Server) registerSettingsRoutes(r *gin.Engine) {
	settingshandler.New(s.logger, s.deps.SettingsService, s.deps.Authenticator).Register(r)
}

func (s Server) registerProjectRoutes(r *gin.Engine) {
	projecthandler.New(s.logger, s.deps.ProjectService, s.deps.Authenticator).Register(r)
}

func (s Server) registerCIRoutes(r *gin.Engine) {
	ciHandler := cihandler.New(s.logger, s.deps.CIService, s.deps.Authenticator)
	ciHandler.RegisterRepositoryRoutes(r)
	ciHandler.RegisterTemplateRoutes(r)
	ciHandler.RegisterBuildStageRoutes(r)
	ciHandler.RegisterPipelineRunRoutes(r)
	ciHandler.RegisterSnapshotRoutes(r)
	ciHandler.RegisterArtifactRoutes(r)
	ciHandler.RegisterCredentialRoutes(r)
}

func (s Server) registerCDRoutes(r *gin.Engine) {
	cdHandler := cdhandler.New(s.logger, s.deps.CDService, s.deps.Authenticator)
	cdHandler.RegisterApplicationRoutes(r)
	cdHandler.RegisterDeploymentRoutes(r)
	cdHandler.RegisterApplicationExtraRoutes(r)
	cdHandler.RegisterRouteRoutes(r)
	cdHandler.RegisterTraefikRouteRoutes(r)
}

func (s Server) registerTaskRoutes(r *gin.Engine) {
	taskhandler.New(s.logger, s.deps.TaskService).Register(r)
}

func (s Server) registerFallbackRoutes(r *gin.Engine) {
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

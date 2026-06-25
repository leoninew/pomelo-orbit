package transporthttp

import (
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"backend/internal/config"
	"backend/internal/repository"
	cdrepo "backend/internal/repository/cd"
	cirepo "backend/internal/repository/ci"
	projectrepo "backend/internal/repository/project"
	rolerepo "backend/internal/repository/role"
	taskrepo "backend/internal/repository/task"
	userrepo "backend/internal/repository/user"
	authsvc "backend/internal/service/auth"
	cdsvc "backend/internal/service/cd"
	cisvc "backend/internal/service/ci"
	projectsvc "backend/internal/service/project"
	rolesvc "backend/internal/service/role"
	settingssvc "backend/internal/service/settings"
	tasksvc "backend/internal/service/task"
	usersvc "backend/internal/service/user"
	authhandler "backend/internal/transport/http/handler/auth"
	"backend/internal/transport/http/handler/authz"
	cdhandler "backend/internal/transport/http/handler/cd"
	cihandler "backend/internal/transport/http/handler/ci"
	projecthandler "backend/internal/transport/http/handler/project"
	rolehandler "backend/internal/transport/http/handler/role"
	settingshandler "backend/internal/transport/http/handler/settings"
	taskhandler "backend/internal/transport/http/handler/task"
	userhandler "backend/internal/transport/http/handler/user"
	transportmiddleware "backend/internal/transport/http/middleware"
	transportresponse "backend/internal/transport/http/response"
)

type chiRouter interface {
	Get(pattern string, handlerFn http.HandlerFunc)
	Post(pattern string, handlerFn http.HandlerFunc)
	Put(pattern string, handlerFn http.HandlerFunc)
	Delete(pattern string, handlerFn http.HandlerFunc)
}

type Server struct {
	appCfg            config.Config
	cfg               config.ServerConfig
	logger            *slog.Logger
	store             repository.Store
	ciRepository      cirepo.Repository
	cdRepository      cdrepo.Repository
	authService       authsvc.Service
	roleService       rolesvc.Service
	userService       usersvc.Service
	projectService    projectsvc.Service
	settingsService   settingssvc.Service
	ciService         cisvc.Service
	cdService         cdsvc.Service
	tokenService      authsvc.TokenService
	taskService       tasksvc.Service
	userRepository    userrepo.Repository
	roleRepository    rolerepo.Repository
	turnstileVerifier turnstileVerifier
}

func New(cfg config.Config, logger *slog.Logger, store repository.Store, tasks taskrepo.Repository, defaultMaxAttempts int) Server {
	tokenService := authsvc.NewTokenService(jwtSecret(cfg))
	userRepository := userrepo.NewRepository(store.DB(), store.Driver())
	roleRepository := rolerepo.NewRepository(store.DB(), store.Driver())
	projectRepository := projectrepo.NewRepository(store.DB(), store.Driver())
	taskService := tasksvc.New(tasks, defaultMaxAttempts)
	ciRepository := cirepo.NewRepository(store.DB(), store.Driver())
	cdRepository := cdrepo.NewRepository(store.DB(), store.Driver())
	return Server{appCfg: cfg, cfg: cfg.Server, logger: logger, store: store, ciRepository: ciRepository, cdRepository: cdRepository, authService: authsvc.New(userRepository, tokenService, logger), roleService: rolesvc.New(roleRepository), userService: usersvc.New(userRepository), projectService: projectsvc.New(projectRepository, userRepository), settingsService: settingssvc.New(cfg), ciService: cisvc.New(ciRepository, taskService, cfg.DataRoot(), cfg.JWT.SecretKey, logger), cdService: cdsvc.New(cdRepository, taskService, cfg, logger), tokenService: tokenService, taskService: taskService, userRepository: userRepository, roleRepository: roleRepository, turnstileVerifier: newTurnstileVerifier(cfg.Turnstile)}
}

func (s Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(transportmiddleware.LogRequest(s.logger, transportmiddleware.LogRequestConfig{BodyEnabled: s.appCfg.Logging.HTTPBodyEnabled, BodyMaxBytes: s.appCfg.Logging.HTTPBodyMaxBytes}))
	r.Use(middleware.Recoverer)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		transportresponse.JSON(s.logger, w, http.StatusOK, map[string]string{"status": "ok"})
	})
	authenticator := authz.New(s.logger, s.userRepository, s.tokenService)
	authhandler.New(s.logger, s.appCfg.Turnstile, s.authService, authenticator, s.turnstileVerifier, s.userRepository).Register(r)
	userhandler.New(s.logger, s.userService, authenticator, s.userRepository, s.roleRepository).Register(r)
	rolehandler.New(s.logger, s.roleService, authenticator, s.roleRepository).Register(r)
	settingshandler.New(s.logger, s.settingsService, authenticator).Register(r)
	projecthandler.New(s.logger, s.projectService, authenticator).Register(r)
	ciService := cisvc.New(s.ciRepository, s.taskService, s.appCfg.DataRoot(), s.appCfg.JWT.SecretKey, s.logger)
	ciHandler := cihandler.New(s.logger, ciService, authenticator)
	ciHandler.RegisterRepositoryRoutes(r)
	ciHandler.RegisterTemplateRoutes(r)
	ciHandler.RegisterBuildStageRoutes(r)
	ciHandler.RegisterPipelineRunRoutes(r)
	ciHandler.RegisterSnapshotRoutes(r)
	ciHandler.RegisterArtifactRoutes(r)
	ciHandler.RegisterCredentialRoutes(r)
	cdService := cdsvc.New(s.cdRepository, s.taskService, s.appCfg, s.logger)
	cdHandler := cdhandler.New(s.logger, cdService, authenticator)
	cdHandler.RegisterApplicationRoutes(r)
	cdHandler.RegisterDeploymentRoutes(r)
	cdHandler.RegisterApplicationExtraRoutes(r)
	cdHandler.RegisterRouteRoutes(r)
	cdHandler.RegisterTraefikRouteRoutes(r)
	taskhandler.New(s.logger, s.taskService).Register(r)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasPrefix(r.URL.Path, "/api/") {
			transportresponse.JSON(s.logger, w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
			return
		}
		if serveStatic(w, r, "static") {
			return
		}
		transportresponse.JSON(s.logger, w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
	})
	return r
}

func serveStatic(w http.ResponseWriter, r *http.Request, staticDir string) bool {
	if r.Method != http.MethodGet && r.Method != http.MethodHead {
		return false
	}

	indexPath := filepath.Join(staticDir, "index.html")
	if _, err := os.Stat(indexPath); err != nil {
		return false
	}

	requestPath := path.Clean("/" + r.URL.Path)
	if requestPath != "/" {
		filePath := filepath.Join(staticDir, filepath.FromSlash(strings.TrimPrefix(requestPath, "/")))
		info, err := os.Stat(filePath)
		if err == nil && !info.IsDir() {
			http.ServeFile(w, r, filePath)
			return true
		}
	}

	http.ServeFile(w, r, indexPath)
	return true
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
}

func urlParam(r *http.Request, key string) string {
	return strings.TrimSpace(chi.URLParam(r, key))
}

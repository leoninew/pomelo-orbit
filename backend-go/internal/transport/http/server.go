package transporthttp

import (
	"fmt"
	"log/slog"
	"net/http"
	"strings"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"

	"backend/internal/config"
	"backend/internal/repository"
	projectrepo "backend/internal/repository/project"
	rolerepo "backend/internal/repository/role"
	taskrepo "backend/internal/repository/task"
	userrepo "backend/internal/repository/user"
	authsvc "backend/internal/service/auth"
	cisvc "backend/internal/service/ci"
	projectsvc "backend/internal/service/project"
	rolesvc "backend/internal/service/role"
	settingssvc "backend/internal/service/settings"
	tasksvc "backend/internal/service/task"
	usersvc "backend/internal/service/user"
	authhandler "backend/internal/transport/http/handler/auth"
	"backend/internal/transport/http/handler/authz"
	cihandler "backend/internal/transport/http/handler/ci"
	projecthandler "backend/internal/transport/http/handler/project"
	rolehandler "backend/internal/transport/http/handler/role"
	settingshandler "backend/internal/transport/http/handler/settings"
	taskhandler "backend/internal/transport/http/handler/task"
	userhandler "backend/internal/transport/http/handler/user"
	transportmiddleware "backend/internal/transport/http/middleware"
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
	authService       authsvc.Service
	roleService       rolesvc.Service
	userService       usersvc.Service
	projectService    projectsvc.Service
	settingsService   settingssvc.Service
	ciService         cisvc.Service
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
	return Server{appCfg: cfg, cfg: cfg.Server, logger: logger, store: store, authService: authsvc.New(userRepository, tokenService, logger), roleService: rolesvc.New(roleRepository), userService: usersvc.New(userRepository), projectService: projectsvc.New(projectRepository, userRepository), settingsService: settingssvc.New(cfg), ciService: cisvc.New(store, taskService, cfg.DataRoot()), tokenService: tokenService, taskService: taskService, userRepository: userRepository, roleRepository: roleRepository, turnstileVerifier: newTurnstileVerifier(cfg.Turnstile)}
}

func (s Server) Handler() http.Handler {
	r := chi.NewRouter()
	r.Use(middleware.RequestID)
	r.Use(middleware.RealIP)
	r.Use(transportmiddleware.LogRequest(s.logger))
	r.Use(middleware.Recoverer)

	r.Get("/api/health", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	authenticator := authz.New(s.logger, s.userRepository, s.tokenService)
	authhandler.New(s.logger, s.appCfg.Turnstile, s.authService, authenticator, s.turnstileVerifier, s.userRepository).Register(r)
	userhandler.New(s.logger, s.userService, authenticator, s.userRepository, s.roleRepository).Register(r)
	rolehandler.New(s.logger, s.roleService, authenticator, s.roleRepository).Register(r)
	settingshandler.New(s.logger, s.settingsService, authenticator).Register(r)
	projecthandler.New(s.logger, s.projectService, authenticator).Register(r)
	ciService := cisvc.New(s.store, s.taskService, s.appCfg.DataRoot())
	ciHandler := cihandler.New(s.logger, ciService, authenticator)
	ciHandler.RegisterRepositoryRoutes(r)
	ciHandler.RegisterTemplateRoutes(r)
	ciHandler.RegisterBuildStageRoutes(r)
	ciHandler.RegisterPipelineRunRoutes(r)
	s.registerDashboardRoutes(r)
	taskhandler.New(s.logger, s.taskService).Register(r)
	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		if len(r.URL.Path) >= 5 && r.URL.Path[:5] == "/api/" {
			writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
			return
		}
		writeJSON(w, http.StatusNotFound, map[string]string{"detail": "Not Found"})
	})
	return r
}

func (s Server) Addr() string {
	return fmt.Sprintf("%s:%d", s.cfg.Host, s.cfg.Port)
}

func urlParam(r *http.Request, key string) string {
	return strings.TrimSpace(chi.URLParam(r, key))
}

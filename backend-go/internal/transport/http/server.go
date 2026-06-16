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
	rolerepo "backend/internal/repository/role"
	taskrepo "backend/internal/repository/task"
	userrepo "backend/internal/repository/user"
	authsvc "backend/internal/service/auth"
	rolesvc "backend/internal/service/role"
	tasksvc "backend/internal/service/task"
	usersvc "backend/internal/service/user"
	authhandler "backend/internal/transport/http/handler/auth"
	"backend/internal/transport/http/handler/authz"
	rolehandler "backend/internal/transport/http/handler/role"
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
	return Server{appCfg: cfg, cfg: cfg.Server, logger: logger, store: store, authService: authsvc.New(userRepository, tokenService, logger), roleService: rolesvc.New(roleRepository), userService: usersvc.New(userRepository), tokenService: tokenService, taskService: tasksvc.New(tasks, defaultMaxAttempts), userRepository: userRepository, roleRepository: roleRepository, turnstileVerifier: newTurnstileVerifier(cfg.Turnstile)}
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
	s.registerSettingsRoutes(r)
	s.registerProjectRoutes(r)
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

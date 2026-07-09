package bootstrap

import (
	"log/slog"

	transporthttp "gitee.com/leoninew/PomeloOrbit-go/internal/api/http"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
	cdsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/usecase"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/usecase"
	projectsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/project/usecase"
	rolesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/role/usecase"
	settingssvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/usecase"
	usersvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/usecase"
	jwt "gitee.com/leoninew/PomeloOrbit-go/internal/auth/jwt"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/config/envfile"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/traefik"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/turnstile"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	store "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc"
	cdrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/cd"
	cirepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/ci"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	rolerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/role"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
	userrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/user"
)

func NewHTTPServer(cfg config.Config, logger *slog.Logger, repositoryStore store.Store, taskRepo taskrepo.Repository) transporthttp.Server {
	return transporthttp.New(cfg, logger, newHTTPServerDependencies(cfg, logger, repositoryStore, taskRepo))
}

func newHTTPServerDependencies(cfg config.Config, logger *slog.Logger, repositoryStore store.Store, taskRepo taskrepo.Repository) transporthttp.ServerDependencies {
	tokenService := jwt.NewTokenService(cfg.JWT.SecretKey)
	userRepository := userrepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	roleRepository := rolerepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	projectRepository := projectrepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	ciRepository := cirepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	cdRepository := cdrepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	taskService := tasksvc.New(taskRepo, cfg.Worker.MaxAttempts)
	logStore := executionlog.Store{}
	routeManager := traefik.NewRouteManager(cfg)
	return transporthttp.ServerDependencies{
		Authenticator:     authz.New(logger, userRepository, tokenService),
		AuthService:       authsvc.New(userRepository, tokenService, logger),
		RoleService:       rolesvc.New(roleRepository),
		UserService:       usersvc.New(userRepository),
		ProjectService:    projectsvc.New(projectRepository, userRepository),
		SettingsService:   settingssvc.New(cfg, envfile.NewStore(cfg)),
		CIService:         cisvc.New(ciRepository, taskService, cfg.DataRoot(), cfg.JWT.SecretKey, logger, logStore),
		CDService:         cdsvc.New(cdRepository, taskService, cfg, logger, logStore, routeManager, traefik.MkcertGenerator{}, routeManager),
		TaskService:       taskService,
		AuthHandlerStore:  userRepository,
		UserStore:         userRepository,
		RoleStore:         roleRepository,
		UserRoleStore:     roleRepository,
		TurnstileVerifier: turnstile.NewVerifier(cfg.Turnstile),
	}
}

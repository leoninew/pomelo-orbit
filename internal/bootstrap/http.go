package bootstrap

import (
	"log/slog"

	"github.com/jmoiron/sqlx"

	transporthttp "gitee.com/leoninew/PomeloOrbit-go/internal/api/http"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/routes"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth/usecase"
	cdsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/usecase"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/usecase"
	projectsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/project/usecase"
	rolesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/role/usecase"
	settingssvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings/usecase"
	usersvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/user/usecase"
	jwt "gitee.com/leoninew/PomeloOrbit-go/internal/auth/jwt"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/external/traefik"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/external/turnstile"
	cdrunner "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/runner/cd"
	dockerci "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/runner/dockerci"
	runtimepath "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/cdworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/ciworkspace"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/envfile"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	queuedispatch "gitee.com/leoninew/PomeloOrbit-go/internal/queue/dispatch"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	rolerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/role"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
	userrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/user"
	cdrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlx/cd"
	cirepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlx/ci"
)

func NewHTTPServer(cfg config.Config, logger *slog.Logger, database *sqlx.DB, taskRepo taskrepo.Repository) transporthttp.Server {
	return transporthttp.New(cfg, logger, newHTTPServerDependencies(cfg, logger, database, taskRepo))
}

func newHTTPServerDependencies(cfg config.Config, logger *slog.Logger, database *sqlx.DB, taskRepo taskrepo.Repository) routes.Dependencies {
	tokenService := jwt.NewTokenService(cfg.JWT.SecretKey)
	userRepository := userrepo.NewRepository(database, cfg.Database.Driver)
	roleRepository := rolerepo.NewRepository(database, cfg.Database.Driver)
	projectRepository := projectrepo.NewRepository(database, cfg.Database.Driver)
	ciRepository := cirepo.NewRepository(database, cfg.Database.Driver)
	cdRepository := cdrepo.NewRepository(database, cfg.Database.Driver)
	taskService := tasksvc.New(taskRepo, cfg.Worker.MaxAttempts)
	ciDispatcher := queuedispatch.NewCIDispatcher(taskService)
	cdDispatcher := queuedispatch.NewCDDispatcher(taskService)
	authService := authsvc.New(userRepository, tokenService, logger)
	logStore := executionlog.Store{}
	ciWorkspace := ciworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	cdWorkspace := cdworkspace.NewWithResolver(cfg.DataRoot(), runtimepath.ResolvePhysicalDataRoot)
	routeManager := traefik.NewRouteManager(cfg)
	return routes.Dependencies{
		Authenticator:     authz.New(logger, authService),
		AuthService:       authService,
		RoleService:       rolesvc.New(roleRepository),
		UserService:       usersvc.New(userRepository, roleRepository),
		ProjectService:    projectsvc.New(projectRepository, userRepository),
		SettingsService:   settingssvc.New(cfg, envfile.NewStore(cfg.EnvFilePath)),
		CIService:         cisvc.New(ciRepository, ciDispatcher, ciWorkspace, cfg.JWT.SecretKey, logger, dockerci.DockerRunner{}, logStore),
		CDService:         cdsvc.New(cdRepository, cdDispatcher, cfg, logger, cdWorkspace, cdrunner.CommandQueryRunner{}, logStore, routeManager, traefik.MkcertGenerator{}, routeManager),
		TaskService:       taskService,
		TurnstileVerifier: turnstile.NewVerifier(cfg.Turnstile),
	}
}

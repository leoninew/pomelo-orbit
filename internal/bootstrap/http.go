package bootstrap

import (
	"log/slog"

	"github.com/jmoiron/sqlx"

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
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/external/traefik"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/external/turnstile"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
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

func newHTTPServerDependencies(cfg config.Config, logger *slog.Logger, database *sqlx.DB, taskRepo taskrepo.Repository) transporthttp.ServerDependencies {
	tokenService := jwt.NewTokenService(cfg.JWT.SecretKey)
	userRepository := userrepo.NewRepository(database, cfg.Database.Driver)
	roleRepository := rolerepo.NewRepository(database, cfg.Database.Driver)
	projectRepository := projectrepo.NewRepository(database, cfg.Database.Driver)
	ciRepository := cirepo.NewRepository(database, cfg.Database.Driver)
	cdRepository := cdrepo.NewRepository(database, cfg.Database.Driver)
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

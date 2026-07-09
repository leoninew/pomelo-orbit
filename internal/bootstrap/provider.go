package bootstrap

import (
	"log/slog"

	"github.com/jmoiron/sqlx"

	transporthttp "gitee.com/leoninew/PomeloOrbit-go/internal/api/http"
	"gitee.com/leoninew/PomeloOrbit-go/internal/api/http/handler/authz"
	authsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/auth"
	cdsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci"
	projectsvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/project"
	rolesvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/role"
	settingssvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/settings"
	usersvc "gitee.com/leoninew/PomeloOrbit-go/internal/application/user"
	jwt "gitee.com/leoninew/PomeloOrbit-go/internal/auth/jwt"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/config/envfile"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
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
	"gitee.com/leoninew/PomeloOrbit-go/internal/worker"
	cdworker "gitee.com/leoninew/PomeloOrbit-go/internal/worker/handler/cd"
	ciworker "gitee.com/leoninew/PomeloOrbit-go/internal/worker/handler/ci"
	cdactivity "gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/cd"
	ciactivity "gitee.com/leoninew/PomeloOrbit-go/internal/workflow/activity/ci"
)

func OpenDatabase(cfg config.Config) (*sqlx.DB, error) {
	return db.Open(cfg.Database)
}

func RunMigrations(database *sqlx.DB, driver string) error {
	return db.MigrateUp(database, driver)
}

func MigrationVersion(database *sqlx.DB, driver string) (db.MigrationVersion, error) {
	return db.ReadMigrationVersion(database, driver)
}

func NewRepositoryStore(database *sqlx.DB, driver string) store.Store {
	return store.NewStore(database, driver)
}

func NewTaskRepository(database *sqlx.DB, driver string) taskrepo.Repository {
	return taskrepo.NewRepository(database, driver)
}

func NewHTTPServer(cfg config.Config, logger *slog.Logger, repositoryStore store.Store, taskRepo taskrepo.Repository) transporthttp.Server {
	tokenService := jwt.NewTokenService(cfg.JWT.SecretKey)
	userRepository := userrepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	roleRepository := rolerepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	projectRepository := projectrepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	ciRepository := cirepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	cdRepository := cdrepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	taskService := tasksvc.New(taskRepo, cfg.Worker.MaxAttempts)
	logStore := executionlog.Store{}
	routeManager := traefik.NewRouteManager(cfg)
	deps := transporthttp.ServerDependencies{
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
	return transporthttp.New(cfg, logger, deps)
}

func NewTaskRouter(repositoryStore store.Store, cfg config.Config, logger *slog.Logger) *worker.Router {
	router := worker.NewRouter()
	ciRepository := cirepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	cdRepository := cdrepo.NewRepository(repositoryStore.DB(), repositoryStore.Driver())
	logStore := executionlog.Store{}
	ciService := ciactivity.NewExecutionService(ciRepository, cfg.DataRoot(), cfg.JWT.SecretKey, logger, ciactivity.DockerRunner{}, logStore)
	cdService := cdactivity.NewExecutionService(cdRepository, cfg, logger, cdactivity.ShellRunner{}, logStore)
	router.Register(status.TaskTypeCIPipelineRunExecute, ciworker.NewHandler(ciService))
	router.Register(status.TaskTypeCDApplicationDeploy, cdworker.NewDeployHandler(cdService))
	router.Register(status.TaskTypeCDApplicationRestart, cdworker.NewRestartHandler(cdService))
	router.Register(status.TaskTypeCDApplicationStop, cdworker.NewStopHandler(cdService))
	return router
}

func NewWorker(cfg config.Config, logger *slog.Logger, taskRepo taskrepo.Repository, router worker.Handler) *worker.Worker {
	return worker.New(taskRepo, router, logger, worker.Config{
		WorkerId:      cfg.Worker.Id,
		PollInterval:  cfg.Worker.PollInterval,
		LeaseDuration: cfg.Worker.LeaseDuration,
		Concurrency:   cfg.Worker.Concurrency,
	})
}

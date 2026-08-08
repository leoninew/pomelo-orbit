package bootstrap

import (
	"context"
	"log/slog"
	"net/http"

	deliverymcp "gitee.com/leoninew/PomeloOrbit-go/internal/api/mcp/delivery"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	database "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	mcpinfra "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/mcp"
	"gitee.com/leoninew/PomeloOrbit-go/internal/queue/worker"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

type App struct {
	cfg    config.Config
	logger *slog.Logger
}

func New(cfg config.Config, logger *slog.Logger) App {
	return App{cfg: cfg, logger: logger}
}

func (a App) Migrate() error {
	database, err := OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	return RunMigrations(database, a.cfg.Database.Driver, a.logger)
}

func (a App) RunWorker(ctx context.Context) error {
	database, err := OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	if err := RunMigrations(database, a.cfg.Database.Driver, a.logger); err != nil {
		return err
	}

	taskRepo := taskrepo.NewRepository(database)
	router := NewTaskRouter(database, a.cfg, a.logger)
	backgroundWorker := worker.New(database, taskRepo, router, a.logger, worker.Config{
		WorkerId:      a.cfg.Worker.Id,
		PollInterval:  a.cfg.Worker.PollInterval,
		LeaseDuration: a.cfg.Worker.LeaseDuration,
		Concurrency:   a.cfg.Worker.Concurrency,
	})
	return backgroundWorker.Run(ctx)
}

// RunMCP starts the local stdio transport. It reuses the same application
// service composition as HTTP but never proxies MCP calls through /api.
func (a App) RunMCP(ctx context.Context) error {
	if err := a.cfg.ValidateMCPClient(); err != nil {
		return err
	}
	database, err := OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	taskRepo := taskrepo.NewRepository(database)
	deps := newHTTPServerDependencies(a.cfg, a.logger, database, taskRepo)

	server, err := deliverymcp.NewServer(deliverymcp.Dependencies{
		ActorAuthorizer: func(ctx context.Context) (string, error) {
			tokenStore, err := mcpinfra.NewDefaultTokenStore()
			if err != nil {
				return "", err
			}
			token, err := tokenStore.Load()
			if err != nil {
				return "", err
			}
			authenticated, err := deps.AuthService.Authenticate(ctx, token)
			if err == nil {
				return authenticated.User.Id, nil
			}
			if !apperror.IsKind(err, apperror.KindUnauthorized) {
				return "", err
			}
			authorizer, err := mcpinfra.NewBrowserAuthorizer(mcpinfra.AuthorizerConfig{
				APIURL:  a.cfg.MCPAPIUrl(),
				WebURL:  a.cfg.MCP.WebUrl,
				Timeout: a.cfg.MCP.AuthTimeout,
			})
			if err != nil {
				return "", err
			}
			token, err = authorizer.Authorize(ctx)
			if err != nil {
				return "", err
			}
			authenticated, err = deps.AuthService.Authenticate(ctx, token)
			if err != nil {
				return "", err
			}
			if err := tokenStore.Save(token); err != nil {
				return "", err
			}
			return authenticated.User.Id, nil
		},
		Project:     deps.ProjectService,
		Application: deps.ApplicationService,
		Service:     deps.ServiceService,
		Deployment:  deps.DeploymentService,
		Gateway:     deps.GatewayService,
	})
	if err != nil {
		return err
	}
	session, err := server.Connect(ctx, &mcp.StdioTransport{}, nil)
	if err != nil {
		return err
	}
	return session.Wait()
}

func (a App) MigrationVersion() (database.MigrationVersion, error) {
	dbConn, err := OpenDatabase(a.cfg)
	if err != nil {
		return database.MigrationVersion{}, err
	}
	defer func() { _ = dbConn.Close() }()
	return MigrationVersion(dbConn, a.cfg.Database.Driver)
}

func (a App) Serve(ctx context.Context) error {
	database, err := OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	if err := RunMigrations(database, a.cfg.Database.Driver, a.logger); err != nil {
		return err
	}

	taskRepo := taskrepo.NewRepository(database)
	server := NewHTTPServer(a.cfg, a.logger, database, taskRepo)
	httpServer := &http.Server{Addr: server.Addr(), Handler: server.Handler()}
	router := NewTaskRouter(database, a.cfg, a.logger)
	backgroundWorker := worker.New(database, taskRepo, router, a.logger, worker.Config{
		WorkerId:      a.cfg.Worker.Id,
		PollInterval:  a.cfg.Worker.PollInterval,
		LeaseDuration: a.cfg.Worker.LeaseDuration,
		Concurrency:   a.cfg.Worker.Concurrency,
	})

	return runHTTPServerAndWorker(ctx, a.logger, server.Addr(), httpServer, backgroundWorker)
}

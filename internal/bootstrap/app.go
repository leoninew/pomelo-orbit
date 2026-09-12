package bootstrap

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	deliverymcp "github.com/leoninew/pomelo-orbit/internal/api/mcp/delivery"
	"github.com/leoninew/pomelo-orbit/internal/config"
	database "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	runtimepath "github.com/leoninew/pomelo-orbit/internal/infrastructure/storage/local"
	"github.com/leoninew/pomelo-orbit/internal/queue/worker"
	taskrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/task"
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
	if err := a.validateContainerWorkspaceMounts(ctx); err != nil {
		return err
	}
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
	if err := a.validateContainerWorkspaceMounts(ctx); err != nil {
		return err
	}
	database, err := OpenDatabase(a.cfg)
	if err != nil {
		return err
	}
	defer func() { _ = database.Close() }()

	taskRepo := taskrepo.NewRepository(database)
	services := newApplicationServices(a.cfg, a.logger, database, taskRepo)

	accessToken := a.cfg.MCP.AccessToken
	mcpDeps := newDeliveryMCPDependencies(services)
	mcpDeps.ActorAuthenticator = func(ctx context.Context) (string, error) {
		authenticated, err := services.AuthService.AuthenticateMCPAccessToken(ctx, accessToken)
		if err != nil {
			return "", err
		}
		return authenticated.User.Id, nil
	}
	server, err := deliverymcp.NewServer(mcpDeps)
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
	if err := a.validateContainerWorkspaceMounts(ctx); err != nil {
		return err
	}
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

type dockerDaemonPathResolverFn func(context.Context, string) (string, error)

func dockerDaemonPathResolver() func(context.Context, string) (string, error) {
	return runtimepath.ResolveDockerDaemonPath
}

func (a App) validateContainerWorkspaceMounts(ctx context.Context) error {
	return validateContainerWorkspaceMounts(
		ctx,
		a.cfg,
		runtimepath.IsRunningInContainer(),
		runtimepath.ResolveDockerDaemonPath,
	)
}

func validateContainerWorkspaceMounts(ctx context.Context, cfg config.Config, runningInContainer bool, resolver dockerDaemonPathResolverFn) error {
	if !runningInContainer {
		return nil
	}
	for _, workspace := range []struct {
		key  string
		path string
	}{
		{key: "workspace.pipeline", path: cfg.Workspace.Pipeline},
		{key: "logging.deployment_root", path: cfg.Logging.DeploymentRoot},
	} {
		if _, err := resolver(ctx, workspace.path); err != nil {
			return fmt.Errorf("%s must be bind mounted when Orbit runs in a container: %w", workspace.key, err)
		}
	}
	return nil
}

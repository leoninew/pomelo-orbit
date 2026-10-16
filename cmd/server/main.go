package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	deliverycli "github.com/leoninew/pomelo-orbit/internal/api/cli"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/bootstrap"
	"github.com/leoninew/pomelo-orbit/internal/config"
	"github.com/spf13/cobra"
)

func main() { os.Exit(runMain()) }

func runMain() int {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	if err := newRootCommand().ExecuteContext(ctx); err != nil {
		if err.Error() != "" {
			_, _ = fmt.Fprintln(os.Stderr, err)
		}
		var exit *deliverycli.ExitError
		if errors.As(err, &exit) {
			return exit.Code
		}
		return 2
	}
	return 0
}

func configuredApplication(command string) (bootstrap.App, func(), error) {
	cfg, err := config.Load()
	if err != nil {
		return bootstrap.App{}, nil, &deliverycli.ExitError{Code: 1, Err: fmt.Errorf("load config: %w", err)}
	}
	newLogger := bootstrap.NewLogger
	if command == "mcp" || command == "cli" {
		newLogger = bootstrap.NewMCPLogger
	}
	logger, closeLogger, err := newLogger(cfg)
	if err != nil {
		return bootstrap.App{}, nil, &deliverycli.ExitError{Code: 1, Err: err}
	}
	return bootstrap.New(cfg, logger), func() { _ = closeLogger() }, nil
}

func newRootCommand() *cobra.Command {
	runServer := func(ctx context.Context, name string) error {
		app, closeApp, err := configuredApplication(name)
		if err != nil {
			return err
		}
		defer closeApp()
		err = run(ctx, app, name)
		if errors.Is(err, context.Canceled) {
			return nil
		}
		if err != nil {
			return &deliverycli.ExitError{Code: 1, Err: err}
		}
		return nil
	}
	root := &cobra.Command{Use: "pomelo-orbit", Short: "Orbit server and local environment management",
		Args: cobra.NoArgs, SilenceErrors: true, SilenceUsage: true,
		RunE: func(command *cobra.Command, _ []string) error { return runServer(command.Context(), "serve") }}
	root.CompletionOptions.DisableDefaultCmd = true
	root.SetFlagErrorFunc(func(_ *cobra.Command, err error) error { return &deliverycli.ExitError{Code: 2, Err: err} })
	for _, name := range []string{"serve", "worker", "mcp"} {
		root.AddCommand(&cobra.Command{Use: name, Args: cobra.NoArgs, RunE: func(command *cobra.Command, _ []string) error {
			return runServer(command.Context(), command.Name())
		}})
	}
	root.AddCommand(deliverycli.NewCommand(deliverycli.Dependencies{
		Open: func(ctx context.Context) (deliverycli.Services, func(), error) {
			app, closeApp, err := configuredApplication("cli")
			if err != nil {
				return deliverycli.Services{}, nil, err
			}
			services, closeServices, err := app.OpenCLI(ctx)
			if err != nil {
				closeApp()
				return deliverycli.Services{}, nil, err
			}
			return services, func() { closeServices(); closeApp() }, nil
		},
		Processes: func(context.Context) (environmentport.ForwardProcesses, error) {
			if _, err := config.Load(); err != nil {
				return nil, &deliverycli.ExitError{Code: 1, Err: fmt.Errorf("load config: %w", err)}
			}
			return bootstrap.LocalForwardProcesses()
		},
		RunForward: func(ctx context.Context, port int, runId string) error {
			app, closeApp, err := configuredApplication("cli")
			if err != nil {
				return err
			}
			defer closeApp()
			if err := app.RunCLIForward(ctx, port, runId); err != nil {
				return &deliverycli.ExitError{Code: 1, Err: err}
			}
			return nil
		},
	}))
	return root
}

func run(ctx context.Context, app bootstrap.App, command string) error {
	switch command {
	case "serve":
		return app.Serve(ctx)
	case "worker":
		return app.RunWorker(ctx)
	case "mcp":
		return app.RunMCP(ctx)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

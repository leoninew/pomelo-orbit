package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gitee.com/leoninew/pomelo-orbit/internal/app"
	"gitee.com/leoninew/pomelo-orbit/internal/config"
	"gitee.com/leoninew/pomelo-orbit/internal/logging"
)

func main() {
	migrateOnly := flag.Bool("migrate-only", false, "run database migrations and exit")
	flag.Parse()

	command := "serve"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}
	if *migrateOnly {
		command = "migrate-up"
	}

	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		os.Exit(1)
	}

	logger, closeLogger, err := logging.New(cfg.Logging)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "init logger failed: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := closeLogger(); err != nil {
			logger.Error("close logger failed", "error", err)
		}
	}()
	backgroundApp := app.New(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, backgroundApp, command); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("backend-go exited with error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, backgroundApp app.App, command string) error {
	switch command {
	case "serve":
		return backgroundApp.Serve(ctx)
	case "worker":
		return backgroundApp.RunWorker(ctx)
	case "migrate-up", "migrate":
		return backgroundApp.Migrate()
	case "migrate-status":
		version, err := backgroundApp.MigrationVersion()
		if err != nil {
			return err
		}
		state := "clean"
		if version.Dirty {
			state = "dirty"
		}
		fmt.Printf("version=%d state=%s\n", version.Version, state)
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

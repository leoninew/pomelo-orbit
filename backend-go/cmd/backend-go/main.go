package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"backend/internal/app"
	"backend/internal/config"
	"backend/internal/logging"
)

func main() {
	configPath := flag.String("config", "", "path to config yaml")
	migrateOnly := flag.Bool("migrate-only", false, "run database migrations and exit")
	flag.Parse()

	command := "worker"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}
	if *migrateOnly {
		command = "migrate-up"
	}

	cfg, err := config.Load(*configPath)
	if err != nil {
		slog.Error("load config failed", "error", err)
		os.Exit(1)
	}

	logger := logging.New(cfg.Logging.Level)
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
		statuses, err := backgroundApp.MigrationStatus()
		if err != nil {
			return err
		}
		for _, status := range statuses {
			state := "pending"
			if status.Applied {
				state = "applied"
			}
			fmt.Printf("%s %s %s\n", state, status.Filename, status.Checksum)
		}
		return nil
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"gitee.com/leoninew/PomeloOrbit-go/internal/bootstrap"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

func main() {
	flag.Parse()

	command := "serve"
	if flag.NArg() > 0 {
		command = flag.Arg(0)
	}

	cfg, err := config.Load()
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "load config failed: %v\n", err)
		os.Exit(1)
	}

	logger, closeLogger, err := bootstrap.NewLogger(cfg)
	if err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "init logger failed: %v\n", err)
		os.Exit(1)
	}
	defer func() {
		if err := closeLogger(); err != nil {
			logger.Error("close logger failed", "error", err)
		}
	}()
	backgroundApp := bootstrap.New(cfg, logger)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	if err := run(ctx, backgroundApp, command); err != nil && !errors.Is(err, context.Canceled) {
		logger.Error("pomelo-orbit exited with error", "error", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, backgroundApp bootstrap.App, command string) error {
	switch command {
	case "serve":
		return backgroundApp.Serve(ctx)
	case "worker":
		return backgroundApp.RunWorker(ctx)
	default:
		return fmt.Errorf("unknown command: %s", command)
	}
}

package main

import (
	"flag"
	"fmt"
	"os"

	"gitee.com/leoninew/PomeloOrbit-go/internal/bootstrap"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
)

func main() {
	statusOnly := flag.Bool("status", false, "print migration version and exit")
	flag.Parse()

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

	app := bootstrap.New(cfg, logger)

	if *statusOnly {
		version, err := app.MigrationVersion()
		if err != nil {
			logger.Error("read migration version failed", "error", err)
			os.Exit(1)
		}
		state := "clean"
		if version.Dirty {
			state = "dirty"
		}
		fmt.Printf("version=%d state=%s\n", version.Version, state)
		return
	}

	if err := app.Migrate(); err != nil {
		logger.Error("migration failed", "error", err)
		os.Exit(1)
	}
	logger.Info("migration complete")
}

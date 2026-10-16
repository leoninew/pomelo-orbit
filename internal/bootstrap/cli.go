package bootstrap

import (
	"context"
	"path/filepath"

	deliverycli "github.com/leoninew/pomelo-orbit/internal/api/cli"
	authsvc "github.com/leoninew/pomelo-orbit/internal/application/auth/usecase"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	environmentsvc "github.com/leoninew/pomelo-orbit/internal/application/environment/usecase"
	jwt "github.com/leoninew/pomelo-orbit/internal/auth/jwt"
	environmentrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/environment"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/forwardprocess"
	sshrunner "github.com/leoninew/pomelo-orbit/internal/infrastructure/runner/ssh"
)

// LocalForwardProcesses has no database or PAT dependency. Its directory uses
// the same startup cwd as config.Load, without configuration discovery.
func LocalForwardProcesses() (*forwardprocess.Manager, error) {
	return forwardprocess.New(filepath.Join("data", "cli", "forward"))
}

func (a App) OpenCLI(ctx context.Context) (deliverycli.Services, func(), error) {
	processes, err := LocalForwardProcesses()
	if err != nil {
		return deliverycli.Services{}, nil, err
	}
	return a.openCLI(ctx, processes)
}

func (a App) openCLI(ctx context.Context, processes environmentport.ForwardProcesses) (deliverycli.Services, func(), error) {
	database, err := OpenDatabase(a.cfg)
	if err != nil {
		return deliverycli.Services{}, nil, err
	}
	closeDatabase := func() { _ = database.Close() }
	stores := newDomainStores(database)
	auth := authsvc.New(stores.user, stores.auth, jwt.NewTokenService(a.cfg.Jwt.SecretKey), a.logger, a.cfg.Jwt.SecretKey)
	actor, err := auth.AuthenticateMCPAccessToken(ctx, a.cfg.MCP.AccessToken)
	if err != nil {
		closeDatabase()
		return deliverycli.Services{}, nil, err
	}
	// These are the same application constructors used by WebAPI. No deployment
	// runtime, worker, migrations, or browser transport is started for this entry.
	environments := environmentsvc.New(stores.environment, stores.project, stores.environmentCredential,
		a.cfg.Jwt.SecretKey, environmentrunner.NewProber(nil, sshrunner.NewEnvironmentProber()), a.logger).WithLocalDisplay(localEnvironmentDisplay())
	terminal := environmentsvc.NewTerminalService(environments)
	access := environmentsvc.NewAccessService(environments, stores.project, terminal)
	runtime := sshrunner.NewRuntime()
	return deliverycli.Services{ActorId: actor.User.Id, Access: access, Terminal: terminal, TerminalRunner: runtime,
		Command: environmentsvc.NewCommandService(access, sshrunner.NewCommandRunner()),
		Forward: environmentsvc.NewForwardService(access, runtime, processes)}, closeDatabase, nil
}

func (a App) RunCLIForward(ctx context.Context, port int, runId string) error {
	processes, err := LocalForwardProcesses()
	if err != nil {
		return err
	}
	return processes.Run(ctx, port, runId, func(ctx context.Context, input environmentdto.ForwardInput, ready func(environmentdto.ForwardSpec) error) error {
		services, closeServices, err := a.openCLI(ctx, processes)
		if err != nil {
			return err
		}
		defer closeServices()
		return services.Forward.Run(ctx, services.ActorId, input, ready)
	})
}

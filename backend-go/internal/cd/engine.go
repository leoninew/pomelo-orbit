package cd

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"backend/internal/config"
	"backend/internal/repository/model"
	"backend/internal/status"
)

type Store interface {
	Application(ctx context.Context, id string) (model.Application, error)
	Deployment(ctx context.Context, id string) (model.Deployment, error)
	ConfigFiles(ctx context.Context, applicationId string) ([]model.ApplicationConfigFile, error)
	ServiceConfigs(ctx context.Context, applicationId string) ([]model.ApplicationServiceConfig, error)
	Routes(ctx context.Context, applicationId string) ([]model.ApplicationRoute, error)
	MarkApplicationStatus(ctx context.Context, id string, status string) error
	MarkDeploymentRunning(ctx context.Context, id string) error
	CompleteDeployment(ctx context.Context, id string, status string, message string) error
}

type Engine struct {
	store  Store
	cfg    config.Config
	logger *slog.Logger
	runner CommandRunner
}

func NewEngine(store Store, cfg config.Config, logger *slog.Logger) Engine {
	return Engine{store: store, cfg: cfg, logger: logger, runner: ShellRunner{}}
}

func NewEngineWithRunner(store Store, cfg config.Config, logger *slog.Logger, runner CommandRunner) Engine {
	return Engine{store: store, cfg: cfg, logger: logger, runner: runner}
}

func (e Engine) Deploy(ctx context.Context, applicationId string, deploymentId string) error {
	app, deployment, err := e.load(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	if err := e.store.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeploying); err != nil {
		return err
	}
	if err := e.store.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	if err := e.writeAndDeploy(ctx, app, deployment.Id); err != nil {
		_ = e.store.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeployFailed)
		_ = e.store.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := e.store.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeployed); err != nil {
		return err
	}
	return e.store.CompleteDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (e Engine) Restart(ctx context.Context, applicationId string, deploymentId string) error {
	app, deployment, err := e.load(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	if err := e.store.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeploying); err != nil {
		return err
	}
	if err := e.store.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	appDir := filepath.Join(e.cfg.DataRoot(), "cd", app.Code)
	logFile, closeLog, err := e.deploymentLog(app.Code, deployment.Id)
	if err != nil {
		return err
	}
	defer closeLog()
	if err := e.runner.Run(ctx, appDir, logFile, "docker", "compose", "-f", "docker-compose.yml", "restart"); err != nil {
		_ = e.store.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeployFailed)
		_ = e.store.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := e.store.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeployed); err != nil {
		return err
	}
	return e.store.CompleteDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (e Engine) load(ctx context.Context, applicationId string, deploymentId string) (model.Application, model.Deployment, error) {
	app, err := e.store.Application(ctx, applicationId)
	if err != nil {
		return model.Application{}, model.Deployment{}, err
	}
	deployment, err := e.store.Deployment(ctx, deploymentId)
	if err != nil {
		return model.Application{}, model.Deployment{}, err
	}
	return app, deployment, nil
}

func (e Engine) writeAndDeploy(ctx context.Context, app model.Application, deploymentId string) error {
	files, err := e.store.ConfigFiles(ctx, app.Id)
	if err != nil {
		return err
	}
	services, err := e.store.ServiceConfigs(ctx, app.Id)
	if err != nil {
		return err
	}
	var routes []model.ApplicationRoute
	if app.RouteManaged {
		routes, err = e.store.Routes(ctx, app.Id)
		if err != nil {
			return err
		}
	}

	appDir := filepath.Join(e.cfg.DataRoot(), "cd", app.Code)
	logFile, closeLog, err := e.deploymentLog(app.Code, deploymentId)
	if err != nil {
		return err
	}
	defer closeLog()
	_, _ = fmt.Fprintf(logFile, "Working directory: %s\n", appDir)

	hasInit := false
	for _, file := range files {
		path := file.Path
		content := file.Content
		if strings.HasSuffix(path, ".jinja") {
			path = strings.TrimSuffix(path, ".jinja")
			content, err = renderTemplate(content, app.Code, e.cfg)
			if err != nil {
				return err
			}
		}
		if path == "docker-compose.yml" {
			content = applyServiceConfigs(content, services)
			if app.RouteManaged {
				content, err = injectRouteLabels(content, routes, e.cfg.Cert.LetsEncrypt.Enabled)
				if err != nil {
					return err
				}
			}
		}
		if err := writeFile(appDir, path, content); err != nil {
			return err
		}
		if file.Path == "init.sh" {
			hasInit = true
		}
	}
	if hasInit {
		if err := e.runner.Run(ctx, appDir, logFile, "bash", "-x", "init.sh"); err != nil {
			return err
		}
	}
	return e.runner.Run(ctx, appDir, logFile, "docker", "compose", "-f", "docker-compose.yml", "up", "-d", "--remove-orphans", "--pull", app.ImagePullPolicy)
}

func (e Engine) deploymentLog(applicationCode string, deploymentId string) (*os.File, func(), error) {
	path := filepath.Join(e.cfg.DataRoot(), "cd", applicationCode, "deployments", deploymentId+".log")
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return file, func() { _ = file.Close() }, nil
}

func writeFile(appDir string, name string, content string) error {
	path := filepath.Join(appDir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	if filepath.Base(path) == "init.sh" {
		return os.Chmod(path, 0o755)
	}
	return nil
}

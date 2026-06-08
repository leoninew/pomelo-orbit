package cd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"backend/internal/config"
	"backend/internal/orbit"
	"backend/internal/task"
)

type Store interface {
	Application(ctx context.Context, id string) (orbit.Application, error)
	Deployment(ctx context.Context, id string) (orbit.Deployment, error)
	ConfigFiles(ctx context.Context, applicationId string) ([]orbit.ApplicationConfigFile, error)
	ServiceConfigs(ctx context.Context, applicationId string) ([]orbit.ApplicationServiceConfig, error)
	Routes(ctx context.Context, applicationId string) ([]orbit.ApplicationRoute, error)
	MarkApplicationStatus(ctx context.Context, id string, status string) error
	MarkDeploymentRunning(ctx context.Context, id string) error
	CompleteDeployment(ctx context.Context, id string, status string, message string) error
}

type Payload struct {
	ApplicationId string `json:"application_id"`
	DeploymentId  string `json:"deployment_id"`
}

type Handler struct {
	store   Store
	cfg     config.Config
	logger  *slog.Logger
	runner  CommandRunner
	restart bool
}

func NewDeployHandler(store Store, cfg config.Config, logger *slog.Logger) Handler {
	return Handler{store: store, cfg: cfg, logger: logger, runner: ShellRunner{}}
}

func NewRestartHandler(store Store, cfg config.Config, logger *slog.Logger) Handler {
	return Handler{store: store, cfg: cfg, logger: logger, runner: ShellRunner{}, restart: true}
}

func (h Handler) Handle(ctx context.Context, item task.Task) error {
	var payload Payload
	if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("parse cd task payload: %w", err)
	}
	if payload.ApplicationId == "" || payload.DeploymentId == "" {
		return fmt.Errorf("application_id and deployment_id are required")
	}
	if h.restart {
		return h.restartApplication(ctx, payload)
	}
	return h.deploy(ctx, payload)
}

func (h Handler) deploy(ctx context.Context, payload Payload) error {
	app, deployment, err := h.load(ctx, payload)
	if err != nil {
		return err
	}
	if err := h.store.MarkApplicationStatus(ctx, app.Id, orbit.ApplicationStatusDeploying); err != nil {
		return err
	}
	if err := h.store.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	if err := h.writeAndDeploy(ctx, app, deployment.Id); err != nil {
		_ = h.store.MarkApplicationStatus(ctx, app.Id, orbit.ApplicationStatusDeployFailed)
		_ = h.store.CompleteDeployment(ctx, deployment.Id, orbit.WorkStatusFaulted, err.Error())
		return err
	}
	if err := h.store.MarkApplicationStatus(ctx, app.Id, orbit.ApplicationStatusDeployed); err != nil {
		return err
	}
	return h.store.CompleteDeployment(ctx, deployment.Id, orbit.WorkStatusRanToCompletion, "")
}

func (h Handler) restartApplication(ctx context.Context, payload Payload) error {
	app, deployment, err := h.load(ctx, payload)
	if err != nil {
		return err
	}
	if err := h.store.MarkApplicationStatus(ctx, app.Id, orbit.ApplicationStatusDeploying); err != nil {
		return err
	}
	if err := h.store.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	appDir := filepath.Join(h.cfg.DataRoot(), "cd", app.Code)
	logFile, closeLog, err := h.deploymentLog(app.Code, deployment.Id)
	if err != nil {
		return err
	}
	defer closeLog()
	if err := h.runner.Run(ctx, appDir, logFile, "docker", "compose", "-f", "docker-compose.yml", "restart"); err != nil {
		_ = h.store.MarkApplicationStatus(ctx, app.Id, orbit.ApplicationStatusDeployFailed)
		_ = h.store.CompleteDeployment(ctx, deployment.Id, orbit.WorkStatusFaulted, err.Error())
		return err
	}
	if err := h.store.MarkApplicationStatus(ctx, app.Id, orbit.ApplicationStatusDeployed); err != nil {
		return err
	}
	return h.store.CompleteDeployment(ctx, deployment.Id, orbit.WorkStatusRanToCompletion, "")
}

func (h Handler) load(ctx context.Context, payload Payload) (orbit.Application, orbit.Deployment, error) {
	app, err := h.store.Application(ctx, payload.ApplicationId)
	if err != nil {
		return orbit.Application{}, orbit.Deployment{}, err
	}
	deployment, err := h.store.Deployment(ctx, payload.DeploymentId)
	if err != nil {
		return orbit.Application{}, orbit.Deployment{}, err
	}
	return app, deployment, nil
}

func (h Handler) writeAndDeploy(ctx context.Context, app orbit.Application, deploymentId string) error {
	files, err := h.store.ConfigFiles(ctx, app.Id)
	if err != nil {
		return err
	}
	services, err := h.store.ServiceConfigs(ctx, app.Id)
	if err != nil {
		return err
	}
	var routes []orbit.ApplicationRoute
	if app.RouteManaged {
		routes, err = h.store.Routes(ctx, app.Id)
		if err != nil {
			return err
		}
	}

	appDir := filepath.Join(h.cfg.DataRoot(), "cd", app.Code)
	logFile, closeLog, err := h.deploymentLog(app.Code, deploymentId)
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
			content, err = renderTemplate(content, app.Code, h.cfg)
			if err != nil {
				return err
			}
		}
		if path == "docker-compose.yml" {
			content = applyServiceConfigs(content, services)
			if app.RouteManaged {
				content, err = injectRouteLabels(content, routes, h.cfg.Cert.LetsEncrypt.Enabled)
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
		if err := h.runner.Run(ctx, appDir, logFile, "bash", "-x", "init.sh"); err != nil {
			return err
		}
	}
	return h.runner.Run(ctx, appDir, logFile, "docker", "compose", "-f", "docker-compose.yml", "up", "-d", "--remove-orphans", "--pull", app.ImagePullPolicy)
}

func (h Handler) deploymentLog(applicationCode string, deploymentId string) (*os.File, func(), error) {
	path := filepath.Join(h.cfg.DataRoot(), "cd", applicationCode, "deployments", deploymentId+".log")
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

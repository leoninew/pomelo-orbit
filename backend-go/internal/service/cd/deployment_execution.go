package cdsvc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"backend/internal/repository/model"
	"backend/internal/status"
)

func (s Service) ExecuteApplicationDeploy(ctx context.Context, applicationId string, deploymentId string) error {
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	if err := s.executionStore.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeploying); err != nil {
		return err
	}
	if err := s.executionStore.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	if err := s.writeAndDeploy(ctx, app, deployment.Id); err != nil {
		_ = s.executionStore.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeployFailed)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeployed); err != nil {
		return err
	}
	return s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) ExecuteApplicationRestart(ctx context.Context, applicationId string, deploymentId string) error {
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	if err := s.executionStore.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeploying); err != nil {
		return err
	}
	if err := s.executionStore.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	appDir := s.workspace.AppDir(app.Code)
	logFile, closeLog, err := s.deploymentLog(app.Code, deployment.Id)
	if err != nil {
		return err
	}
	defer closeLog()
	if err := s.runner.Run(ctx, appDir, logFile, "docker", "compose", "-f", "docker-compose.yml", "restart"); err != nil {
		_ = s.executionStore.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeployFailed)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeployed); err != nil {
		return err
	}
	return s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) loadDeploymentExecution(ctx context.Context, applicationId string, deploymentId string) (model.Application, model.Deployment, error) {
	app, err := s.executionStore.Application(ctx, applicationId)
	if err != nil {
		return model.Application{}, model.Deployment{}, err
	}
	deployment, err := s.executionStore.Deployment(ctx, deploymentId)
	if err != nil {
		return model.Application{}, model.Deployment{}, err
	}
	return app, deployment, nil
}

func (s Service) writeAndDeploy(ctx context.Context, app model.Application, deploymentId string) error {
	files, err := s.executionStore.ConfigFiles(ctx, app.Id)
	if err != nil {
		return err
	}
	services, err := s.executionStore.ServiceConfigs(ctx, app.Id)
	if err != nil {
		return err
	}
	var routes []model.ApplicationRoute
	if app.RouteManaged {
		routes, err = s.executionStore.Routes(ctx, app.Id)
		if err != nil {
			return err
		}
	}

	appDir := s.workspace.AppDir(app.Code)
	logFile, closeLog, err := s.deploymentLog(app.Code, deploymentId)
	if err != nil {
		return err
	}
	defer closeLog()
	_, _ = fmt.Fprintf(logFile, "Working directory: %s\n", appDir)

	hasInit := false
	for _, file := range files {
		path := file.Path
		content := file.Content
		if strings.HasSuffix(path, ".liquid") {
			path = strings.TrimSuffix(path, ".liquid")
			content, err = s.renderApplicationTemplate(ctx, content, app.Code)
			if err != nil {
				return err
			}
		}
		if path == "docker-compose.yml" {
			content = applyApplicationServiceConfigs(content, services)
			if app.RouteManaged {
				content, err = injectApplicationRouteLabels(content, routes, s.cfg.Cert.LetsEncrypt.Enabled)
				if err != nil {
					return err
				}
			}
		}
		if err := writeDeploymentFile(appDir, path, content); err != nil {
			return err
		}
		if file.Path == "init.sh" {
			hasInit = true
		}
	}
	if hasInit {
		if err := s.runner.Run(ctx, appDir, logFile, "bash", "-x", "init.sh"); err != nil {
			return err
		}
	}
	return s.runner.Run(ctx, appDir, logFile, "docker", "compose", "-f", "docker-compose.yml", "up", "-d", "--remove-orphans", "--pull", app.ImagePullPolicy)
}

func (s Service) deploymentLog(applicationCode string, deploymentId string) (*os.File, func(), error) {
	path := s.workspace.DeploymentLogPath(applicationCode, deploymentId)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, nil, err
	}
	file, err := os.Create(path)
	if err != nil {
		return nil, nil, err
	}
	return file, func() { _ = file.Close() }, nil
}

func writeDeploymentFile(appDir string, name string, content string) error {
	path := filepath.Join(appDir, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	if filepath.Base(path) == "init.sh" {
		content = normalizeShellScriptLineEndings(content)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return err
	}
	if filepath.Base(path) == "init.sh" {
		return os.Chmod(path, 0o755)
	}
	return nil
}

func normalizeShellScriptLineEndings(content string) string {
	content = strings.ReplaceAll(content, "\r\n", "\n")
	return strings.ReplaceAll(content, "\r", "\n")
}

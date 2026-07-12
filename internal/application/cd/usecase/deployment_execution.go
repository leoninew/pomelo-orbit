package cdsvc

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (s Service) ExecuteApplicationDeploy(ctx context.Context, applicationId string, deploymentId string, forceRecreate bool) error {
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

	if err := s.writeAndDeploy(ctx, app, deployment.Id, forceRecreate); err != nil {
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
	logPath := s.workspace.DeploymentLogPath(app.Code, deployment.Id)
	logWriter, err := s.executionLogStore.Writer(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = logWriter.Close() }()

	if err := writeWorkingDirectory(logWriter, appDir); err != nil {
		return err
	}
	command := restartComposeCommand()
	if err := s.runner.Run(ctx, appDir, logWriter, command.Name, command.Args...); err != nil {
		_ = s.executionStore.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeployFailed)
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusDeployed); err != nil {
		return err
	}
	return s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusRanToCompletion, "")
}

func (s Service) ExecuteApplicationStop(ctx context.Context, applicationId string, deploymentId string, removeVolumes bool) error {
	app, deployment, err := s.loadDeploymentExecution(ctx, applicationId, deploymentId)
	if err != nil {
		return err
	}
	if err := s.executionStore.MarkDeploymentRunning(ctx, deployment.Id); err != nil {
		return err
	}

	appDir := s.workspace.AppDir(app.Code)
	logPath := s.workspace.DeploymentLogPath(app.Code, deployment.Id)
	logWriter, err := s.executionLogStore.Writer(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = logWriter.Close() }()

	if err := writeWorkingDirectory(logWriter, appDir); err != nil {
		return err
	}
	command := stopComposeCommand(removeVolumes)
	if err := s.runner.Run(ctx, appDir, logWriter, command.Name, command.Args...); err != nil {
		_ = s.executionStore.CompleteDeployment(ctx, deployment.Id, status.WorkStatusFaulted, err.Error())
		return err
	}
	if err := s.executionStore.MarkApplicationStatus(ctx, app.Id, status.ApplicationStatusUndeployed); err != nil {
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

func (s Service) writeAndDeploy(ctx context.Context, app model.Application, deploymentId string, forceRecreate bool) error {
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
	logPath := s.workspace.DeploymentLogPath(app.Code, deploymentId)
	logWriter, err := s.executionLogStore.Writer(logPath)
	if err != nil {
		return err
	}
	defer func() { _ = logWriter.Close() }()

	if err := writeWorkingDirectory(logWriter, appDir); err != nil {
		return err
	}

	hasInit := false
	for _, file := range files {
		path, content, err := s.renderDeploymentConfigFile(ctx, app, file.Path, file.Content, services, routes)
		if err != nil {
			return err
		}
		if err := writeDeploymentFile(appDir, path, content); err != nil {
			return err
		}
		if path == "init.sh" {
			hasInit = true
		}
	}
	if hasInit {
		if err := s.runner.Run(ctx, appDir, logWriter, "bash", "-x", "init.sh"); err != nil {
			return err
		}
	}
	command := deployComposeCommand(app.ImagePullPolicy, forceRecreate)
	return s.runner.Run(ctx, appDir, logWriter, command.Name, command.Args...)
}

func writeWorkingDirectory(writer io.Writer, appDir string) error {
	_, err := fmt.Fprintf(writer, "Working directory: %s\n", appDir)
	return err
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

package cdsvc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"backend/internal/config"
	"backend/internal/infrastructure/logstore"
	"backend/internal/repository/model"
	"backend/internal/status"
)

func TestExecuteApplicationRestartRestartsApplication(t *testing.T) {
	store := &fakeDeploymentExecutionStore{app: model.Application{Id: "app-1", Code: "demo"}, deployment: model.Deployment{Id: "deploy-1"}}
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), fakeCommandRunner{}, logstore.LogStore{})

	err := service.ExecuteApplicationRestart(context.Background(), "app-1", "deploy-1")
	if err != nil {
		t.Fatalf("ExecuteApplicationRestart returned error: %v", err)
	}
	if store.appStatus != status.ApplicationStatusDeployed {
		t.Fatalf("unexpected app status: %s", store.appStatus)
	}
	if store.deploymentStatus != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestExecuteApplicationStopStopsApplication(t *testing.T) {
	store := &fakeDeploymentExecutionStore{app: model.Application{Id: "app-1", Code: "demo"}, deployment: model.Deployment{Id: "deploy-1"}}
	runner := &recordingCommandRunner{}
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), runner, logstore.LogStore{})

	err := service.ExecuteApplicationStop(context.Background(), "app-1", "deploy-1", true)
	if err != nil {
		t.Fatalf("ExecuteApplicationStop returned error: %v", err)
	}
	if store.appStatus != status.ApplicationStatusUndeployed {
		t.Fatalf("unexpected app status: %s", store.appStatus)
	}
	if store.deploymentStatus != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
	if runner.name != "docker" || strings.Join(runner.args, " ") != "compose -f docker-compose.yml down -v" {
		t.Fatalf("unexpected command: %s %s", runner.name, strings.Join(runner.args, " "))
	}
}

func TestExecuteApplicationStopMarksDeploymentFaultedOnRunnerError(t *testing.T) {
	store := &fakeDeploymentExecutionStore{app: model.Application{Id: "app-1", Code: "demo"}, deployment: model.Deployment{Id: "deploy-1"}}
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), failingCommandRunner{}, logstore.LogStore{})

	err := service.ExecuteApplicationStop(context.Background(), "app-1", "deploy-1", false)
	if err == nil {
		t.Fatal("expected runner error")
	}
	if store.appStatus != "" {
		t.Fatalf("expected app status to remain unchanged, got %s", store.appStatus)
	}
	if store.deploymentStatus != status.WorkStatusFaulted {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestExecuteApplicationDeployMarksDeploymentFaultedOnRunnerError(t *testing.T) {
	store := &fakeDeploymentExecutionStore{
		app:        model.Application{Id: "app-1", Code: "demo", ImagePullPolicy: "missing"},
		deployment: model.Deployment{Id: "deploy-1"},
		files:      []model.ApplicationConfigFile{{Path: "docker-compose.yml", Content: "services:\n  web:\n    image: nginx\n"}},
	}
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), failingCommandRunner{}, logstore.LogStore{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false)
	if err == nil {
		t.Fatal("expected runner error")
	}
	if store.appStatus != status.ApplicationStatusDeployFailed {
		t.Fatalf("unexpected app status: %s", store.appStatus)
	}
	if store.deploymentStatus != status.WorkStatusFaulted {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestExecuteApplicationDeployDeploysApplication(t *testing.T) {
	store := &fakeDeploymentExecutionStore{
		app:        model.Application{Id: "app-1", Code: "demo", ImagePullPolicy: "missing"},
		deployment: model.Deployment{Id: "deploy-1"},
		files: []model.ApplicationConfigFile{
			{Path: "docker-compose.yml", Content: "services:\n  web:\n    image: nginx\n"},
		},
	}
	runner := &recordingCommandRunner{}
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), runner, logstore.LogStore{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false)
	if err != nil {
		t.Fatalf("ExecuteApplicationDeploy returned error: %v", err)
	}
	if store.appStatus != status.ApplicationStatusDeployed {
		t.Fatalf("unexpected app status: %s", store.appStatus)
	}
	if store.deploymentStatus != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
	if runner.name != "docker" || strings.Join(runner.args, " ") != "compose -f docker-compose.yml up -d --remove-orphans --pull missing" {
		t.Fatalf("unexpected command: %s %s", runner.name, strings.Join(runner.args, " "))
	}
}

func TestExecuteApplicationDeployForceRecreatesApplication(t *testing.T) {
	store := &fakeDeploymentExecutionStore{
		app:        model.Application{Id: "app-1", Code: "demo", ImagePullPolicy: "missing"},
		deployment: model.Deployment{Id: "deploy-1"},
		files: []model.ApplicationConfigFile{
			{Path: "docker-compose.yml", Content: "services:\n  web:\n    image: nginx\n"},
		},
	}
	runner := &recordingCommandRunner{}
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), runner, logstore.LogStore{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", true)
	if err != nil {
		t.Fatalf("ExecuteApplicationDeploy returned error: %v", err)
	}
	if runner.name != "docker" || strings.Join(runner.args, " ") != "compose -f docker-compose.yml up -d --remove-orphans --pull missing --force-recreate" {
		t.Fatalf("unexpected command: %s %s", runner.name, strings.Join(runner.args, " "))
	}
}

func TestExecuteApplicationDeployFailsWhenPhysicalDataRootCannotBeResolved(t *testing.T) {
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}
	store := &fakeDeploymentExecutionStore{
		app:        model.Application{Id: "app-1", Code: "demo", ImagePullPolicy: "missing"},
		deployment: model.Deployment{Id: "deploy-1"},
		files: []model.ApplicationConfigFile{
			{Path: "docker-compose.yml.liquid", Content: "services:\n  web:\n    image: nginx\n    volumes:\n      - {{ app.physical_app_dir }}/data:/data\n"},
		},
	}
	service := NewExecutionService(store, cfg, slog.Default(), fakeCommandRunner{}, logstore.LogStore{})
	service.workspace = newWorkspaceWithResolver(cfg.DataRoot(), func(ctx context.Context, logicalDataRoot string) (string, error) {
		return "", errors.New("missing host mount")
	})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false)
	if err == nil || !strings.Contains(err.Error(), "missing host mount") {
		t.Fatalf("expected physical data root error, got %v", err)
	}
	if store.appStatus != status.ApplicationStatusDeployFailed {
		t.Fatalf("unexpected app status: %s", store.appStatus)
	}
	if store.deploymentStatus != status.WorkStatusFaulted {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestExecuteApplicationDeployRendersLiquidFiles(t *testing.T) {
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}
	store := &fakeDeploymentExecutionStore{
		app:        model.Application{Id: "app-1", Code: "demo", ImagePullPolicy: "missing"},
		deployment: model.Deployment{Id: "deploy-1"},
		files: []model.ApplicationConfigFile{
			{Path: "docker-compose.yml.liquid", Content: "services:\n  web:\n    image: nginx\n    volumes:\n      - {{ app.physical_app_dir }}/data:/data\n"},
		},
	}
	service := NewExecutionService(store, cfg, slog.Default(), fakeCommandRunner{}, logstore.LogStore{})
	physicalRoot := filepath.Join(t.TempDir(), "host-data")
	service.workspace = newWorkspaceWithResolver(cfg.DataRoot(), func(ctx context.Context, logicalDataRoot string) (string, error) {
		return physicalRoot, nil
	})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false)
	if err != nil {
		t.Fatalf("ExecuteApplicationDeploy returned error: %v", err)
	}
	composePath := filepath.Join(cfg.DataRoot(), "cd", "demo", "docker-compose.yml")
	content, err := os.ReadFile(composePath)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.ToSlash(filepath.Join(physicalRoot, "cd", "demo", "data"))
	got := filepath.ToSlash(string(content))
	if !strings.Contains(got, want) {
		t.Fatalf("expected rendered compose to contain %q, got:\n%s", want, string(content))
	}
}

type fakeDeploymentExecutionStore struct {
	app              model.Application
	deployment       model.Deployment
	files            []model.ApplicationConfigFile
	appStatus        string
	deploymentStatus string
}

func (s *fakeDeploymentExecutionStore) Application(_ context.Context, id string) (model.Application, error) {
	return s.app, nil
}

func (s *fakeDeploymentExecutionStore) Deployment(_ context.Context, id string) (model.Deployment, error) {
	return s.deployment, nil
}

func (s *fakeDeploymentExecutionStore) ConfigFiles(_ context.Context, applicationId string) ([]model.ApplicationConfigFile, error) {
	return s.files, nil
}

func (s *fakeDeploymentExecutionStore) ServiceConfigs(ctx context.Context, applicationId string) ([]model.ApplicationServiceConfig, error) {
	return nil, nil
}

func (s *fakeDeploymentExecutionStore) Routes(ctx context.Context, applicationId string) ([]model.ApplicationRoute, error) {
	return nil, nil
}

func (s *fakeDeploymentExecutionStore) MarkApplicationStatus(ctx context.Context, id string, status string) error {
	s.appStatus = status
	return nil
}

func (s *fakeDeploymentExecutionStore) MarkDeploymentRunning(ctx context.Context, id string) error {
	s.deploymentStatus = status.WorkStatusRunning
	return nil
}

func (s *fakeDeploymentExecutionStore) CompleteDeployment(ctx context.Context, id string, status string, message string) error {
	s.deploymentStatus = status
	return nil
}

type fakeCommandRunner struct{}

func (fakeCommandRunner) Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error {
	return nil
}

type recordingCommandRunner struct {
	cwd  string
	name string
	args []string
}

func (r *recordingCommandRunner) Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error {
	r.cwd = cwd
	r.name = name
	r.args = append([]string{}, args...)
	return nil
}

type failingCommandRunner struct{}

func (failingCommandRunner) Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error {
	return errors.New("boom")
}

func TestWriteDeploymentFileNormalizesInitScriptLineEndings(t *testing.T) {
	appDir := t.TempDir()
	content := "#!/bin/bash\r\nset -e\r\necho ok\r"

	if err := writeDeploymentFile(appDir, "init.sh", content); err != nil {
		t.Fatalf("writeDeploymentFile returned error: %v", err)
	}

	path := filepath.Join(appDir, "init.sh")
	written, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(written), "\r") {
		t.Fatalf("expected init.sh line endings to be normalized, got %q", string(written))
	}
	if string(written) != "#!/bin/bash\nset -e\necho ok\n" {
		t.Fatalf("unexpected init.sh content: %q", string(written))
	}
}

func TestWriteDeploymentFilePreservesNonInitFileContent(t *testing.T) {
	appDir := t.TempDir()
	content := "services:\r\n  app:\r\n    image: nginx\r\n"

	if err := writeDeploymentFile(appDir, "docker-compose.yml", content); err != nil {
		t.Fatalf("writeDeploymentFile returned error: %v", err)
	}

	written, err := os.ReadFile(filepath.Join(appDir, "docker-compose.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if string(written) != content {
		t.Fatalf("expected non-init content to be preserved, got %q", string(written))
	}
}

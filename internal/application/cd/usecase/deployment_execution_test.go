package cdsvc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"path/filepath"
	"strings"
	"testing"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func TestExecuteApplicationRestartRestartsApplication(t *testing.T) {
	store := &fakeDeploymentExecutionStore{app: model.Application{Id: "app-1", Code: "demo"}, deployment: model.Deployment{Id: "deploy-1"}}
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), testWorkspace(t.TempDir()), fakeCommandRunner{}, executionlog.Store{})

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
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), testWorkspace(t.TempDir()), runner, executionlog.Store{})

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
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), testWorkspace(t.TempDir()), failingCommandRunner{}, executionlog.Store{})

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
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), testWorkspace(t.TempDir()), failingCommandRunner{}, executionlog.Store{})

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
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), testWorkspace(t.TempDir()), runner, executionlog.Store{})

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
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), testWorkspace(t.TempDir()), runner, executionlog.Store{})

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
	service := NewExecutionService(store, cfg, slog.Default(), testWorkspaceWithPhysicalRoot(cfg.DataRoot(), "", errors.New("missing host mount")), fakeCommandRunner{}, executionlog.Store{})

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
	physicalRoot := filepath.Join(t.TempDir(), "host-data")
	workspace := testWorkspaceWithPhysicalRoot(cfg.DataRoot(), physicalRoot, nil)
	service := NewExecutionService(store, cfg, slog.Default(), workspace, fakeCommandRunner{}, executionlog.Store{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false)
	if err != nil {
		t.Fatalf("ExecuteApplicationDeploy returned error: %v", err)
	}
	content, ok := workspace.Config("demo", "docker-compose.yml")
	if !ok {
		t.Fatal("expected rendered compose to be written through workspace port")
	}
	want := filepath.ToSlash(filepath.Join(physicalRoot, "cd", "demo", "data"))
	if !strings.Contains(filepath.ToSlash(content), want) {
		t.Fatalf("expected rendered compose to contain %q, got:\n%s", want, content)
	}
}

func TestApplicationComposePreviewMatchesDeployRouteLabels(t *testing.T) {
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}
	app := model.Application{Id: "app-1", Code: "demo", ImagePullPolicy: "missing", RouteManaged: true}
	compose := model.ApplicationConfigFile{Path: "docker-compose.yml", Content: "services:\n  web:\n    image: nginx\n"}
	routes := []model.ApplicationRoute{
		{ServiceName: "web", Domain: "web.example.com", Port: 80},
		{ServiceName: "web", Domain: "alt.example.com", Port: 80},
	}
	store := &fakeDeploymentExecutionStore{app: app, deployment: model.Deployment{Id: "deploy-1"}, files: []model.ApplicationConfigFile{compose}, routes: routes}
	workspace := testWorkspace(cfg.DataRoot())
	service := NewExecutionService(store, cfg, slog.Default(), workspace, fakeCommandRunner{}, executionlog.Store{})

	preview, err := service.renderDeploymentCompose(context.Background(), app, compose)
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false); err != nil {
		t.Fatalf("ExecuteApplicationDeploy returned error: %v", err)
	}
	content, ok := workspace.Config("demo", "docker-compose.yml")
	if !ok {
		t.Fatal("expected rendered compose to be written through workspace port")
	}
	if preview != content {
		t.Fatalf("expected preview to match deployed compose\npreview:\n%s\ndeployed:\n%s", preview, content)
	}
	wantRule := "traefik.http.routers.web.rule=Host(`web.example.com`) || Host(`alt.example.com`)"
	if !strings.Contains(preview, wantRule) {
		t.Fatalf("expected merged route rule %q, got:\n%s", wantRule, preview)
	}
}

type fakeDeploymentExecutionStore struct {
	app              model.Application
	deployment       model.Deployment
	files            []model.ApplicationConfigFile
	routes           []model.ApplicationRoute
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
	return s.routes, nil
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

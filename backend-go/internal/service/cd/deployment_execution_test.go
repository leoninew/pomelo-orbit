package cdsvc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"testing"

	"backend/internal/config"
	"backend/internal/repository/model"
	"backend/internal/status"
)

func TestExecuteApplicationRestartRestartsApplication(t *testing.T) {
	store := &fakeDeploymentExecutionStore{app: model.Application{Id: "app-1", Code: "demo"}, deployment: model.Deployment{Id: "deploy-1"}}
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), fakeCommandRunner{})

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

func TestExecuteApplicationDeployMarksDeploymentFaultedOnRunnerError(t *testing.T) {
	store := &fakeDeploymentExecutionStore{
		app:        model.Application{Id: "app-1", Code: "demo", ImagePullPolicy: "missing"},
		deployment: model.Deployment{Id: "deploy-1"},
		files:      []model.ApplicationConfigFile{{Path: "docker-compose.yml", Content: "services:\n  web:\n    image: nginx\n"}},
	}
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), failingCommandRunner{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1")
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
	service := NewExecutionService(store, config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, slog.Default(), fakeCommandRunner{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1")
	if err != nil {
		t.Fatalf("ExecuteApplicationDeploy returned error: %v", err)
	}
	if store.appStatus != status.ApplicationStatusDeployed {
		t.Fatalf("unexpected app status: %s", store.appStatus)
	}
	if store.deploymentStatus != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
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
	if log != nil {
		_, _ = log.Write([]byte("ok"))
	}
	return nil
}

type failingCommandRunner struct{}

func (failingCommandRunner) Run(ctx context.Context, cwd string, log io.Writer, name string, args ...string) error {
	return errors.New("boom")
}

package cd

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"

	"backend/internal/config"
	"backend/internal/orbit"
	"backend/internal/task"
)

func TestHandleRejectsInvalidPayload(t *testing.T) {
	handler := Handler{store: orbit.Store{}, cfg: config.Config{}, logger: slog.Default()}
	err := handler.Handle(context.Background(), task.Task{PayloadJSON: `{invalid`})
	if err == nil || !strings.Contains(err.Error(), "parse cd task payload") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestHandleRequiresApplicationAndDeploymentId(t *testing.T) {
	handler := Handler{store: orbit.Store{}, cfg: config.Config{}, logger: slog.Default()}
	err := handler.Handle(context.Background(), task.Task{PayloadJSON: `{}`})
	if err == nil || !strings.Contains(err.Error(), "application_id and deployment_id are required") {
		t.Fatalf("expected required field error, got %v", err)
	}
}

func TestHandleRestartsApplication(t *testing.T) {
	store := &fakeStore{app: orbit.Application{Id: "app-1", Code: "demo"}, deployment: orbit.Deployment{Id: "deploy-1"}}
	handler := Handler{store: store, cfg: config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, logger: slog.Default(), runner: fakeCommandRunner{}, restart: true}

	err := handler.Handle(context.Background(), task.Task{PayloadJSON: `{"application_id":"app-1","deployment_id":"deploy-1"}`})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if store.appStatus != orbit.ApplicationStatusDeployed {
		t.Fatalf("unexpected app status: %s", store.appStatus)
	}
	if store.deploymentStatus != orbit.WorkStatusRanToCompletion {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestHandleMarksDeploymentFaultedOnRunnerError(t *testing.T) {
	store := &fakeStore{
		app:        orbit.Application{Id: "app-1", Code: "demo", ImagePullPolicy: "missing"},
		deployment: orbit.Deployment{Id: "deploy-1"},
		files:      []orbit.ApplicationConfigFile{{Path: "docker-compose.yml", Content: "services:\n  web:\n    image: nginx\n"}},
	}
	handler := Handler{store: store, cfg: config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}, logger: slog.Default(), runner: failingCommandRunner{}}

	err := handler.Handle(context.Background(), task.Task{PayloadJSON: `{"application_id":"app-1","deployment_id":"deploy-1"}`})
	if err == nil {
		t.Fatal("expected runner error")
	}
	if store.appStatus != orbit.ApplicationStatusDeployFailed {
		t.Fatalf("unexpected app status: %s", store.appStatus)
	}
	if store.deploymentStatus != orbit.WorkStatusFaulted {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestHandleDeploysApplication(t *testing.T) {
	store := &fakeStore{
		app:        orbit.Application{Id: "app-1", Code: "demo", ImagePullPolicy: "missing"},
		deployment: orbit.Deployment{Id: "deploy-1"},
		files: []orbit.ApplicationConfigFile{
			{Path: "docker-compose.yml", Content: "services:\n  web:\n    image: nginx\n"},
		},
	}
	handler := Handler{
		store:  store,
		cfg:    config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}},
		logger: slog.Default(),
		runner: fakeCommandRunner{},
	}

	err := handler.Handle(context.Background(), task.Task{PayloadJSON: `{"application_id":"app-1","deployment_id":"deploy-1"}`})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if store.appStatus != orbit.ApplicationStatusDeployed {
		t.Fatalf("unexpected app status: %s", store.appStatus)
	}
	if store.deploymentStatus != orbit.WorkStatusRanToCompletion {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

type fakeStore struct {
	app              orbit.Application
	deployment       orbit.Deployment
	files            []orbit.ApplicationConfigFile
	appStatus        string
	deploymentStatus string
}

func (s *fakeStore) Application(_ context.Context, id string) (orbit.Application, error) {
	return s.app, nil
}

func (s *fakeStore) Deployment(_ context.Context, id string) (orbit.Deployment, error) {
	return s.deployment, nil
}

func (s *fakeStore) ConfigFiles(_ context.Context, applicationId string) ([]orbit.ApplicationConfigFile, error) {
	return s.files, nil
}

func (s *fakeStore) ServiceConfigs(ctx context.Context, applicationId string) ([]orbit.ApplicationServiceConfig, error) {
	return nil, nil
}

func (s *fakeStore) Routes(ctx context.Context, applicationId string) ([]orbit.ApplicationRoute, error) {
	return nil, nil
}

func (s *fakeStore) MarkApplicationStatus(ctx context.Context, id string, status string) error {
	s.appStatus = status
	return nil
}

func (s *fakeStore) MarkDeploymentRunning(ctx context.Context, id string) error {
	s.deploymentStatus = orbit.WorkStatusRunning
	return nil
}

func (s *fakeStore) CompleteDeployment(ctx context.Context, id string, status string, message string) error {
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

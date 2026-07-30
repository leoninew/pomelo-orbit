package deploymentsvc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"os"
	"strings"
	"testing"

	deploymentport "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/port"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	"gitee.com/leoninew/PomeloOrbit-go/internal/repository"
)

func testVersionId() string { return "ver-1" }
func testProjectId() string { return "proj-1" }

func stringPtr(value string) *string { return &value }

func newTestExecutionService(store deploymentport.ExecutionStore, logger *slog.Logger, workspace deploymentport.Workspace, runner deploymentport.CommandRunner, logStore deploymentport.ExecutionLogStore) Service {
	return Service{
		executionStore:     store,
		workspace:          workspace,
		logStore:           logStore,
		executionLogStore:  logStore,
		logger:             logger,
		runner:             runner,
		gatewayCoordinator: &fakeGatewayCoordinator{store: store.(*fakeDeploymentExecutionStore)},
	}
}

func deployStore(t *testing.T) *fakeDeploymentExecutionStore {
	t.Helper()
	versionId := testVersionId()
	projectId := testProjectId()
	optionsJSON := `{"instance_key":"default"}`
	return &fakeDeploymentExecutionStore{
		app: model.Application{Id: "app-1", ProjectId: &projectId, Code: "demo", Kind: status.ApplicationKindStandard, ImagePullPolicy: "missing"},
		deployment: model.Deployment{
			Id:          "deploy-1",
			VersionId:   &versionId,
			ServiceId:   stringPtr("svc-1"),
			OptionsJSON: &optionsJSON,
		},
		version: model.Version{Id: versionId, ApplicationId: "app-1", Label: "v1", Status: status.VersionStatusUnpublished},
		components: []model.VersionComponent{
			{Id: "c1", VersionId: versionId, Name: "web", Image: "nginx"},
		},
		service: model.Service{
			Id:            "svc-1",
			ApplicationId: "app-1",
			InstanceKey:   "default",
			VersionId:     versionId,
			Status:        status.ServiceStatusRunning,
		},
	}
}

func TestExecuteApplicationRestartRestartsApplication(t *testing.T) {
	store := deployStore(t)
	service := newTestExecutionService(store, slog.Default(), testWorkspace(t.TempDir()), fakeCommandRunner{}, executionlog.Store{})

	err := service.ExecuteApplicationRestart(context.Background(), "app-1", "deploy-1")
	if err != nil {
		t.Fatalf("ExecuteApplicationRestart returned error: %v", err)
	}
	if store.serviceStatus != status.ServiceStatusRunning {
		t.Fatalf("unexpected service status: %s", store.serviceStatus)
	}
	if store.deploymentStatus != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestExecuteApplicationStopStopsApplication(t *testing.T) {
	store := deployStore(t)
	runner := &recordingCommandRunner{}
	workspace := testWorkspace(t.TempDir())
	service := newTestExecutionService(store, slog.Default(), workspace, runner, executionlog.Store{})

	err := service.ExecuteApplicationStop(context.Background(), "app-1", "deploy-1", true)
	if err != nil {
		t.Fatalf("ExecuteApplicationStop returned error: %v", err)
	}
	if store.serviceStatus != status.ServiceStatusStopped {
		t.Fatalf("unexpected service status: %s", store.serviceStatus)
	}
	if store.deploymentStatus != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
	if runner.name != "docker" || strings.Join(runner.args, " ") != "compose -p demo-default -f docker-compose.yml down -v" {
		t.Fatalf("unexpected command: %s %s", runner.name, strings.Join(runner.args, " "))
	}
	log := readDeploymentLog(t, workspace, "deploy-1")
	assertLogContainsOnce(t, log, "Working directory: "+workspace.ServiceDir("demo", "default"))
	for _, step := range []string{"Preparing service shutdown", "Stopping services and removing volumes"} {
		if !strings.Contains(log, step) {
			t.Fatalf("expected log step %q, got:\n%s", step, log)
		}
	}
}

func TestExecuteApplicationStopMarksDeploymentFaultedOnRunnerError(t *testing.T) {
	store := deployStore(t)
	service := newTestExecutionService(store, slog.Default(), testWorkspace(t.TempDir()), failingCommandRunner{}, executionlog.Store{})

	err := service.ExecuteApplicationStop(context.Background(), "app-1", "deploy-1", false)
	if err == nil {
		t.Fatal("expected runner error")
	}
	if store.serviceStatus != status.ServiceStatusFaulted {
		t.Fatalf("expected service faulted, got %s", store.serviceStatus)
	}
	if store.deploymentStatus != status.WorkStatusFaulted {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestExecuteApplicationDeployMarksDeploymentFaultedOnRunnerError(t *testing.T) {
	store := deployStore(t)
	service := newTestExecutionService(store, slog.Default(), testWorkspace(t.TempDir()), failingCommandRunner{}, executionlog.Store{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false)
	if err == nil {
		t.Fatal("expected runner error")
	}
	if store.serviceStatus != status.ServiceStatusFaulted {
		t.Fatalf("unexpected service status: %s", store.serviceStatus)
	}
	if store.deploymentStatus != status.WorkStatusFaulted {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
	if store.errorMessage != "boom" {
		t.Fatalf("unexpected error message: %q", store.errorMessage)
	}
}

func TestExecuteApplicationDeployDeploysApplication(t *testing.T) {
	store := deployStore(t)
	runner := &recordingCommandRunner{}
	workspace := testWorkspace(t.TempDir())
	service := newTestExecutionService(store, slog.Default(), workspace, runner, executionlog.Store{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false)
	if err != nil {
		t.Fatalf("ExecuteApplicationDeploy returned error: %v", err)
	}
	if store.serviceStatus != status.ServiceStatusRunning {
		t.Fatalf("unexpected service status: %s", store.serviceStatus)
	}
	if store.deploymentStatus != status.WorkStatusRanToCompletion {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
	if runner.name != "docker" || strings.Join(runner.args, " ") != "compose -p demo-default -f docker-compose.yml up -d --remove-orphans --pull missing" {
		t.Fatalf("unexpected command: %s %s", runner.name, strings.Join(runner.args, " "))
	}
	log := readDeploymentLog(t, workspace, "deploy-1")
	assertLogContainsOnce(t, log, "Working directory: "+workspace.ServiceDir("demo", "default"))
	for _, step := range []string{
		"Preparing deployment",
		"Rendering version v1 (ver-1) with 1 component(s) into instance default",
		"Writing deployment configuration",
		"Starting services",
	} {
		if !strings.Contains(log, step) {
			t.Fatalf("expected log step %q, got:\n%s", step, log)
		}
	}
}

func TestExecuteApplicationDeployRequiresServiceId(t *testing.T) {
	store := deployStore(t)
	store.deployment.ServiceId = nil
	service := newTestExecutionService(store, slog.Default(), testWorkspace(t.TempDir()), fakeCommandRunner{}, executionlog.Store{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false)
	if err == nil || !strings.Contains(err.Error(), "missing service_id") {
		t.Fatalf("expected missing service_id error, got %v", err)
	}
	if store.deploymentStatus != status.WorkStatusFaulted {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestExecuteApplicationDeployForceRecreatesApplication(t *testing.T) {
	store := deployStore(t)
	runner := &recordingCommandRunner{}
	service := newTestExecutionService(store, slog.Default(), testWorkspace(t.TempDir()), runner, executionlog.Store{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", true)
	if err != nil {
		t.Fatalf("ExecuteApplicationDeploy returned error: %v", err)
	}
	if runner.name != "docker" || strings.Join(runner.args, " ") != "compose -p demo-default -f docker-compose.yml up -d --remove-orphans --pull missing --force-recreate" {
		t.Fatalf("unexpected command: %s %s", runner.name, strings.Join(runner.args, " "))
	}
}

func TestExecuteApplicationDeployRequiresVersionId(t *testing.T) {
	store := deployStore(t)
	store.deployment.VersionId = nil
	service := newTestExecutionService(store, slog.Default(), testWorkspace(t.TempDir()), fakeCommandRunner{}, executionlog.Store{})

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false)
	if err == nil {
		t.Fatal("expected missing version_id error")
	}
	if store.deploymentStatus != status.WorkStatusFaulted {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestExecuteApplicationDeployFaultsWhenGatewayResolutionFails(t *testing.T) {
	store := deployStore(t)
	service := newTestExecutionService(store, slog.Default(), testWorkspace(t.TempDir()), fakeCommandRunner{}, executionlog.Store{})
	service.gatewayCoordinator = &fakeGatewayCoordinator{gatewayErr: errors.New("gateway resolution failed")}

	err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false)
	if err == nil || !strings.Contains(err.Error(), "gateway resolution failed") {
		t.Fatalf("expected gateway resolution error, got %v", err)
	}
	if store.deploymentStatus != status.WorkStatusFaulted {
		t.Fatalf("unexpected deployment status: %s", store.deploymentStatus)
	}
}

func TestApplicationComposePreviewMatchesDeployExposeLabels(t *testing.T) {
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}
	store := deployStore(t)
	store.exposes = []model.ServiceExpose{
		{ComponentName: "web", Protocol: "http", Access: exposeAccessPublic, ContainerPort: 80},
	}
	store.gateway = model.GatewayConfig{
		ApplicationId:     "gw-1",
		RestApiUrl:        "http://traefik:8080",
		BaseDomain:        "example.com",
		DefaultEntrypoint: "websecure",
		TLSMode:           "letsencrypt",
	}
	workspace := testWorkspace(cfg.DataRoot())
	service := newTestExecutionService(store, slog.Default(), workspace, fakeCommandRunner{}, executionlog.Store{})

	preview, err := service.RenderCompose(context.Background(), RenderInput{
		App:        store.app,
		Version:    store.version,
		Components: store.components,
		Exposes:    store.exposes,
		Service:    store.service,
		Gateway:    &store.gateway,
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := service.ExecuteApplicationDeploy(context.Background(), "app-1", "deploy-1", false); err != nil {
		t.Fatalf("ExecuteApplicationDeploy returned error: %v", err)
	}
	content, ok := workspace.Config("demo", "default", "docker-compose.yml")
	if !ok {
		t.Fatal("expected rendered compose to be written through workspace port")
	}
	if preview != content {
		t.Fatalf("expected preview to match deployed compose\npreview:\n%s\ndeployed:\n%s", preview, content)
	}
	wantRule := "traefik.http.routers.demo-default-web-http.rule=Host(`demo.example.com`)"
	if !strings.Contains(preview, wantRule) {
		t.Fatalf("expected derived host rule %q, got:\n%s", wantRule, preview)
	}
	if strings.Contains(preview, "pomelo.orbit.version-") {
		t.Fatalf("version labels must not be injected into compose services, got:\n%s", preview)
	}
}

type fakeDeploymentExecutionStore struct {
	app              model.Application
	deployment       model.Deployment
	version          model.Version
	components       []model.VersionComponent
	exposes          []model.ServiceExpose
	gateway          model.GatewayConfig
	service          model.Service
	serviceStatus    string
	deploymentStatus string
	errorMessage     string
}

type fakeGatewayCoordinator struct {
	store      *fakeDeploymentExecutionStore
	gatewayErr error
}

func (f *fakeGatewayCoordinator) EnsureGatewayRunning(_ context.Context, _ model.Application) error {
	return nil
}

func (f *fakeGatewayCoordinator) GatewayForDeployment(_ context.Context, app model.Application, exposes []model.ServiceExpose) (*model.GatewayConfig, error) {
	if f.gatewayErr != nil {
		return nil, f.gatewayErr
	}
	if f.store != nil && f.store.gateway.BaseDomain != "" {
		return &f.store.gateway, nil
	}
	return nil, nil
}

func (s *fakeDeploymentExecutionStore) Application(_ context.Context, id string) (model.Application, error) {
	return s.app, nil
}

func (s *fakeDeploymentExecutionStore) Deployment(_ context.Context, id string) (model.Deployment, error) {
	return s.deployment, nil
}

func (s *fakeDeploymentExecutionStore) Version(_ context.Context, id string) (model.Version, error) {
	if s.version.Id != id {
		return model.Version{}, repository.ErrNotFound
	}
	return s.version, nil
}

func (s *fakeDeploymentExecutionStore) VersionComponentsByVersion(_ context.Context, versionId string) ([]model.VersionComponent, error) {
	return s.components, nil
}

func (s *fakeDeploymentExecutionStore) ServiceExposesByService(_ context.Context, serviceId string) ([]model.ServiceExpose, error) {
	return s.exposes, nil
}

func (s *fakeDeploymentExecutionStore) Service(_ context.Context, id string) (model.Service, error) {
	if s.service.Id == id {
		return s.service, nil
	}
	return model.Service{}, repository.ErrNotFound
}

func (s *fakeDeploymentExecutionStore) UpdateServiceStatus(_ context.Context, id string, status string) error {
	s.service.Status = status
	s.serviceStatus = status
	return nil
}

func (s *fakeDeploymentExecutionStore) UpdateServiceAfterDeploy(_ context.Context, id string, status string, versionId string) error {
	s.service.Status = status
	s.service.VersionId = versionId
	s.serviceStatus = status
	return nil
}

func (s *fakeDeploymentExecutionStore) MarkDeploymentRunning(_ context.Context, id string) error {
	s.deploymentStatus = status.WorkStatusRunning
	return nil
}

func (s *fakeDeploymentExecutionStore) CompleteDeployment(_ context.Context, id string, status string, message string) error {
	s.deploymentStatus = status
	s.errorMessage = message
	return nil
}

func (s *fakeDeploymentExecutionStore) GatewayConfig(_ context.Context, applicationId string) (model.GatewayConfig, error) {
	if s.gateway.ApplicationId != "" && (applicationId == s.gateway.ApplicationId || applicationId == s.app.Id) {
		return s.gateway, nil
	}
	if s.gateway.BaseDomain != "" {
		return s.gateway, nil
	}
	return model.GatewayConfig{}, repository.ErrNotFound
}

func (s *fakeDeploymentExecutionStore) ResolveActiveGatewayConfig(_ context.Context) (model.GatewayConfig, error) {
	if s.gateway.BaseDomain != "" || s.gateway.RestApiUrl != "" {
		return s.gateway, nil
	}
	return model.GatewayConfig{}, repository.ErrNotFound
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

func readDeploymentLog(t *testing.T, workspace *workspaceFake, deploymentId string) string {
	t.Helper()
	path := workspace.DeploymentLogPath("demo", "default", deploymentId)
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read deployment log %s: %v", path, err)
	}
	return string(content)
}

func assertLogContainsOnce(t *testing.T, log string, text string) {
	t.Helper()
	if count := strings.Count(log, text); count != 1 {
		t.Fatalf("expected %q once, found %d time(s) in:\n%s", text, count, log)
	}
}

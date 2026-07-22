package cdsvc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"testing"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	cdport "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/port"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	queuedispatch "gitee.com/leoninew/PomeloOrbit-go/internal/queue/dispatch"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
	cdrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlx/cd"
)

const (
	cdTestUserId        = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	cdTestProjectId     = "01KRRKK0K3T519ZQZES3M4QA9Z"
	cdTestApplicationId = "01KN8CG4A5S4VVH6NKNJF4F9NJ"
)

func TestApplicationServiceCRUDDeployStopAndVersions(t *testing.T) {
	service, database := newCDIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	created, err := service.CreateApplication(ctx, cdTestUserId, cdto.ApplicationCreateInput{ProjectId: cdTestProjectId, Name: "Route App", Code: "route-app", ImagePullPolicy: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.ProjectId == nil || *created.ProjectId != cdTestProjectId || created.Name != "Route App" || created.Code != "route-app" {
		t.Fatalf("unexpected created application: %+v", created)
	}

	updated, err := service.UpdateApplication(ctx, cdTestUserId, created.Id, cdto.ApplicationUpdateInput{Name: stringPtr("Route App 2"), Code: stringPtr("route-app-2"), ImagePullPolicy: stringPtr("always")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Route App 2" || updated.Code != "route-app-2" || updated.ImagePullPolicy != "always" {
		t.Fatalf("unexpected updated application: %+v", updated)
	}

	version, err := service.CreateVersion(ctx, cdTestUserId, cdto.VersionCreateInput{
		ApplicationId: created.Id,
		Label:         "v1",
		Components:    []cdto.VersionComponentInput{{Name: "web", Image: "nginx"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if version.Version.Label != "v1" || len(version.Components) != 1 || version.Components[0].Name != "web" {
		t.Fatalf("unexpected version: %+v", version)
	}

	localEnv := loadLocalEnvironment(t, service)
	deploymentId, err := service.DeployApplication(ctx, cdTestUserId, created.Id, cdto.ApplicationDeployInput{
		VersionId:     version.Version.Id,
		EnvironmentId: localEnv.Id,
		ForceRecreate: true,
	})
	if err != nil {
		t.Fatal(err)
	}
	deployment := loadDeploymentForTest(t, database, deploymentId)
	wantCommand := "docker compose -p route-app-2-local-default -f docker-compose.yml up -d --remove-orphans --pull always --force-recreate"
	if deployment.Status != status.WorkStatusWaitingToRun || deployment.CommandText != wantCommand {
		t.Fatalf("unexpected deployment: %+v", deployment)
	}
	if deployment.VersionId == nil || *deployment.VersionId != version.Version.Id {
		t.Fatalf("unexpected deployment version: %+v", deployment)
	}
	if deployment.EnvironmentId == nil || *deployment.EnvironmentId != localEnv.Id {
		t.Fatalf("unexpected deployment environment: %+v", deployment)
	}
	assertLatestTaskPayload(t, database, status.TaskTypeCDApplicationDeploy, deploymentId, "force_recreate", true)

	// Simulate successful runtime binding so stop is allowed.
	if err := service.store.UpsertService(ctx, model.Service{
		Id:            "01KSERVICE00000000000000001",
		ApplicationId: created.Id,
		EnvironmentId: localEnv.Id,
		InstanceKey:   "default",

		VersionId: version.Version.Id,
		Status:    status.ServiceStatusRunning,
	}); err != nil {
		t.Fatal(err)
	}
	stopId, err := service.StopApplication(ctx, cdTestUserId, created.Id, cdto.ApplicationServiceTargetInput{RemoveVolumes: true})
	if err != nil {
		t.Fatal(err)
	}
	stopDeployment := loadDeploymentForTest(t, database, stopId)
	wantStop := "docker compose -p route-app-2-local-default -f docker-compose.yml down -v"
	if stopDeployment.Status != status.WorkStatusWaitingToRun || stopDeployment.CommandText != wantStop {
		t.Fatalf("unexpected stop deployment: %+v", stopDeployment)
	}
	assertLatestTaskPayload(t, database, status.TaskTypeCDApplicationStop, stopId, "remove_volumes", true)

	if err := service.store.UpdateServiceStatus(ctx, "01KSERVICE00000000000000001", status.ServiceStatusStopped); err != nil {
		t.Fatal(err)
	}
	workspace := service.workspace.(*workspaceFake)
	if err := service.DeleteApplication(ctx, cdTestUserId, created.Id, cdto.ApplicationDeleteInput{RemoveDir: true}); err != nil {
		t.Fatal(err)
	}
	if len(workspace.removedApps) != 1 || workspace.removedApps[0] != updated.Code {
		t.Fatalf("expected workspace removal for application %q, got %+v", updated.Code, workspace.removedApps)
	}
}

func TestWaitingDeploymentContainerLogDoesNotRequireWorkspace(t *testing.T) {
	service, database := newCDIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	created, err := service.CreateApplication(ctx, cdTestUserId, cdto.ApplicationCreateInput{ProjectId: cdTestProjectId, Name: "Pending Logs App", Code: "pending-logs-app", ImagePullPolicy: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	version, err := service.CreateVersion(ctx, cdTestUserId, cdto.VersionCreateInput{
		ApplicationId: created.Id,
		Label:         "v1",
		Components:    []cdto.VersionComponentInput{{Name: "web", Image: "nginx"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	localEnv := loadLocalEnvironment(t, service)
	deploymentId, err := service.DeployApplication(ctx, cdTestUserId, created.Id, cdto.ApplicationDeployInput{
		VersionId:     version.Version.Id,
		EnvironmentId: localEnv.Id,
	})
	if err != nil {
		t.Fatal(err)
	}

	log, err := service.DeploymentContainerLog(ctx, cdTestUserId, deploymentId, 200)
	if err != nil {
		t.Fatal(err)
	}
	if log.Logs != "" || log.Source != "pending" || !log.IsRealtimeSupported {
		t.Fatalf("unexpected pending container log response: %+v", log)
	}
}

func TestApplicationImportExportAndVersion(t *testing.T) {
	service, database := newCDIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	ports := `["8080:80"]`

	imported, err := service.ImportApplication(ctx, cdTestUserId, cdto.ApplicationImportInput{
		ProjectId:       cdTestProjectId,
		Name:            "Imported App",
		Code:            "imported-app",
		ImagePullPolicy: "missing",
		VersionLabel:    "import-v1",
		Components:      []cdto.VersionComponentInput{{Name: "web", Image: "nginx:1.27", PortsJSON: &ports}},
		Exposes:         []cdto.VersionExposeInput{{ComponentName: "web", Protocol: "http", ContainerPort: 80}},
	})
	if err != nil {
		t.Fatal(err)
	}
	exported, err := service.ExportApplication(ctx, cdTestUserId, imported.Id)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Application.Name != "Imported App" || len(exported.Versions) != 1 || len(exported.Versions[0].Components) != 1 || len(exported.Versions[0].Exposes) != 1 {
		t.Fatalf("unexpected exported application: %+v", exported)
	}
	if exported.Versions[0].Version.Label != "import-v1" || exported.Versions[0].Components[0].Image != "nginx:1.27" {
		t.Fatalf("unexpected exported version: %+v", exported.Versions[0])
	}

	localEnv := loadLocalEnvironment(t, service)
	preview, err := service.PreviewVersion(ctx, cdTestUserId, exported.Versions[0].Version.Id, localEnv.Id, "default")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(preview, "image: nginx:1.27") {
		t.Fatalf("unexpected preview:\n%s", preview)
	}

	published, err := service.PublishVersion(ctx, cdTestUserId, exported.Versions[0].Version.Id)
	if err != nil {
		t.Fatal(err)
	}
	if published.Version.Status != status.VersionStatusPublished {
		t.Fatalf("unexpected published status: %s", published.Version.Status)
	}
	if _, err := service.UpdateVersion(ctx, cdTestUserId, published.Version.Id, cdto.VersionUpdateInput{Label: stringPtr("blocked")}); err == nil {
		t.Fatal("expected published version to be immutable")
	}
}

func TestRouteServicePublishesCertificatesAndTraefikViews(t *testing.T) {
	service, database := newCDIntegrationService(t)
	publisher := service.routePublisher.(*recordingRoutePublisher)
	client := service.traefikRouterClient.(*recordingTraefikClient)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	route, err := service.CreateRoute(ctx, cdTestUserId, cdTestProjectId, cdto.RouteCreateInput{Name: "api-route", Domain: "api.example.test", PathPrefix: "/api", TargetURL: "http://host.docker.internal:8081", Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if route.Id == "" || route.CertType != "manual" || route.HTTPSEnabled {
		t.Fatalf("unexpected created route: %+v", route)
	}
	if err := service.DeleteRoute(ctx, cdTestUserId, route.Id); err != nil {
		t.Fatal(err)
	}

	enabled, err := service.CreateRoute(ctx, cdTestUserId, cdTestProjectId, cdto.RouteCreateInput{Name: "enabled-route", Domain: "enabled.example.test", PathPrefix: "/", TargetURL: "http://host.docker.internal:8082", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(publisher.snapshots) == 0 {
		t.Fatal("expected rest snapshot after enabling route")
	}
	lastSnap := publisher.snapshots[len(publisher.snapshots)-1]
	if len(lastSnap) != 1 || lastSnap[0].Id != enabled.Id {
		t.Fatalf("expected enabled route in snapshot, got %+v", lastSnap)
	}
	if err := service.DeleteRoute(ctx, cdTestUserId, enabled.Id); err == nil || apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("expected enabled route delete validation error, got %v", err)
	}
	disabled, err := service.DisableRoute(ctx, cdTestUserId, enabled.Id)
	if err != nil {
		t.Fatal(err)
	}
	if disabled.Enabled {
		t.Fatalf("expected disabled route, got %+v", disabled)
	}
	lastSnap = publisher.snapshots[len(publisher.snapshots)-1]
	if len(lastSnap) != 0 {
		t.Fatalf("expected empty snapshot after disable, got %+v", lastSnap)
	}

	certRoute, err := service.UploadRouteCert(ctx, cdTestUserId, disabled.Id, "CERT", "KEY")
	if err != nil {
		t.Fatal(err)
	}
	if !certRoute.HTTPSEnabled || certRoute.CertType != "manual" {
		t.Fatalf("unexpected manual cert route: %+v", certRoute)
	}
	certRoute, err = service.EnableRouteLetsEncrypt(ctx, cdTestUserId, disabled.Id)
	if err != nil {
		t.Fatal(err)
	}
	if !certRoute.HTTPSEnabled || certRoute.CertType != "letsencrypt" || certRoute.CertPEM != nil || certRoute.CertKey != nil {
		t.Fatalf("unexpected letsencrypt route: %+v", certRoute)
	}
	if len(publisher.revokedCerts) == 0 || publisher.revokedCerts[len(publisher.revokedCerts)-1] != "enabled-route" {
		t.Fatalf("expected cert revoke, got %+v", publisher.revokedCerts)
	}

	dashboardProject := cdTestProjectId
	if err := service.store.CreateRoute(ctx, model.Route{Id: "01KTRAETFIKROUTE0000000001", ProjectId: &dashboardProject, Name: "traefik-dashboard", Domain: "traefik.lvh.me", PathPrefix: "/", TargetURL: "http://traefik:8080", Enabled: true, HTTPSEnabled: true, CertType: "manual"}); err != nil {
		t.Fatal(err)
	}
	cfg, err := service.TraefikRouteConfig(ctx, cdTestUserId, cdTestProjectId)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.DashboardDomain != "traefik.lvh.me" || !cfg.HTTPSEnabled {
		t.Fatalf("unexpected traefik config: %+v", cfg)
	}
	client.routers = []cdport.TraefikRouter{{Name: "api@docker", Provider: "docker", Status: "enabled", Rule: "Host(`api.lvh.me`)", Service: "api-service", Entrypoints: []string{"websecure"}, TLS: true}}
	routes, err := service.ListTraefikRoutes(ctx, cdTestUserId, cdTestProjectId)
	if err != nil {
		t.Fatal(err)
	}
	if routes.Total != 1 || routes.Items[0].Name != "api@docker" || routes.Items[0].Provider != "docker" || routes.Items[0].Status != "enabled" || routes.Items[0].Rule != "Host(`api.lvh.me`)" || routes.Items[0].Service != "api-service" || len(routes.Items[0].Entrypoints) != 1 || routes.Items[0].Entrypoints[0] != "websecure" || !routes.Items[0].TLS {
		t.Fatalf("unexpected traefik routes: %+v", routes)
	}
	client.routers[0].Name = "mutated@docker"
	client.routers[0].Entrypoints[0] = "web"
	if routes.Items[0].Name != "api@docker" || routes.Items[0].Entrypoints[0] != "websecure" {
		t.Fatalf("expected application DTO to be independent from the port data, got %+v", routes)
	}
	routes.Items[0].Entrypoints[0] = "mutated-response"
	if client.routers[0].Entrypoints[0] != "web" {
		t.Fatalf("expected port data to be independent from the application DTO, got %+v", client.routers)
	}
	client.err = errors.New("connection refused")
	if _, err := service.ListTraefikRoutes(ctx, cdTestUserId, cdTestProjectId); err == nil || apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("expected traefik connection validation error, got %v", err)
	}
}

func newCDIntegrationService(t *testing.T) (Service, *sqlx.DB) {
	t.Helper()
	database, err := sqlx.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}}
	cfg.Cert.LetsEncrypt.Enabled = true
	cfg.Cert.LetsEncrypt.Email = "admin@example.test"
	store := cdrepo.NewRepository(database, config.DatabaseDriverSQLite)
	tasks := tasksvc.New(taskrepo.NewRepository(database, config.DatabaseDriverSQLite), 3)
	service := New(store, queuedispatch.NewCDDispatcher(tasks), cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), testWorkspace(cfg.DataRoot()), &recordingQueryRunner{}, executionlog.Store{}, &recordingRoutePublisher{}, recordingCertificateGenerator{}, &recordingTraefikClient{})
	seedTestGateway(t, service)
	return service, database
}

func seedTestGateway(t *testing.T, service Service) {
	t.Helper()
	ctx := context.Background()
	_, err := service.CreateGateway(ctx, cdTestUserId, cdto.GatewayCreateInput{
		ProjectId:       cdTestProjectId,
		Code:            "test-gateway",
		Name:            "Test Gateway",
		RestAPIURL:      "http://traefik:8080",
		BaseDomain:      "lvh.me",
		ImagePullPolicy: "missing",
	})
	if err != nil {
		t.Fatalf("seed gateway: %v", err)
	}
}

func TestCreateGatewayCompilesManagedVersion(t *testing.T) {
	service, database := newCDIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	// seedTestGateway already created one gateway; create another to assert compile output.
	img := "traefik:v3.9-test"
	view, err := service.CreateGateway(ctx, cdTestUserId, cdto.GatewayCreateInput{
		ProjectId:       cdTestProjectId,
		Code:            "edge-gw",
		Name:            "Edge GW",
		RestAPIURL:      "http://127.0.0.1:8080",
		BaseDomain:      "example.test",
		Image:           &img,
		ImagePullPolicy: "always",
	})
	if err != nil {
		t.Fatal(err)
	}
	if view.Application.ImagePullPolicy != "always" {
		t.Fatalf("gateway ImagePullPolicy = %q, want always from create input", view.Application.ImagePullPolicy)
	}
	policy := "never"
	updated, err := service.UpdateGateway(ctx, cdTestUserId, view.Application.Id, cdto.GatewayUpdateInput{
		ImagePullPolicy: &policy,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Application.ImagePullPolicy != "never" {
		t.Fatalf("after update ImagePullPolicy = %q, want never", updated.Application.ImagePullPolicy)
	}
	versions, err := service.ListVersions(ctx, cdTestUserId, view.Application.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 {
		t.Fatalf("expected one compiled version, got %d", len(versions))
	}
	v := versions[0]
	if v.Version.Status != status.VersionStatusUnpublished || v.Version.Label != gatewayCompileVersionLabel {
		t.Fatalf("unexpected version meta: %+v", v.Version)
	}
	if len(v.Components) != 1 || v.Components[0].Name != gatewayManagedComponentName {
		t.Fatalf("unexpected components: %+v", v.Components)
	}
	if v.Components[0].Image != img {
		t.Fatalf("expected image %s, got %s", img, v.Components[0].Image)
	}
	if v.Components[0].MountsJSON == nil || !strings.Contains(*v.Components[0].MountsJSON, "providers") {
		t.Fatalf("expected traefik.yml content in mounts, got %v", v.Components[0].MountsJSON)
	}
	if v.Components[0].PortsJSON == nil || !strings.Contains(*v.Components[0].PortsJSON, "8080:8080") {
		t.Fatalf("expected ports, got %v", v.Components[0].PortsJSON)
	}

	// Update image recompiles same unpublished version.
	img2 := "traefik:v3.9-updated"
	updated, err = service.UpdateGateway(ctx, cdTestUserId, view.Application.Id, cdto.GatewayUpdateInput{Image: &img2})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Config.Image == nil || *updated.Config.Image != img2 {
		t.Fatalf("config image not updated: %+v", updated.Config)
	}
	versions2, err := service.ListVersions(ctx, cdTestUserId, view.Application.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions2) != 1 {
		t.Fatalf("update must not create extra version, got %d", len(versions2))
	}
	if versions2[0].Components[0].Image != img2 {
		t.Fatalf("compile did not refresh image, got %s", versions2[0].Components[0].Image)
	}
}

func TestApplicationStatusUsesQueryRunner(t *testing.T) {
	service, database := newCDIntegrationService(t)
	defer func() { _ = database.Close() }()
	queryRunner := service.queryRunner.(*recordingQueryRunner)
	app, env, _ := createQueryTestApplicationWithService(t, service)
	queryRunner.output = "[{\"Name\":\"demo\"}]"

	output, err := service.ApplicationStatus(context.Background(), cdTestUserId, app.Id, cdto.ApplicationServiceTargetInput{})
	if err != nil {
		t.Fatal(err)
	}
	wantCwd := service.workspace.ServiceDir(app.Code, env.Code, "default")
	wantArgs := "compose -p query-app-local-default -f docker-compose.yml ps --format json"
	if output != queryRunner.output || queryRunner.cwd != wantCwd || queryRunner.name != "docker" || strings.Join(queryRunner.args, " ") != wantArgs {
		t.Fatalf("unexpected query invocation: %+v", queryRunner)
	}
}

func TestDeploymentContainerLogFallsBackToTailQuery(t *testing.T) {
	service, database := newCDIntegrationService(t)
	defer func() { _ = database.Close() }()
	queryRunner := service.queryRunner.(*recordingQueryRunner)
	app, env, svc := createQueryTestApplicationWithService(t, service)
	deployment := model.Deployment{
		Id:              "01KDEPLOYMENTLOG00000000001",
		ProjectId:       app.ProjectId,
		ApplicationId:   &app.Id,
		ApplicationName: app.Name,
		ServiceId:       &svc.Id,
		EnvironmentId:   &env.Id,
		VersionId:       &svc.VersionId,
		OperationType:   "deploy",
		TriggerType:     "manual",
		Status:          status.WorkStatusRunning,
	}
	if err := service.store.CreateDeployment(context.Background(), deployment); err != nil {
		t.Fatal(err)
	}
	queryRunner.responses = []queryResponse{{output: "since unavailable", err: errors.New("compose unavailable")}, {output: "tail logs"}}

	result, err := service.DeploymentContainerLog(context.Background(), cdTestUserId, deployment.Id, 25)
	if err != nil {
		t.Fatal(err)
	}
	wantTailArgs := "compose -p query-app-local-default -f docker-compose.yml logs --tail 25"
	if result.Logs != "tail logs" || result.Source != "tail" || len(queryRunner.calls) != 2 || strings.Join(queryRunner.calls[1].args, " ") != wantTailArgs {
		t.Fatalf("unexpected fallback result or query calls: result=%+v calls=%+v", result, queryRunner.calls)
	}
}

func createQueryTestApplicationWithService(t *testing.T, service Service) (model.Application, model.Environment, model.Service) {
	t.Helper()
	app, err := service.CreateApplication(context.Background(), cdTestUserId, cdto.ApplicationCreateInput{ProjectId: cdTestProjectId, Name: "Query App", Code: "query-app", ImagePullPolicy: "missing"})
	if err != nil {
		t.Fatal(err)
	}
	version, err := service.CreateVersion(context.Background(), cdTestUserId, cdto.VersionCreateInput{
		ApplicationId: app.Id,
		Label:         "v1",
		Components:    []cdto.VersionComponentInput{{Name: "web", Image: "nginx"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	env := loadLocalEnvironment(t, service)
	svc := model.Service{
		Id:            "01KSERVICEQUERY000000000001",
		ApplicationId: app.Id,
		EnvironmentId: env.Id,
		InstanceKey:   "default",

		VersionId: version.Version.Id,
		Status:    status.ServiceStatusRunning,
	}
	if err := service.store.UpsertService(context.Background(), svc); err != nil {
		t.Fatal(err)
	}
	return app, env, svc
}

func loadLocalEnvironment(t *testing.T, service Service) model.Environment {
	t.Helper()
	env, err := service.store.EnvironmentByProjectCode(context.Background(), cdTestProjectId, "local")
	if err != nil {
		t.Fatalf("expected seeded local environment: %v", err)
	}
	return env
}

func loadDeploymentForTest(t *testing.T, database *sqlx.DB, id string) model.Deployment {
	t.Helper()
	var deployment model.Deployment
	if err := database.Get(&deployment, `SELECT id, project_id, application_id, application_name, version_id, service_id, environment_id, options_json, operation_type, trigger_type, command_text, status, started_at, finished_at, duration_ms, log_text, error_message, is_rollback, rollback_from_deployment_id FROM deployment WHERE id = ?`, id); err != nil {
		t.Fatal(err)
	}
	return deployment
}

func assertLatestTaskPayload(t *testing.T, database *sqlx.DB, taskType string, deploymentId string, key string, want any) {
	t.Helper()
	var task struct {
		TaskType    string `db:"task_type"`
		PayloadJSON string `db:"payload_json"`
	}
	if err := database.Get(&task, `SELECT task_type, payload_json FROM background_task WHERE task_type = ? ORDER BY created_at DESC, id DESC LIMIT 1`, taskType); err != nil {
		t.Fatal(err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(task.PayloadJSON), &payload); err != nil {
		t.Fatal(err)
	}
	if payload["deployment_id"] != deploymentId || payload[key] != want {
		t.Fatalf("unexpected task payload: %+v", payload)
	}
}

type queryCall struct {
	cwd  string
	name string
	args []string
}

type queryResponse struct {
	output string
	err    error
}

type recordingQueryRunner struct {
	output    string
	err       error
	cwd       string
	name      string
	args      []string
	responses []queryResponse
	calls     []queryCall
}

func (r *recordingQueryRunner) Run(_ context.Context, cwd string, name string, args ...string) (string, error) {
	r.cwd = cwd
	r.name = name
	r.args = append([]string{}, args...)
	r.calls = append(r.calls, queryCall{cwd: cwd, name: name, args: append([]string{}, args...)})
	if len(r.responses) > 0 {
		response := r.responses[0]
		r.responses = r.responses[1:]
		return response.output, response.err
	}
	return r.output, r.err
}

type recordingRoutePublisher struct {
	snapshots    [][]model.Route
	revokedCerts []string
}

func (p *recordingRoutePublisher) ApplySnapshot(_ context.Context, _ string, routes []model.Route) error {
	p.snapshots = append(p.snapshots, append([]model.Route(nil), routes...))
	return nil
}

func (p *recordingRoutePublisher) WriteCertificate(_ context.Context, _ string, _ string, _ string) error {
	return nil
}

func (p *recordingRoutePublisher) RevokeCertificate(_ context.Context, routeName string) error {
	p.revokedCerts = append(p.revokedCerts, routeName)
	return nil
}

type recordingCertificateGenerator struct{}

func (recordingCertificateGenerator) Generate(context.Context, string) (string, string, error) {
	return "CERT", "KEY", nil
}

type recordingTraefikClient struct {
	routers []cdport.TraefikRouter
	err     error
}

func (c *recordingTraefikClient) ListRouters(_ context.Context, _ string) ([]cdport.TraefikRouter, error) {
	if c.err != nil {
		return nil, c.err
	}
	return c.routers, nil
}

func (c *recordingTraefikClient) IsConnectionError(err error) bool {
	return err != nil && errors.Is(err, c.err)
}

func stringPtr(value string) *string {
	return &value
}

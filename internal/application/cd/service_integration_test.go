package cdsvc

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"github.com/jmoiron/sqlx"
	_ "modernc.org/sqlite"

	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/storage/local/executionlog"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
	cdrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/cd"
	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/task"
)

const (
	cdTestUserId        = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	cdTestProjectId     = "01KRRKK0K3T519ZQZES3M4QA9Z"
	cdTestApplicationId = "01KN8CG4A5S4VVH6NKNJF4F9NJ"
)

func TestApplicationServiceCRUDDeployStopAndFiles(t *testing.T) {
	service, database := newCDIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	created, err := service.CreateApplication(ctx, cdTestUserId, ApplicationCreateInput{ProjectId: cdTestProjectId, Name: "Route App", Code: "route-app", ImagePullPolicy: "missing", RouteManaged: true})
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.ProjectId == nil || *created.ProjectId != cdTestProjectId || created.Name != "Route App" || created.Code != "route-app" || created.Status != status.ApplicationStatusUndeployed || !created.RouteManaged {
		t.Fatalf("unexpected created application: %+v", created)
	}

	updated, err := service.UpdateApplication(ctx, cdTestUserId, created.Id, ApplicationUpdateInput{Name: stringPtr("Route App 2"), Code: stringPtr("route-app-2"), ImagePullPolicy: stringPtr("always"), RouteManaged: boolPtr(true)})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Route App 2" || updated.Code != "route-app-2" || updated.ImagePullPolicy != "always" || !updated.RouteManaged {
		t.Fatalf("unexpected updated application: %+v", updated)
	}

	file, err := service.CreateApplicationFile(ctx, cdTestUserId, created.Id, ConfigFileInput{Path: "docker-compose.yml", Content: "services:\n  web:\n    image: nginx\n"})
	if err != nil {
		t.Fatal(err)
	}
	if file.Path != "docker-compose.yml" {
		t.Fatalf("unexpected config file: %+v", file)
	}

	deploymentId, err := service.DeployApplication(ctx, cdTestUserId, created.Id, ApplicationDeployInput{ForceRecreate: true})
	if err != nil {
		t.Fatal(err)
	}
	deployment := loadDeploymentForTest(t, database, deploymentId)
	if deployment.Status != status.WorkStatusWaitingToRun || deployment.CommandText != "docker compose -f docker-compose.yml up -d --remove-orphans --pull always --force-recreate" {
		t.Fatalf("unexpected deployment: %+v", deployment)
	}
	assertLatestTaskPayload(t, database, status.TaskTypeCDApplicationDeploy, deploymentId, "force_recreate", true)

	if _, err := database.ExecContext(ctx, `UPDATE application SET status = ? WHERE id = ?`, status.ApplicationStatusDeployed, created.Id); err != nil {
		t.Fatal(err)
	}
	stopId, err := service.StopApplication(ctx, cdTestUserId, created.Id, true)
	if err != nil {
		t.Fatal(err)
	}
	stopDeployment := loadDeploymentForTest(t, database, stopId)
	if stopDeployment.Status != status.WorkStatusWaitingToRun || stopDeployment.CommandText != "docker compose -f docker-compose.yml down -v" {
		t.Fatalf("unexpected stop deployment: %+v", stopDeployment)
	}
	assertLatestTaskPayload(t, database, status.TaskTypeCDApplicationStop, stopId, "remove_volumes", true)

	if _, err := database.ExecContext(ctx, `UPDATE application SET status = ? WHERE id = ?`, status.ApplicationStatusUndeployed, created.Id); err != nil {
		t.Fatal(err)
	}
	appDir := filepath.Join(service.cfg.DataRoot(), "cd", updated.Code)
	if err := os.MkdirAll(appDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteApplication(ctx, cdTestUserId, created.Id, ApplicationDeleteInput{RemoveDir: true}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(appDir); !os.IsNotExist(err) {
		t.Fatalf("expected application directory to be removed, stat error: %v", err)
	}
}

func TestApplicationImportExportAndServiceConfig(t *testing.T) {
	service, database := newCDIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	imported, err := service.ImportApplication(ctx, cdTestUserId, ApplicationImportInput{
		ProjectId:         cdTestProjectId,
		Name:              "Imported App",
		Code:              "imported-app",
		ImagePullPolicy:   "missing",
		RouteManaged:      true,
		ConfigFiles:       []ConfigFileInput{{Path: "docker-compose.yml", Content: "services:\n  web:\n    image: nginx\n    ports:\n      - '8080:80'\n"}},
		ServiceConfigs:    []ApplicationServiceConfigImportInput{{ServiceName: "web", Image: stringPtr("nginx:1.27")}},
		ApplicationRoutes: []ApplicationRouteInput{{ServiceName: "web", Domain: "imported.example.test", Port: 80}},
	})
	if err != nil {
		t.Fatal(err)
	}
	exported, err := service.ExportApplication(ctx, cdTestUserId, imported.Id)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Application.Name != "Imported App" || len(exported.ConfigFiles) != 1 || len(exported.ServiceConfigs) != 1 || len(exported.Routes) != 1 {
		t.Fatalf("unexpected exported application: %+v", exported)
	}

	services, err := service.ApplicationComposeServices(ctx, cdTestUserId, imported.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(services) != 1 || services[0].ServiceName != "web" || services[0].DefaultPort != 80 {
		t.Fatalf("unexpected compose services: %+v", services)
	}
	configView, err := service.UpdateApplicationServiceConfig(ctx, cdTestUserId, imported.Id, "web", stringPtr("nginx:1.28"))
	if err != nil {
		t.Fatal(err)
	}
	if configView.Image == nil || *configView.Image != "nginx:1.28" {
		t.Fatalf("unexpected service config: %+v", configView)
	}
	resetView, err := service.UpdateApplicationServiceConfig(ctx, cdTestUserId, imported.Id, "web", nil)
	if err != nil {
		t.Fatal(err)
	}
	if resetView.Image != nil || resetView.BaseImage == nil || *resetView.BaseImage != "nginx" {
		t.Fatalf("unexpected reset service config: %+v", resetView)
	}
}

func TestRouteServicePublishesCertificatesAndTraefikViews(t *testing.T) {
	service, database := newCDIntegrationService(t)
	publisher := service.routePublisher.(*recordingRoutePublisher)
	client := service.traefikRouterClient.(*recordingTraefikClient)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	route, err := service.CreateRoute(ctx, cdTestUserId, cdTestProjectId, RouteCreateInput{Name: "api-route", Domain: "api.example.test", PathPrefix: "/api", TargetURL: "http://host.docker.internal:8081", Enabled: false})
	if err != nil {
		t.Fatal(err)
	}
	if route.Id == "" || route.CertType != "manual" || route.HTTPSEnabled {
		t.Fatalf("unexpected created route: %+v", route)
	}
	if err := service.DeleteRoute(ctx, cdTestUserId, route.Id); err != nil {
		t.Fatal(err)
	}

	enabled, err := service.CreateRoute(ctx, cdTestUserId, cdTestProjectId, RouteCreateInput{Name: "enabled-route", Domain: "enabled.example.test", PathPrefix: "/", TargetURL: "http://host.docker.internal:8082", Enabled: true})
	if err != nil {
		t.Fatal(err)
	}
	if len(publisher.synced) == 0 || publisher.synced[len(publisher.synced)-1].Id != enabled.Id {
		t.Fatalf("expected enabled route to be synced, got %+v", publisher.synced)
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
	if len(publisher.revoked) == 0 || publisher.revoked[len(publisher.revoked)-1] != "enabled-route" {
		t.Fatalf("expected route config revoke, got %+v", publisher.revoked)
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
	client.routers = []model.TraefikRouter{{Name: "api@docker", Provider: "docker", Status: "enabled", Rule: "Host(`api.lvh.me`)", Service: "api-service", Entrypoints: []string{"websecure"}, TLS: true}}
	routes, err := service.ListTraefikRoutes(ctx, cdTestUserId, cdTestProjectId)
	if err != nil {
		t.Fatal(err)
	}
	if routes.Total != 1 || routes.Items[0].Name != "api@docker" || !routes.Items[0].TLS {
		t.Fatalf("unexpected traefik routes: %+v", routes)
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
	cfg := config.Config{Orbit: config.OrbitConfig{Root: t.TempDir()}, Traefik: config.TraefikConfig{DomainSuffix: "lvh.me"}}
	cfg.Cert.LetsEncrypt.Enabled = true
	cfg.Cert.LetsEncrypt.Email = "admin@example.test"
	tasks := tasksvc.New(taskrepo.NewRepository(database, config.DatabaseDriverSQLite), 3)
	service := New(cdrepo.NewRepository(database, config.DatabaseDriverSQLite), tasks, cfg, slog.New(slog.NewTextHandler(io.Discard, nil)), executionlog.Store{}, &recordingRoutePublisher{}, recordingCertificateGenerator{}, &recordingTraefikClient{})
	return service, database
}

func loadDeploymentForTest(t *testing.T, database *sqlx.DB, id string) model.Deployment {
	t.Helper()
	var deployment model.Deployment
	if err := database.Get(&deployment, `SELECT id, project_id, application_id, application_name, operation_type, trigger_type, command_text, status, started_at, finished_at, duration_ms, log_text, error_message, is_rollback, rollback_from_deployment_id FROM deployment WHERE id = ?`, id); err != nil {
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

type recordingRoutePublisher struct {
	synced       []model.Route
	revoked      []string
	revokedCerts []string
}

func (p *recordingRoutePublisher) Sync(_ context.Context, route model.Route) error {
	p.synced = append(p.synced, route)
	return nil
}

func (p *recordingRoutePublisher) Revoke(_ context.Context, routeName string) error {
	p.revoked = append(p.revoked, routeName)
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
	routers []model.TraefikRouter
	err     error
}

func (c *recordingTraefikClient) ListRouters(context.Context) ([]model.TraefikRouter, error) {
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

func boolPtr(value bool) *bool {
	return &value
}

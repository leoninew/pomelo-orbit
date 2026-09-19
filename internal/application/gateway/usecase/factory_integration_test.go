package gatewaysvc

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"testing"

	_ "modernc.org/sqlite"

	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	gatewayport "github.com/leoninew/pomelo-orbit/internal/application/gateway/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	databasepkg "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	databasetx "github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	deploymentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/deployment"
	environmentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment"
	gatewayrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/gateway"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	routerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/route"
	servicerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/service"
)

const gatewayFactoryUserId = "01KKX2YNPF6VJ9N7QYCWG61KVK"
const gatewayFactoryProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"
const seededGatewayApplicationId = "01M10RRA8F863EJ2N9TC3Z2EC1"
const seededGatewayServiceId = "01M10RRA8F863EJ2N9TTG49G6S"
const seededGatewayDashboardRouteId = "01M10RRA8F863EJ2N9TYPF0CV7"

func TestCreateGatewayCreatesAtomicServiceBundle(t *testing.T) {
	service, applications, services, database := newGatewayFactoryService(t, nil)
	defer func() { _ = database.Close() }()

	created, err := service.CreateGateway(context.Background(), gatewayFactoryUserId, gatewayFactoryProjectId, managedGatewayCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	if created.Service == nil {
		t.Fatal("gateway response is missing service")
	}
	if got := created.Service; got.Code != "traefik-default" || got.Status != status.ServiceStatusStopped {
		t.Fatalf("gateway service = %#v", got)
	}
	if created.Application.Code != "traefik" {
		t.Fatalf("application code = %q", created.Application.Code)
	}
	if len(created.Config.VersionBindings) != 4 {
		t.Fatalf("gateway Version bindings = %#v", created.Config.VersionBindings)
	}
	for _, role := range []string{"base", "http", "dns", "http-dns"} {
		if created.Config.VersionIdForProfile(role) == "" {
			t.Fatalf("missing Gateway Version binding for %q: %#v", role, created.Config.VersionBindings)
		}
	}
	versions, err := applications.ListVersions(context.Background(), gatewayFactoryProjectId, created.Application.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 4 {
		t.Fatalf("gateway Versions = %#v", versions)
	}

	version, err := applications.Version(context.Background(), gatewayFactoryProjectId, created.Service.VersionId)
	if err != nil {
		t.Fatal(err)
	}
	if version.Status != status.VersionStatusUnpublished {
		t.Fatalf("initial version status = %q", version.Status)
	}
	components, err := applications.VersionComponentsByVersion(context.Background(), gatewayFactoryProjectId, version.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(components) != 1 || components[0].Image != "traefik:3.6" || components[0].PullPolicy != "missing" {
		t.Fatalf("initial components = %#v", components)
	}
	gatewayConfig, err := gatewayrepo.NewRepository(database).GatewayConfigByProject(context.Background(), gatewayFactoryProjectId)
	if err != nil {
		t.Fatal(err)
	}
	if gatewayConfig.NetworkName != "traefik" {
		t.Fatalf("gateway network = %q", gatewayConfig.NetworkName)
	}
	if gatewayConfig.RestApiUrl != model.GatewayRestApiContainerUrl || gatewayConfig.RestApiHostUrl != model.GatewayRestApiHostUrl {
		t.Fatalf("gateway REST endpoints = container:%q host:%q", gatewayConfig.RestApiUrl, gatewayConfig.RestApiHostUrl)
	}
	var staticConfig string
	for _, mount := range components[0].Mounts {
		if mount.Target == "/etc/traefik/traefik.yml" {
			staticConfig = mount.Content
			break
		}
	}
	if !strings.Contains(staticConfig, "network: traefik") {
		t.Fatalf("initial Traefik config missing shared network: %q", staticConfig)
	}
	if len(components[0].Endpoints) != 3 || components[0].Endpoints[2].Mode != "local" || components[0].Endpoints[2].BindAddress == nil || *components[0].Endpoints[2].BindAddress != "127.0.0.1" || components[0].Endpoints[2].ListenPort == nil || *components[0].Endpoints[2].ListenPort != 8080 {
		t.Fatalf("gateway REST endpoint = %#v", components[0].Endpoints)
	}
	mappings, err := services.ServiceComponentsByService(context.Background(), gatewayFactoryProjectId, created.Service.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 1 || mappings[0].SourceVersionComponentId != components[0].Id {
		t.Fatalf("service mappings = %#v", mappings)
	}
	routes, err := routerepo.NewRepository(database).ListAllRoutes(context.Background(), gatewayFactoryProjectId)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 0 {
		t.Fatalf("gateway routes = %#v", routes)
	}
}

func TestCreateGatewayInitializationCodesDoNotRepeatProjectDefault(t *testing.T) {
	service, _, _, database := newGatewayFactoryService(t, nil)
	defer func() { _ = database.Close() }()

	input := managedGatewayCreateInput()
	input.Code = ManagedGatewayCode()
	input.Name = ManagedGatewayName()
	created, err := service.CreateGateway(context.Background(), gatewayFactoryUserId, gatewayFactoryProjectId, input)
	if err != nil {
		t.Fatal(err)
	}
	if created.Application.Code != "traefik" || created.Application.Name != "Traefik" {
		t.Fatalf("application = %#v", created.Application)
	}
	if created.Service == nil || created.Service.Code != "traefik-default" {
		t.Fatalf("gateway service = %#v", created.Service)
	}
}

func TestCreateGatewayRequiresFreshEnvironmentProbe(t *testing.T) {
	service, _, _, database := newGatewayFactoryService(t, nil)
	defer func() { _ = database.Close() }()
	if _, err := database.Exec(`UPDATE environment SET last_probe_revision = NULL, last_probe_status = NULL WHERE project_id = ?`, gatewayFactoryProjectId); err != nil {
		t.Fatal(err)
	}

	_, err := service.CreateGateway(context.Background(), gatewayFactoryUserId, gatewayFactoryProjectId, managedGatewayCreateInput())
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) || !strings.Contains(err.Error(), "must pass probe") {
		t.Fatalf("CreateGateway error = %v, want fresh Probe validation", err)
	}
}
func TestProvisionGatewayOnlyPreparesStoppedServices(t *testing.T) {
	service, applications, _, database := newGatewayFactoryService(t, nil)
	defer func() { _ = database.Close() }()
	created, err := service.CreateGateway(context.Background(), gatewayFactoryUserId, gatewayFactoryProjectId, managedGatewayCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	defaultResult, err := service.ProvisionGateway(context.Background(), gatewayFactoryUserId, gatewaydto.ProvisionGatewayInput{ProjectId: gatewayFactoryProjectId})
	if err != nil {
		t.Fatal(err)
	}
	if defaultResult.GatewayCreated || defaultResult.ServiceCreated || defaultResult.Service.Id != created.Service.Id {
		t.Fatalf("provision result = %#v", defaultResult)
	}
	version, err := applications.Version(context.Background(), gatewayFactoryProjectId, created.Service.VersionId)
	if err != nil {
		t.Fatal(err)
	}
	if version.Status != status.VersionStatusUnpublished {
		t.Fatalf("provision published version: %#v", version)
	}

}

func TestSelectGatewayDeploymentVersionUsesProfileBinding(t *testing.T) {
	service, applications, services, database := newGatewayFactoryService(t, nil)
	defer func() { _ = database.Close() }()
	created, err := service.CreateGateway(context.Background(), gatewayFactoryUserId, gatewayFactoryProjectId, managedGatewayCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	profile, email, token := "dns", "ops@example.test", "cfat_gateway_token"
	updated, err := service.UpdateGateway(context.Background(), gatewayFactoryUserId, gatewayFactoryProjectId, created.Application.Id, gatewaydto.GatewayUpdateInput{
		AcmeProfile: &profile, AcmeEmail: &email, DNSApiToken: &token,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Service == nil {
		t.Fatal("updated gateway service is missing")
	}
	selected, err := service.SelectGatewayDeploymentVersion(context.Background(), gatewayFactoryProjectId, updated.Application, *updated.Service)
	if err != nil {
		t.Fatal(err)
	}
	if selected.VersionId != updated.Config.VersionIdForProfile("dns") {
		t.Fatalf("selected Version = %q, bindings = %#v", selected.VersionId, updated.Config.VersionBindings)
	}
	component, err := applications.VersionComponentsByVersion(context.Background(), gatewayFactoryProjectId, selected.VersionId)
	if err != nil {
		t.Fatal(err)
	}
	mappings, err := services.ServiceComponentsByService(context.Background(), gatewayFactoryProjectId, selected.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(component) != 1 || len(mappings) != 1 || mappings[0].SourceVersionComponentId != component[0].Id {
		t.Fatalf("selected Gateway mappings = %#v, declarations = %#v", mappings, component)
	}
}

func TestCreateGatewayRollsBackWhenGatewayConfigWriteFails(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	database.SetMaxOpenConns(1)
	if err := databasepkg.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	removeSeededGateway(t, database)
	seedGatewayFactoryEnvironment(t, database)
	applications := applicationrepo.NewRepository(database)
	service := New(
		projectrepo.NewRepository(database),
		environmentrepo.NewRepository(database),
		applications,
		failingGatewayConfigStore{GatewayStore: gatewayrepo.NewRepository(database), err: errors.New("gateway config write failed")},
		servicerepo.NewRepository(database),
		deploymentrepo.NewRepository(database),
		resolveGatewayPathForTest,
		databasetx.NewTransactionRunner(database),
	)

	_, err = service.CreateGateway(context.Background(), gatewayFactoryUserId, gatewayFactoryProjectId, managedGatewayCreateInput())
	if err == nil {
		t.Fatal("expected gateway creation failure")
	}
	if _, err := applications.ApplicationByCode(context.Background(), gatewayFactoryProjectId, managedGatewayCode); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("gateway application survived failed factory: %v", err)
	}
}

type failingGatewayConfigStore struct {
	repository.GatewayStore
	err error
}

func (s failingGatewayConfigStore) UpsertGatewayConfig(context.Context, model.GatewayConfig) error {
	return s.err
}

func newGatewayFactoryService(t *testing.T, configStore gatewayport.ConfigStore) (Service, applicationrepo.Repository, servicerepo.Repository, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := databasepkg.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	removeSeededGateway(t, database)
	seedGatewayFactoryEnvironment(t, database)
	applications := applicationrepo.NewRepository(database)
	services := servicerepo.NewRepository(database)
	if configStore == nil {
		configStore = gatewayrepo.NewRepository(database)
	}
	return New(
		projectrepo.NewRepository(database),
		environmentrepo.NewRepository(database),
		applications,
		configStore,
		services,
		deploymentrepo.NewRepository(database),
		resolveGatewayPathForTest,
		databasetx.NewTransactionRunner(database),
	), applications, services, database
}

func resolveGatewayPathForTest(_ context.Context, logicalPath string) (string, error) {
	return logicalPath, nil
}

func seedGatewayFactoryEnvironment(t *testing.T, database *sql.DB) {
	t.Helper()
	probeRevision := int64(1)
	probeStatus := model.EnvironmentProbeStatusSucceeded
	environment := model.Environment{
		Id:                "01KROUTEGATEWAYENV00000001",
		ProjectId:         gatewayFactoryProjectId,
		Code:              "gateway-factory",
		TargetType:        model.EnvironmentTargetTypeSSH,
		WorkspaceRoot:     "/srv/pomelo-orbit",
		TargetRevision:    1,
		LastProbeRevision: &probeRevision,
		LastProbeStatus:   &probeStatus,
		SSH: &model.EnvironmentSSHTarget{
			Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22, Username: "deploy",
			CredentialId: "gateway-factory-credential", CredentialRevision: 1,
			HostKeyFingerprint: "SHA256:AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		},
	}
	if err := environmentrepo.NewRepository(database).CreateEnvironment(context.Background(), environment); err != nil {
		t.Fatalf("seed gateway factory environment: %v", err)
	}
}

func removeSeededGateway(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, err := database.Exec("DELETE FROM route WHERE id = ?", seededGatewayDashboardRouteId); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM service_component WHERE service_id = ?", seededGatewayServiceId); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM service WHERE id = ?", seededGatewayServiceId); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM application WHERE id = ?", seededGatewayApplicationId); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM environment WHERE project_id = ?", gatewayFactoryProjectId); err != nil {
		t.Fatal(err)
	}
}

func managedGatewayCreateInput() gatewaydto.GatewayCreateInput {
	image := "traefik:3.6"
	entrypoint := "web"
	tlsMode := "none"
	timeout := 20
	return gatewaydto.GatewayCreateInput{
		ProjectId:               gatewayFactoryProjectId,
		Code:                    managedGatewayCode,
		Name:                    managedGatewayName,
		RestApiUrl:              model.GatewayRestApiContainerUrl,
		RestApiHostUrl:          model.GatewayRestApiHostUrl,
		RestReadyTimeoutSeconds: &timeout,
		BaseDomain:              "example.test",
		InitialComponentImage:   &image,
		DefaultEntrypoint:       &entrypoint,
		TLSMode:                 &tlsMode,
	}
}

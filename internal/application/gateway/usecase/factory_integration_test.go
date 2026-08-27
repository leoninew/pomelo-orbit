package gatewaysvc

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	_ "modernc.org/sqlite"

	gatewaydto "github.com/leoninew/pomelo-orbit/internal/application/gateway/dto"
	gatewayport "github.com/leoninew/pomelo-orbit/internal/application/gateway/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/config"
	databasepkg "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	databasetx "github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	deploymentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/deployment"
	gatewayrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/gateway"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	routerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/route"
	servicerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/service"
)

const gatewayFactoryUserID = "01KKX2YNPF6VJ9N7QYCWG61KVK"
const gatewayFactoryProjectID = "01KRRKK0K3T519ZQZES3M4QA9Z"
const seededGatewayApplicationID = "01M10RRA8F863EJ2N9TC3Z2EC1"
const seededGatewayServiceID = "01M10RRA8F863EJ2N9TTG49G6S"
const seededGatewayDashboardRouteID = "01M10RRA8F863EJ2N9TYPF0CV7"

func TestCreateGatewayCreatesAtomicDefaultServiceBundle(t *testing.T) {
	service, applications, services, database := newGatewayFactoryService(t, nil)
	defer func() { _ = database.Close() }()

	created, err := service.CreateGateway(context.Background(), gatewayFactoryUserID, managedGatewayCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	if created.DefaultService == nil {
		t.Fatal("gateway response is missing default service")
	}
	if got := created.DefaultService; got.InstanceKey != "default" || got.Code != "traefik-default" || got.Status != status.ServiceStatusStopped {
		t.Fatalf("default service = %#v", got)
	}
	if len(created.Services) != 1 || created.Services[0].Id != created.DefaultService.Id {
		t.Fatalf("gateway services = %#v", created.Services)
	}
	if len(created.Config.VersionBindings) != 4 {
		t.Fatalf("gateway Version bindings = %#v", created.Config.VersionBindings)
	}
	for _, role := range []string{"base", "http", "dns", "http-dns"} {
		if created.Config.VersionIDForProfile(role) == "" {
			t.Fatalf("missing Gateway Version binding for %q: %#v", role, created.Config.VersionBindings)
		}
	}
	versions, err := applications.ListVersions(context.Background(), created.Application.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 4 {
		t.Fatalf("gateway Versions = %#v", versions)
	}

	version, err := applications.Version(context.Background(), created.DefaultService.VersionId)
	if err != nil {
		t.Fatal(err)
	}
	if version.Status != status.VersionStatusUnpublished {
		t.Fatalf("initial version status = %q", version.Status)
	}
	components, err := applications.VersionComponentsByVersion(context.Background(), version.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(components) != 1 || components[0].Image != "traefik:3.6" || components[0].PullPolicy != "missing" {
		t.Fatalf("initial components = %#v", components)
	}
	mappings, err := services.ServiceComponentsByService(context.Background(), created.DefaultService.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(mappings) != 1 || mappings[0].SourceVersionComponentId != components[0].Id {
		t.Fatalf("service mappings = %#v", mappings)
	}
	routes, err := routerepo.NewRepository(database).ListAllRoutes(context.Background(), gatewayFactoryProjectID)
	if err != nil {
		t.Fatal(err)
	}
	if len(routes) != 1 {
		t.Fatalf("gateway routes = %#v", routes)
	}
	route := routes[0]
	if route.Name != "traefik" || route.Domain != "traefik-dashboard.example.test" || route.TargetUrl != "http://traefik-traefik:8080" || route.Enabled || route.HTTPSEnabled || route.CertType != "manual" || route.AcmeChallenge != "http" {
		t.Fatalf("gateway dashboard route = %#v", route)
	}
	if route.ServiceId != nil || route.ComponentName != nil || route.EndpointProtocol != nil || route.EndpointContainerPort != nil {
		t.Fatalf("gateway dashboard route must use a custom target: %#v", route)
	}
}

func TestProvisionGatewayOnlyPreparesStoppedServices(t *testing.T) {
	service, applications, _, database := newGatewayFactoryService(t, nil)
	defer func() { _ = database.Close() }()
	created, err := service.CreateGateway(context.Background(), gatewayFactoryUserID, managedGatewayCreateInput())
	if err != nil {
		t.Fatal(err)
	}

	defaultResult, err := service.ProvisionGateway(context.Background(), gatewayFactoryUserID, gatewaydto.ProvisionGatewayInput{ProjectId: gatewayFactoryProjectID, InstanceKey: "default"})
	if err != nil {
		t.Fatal(err)
	}
	if defaultResult.GatewayCreated || defaultResult.ServiceCreated || defaultResult.Service.Id != created.DefaultService.Id {
		t.Fatalf("default provision result = %#v", defaultResult)
	}
	version, err := applications.Version(context.Background(), created.DefaultService.VersionId)
	if err != nil {
		t.Fatal(err)
	}
	if version.Status != status.VersionStatusUnpublished {
		t.Fatalf("provision published version: %#v", version)
	}

	stagingResult, err := service.ProvisionGateway(context.Background(), gatewayFactoryUserID, gatewaydto.ProvisionGatewayInput{ProjectId: gatewayFactoryProjectID, InstanceKey: "staging"})
	if err != nil {
		t.Fatal(err)
	}
	if !stagingResult.ServiceCreated || stagingResult.Service.InstanceKey != "staging" || stagingResult.Service.Code != "traefik-staging" || stagingResult.Service.Status != status.ServiceStatusStopped {
		t.Fatalf("staging provision result = %#v", stagingResult)
	}
	if stagingResult.Service.VersionId != created.DefaultService.VersionId {
		t.Fatalf("staging version = %q, want %q", stagingResult.Service.VersionId, created.DefaultService.VersionId)
	}
}

func TestSelectGatewayDeploymentVersionUsesProfileBinding(t *testing.T) {
	service, applications, services, database := newGatewayFactoryService(t, nil)
	defer func() { _ = database.Close() }()
	created, err := service.CreateGateway(context.Background(), gatewayFactoryUserID, managedGatewayCreateInput())
	if err != nil {
		t.Fatal(err)
	}
	profile, email, token := "dns", "ops@example.test", "cfat_gateway_token"
	updated, err := service.UpdateGateway(context.Background(), gatewayFactoryUserID, created.Application.Id, gatewaydto.GatewayUpdateInput{
		AcmeProfile: &profile, AcmeEmail: &email, DNSApiToken: &token,
	})
	if err != nil {
		t.Fatal(err)
	}
	selected, err := service.SelectGatewayDeploymentVersion(context.Background(), updated.Application, *updated.DefaultService)
	if err != nil {
		t.Fatal(err)
	}
	if selected.VersionId != updated.Config.VersionIDForProfile("dns") {
		t.Fatalf("selected Version = %q, bindings = %#v", selected.VersionId, updated.Config.VersionBindings)
	}
	component, err := applications.VersionComponentsByVersion(context.Background(), selected.VersionId)
	if err != nil {
		t.Fatal(err)
	}
	mappings, err := services.ServiceComponentsByService(context.Background(), selected.Id)
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
	if err := databasepkg.MigrateTo(database, config.DatabaseDriverSQLite, 37); err != nil {
		t.Fatal(err)
	}
	removeSeededGateway(t, database)
	applications := applicationrepo.NewRepository(database)
	service := New(
		projectrepo.NewRepository(database),
		applications,
		failingGatewayConfigStore{GatewayStore: gatewayrepo.NewRepository(database), err: errors.New("gateway config write failed")},
		servicerepo.NewRepository(database),
		routerepo.NewRepository(database),
		deploymentrepo.NewRepository(database),
		testGatewayConfig(),
		resolveGatewayPathForTest,
		databasetx.NewTransactionRunner(database),
	)

	_, err = service.CreateGateway(context.Background(), gatewayFactoryUserID, managedGatewayCreateInput())
	if err == nil {
		t.Fatal("expected gateway creation failure")
	}
	if _, err := applications.ApplicationByCode(context.Background(), managedGatewayCode); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("gateway application survived failed factory: %v", err)
	}
}

func TestCreateGatewayRollsBackWhenDashboardRouteWriteFails(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	database.SetMaxOpenConns(1)
	if err := databasepkg.MigrateTo(database, config.DatabaseDriverSQLite, 37); err != nil {
		t.Fatal(err)
	}
	removeSeededGateway(t, database)
	applications := applicationrepo.NewRepository(database)
	service := New(
		projectrepo.NewRepository(database),
		applications,
		gatewayrepo.NewRepository(database),
		servicerepo.NewRepository(database),
		failingGatewayRouteStore{err: errors.New("gateway dashboard route write failed")},
		deploymentrepo.NewRepository(database),
		testGatewayConfig(),
		resolveGatewayPathForTest,
		databasetx.NewTransactionRunner(database),
	)

	if _, err := service.CreateGateway(context.Background(), gatewayFactoryUserID, managedGatewayCreateInput()); err == nil {
		t.Fatal("expected gateway creation failure")
	}
	if _, err := applications.ApplicationByCode(context.Background(), managedGatewayCode); !errors.Is(err, repository.ErrNotFound) {
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

type failingGatewayRouteStore struct {
	repository.RouteStore
	err error
}

func (s failingGatewayRouteStore) CreateRoute(context.Context, model.Route) error {
	return s.err
}

func newGatewayFactoryService(t *testing.T, configStore gatewayport.ConfigStore) (Service, applicationrepo.Repository, servicerepo.Repository, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := databasepkg.MigrateTo(database, config.DatabaseDriverSQLite, 37); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	removeSeededGateway(t, database)
	applications := applicationrepo.NewRepository(database)
	services := servicerepo.NewRepository(database)
	if configStore == nil {
		configStore = gatewayrepo.NewRepository(database)
	}
	return New(
		projectrepo.NewRepository(database),
		applications,
		configStore,
		services,
		routerepo.NewRepository(database),
		deploymentrepo.NewRepository(database),
		testGatewayConfig(),
		resolveGatewayPathForTest,
		databasetx.NewTransactionRunner(database),
	), applications, services, database
}

func resolveGatewayPathForTest(_ context.Context, logicalPath string) (string, error) {
	return logicalPath, nil
}

func removeSeededGateway(t *testing.T, database *sql.DB) {
	t.Helper()
	if _, err := database.Exec("DELETE FROM route WHERE id = ?", seededGatewayDashboardRouteID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM service_component WHERE service_id = ?", seededGatewayServiceID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM service WHERE id = ?", seededGatewayServiceID); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("DELETE FROM application WHERE id = ?", seededGatewayApplicationID); err != nil {
		t.Fatal(err)
	}
}

func managedGatewayCreateInput() gatewaydto.GatewayCreateInput {
	image := "traefik:3.6"
	entrypoint := "web"
	tlsMode := "none"
	return gatewaydto.GatewayCreateInput{
		ProjectId:                  gatewayFactoryProjectID,
		Code:                       managedGatewayCode,
		Name:                       managedGatewayName,
		RestApiUrl:                 "http://localhost:8080",
		BaseDomain:                 "example.test",
		InitialComponentImage:      &image,
		InitialComponentPullPolicy: "missing",
		DefaultEntrypoint:          &entrypoint,
		TLSMode:                    &tlsMode,
	}
}

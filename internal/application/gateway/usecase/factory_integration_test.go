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
	servicerepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/service"
)

const gatewayFactoryUserID = "01KKX2YNPF6VJ9N7QYCWG61KVK"
const gatewayFactoryProjectID = "01KRRKK0K3T519ZQZES3M4QA9Z"

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

func TestCreateGatewayRollsBackWhenGatewayConfigWriteFails(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	database.SetMaxOpenConns(1)
	if err := databasepkg.MigrateTo(database, config.DatabaseDriverSQLite, 33); err != nil {
		t.Fatal(err)
	}
	applications := applicationrepo.NewRepository(database)
	service := New(
		projectrepo.NewRepository(database),
		applications,
		failingGatewayConfigStore{GatewayStore: gatewayrepo.NewRepository(database), err: errors.New("gateway config write failed")},
		servicerepo.NewRepository(database),
		deploymentrepo.NewRepository(database),
		testGatewayConfig(),
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
	if err := databasepkg.MigrateTo(database, config.DatabaseDriverSQLite, 33); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
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
		deploymentrepo.NewRepository(database),
		testGatewayConfig(),
		databasetx.NewTransactionRunner(database),
	), applications, services, database
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

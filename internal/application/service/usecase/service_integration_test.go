package servicesvc

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	applicationrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/application"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	servicerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/service"

	_ "modernc.org/sqlite"
)

const (
	serviceTestUserId    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	serviceTestProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"
)

func TestServiceRuntimeBindingQueries(t *testing.T) {
	service, database, applicationStore, serviceStore := newServiceIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app, version := createServiceTestApplication(t, applicationStore, "runtime-query-app")
	defaultService := model.Service{
		Id: idutil.NewId(), ApplicationId: app.Id, InstanceKey: "default",
		VersionId: version.Id, Status: status.ServiceStatusRunning,
	}
	canaryService := model.Service{
		Id: idutil.NewId(), ApplicationId: app.Id, InstanceKey: "canary",
		VersionId: version.Id, Status: status.ServiceStatusRunning,
	}
	if err := serviceStore.UpsertService(ctx, defaultService); err != nil {
		t.Fatal(err)
	}
	if err := serviceStore.UpsertService(ctx, canaryService); err != nil {
		t.Fatal(err)
	}

	page, err := service.ListServices(ctx, serviceTestUserId, servicedto.ServiceListInput{ProjectId: serviceTestProjectId, ApplicationId: app.Id})
	if err != nil {
		t.Fatal(err)
	}
	if len(page.Items) != 2 || page.Total != 2 {
		t.Fatalf("unexpected service list: %+v", page)
	}
	for _, item := range page.Items {
		if item.ApplicationName != app.Name || item.ApplicationCode != app.Code || item.VersionLabel != version.Label {
			t.Fatalf("service view is missing application or version labels: %+v", item)
		}
	}

	bindings, err := service.ListServicesByApplication(ctx, serviceTestUserId, app.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 2 {
		t.Fatalf("got %d application services, want 2", len(bindings))
	}
	primary, err := service.PrimaryServiceByApplication(ctx, serviceTestUserId, app.Id)
	if err != nil {
		t.Fatal(err)
	}
	if primary == nil || primary.ApplicationId != app.Id {
		t.Fatalf("unexpected primary service: %+v", primary)
	}

	resolved, err := service.ResolveServiceTarget(ctx, serviceTestUserId, app.Id, servicedto.ServiceTargetInput{ServiceId: canaryService.Id})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Id != canaryService.Id {
		t.Fatalf("resolved service = %s, want %s", resolved.Id, canaryService.Id)
	}
	_, err = service.ResolveServiceTarget(ctx, serviceTestUserId, app.Id, servicedto.ServiceTargetInput{})
	if apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("expected ambiguous target validation error, got %v", err)
	}
}

func TestResolveServiceTargetRejectsServiceFromAnotherApplication(t *testing.T) {
	service, database, applicationStore, serviceStore := newServiceIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	firstApp, firstVersion := createServiceTestApplication(t, applicationStore, "first-runtime-app")
	secondApp, secondVersion := createServiceTestApplication(t, applicationStore, "second-runtime-app")
	firstBinding := model.Service{Id: idutil.NewId(), ApplicationId: firstApp.Id, InstanceKey: "default", VersionId: firstVersion.Id, Status: status.ServiceStatusRunning}
	secondBinding := model.Service{Id: idutil.NewId(), ApplicationId: secondApp.Id, InstanceKey: "default", VersionId: secondVersion.Id, Status: status.ServiceStatusRunning}
	if err := serviceStore.UpsertService(ctx, firstBinding); err != nil {
		t.Fatal(err)
	}
	if err := serviceStore.UpsertService(ctx, secondBinding); err != nil {
		t.Fatal(err)
	}

	_, err := service.ResolveServiceTarget(ctx, serviceTestUserId, firstApp.Id, servicedto.ServiceTargetInput{ServiceId: secondBinding.Id})
	if apperror.StatusCode(err) != http.StatusNotFound {
		t.Fatalf("expected cross-application service to be hidden, got %v", err)
	}
}

func TestUpdateServiceConfigurationPersistsRuntimeConfig(t *testing.T) {
	service, database, applicationStore, serviceStore := newServiceIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app, version := createServiceTestApplication(t, applicationStore, "runtime-config-service")
	binding := model.Service{Id: idutil.NewId(), ApplicationId: app.Id, InstanceKey: "default", VersionId: version.Id, RuntimeConfig: map[string]string{"API_TOKEN": "service-value"}, Status: status.ServiceStatusStopped}
	if err := serviceStore.UpsertService(ctx, binding); err != nil {
		t.Fatal(err)
	}

	updated, err := service.UpdateServiceConfiguration(ctx, serviceTestUserId, binding.Id, servicedto.ServiceConfigInput{
		RuntimeConfig: map[string]string{"API_TOKEN": "next-value"}, Exposes: []servicedto.ServiceExposeInput{},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Service.RuntimeConfig["API_TOKEN"] != "next-value" {
		t.Fatalf("updated runtime config = %+v", updated.Service.RuntimeConfig)
	}
}

func TestUpdateServiceBasicPersistsVersionAndInstanceKey(t *testing.T) {
	service, database, applicationStore, serviceStore := newServiceIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app, firstVersion := createServiceTestApplication(t, applicationStore, "basic-service")
	secondVersion := model.Version{Id: idutil.NewId(), ApplicationId: app.Id, Label: "v2", Status: status.VersionStatusPublished}
	if err := applicationStore.CreateVersion(ctx, secondVersion); err != nil {
		t.Fatal(err)
	}
	binding := model.Service{Id: idutil.NewId(), ApplicationId: app.Id, InstanceKey: "default", VersionId: firstVersion.Id, Status: status.ServiceStatusStopped}
	if err := serviceStore.UpsertService(ctx, binding); err != nil {
		t.Fatal(err)
	}

	updated, err := service.UpdateServiceBasic(ctx, serviceTestUserId, binding.Id, servicedto.ServiceBasicUpdateInput{
		VersionId: secondVersion.Id, InstanceKey: "canary",
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Service.VersionId != secondVersion.Id || updated.Service.InstanceKey != "canary" {
		t.Fatalf("updated basic service = %+v", updated.Service)
	}
}

func newServiceIntegrationService(t *testing.T) (Service, *sql.DB, applicationrepo.Repository, servicerepo.Repository) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		_ = database.Close()
		t.Fatal(err)
	}
	projectStore := projectrepo.NewRepository(database)
	applicationStore := applicationrepo.NewRepository(database)
	serviceStore := servicerepo.NewRepository(database)
	return New(projectStore, applicationStore, serviceStore), database, applicationStore, serviceStore
}

func createServiceTestApplication(t *testing.T, applicationStore applicationrepo.Repository, code string) (model.Application, model.Version) {
	t.Helper()
	ctx := context.Background()
	projectId := serviceTestProjectId
	app := model.Application{
		Id: idutil.NewId(), ProjectId: &projectId, Name: code, Code: code, Kind: "application", ImagePullPolicy: "missing",
	}
	if err := applicationStore.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}
	version := model.Version{Id: idutil.NewId(), ApplicationId: app.Id, Label: "v1", Status: status.VersionStatusPublished}
	if err := applicationStore.CreateVersion(ctx, version); err != nil {
		t.Fatal(err)
	}
	return app, version
}

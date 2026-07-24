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
	environmentrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/environment"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	servicerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/service"

	_ "modernc.org/sqlite"
)

const (
	serviceTestUserID    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	serviceTestProjectID = "01KRRKK0K3T519ZQZES3M4QA9Z"
)

func TestServiceRuntimeBindingQueries(t *testing.T) {
	service, database, applicationStore, environmentStore, serviceStore := newServiceIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app, version := createServiceTestApplication(t, applicationStore, "runtime-query-app")
	local, err := environmentStore.EnvironmentByProjectCode(ctx, serviceTestProjectID, "local")
	if err != nil {
		t.Fatal(err)
	}
	staging := model.Environment{Id: idutil.NewId(), ProjectId: serviceTestProjectID, Code: "staging", Name: "Staging"}
	if err := environmentStore.CreateEnvironment(ctx, staging); err != nil {
		t.Fatal(err)
	}
	localService := model.Service{
		Id: idutil.NewId(), ApplicationId: app.Id, EnvironmentId: local.Id, InstanceKey: "default",
		VersionId: version.Id, Status: status.ServiceStatusRunning,
	}
	stagingService := model.Service{
		Id: idutil.NewId(), ApplicationId: app.Id, EnvironmentId: staging.Id, InstanceKey: "default",
		VersionId: version.Id, Status: status.ServiceStatusRunning,
	}
	if err := serviceStore.UpsertService(ctx, localService); err != nil {
		t.Fatal(err)
	}
	if err := serviceStore.UpsertService(ctx, stagingService); err != nil {
		t.Fatal(err)
	}

	page, err := service.ListServices(ctx, serviceTestUserID, servicedto.ServiceListInput{ProjectId: serviceTestProjectID, ApplicationId: app.Id})
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

	bindings, err := service.ListServicesByApplication(ctx, serviceTestUserID, app.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(bindings) != 2 {
		t.Fatalf("got %d application services, want 2", len(bindings))
	}
	primary, err := service.PrimaryServiceByApplication(ctx, serviceTestUserID, app.Id)
	if err != nil {
		t.Fatal(err)
	}
	if primary == nil || primary.ApplicationId != app.Id {
		t.Fatalf("unexpected primary service: %+v", primary)
	}

	resolved, err := service.ResolveServiceTarget(ctx, serviceTestUserID, app.Id, servicedto.ServiceTargetInput{EnvironmentId: staging.Id})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Id != stagingService.Id {
		t.Fatalf("resolved service = %s, want %s", resolved.Id, stagingService.Id)
	}
	_, err = service.ResolveServiceTarget(ctx, serviceTestUserID, app.Id, servicedto.ServiceTargetInput{})
	if apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("expected ambiguous target validation error, got %v", err)
	}
}

func TestResolveServiceTargetRejectsServiceFromAnotherApplication(t *testing.T) {
	service, database, applicationStore, environmentStore, serviceStore := newServiceIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	firstApp, firstVersion := createServiceTestApplication(t, applicationStore, "first-runtime-app")
	secondApp, secondVersion := createServiceTestApplication(t, applicationStore, "second-runtime-app")
	local, err := environmentStore.EnvironmentByProjectCode(ctx, serviceTestProjectID, "local")
	if err != nil {
		t.Fatal(err)
	}
	firstBinding := model.Service{Id: idutil.NewId(), ApplicationId: firstApp.Id, EnvironmentId: local.Id, InstanceKey: "default", VersionId: firstVersion.Id, Status: status.ServiceStatusRunning}
	secondBinding := model.Service{Id: idutil.NewId(), ApplicationId: secondApp.Id, EnvironmentId: local.Id, InstanceKey: "default", VersionId: secondVersion.Id, Status: status.ServiceStatusRunning}
	if err := serviceStore.UpsertService(ctx, firstBinding); err != nil {
		t.Fatal(err)
	}
	if err := serviceStore.UpsertService(ctx, secondBinding); err != nil {
		t.Fatal(err)
	}

	_, err = service.ResolveServiceTarget(ctx, serviceTestUserID, firstApp.Id, servicedto.ServiceTargetInput{ServiceId: secondBinding.Id})
	if apperror.StatusCode(err) != http.StatusNotFound {
		t.Fatalf("expected cross-application service to be hidden, got %v", err)
	}
}

func newServiceIntegrationService(t *testing.T) (Service, *sql.DB, applicationrepo.Repository, environmentrepo.Repository, servicerepo.Repository) {
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
	environmentStore := environmentrepo.NewRepository(database)
	serviceStore := servicerepo.NewRepository(database)
	return New(projectStore, applicationStore, serviceStore), database, applicationStore, environmentStore, serviceStore
}

func createServiceTestApplication(t *testing.T, applicationStore applicationrepo.Repository, code string) (model.Application, model.Version) {
	t.Helper()
	ctx := context.Background()
	projectID := serviceTestProjectID
	app := model.Application{
		Id: idutil.NewId(), ProjectId: &projectID, Name: code, Code: code, Kind: "application", ImagePullPolicy: "missing",
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

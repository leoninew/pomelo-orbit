package servicesvc

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	servicedto "gitee.com/leoninew/PomeloOrbit-go/internal/application/service/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	security "gitee.com/leoninew/PomeloOrbit-go/internal/common/crypto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	applicationrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/application"
	credentialrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/credential"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	servicerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/service"

	_ "modernc.org/sqlite"
)

const (
	serviceTestUserID    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	serviceTestProjectID = "01KRRKK0K3T519ZQZES3M4QA9Z"
	serviceTestSecretKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
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

	resolved, err := service.ResolveServiceTarget(ctx, serviceTestUserID, app.Id, servicedto.ServiceTargetInput{InstanceKey: "canary"})
	if err != nil {
		t.Fatal(err)
	}
	if resolved.Id != canaryService.Id {
		t.Fatalf("resolved service = %s, want %s", resolved.Id, canaryService.Id)
	}
	_, err = service.ResolveServiceTarget(ctx, serviceTestUserID, app.Id, servicedto.ServiceTargetInput{})
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

	_, err := service.ResolveServiceTarget(ctx, serviceTestUserID, firstApp.Id, servicedto.ServiceTargetInput{ServiceId: secondBinding.Id})
	if apperror.StatusCode(err) != http.StatusNotFound {
		t.Fatalf("expected cross-application service to be hidden, got %v", err)
	}
}

func TestServiceRuntimeEnvResolvesCurrentVersionReferences(t *testing.T) {
	service, database, applicationStore, serviceStore := newServiceIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app, version := createServiceTestApplication(t, applicationStore, "runtime-env-service")
	credentialStore := credentialrepo.NewRepository(database)
	projectID := serviceTestProjectID
	encrypted, err := security.EncryptString(serviceTestSecretKey, `{"API_TOKEN":"service-secret","DB_PASSWORD":"database-secret"}`)
	if err != nil {
		t.Fatal(err)
	}
	credential := model.Credential{
		Id: idutil.NewId(), ProjectId: &projectID, Name: "runtime credentials", Type: "runtime_env", EncryptedData: encrypted,
	}
	if err := credentialStore.CreateCredential(ctx, credential); err != nil {
		t.Fatal(err)
	}
	if err := applicationStore.ReplaceVersionComponents(ctx, version.Id, []model.VersionComponent{
		{
			Id: idutil.NewId(), VersionId: version.Id, Name: "api", Image: "api:latest",
			SecretEnvRefs: []model.VersionComponentSecretEnvRef{
				{EnvKey: "API_TOKEN", CredentialId: credential.Id, DataKey: "API_TOKEN"},
				{EnvKey: "DB_PASSWORD", CredentialId: credential.Id, DataKey: "DB_PASSWORD"},
			},
		},
	}); err != nil {
		t.Fatal(err)
	}
	binding := model.Service{
		Id: idutil.NewId(), ApplicationId: app.Id, InstanceKey: "default", VersionId: version.Id, Status: status.ServiceStatusRunning,
	}
	if err := serviceStore.UpsertService(ctx, binding); err != nil {
		t.Fatal(err)
	}

	view, err := service.RuntimeEnv(ctx, serviceTestUserID, binding.Id)
	if err != nil {
		t.Fatal(err)
	}
	if view.ServiceId != binding.Id || view.VersionId != version.Id || len(view.Items) != 2 {
		t.Fatalf("unexpected runtime env view: %+v", view)
	}
	if view.Items[0].EnvKey != "API_TOKEN" || view.Items[0].Value != "service-secret" || view.Items[1].EnvKey != "DB_PASSWORD" || view.Items[1].Value != "database-secret" {
		t.Fatalf("runtime env values were not resolved in stable order: %+v", view.Items)
	}

	_, err = service.RuntimeEnv(ctx, "not-a-project-member", binding.Id)
	if apperror.StatusCode(err) != http.StatusForbidden {
		t.Fatalf("expected forbidden runtime env access, got %v", err)
	}

	credential.EncryptedData = "not-a-valid-ciphertext"
	if err := credentialStore.UpdateCredential(ctx, credential); err != nil {
		t.Fatal(err)
	}
	_, err = service.RuntimeEnv(ctx, serviceTestUserID, binding.Id)
	if apperror.StatusCode(err) != http.StatusInternalServerError {
		t.Fatalf("expected unreadable credential to be internal, got %v", err)
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
	credentialStore := credentialrepo.NewRepository(database)
	serviceStore := servicerepo.NewRepository(database)
	return New(projectStore, applicationStore, credentialStore, serviceTestSecretKey, serviceStore), database, applicationStore, serviceStore
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

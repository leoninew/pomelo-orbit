package applicationsvc

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	_ "modernc.org/sqlite"

	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/config"
	db "gitee.com/leoninew/PomeloOrbit-go/internal/infrastructure/database"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	applicationrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/application"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
	servicerepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/service"
)

const (
	versionTestUserId    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	versionTestProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"
)

func TestPublishedVersionCanBeUpdatedAndDeletedWhenUnreferenced(t *testing.T) {
	service, database, applicationStore := newVersionIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app, version := createPublishedVersion(t, ctx, applicationStore, "published-editable")
	label := "v2"
	components := []applicationdto.VersionComponentInput{{Name: "web", Image: "nginx:latest"}}
	exposes := []applicationdto.VersionExposeInput{{ComponentName: "web", Protocol: "http", ContainerPort: 8080, Access: "local"}}
	updated, err := service.UpdateVersion(ctx, versionTestUserId, version.Id, applicationdto.VersionUpdateInput{
		Label:      &label,
		Components: &components,
		Exposes:    &exposes,
	})
	if err != nil {
		t.Fatalf("update published version: %v", err)
	}
	if updated.Version.Label != label || updated.Version.Status != status.VersionStatusPublished {
		t.Fatalf("unexpected updated version: %+v", updated.Version)
	}
	if len(updated.Components) != 1 || updated.Components[0].Image != "nginx:latest" {
		t.Fatalf("unexpected updated components: %+v", updated.Components)
	}
	if len(updated.Exposes) != 1 || updated.Exposes[0].ContainerPort != 8080 {
		t.Fatalf("unexpected updated exposes: %+v", updated.Exposes)
	}
	if err := service.DeleteVersion(ctx, versionTestUserId, version.Id); err != nil {
		t.Fatalf("delete published version: %v", err)
	}
	if _, err := applicationStore.Version(ctx, version.Id); err == nil {
		t.Fatalf("deleted version %s is still present for app %s", version.Id, app.Id)
	}
}

func TestDeletePublishedVersionKeepsRuntimeReferenceProtection(t *testing.T) {
	service, database, applicationStore := newVersionIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app, version := createPublishedVersion(t, ctx, applicationStore, "published-referenced")
	if err := servicerepo.NewRepository(database).UpsertService(ctx, model.Service{
		Id:            idutil.NewId(),
		ApplicationId: app.Id,
		InstanceKey:   "default",
		VersionId:     version.Id,
		Status:        status.ServiceStatusRunning,
	}); err != nil {
		t.Fatal(err)
	}

	err := service.DeleteVersion(ctx, versionTestUserId, version.Id)
	if apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("expected referenced published version to be rejected, got %v", err)
	}
}

func TestUnpublishVersionChangesPublishedVersionToUnpublished(t *testing.T) {
	service, database, applicationStore := newVersionIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	_, version := createPublishedVersion(t, ctx, applicationStore, "published-unpublish")
	view, err := service.UnpublishVersion(ctx, versionTestUserId, version.Id)
	if err != nil {
		t.Fatalf("unpublish version: %v", err)
	}
	if view.Version.Status != status.VersionStatusUnpublished {
		t.Fatalf("expected unpublished version view, got %+v", view.Version)
	}
	persisted, err := applicationStore.Version(ctx, version.Id)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.Status != status.VersionStatusUnpublished {
		t.Fatalf("expected persisted unpublished version, got %+v", persisted)
	}
}

func TestVersionComponentSummaryTracksComponentUpdatesAndListIsLightweight(t *testing.T) {
	service, database, applicationStore := newVersionIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app := model.Application{
		Id:              idutil.NewId(),
		Name:            "component-summary-" + idutil.NewId(),
		Code:            "component-summary-" + idutil.NewId(),
		Kind:            status.ApplicationKindStandard,
		ImagePullPolicy: "missing",
	}
	if err := applicationStore.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateVersion(ctx, versionTestUserId, applicationdto.VersionCreateInput{
		ApplicationId: app.Id,
		Label:         "v1",
		Components: []applicationdto.VersionComponentInput{
			{Name: "worker", Image: "busybox:1.36"},
			{Name: "api", Image: "nginx:1.27"},
		},
	})
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	if got, want := created.Version.ComponentSummary, "nginx:1.27, busybox:1.36"; got != want {
		t.Fatalf("component summary = %q, want %q", got, want)
	}

	page, err := service.ListVersionsPage(ctx, versionTestUserId, app.Id, 1, 10, "")
	if err != nil {
		t.Fatalf("list versions page: %v", err)
	}
	if len(page.Items) != 1 {
		t.Fatalf("list versions page count = %d, want 1", len(page.Items))
	}
	item := page.Items[0]
	if item.Version.ComponentSummary != created.Version.ComponentSummary {
		t.Fatalf("list component summary = %q, want %q", item.Version.ComponentSummary, created.Version.ComponentSummary)
	}
	if item.Components != nil || item.Exposes != nil {
		t.Fatalf("version list unexpectedly loaded details: %+v", item)
	}

	components := []applicationdto.VersionComponentInput{{Name: "web", Image: "caddy:2.8"}}
	updated, err := service.UpdateVersion(ctx, versionTestUserId, created.Version.Id, applicationdto.VersionUpdateInput{
		Components: &components,
	})
	if err != nil {
		t.Fatalf("update version components: %v", err)
	}
	if got, want := updated.Version.ComponentSummary, "caddy:2.8"; got != want {
		t.Fatalf("updated component summary = %q, want %q", got, want)
	}
	persisted, err := applicationStore.Version(ctx, created.Version.Id)
	if err != nil {
		t.Fatal(err)
	}
	if persisted.ComponentSummary != updated.Version.ComponentSummary {
		t.Fatalf("persisted component summary = %q, want %q", persisted.ComponentSummary, updated.Version.ComponentSummary)
	}
}

func newVersionIntegrationService(t *testing.T) (Service, *sql.DB, applicationrepo.Repository) {
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
	applicationStore := applicationrepo.NewRepository(database)
	return New(projectrepo.NewRepository(database), applicationStore), database, applicationStore
}

func createPublishedVersion(t *testing.T, ctx context.Context, applicationStore applicationrepo.Repository, code string) (model.Application, model.Version) {
	t.Helper()
	projectID := versionTestProjectId
	app := model.Application{
		Id:              idutil.NewId(),
		ProjectId:       &projectID,
		Name:            code,
		Code:            code,
		Kind:            status.ApplicationKindStandard,
		ImagePullPolicy: "missing",
	}
	if err := applicationStore.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}
	version := model.Version{
		Id:            idutil.NewId(),
		ApplicationId: app.Id,
		Label:         "v1",
		Status:        status.VersionStatusPublished,
	}
	if err := applicationStore.CreateVersion(ctx, version); err != nil {
		t.Fatal(err)
	}
	return app, version
}

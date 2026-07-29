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
	updated, err := service.UpdateVersion(ctx, versionTestUserId, version.Id, applicationdto.VersionUpdateInput{
		Label: &label,
	})
	if err != nil {
		t.Fatalf("update published version: %v", err)
	}
	if updated.Version.Label != label || updated.Version.Status != status.VersionStatusPublished {
		t.Fatalf("unexpected updated version: %+v", updated.Version)
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

func TestUpdateVersionRejectsMissingRuntimeConfigForBoundService(t *testing.T) {
	_, database, applicationStore := newVersionIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app, version := createPublishedVersion(t, ctx, applicationStore, "runtime-config-validation")
	component := model.VersionComponent{Id: idutil.NewId(), VersionId: version.Id, Name: "web", Image: "nginx"}
	if err := applicationStore.CreateVersionComponent(ctx, component); err != nil {
		t.Fatal(err)
	}
	version.Status = status.VersionStatusUnpublished
	if err := applicationStore.UpdateVersion(ctx, version); err != nil {
		t.Fatal(err)
	}
	serviceStore := servicerepo.NewRepository(database)
	if err := serviceStore.UpsertService(ctx, model.Service{
		Id: idutil.NewId(), ApplicationId: app.Id, InstanceKey: "default", VersionId: version.Id,
		RuntimeConfig: map[string]string{}, Status: status.ServiceStatusStopped,
	}); err != nil {
		t.Fatal(err)
	}
	service := New(projectrepo.NewRepository(database), applicationStore, serviceStore)
	_, err := service.UpdateVersionComponentEnv(ctx, versionTestUserId, version.Id, component.Id, applicationdto.VersionComponentEnvUpdateInput{
		Env: []model.VersionComponentEnv{{Key: "API_TOKEN", Value: "${API_TOKEN}"}},
	})
	if apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("expected missing runtime config to reject version update, got %v", err)
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
	if got, want := created.Version.ComponentSummary, "api, worker"; got != want {
		t.Fatalf("component summary = %q, want %q", got, want)
	}
	for _, component := range created.Components {
		if component.PullPolicy == nil || *component.PullPolicy != "missing" {
			t.Fatalf("component %s pull policy = %v, want missing", component.Name, component.PullPolicy)
		}
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

	var apiId string
	for _, component := range created.Components {
		if component.Name == "api" {
			apiId = component.Id
		}
	}
	updatedComponent, err := service.UpdateVersionComponentBasic(ctx, versionTestUserId, created.Version.Id, apiId, applicationdto.VersionComponentBasicUpdateInput{Name: "api", Image: "caddy:2.8", Command: ""})
	if err != nil {
		t.Fatalf("update version components: %v", err)
	}
	if updatedComponent.Image != "caddy:2.8" {
		t.Fatalf("updated component image = %q", updatedComponent.Image)
	}
	updated, err := service.VersionForUser(ctx, versionTestUserId, created.Version.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := updated.Version.ComponentSummary, "api, worker"; got != want {
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

func TestVersionComponentRenamePreservesValuesAndUpdatesReferences(t *testing.T) {
	service, database, applicationStore := newVersionIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app := model.Application{
		Id:              idutil.NewId(),
		Name:            "component-reference-" + idutil.NewId(),
		Code:            "component-reference-" + idutil.NewId(),
		Kind:            status.ApplicationKindStandard,
		ImagePullPolicy: "missing",
	}
	if err := applicationStore.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateVersion(ctx, versionTestUserId, applicationdto.VersionCreateInput{
		ApplicationId: app.Id,
		Label:         " v1 ",
		Note:          ptr("  keep this note exactly  "),
		Components: []applicationdto.VersionComponentInput{
			{
				Name:  "api",
				Image: "nginx:1.27",
				Env:   []model.VersionComponentEnv{{Key: "TOKEN", Value: "  ${TOKEN}  "}},
			},
			{
				Name: "worker", Image: "busybox:1.36",
				Dependencies: []model.VersionComponentDependency{{
					Name: "api", Condition: "service_started",
				}},
			},
		},
		Exposes: []applicationdto.VersionExposeInput{{
			ComponentName: "api", Protocol: "http", ContainerPort: 8080, Access: "local",
		}},
	})
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	if created.Version.Label != " v1 " || created.Version.Note == nil || *created.Version.Note != "  keep this note exactly  " {
		t.Fatalf("version text was changed: %+v", created.Version)
	}

	var apiId string
	for _, item := range created.Components {
		if item.Name == "api" {
			apiId = item.Id
		}
	}
	updated, err := service.UpdateVersionComponentBasic(ctx, versionTestUserId, created.Version.Id, apiId, applicationdto.VersionComponentBasicUpdateInput{
		Name: "backend", Image: "nginx:1.27", Command: "",
	})
	if err != nil {
		t.Fatalf("rename component: %v", err)
	}
	if updated.Env[0].Value != "  ${TOKEN}  " {
		t.Fatalf("component env value was changed: %q", updated.Env[0].Value)
	}

	view, err := service.VersionForUser(ctx, versionTestUserId, created.Version.Id)
	if err != nil {
		t.Fatal(err)
	}
	if view.Exposes[0].ComponentName != "backend" {
		t.Fatalf("expose reference = %q, want backend", view.Exposes[0].ComponentName)
	}
	for _, item := range view.Components {
		if item.Name == "worker" && item.Dependencies[0].Name != "backend" {
			t.Fatalf("worker dependency = %q, want backend", item.Dependencies[0].Name)
		}
	}
	if err := service.DeleteVersionComponent(ctx, versionTestUserId, created.Version.Id, apiId); apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("expected referenced component delete to fail, got %v", err)
	}
}

func TestVersionComponentGroupUpdatesPreserveOtherConfiguration(t *testing.T) {
	service, database, applicationStore := newVersionIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	app := model.Application{
		Id:              idutil.NewId(),
		Name:            "component-groups-" + idutil.NewId(),
		Code:            "component-groups-" + idutil.NewId(),
		Kind:            status.ApplicationKindStandard,
		ImagePullPolicy: "missing",
	}
	if err := applicationStore.CreateApplication(ctx, app); err != nil {
		t.Fatal(err)
	}
	created, err := service.CreateVersion(ctx, versionTestUserId, applicationdto.VersionCreateInput{
		ApplicationId: app.Id,
		Label:         "v1",
		Components: []applicationdto.VersionComponentInput{{
			Name: "api", Image: "nginx:1.27", Command: "nginx",
			Env:       []model.VersionComponentEnv{{Key: "TOKEN", Value: "before"}},
			Ports:     []model.VersionComponentPort{{HostPort: 8080, ContainerPort: 80}},
			Resources: &model.VersionComponentResources{LimitMemory: ptr("128m")},
			Tmpfs:     []model.VersionComponentTmpfs{{Target: "/run", SizeBytes: 1048576, Mode: "1777"}},
		}},
	})
	if err != nil {
		t.Fatalf("create version: %v", err)
	}
	componentId := created.Components[0].Id

	runtime, err := service.UpdateVersionComponentRuntime(ctx, versionTestUserId, created.Version.Id, componentId, applicationdto.VersionComponentRuntimeUpdateInput{})
	if err != nil {
		t.Fatalf("update runtime group: %v", err)
	}
	if got := runtime.Env[0].Value; got != "before" {
		t.Fatalf("runtime group replaced env = %q", got)
	}
	if got := runtime.Tmpfs[0].Target; got != "/run" {
		t.Fatalf("runtime group replaced tmpfs = %q", got)
	}
	if got := *runtime.Resources.LimitMemory; got != "128m" {
		t.Fatalf("runtime group replaced resources = %q", got)
	}

	ports, err := service.UpdateVersionComponentPorts(ctx, versionTestUserId, created.Version.Id, componentId, applicationdto.VersionComponentPortsUpdateInput{
		Ports: []model.VersionComponentPort{{HostPort: 9090, ContainerPort: 80}},
	})
	if err != nil {
		t.Fatalf("update ports group: %v", err)
	}
	if got := ports.Command[0]; got != "nginx" {
		t.Fatalf("ports group replaced command = %q", got)
	}
	if got := ports.Env[0].Value; got != "before" {
		t.Fatalf("ports group replaced env = %q", got)
	}

	environment, err := service.UpdateVersionComponentEnv(ctx, versionTestUserId, created.Version.Id, componentId, applicationdto.VersionComponentEnvUpdateInput{
		Env: []model.VersionComponentEnv{{Key: "TOKEN", Value: "after"}},
	})
	if err != nil {
		t.Fatalf("update env group: %v", err)
	}
	if got := environment.Ports[0].HostPort; got != 9090 {
		t.Fatalf("env group replaced ports = %d", got)
	}
	if got := environment.Tmpfs[0].Target; got != "/run" {
		t.Fatalf("env group replaced tmpfs = %q", got)
	}

	advanced, err := service.UpdateVersionComponentAdvanced(ctx, versionTestUserId, created.Version.Id, componentId, applicationdto.VersionComponentAdvancedUpdateInput{
		Resources: &model.VersionComponentResources{LimitMemory: ptr("256m")},
		Tmpfs:     []model.VersionComponentTmpfs{{Target: "/cache", SizeBytes: 2097152, Mode: "1777"}},
	})
	if err != nil {
		t.Fatalf("update advanced group: %v", err)
	}
	if got := advanced.Command[0]; got != "nginx" {
		t.Fatalf("advanced group replaced command = %q", got)
	}
	if got := advanced.Env[0].Value; got != "after" {
		t.Fatalf("advanced group replaced env = %q", got)
	}
	if got := *advanced.Resources.LimitMemory; got != "256m" {
		t.Fatalf("advanced group did not update resources = %q", got)
	}

	basic, err := service.UpdateVersionComponentBasic(ctx, versionTestUserId, created.Version.Id, componentId, applicationdto.VersionComponentBasicUpdateInput{
		Name: "backend", Image: "caddy:2.8", Command: "caddy run",
	})
	if err != nil {
		t.Fatalf("update basic group: %v", err)
	}
	if basic.Name != "backend" || basic.Image != "caddy:2.8" || basic.Command[0] != "caddy" || basic.Env[0].Value != "after" {
		t.Fatalf("basic group replaced configuration: %+v", basic)
	}
	view, err := service.VersionForUser(ctx, versionTestUserId, created.Version.Id)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := view.Version.ComponentSummary, "backend"; got != want {
		t.Fatalf("component summary = %q, want %q", got, want)
	}
}

func ptr(value string) *string {
	return &value
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
	projectId := versionTestProjectId
	app := model.Application{
		Id:              idutil.NewId(),
		ProjectId:       &projectId,
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

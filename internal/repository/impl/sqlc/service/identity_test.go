package servicerepo

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestServiceIdentityPresenceRoundTrip(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, statement := range []string{
		`INSERT INTO project (id, name, code) VALUES ('p', 'Project', 'project')`,
		`INSERT INTO application (id, project_id, name, code, kind) VALUES ('a', 'p', 'Orbit', 'orbit', 'standard')`,
		`INSERT INTO version (id, application_id, label, status) VALUES ('v', 'a', '1', 'unpublished')`,
		`INSERT INTO version_component (id, version_id, name, image, pull_policy) VALUES ('vc', 'v', 'orbit', 'orbit:1', 'missing')`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	r := NewRepository(database)
	user, empty := "1000:1000", ""
	component := model.ServiceComponent{Id: "sc", ServiceId: "s", SourceVersionComponentId: "vc", ComponentName: "orbit", Status: "active", User: &user, GroupAdd: []string{"988"}}
	if err := r.CreateServiceWithComponents(ctx, "p", model.Service{Id: "s", ProjectId: "p", ApplicationId: "a", VersionId: "v", Code: "orbit-default", Status: "stopped"}, []model.ServiceComponent{component}); err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		name   string
		user   *string
		groups []string
	}{
		{"replace", &user, []string{"988", "docker"}},
		{"clear", &empty, []string{}},
		{"inherit", nil, nil},
	} {
		t.Run(tc.name, func(t *testing.T) {
			component.User, component.GroupAdd = tc.user, tc.groups
			if err := r.UpdateServiceComponentOverlay(ctx, "p", component); err != nil {
				t.Fatal(err)
			}
			stored, err := r.ServiceComponent(ctx, "p", component.Id)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(stored.User, tc.user) || !reflect.DeepEqual(stored.GroupAdd, tc.groups) {
				t.Fatalf("identity presence lost: %+v", stored)
			}
		})
	}
}

func TestServiceMountOverlayPresenceRoundTrip(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, statement := range []string{
		`INSERT INTO project (id, name, code) VALUES ('p', 'Project', 'project')`,
		`INSERT INTO application (id, project_id, name, code, kind) VALUES ('a', 'p', 'Orbit', 'orbit', 'standard')`,
		`INSERT INTO version (id, application_id, label, status) VALUES ('v', 'a', '1', 'unpublished')`,
		`INSERT INTO version_component (id, version_id, name, image, pull_policy) VALUES ('vc', 'v', 'orbit', 'orbit:1', 'missing')`,
	} {
		if _, err := database.ExecContext(ctx, statement); err != nil {
			t.Fatal(err)
		}
	}
	repo := NewRepository(database)
	component := model.ServiceComponent{Id: "sc", ServiceId: "s", SourceVersionComponentId: "vc", ComponentName: "orbit", Status: "active"}
	if err := repo.CreateServiceWithComponents(ctx, "p", model.Service{Id: "s", ProjectId: "p", ApplicationId: "a", VersionId: "v", Code: "orbit-default", Status: "stopped"}, []model.ServiceComponent{component}); err != nil {
		t.Fatal(err)
	}
	empty, mode := "", "0000"
	disabled := false
	component.Mounts = []model.ServiceComponentMount{{Target: "/config", State: model.ServiceComponentOverlayOverride, Content: &empty, Mode: &mode, IgnoreIfExists: &disabled}}
	if err := repo.UpdateServiceComponentOverlay(ctx, "p", component); err != nil {
		t.Fatal(err)
	}
	stored, err := repo.ServiceComponent(ctx, "p", component.Id)
	if err != nil || !reflect.DeepEqual(stored.Mounts, component.Mounts) {
		t.Fatalf("mounts = %#v, error = %v", stored.Mounts, err)
	}
	component.Mounts[0].Content = nil
	if err := repo.UpdateServiceComponentOverlay(ctx, "p", component); err != nil {
		t.Fatal(err)
	}
	stored, err = repo.ServiceComponent(ctx, "p", component.Id)
	if err != nil || !reflect.DeepEqual(stored.Mounts, component.Mounts) {
		t.Fatalf("inherited fields lost: %#v, error = %v", stored.Mounts, err)
	}
	if _, err := database.Exec(`CREATE TRIGGER fail_mount BEFORE INSERT ON service_component_mount WHEN NEW.target = '/bad' BEGIN SELECT RAISE(ABORT, 'injected failure'); END`); err != nil {
		t.Fatal(err)
	}
	bad := component
	bad.Mounts = []model.ServiceComponentMount{{Target: "/bad", State: model.ServiceComponentOverlayOverride, Content: &empty}}
	if err := tx.NewTransactionRunner(database).RunInTransaction(ctx, func(transactionContext context.Context) error {
		return repo.UpdateServiceComponentOverlay(transactionContext, "p", bad)
	}); err == nil {
		t.Fatal("expected failed insert")
	}
	stored, err = repo.ServiceComponent(ctx, "p", component.Id)
	if err != nil || !reflect.DeepEqual(stored.Mounts, component.Mounts) {
		t.Fatalf("failed transaction did not preserve mounts: %#v, error = %v", stored.Mounts, err)
	}
	if _, err := repo.ServiceComponent(ctx, "other-project", component.Id); err == nil {
		t.Fatal("component visible across projects")
	}
}

package servicerepo

import (
	"context"
	"database/sql"
	"reflect"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
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

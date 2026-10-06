package applicationsvc

import (
	"context"
	"database/sql"
	"testing"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
	applicationport "github.com/leoninew/pomelo-orbit/internal/application/application/port"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	_ "modernc.org/sqlite"
)

func TestCreateVersionFromDefinitionCreatesMappedDraftVersion(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO project_member (project_id, user_id) VALUES ('project-1', 'user-1')`); err != nil {
		t.Fatal(err)
	}
	service := New(projectrepo.NewRepository(database), applicationrepo.NewRepository(database))
	application, err := service.CreateApplication(ctx, "user-1", applicationdto.ApplicationCreateInput{
		ProjectId: "project-1", Name: "Application", Code: "application", Kind: "standard",
	})
	if err != nil {
		t.Fatal(err)
	}
	versions, err := service.ListVersions(ctx, "user-1", "project-1", application.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(versions) != 1 {
		t.Fatalf("initial versions = %d, want 1", len(versions))
	}
	parentId := versions[0].Version.Id
	restartPolicy := "unless-stopped"
	user := "1000:1000"
	input := applicationdto.VersionDefinitionInput{
		Label:                "release-1",
		CreatedFromVersionId: &parentId,
		Components: []model.VersionComponent{{
			Id: "source-component", VersionId: "source-version", Name: "api", Image: "example/api:1",
			Entrypoint: []string{"/bin/api"}, Command: []string{"serve"}, PullPolicy: "missing", RestartPolicy: &restartPolicy,
			User: &user, GroupAdd: []string{"988"},
			Mounts: []model.VersionComponentMount{{SourceType: "directory", Source: "/srv/shared", Target: "/app/data", Shared: true, SourceIsHostPath: true}},
			Env:    []model.VersionComponentEnv{{Key: "PORT", Value: "8080"}},
		}},
	}
	created, err := service.CreateVersionFromDefinition(ctx, "user-1", "project-1", application.Id, input)
	if err != nil {
		t.Fatal(err)
	}
	if created.Version.Id == "source-version" || created.Version.Id == parentId {
		t.Fatalf("created version id = %q, expected a new id", created.Version.Id)
	}
	if created.Version.CreatedFromVersionId == nil || *created.Version.CreatedFromVersionId != parentId {
		t.Fatalf("created_from_version_id = %v, want %q", created.Version.CreatedFromVersionId, parentId)
	}
	if len(created.Components) != 1 {
		t.Fatalf("components = %d, want 1", len(created.Components))
	}
	component := created.Components[0]
	if component.User == nil || *component.User != user || len(component.GroupAdd) != 1 || component.GroupAdd[0] != "988" || len(component.Mounts) != 1 || !component.Mounts[0].Shared {
		t.Fatalf("definition identity/shared mount was lost: %+v", component)
	}
	if component.Id == "source-component" || component.VersionId != created.Version.Id {
		t.Fatalf("created component = %+v, expected generated identity bound to created version", component)
	}
	if component.Name != "api" || component.Image != "example/api:1" || len(component.Env) != 1 || component.Env[0].Key != "PORT" {
		t.Fatalf("created component specification = %+v", component)
	}
	if input.Components[0].Id != "source-component" || input.Components[0].VersionId != "source-version" {
		t.Fatalf("input was mutated: %+v", input.Components[0])
	}
	forked, err := service.ForkVersion(ctx, "user-1", "project-1", created.Version.Id, "forked")
	if err != nil {
		t.Fatal(err)
	}
	if len(forked.Components) != 1 || forked.Components[0].User == nil || *forked.Components[0].User != user || forked.Components[0].GroupAdd[0] != "988" || !forked.Components[0].Mounts[0].Shared {
		t.Fatal("fork lost runtime identity or shared mount")
	}
	built, err := service.ForkVersionForBuild(ctx, applicationport.BuildVersionForkInput{
		ProjectId: "project-1", SourceVersionId: created.Version.Id, Label: "build-1",
		Components: []applicationport.BuildVersionComponentUpdate{{ComponentName: "api", Image: "example/api:build", ArtifactId: "build-artifact"}},
	})
	if err != nil {
		t.Fatal(err)
	}
	builtComponents, err := applicationrepo.NewRepository(database).VersionComponentsByVersion(ctx, "project-1", built.Id)
	if err != nil {
		t.Fatal(err)
	}
	if len(builtComponents) != 1 || builtComponents[0].Image != "example/api:build" || builtComponents[0].User == nil || *builtComponents[0].User != user || len(builtComponents[0].GroupAdd) != 1 || builtComponents[0].GroupAdd[0] != "988" || len(builtComponents[0].Mounts) != 1 || !builtComponents[0].Mounts[0].Shared {
		t.Fatalf("build fork lost runtime configuration: %+v", builtComponents)
	}
	updated, err := service.UpdateVersionComponentBasic(ctx, "user-1", "project-1", created.Version.Id, component.Id, applicationdto.VersionComponentBasicUpdateInput{
		Name: component.Name, Image: "example/api:2", PullPolicy: component.PullPolicy, RestartPolicy: component.RestartPolicy,
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.User == nil || *updated.User != user || len(updated.GroupAdd) != 1 || updated.GroupAdd[0] != "988" {
		t.Fatalf("basic save changed identity: %+v", updated)
	}
	replacementUser := "2000:2000"
	updated, err = service.UpdateVersionComponentIdentity(ctx, "user-1", "project-1", created.Version.Id, component.Id, applicationdto.VersionComponentIdentityUpdateInput{User: &replacementUser, GroupAdd: []string{"999"}})
	if err != nil {
		t.Fatal(err)
	}
	if updated.User == nil || *updated.User != replacementUser || len(updated.GroupAdd) != 1 || updated.GroupAdd[0] != "999" || updated.Image != "example/api:2" || updated.Name != component.Name {
		t.Fatalf("identity save did not preserve basic fields: %+v", updated)
	}
	updated, err = service.UpdateVersionComponentIdentity(ctx, "user-1", "project-1", created.Version.Id, component.Id, applicationdto.VersionComponentIdentityUpdateInput{})
	if err != nil {
		t.Fatal(err)
	}
	if updated.User != nil || len(updated.GroupAdd) != 0 || !updated.Mounts[0].Shared || updated.Image != "example/api:2" {
		t.Fatalf("clearing identity changed unrelated mounts: %+v", updated)
	}
}

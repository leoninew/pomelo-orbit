package applicationsvc

import (
	"context"
	"database/sql"
	"testing"

	applicationdto "github.com/leoninew/pomelo-orbit/internal/application/application/dto"
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
	input := applicationdto.VersionDefinitionInput{
		Label:                "release-1",
		CreatedFromVersionId: &parentId,
		Components: []model.VersionComponent{{
			Id: "source-component", VersionId: "source-version", Name: "api", Image: "example/api:1",
			Entrypoint: []string{"/bin/api"}, Command: []string{"serve"}, PullPolicy: "missing", RestartPolicy: &restartPolicy,
			Env: []model.VersionComponentEnv{{Key: "PORT", Value: "8080"}},
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
	if component.Id == "source-component" || component.VersionId != created.Version.Id {
		t.Fatalf("created component = %+v, expected generated identity bound to created version", component)
	}
	if component.Name != "api" || component.Image != "example/api:1" || len(component.Env) != 1 || component.Env[0].Key != "PORT" {
		t.Fatalf("created component specification = %+v", component)
	}
	if input.Components[0].Id != "source-component" || input.Components[0].VersionId != "source-version" {
		t.Fatalf("input was mutated: %+v", input.Components[0])
	}
}

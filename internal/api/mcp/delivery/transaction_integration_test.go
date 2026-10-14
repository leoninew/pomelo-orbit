package delivery

import (
	"context"
	"database/sql"
	"testing"

	_ "modernc.org/sqlite"

	applicationsvc "github.com/leoninew/pomelo-orbit/internal/application/application/usecase"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	applicationrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/application"
	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// This local integration test verifies rollback across the MCP/application/SQL
// boundary; application doubles in the protocol tests cannot establish it.
func TestComponentEnvToolCommitsOrRollsBackEntireCollection(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := database.Exec(`INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project');
		INSERT INTO application (id, project_id, name, code, kind) VALUES ('app-1', 'project-1', 'App', 'app', 'standard');
		INSERT INTO version (id, application_id, label, status) VALUES ('version-1', 'app-1', 'v1', 'unpublished');`); err != nil {
		t.Fatal(err)
	}
	repo := applicationrepo.NewRepository(database)
	restartPolicy := "unless-stopped"
	component := model.VersionComponent{Id: "component-1", VersionId: "version-1", Name: "api", Image: "example/api:v1", PullPolicy: "missing", RestartPolicy: &restartPolicy, Env: []model.VersionComponentEnv{{Key: "OLD", Value: "old"}}}
	if err := repo.CreateVersionComponent(ctx, "project-1", component); err != nil {
		t.Fatal(err)
	}
	before, err := repo.Version(ctx, "project-1", "version-1")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`CREATE TRIGGER fail_env BEFORE INSERT ON version_component_env
		WHEN NEW.env_key = 'NEW' BEGIN SELECT RAISE(ABORT, 'injected insert failure'); END;`); err != nil {
		t.Fatal(err)
	}
	server := newScopedServer(t, Dependencies{Application: applicationsvc.New(transactionProjectReader{}, repo), TransactionRunner: tx.NewTransactionRunner(database)})
	session := connectInMemory(t, server)
	input := &mcp.CallToolParams{Name: "orbit_update_version_component_env", Arguments: map[string]any{
		"version_id": "version-1", "component_id": "component-1", "env": []map[string]any{{"key": "NEW", "value": "new"}},
	}}
	result, err := session.CallTool(ctx, input)
	if err != nil || !result.IsError {
		t.Fatalf("want tool failure, result=%+v err=%v", result, err)
	}
	components, err := repo.VersionComponentsByVersion(ctx, "project-1", "version-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(components) != 1 || len(components[0].Env) != 1 || components[0].Env[0] != component.Env[0] {
		t.Fatalf("failed write changed original env: %+v", components)
	}
	after, err := repo.Version(ctx, "project-1", "version-1")
	if err != nil {
		t.Fatal(err)
	}
	if after.ComponentSummary != before.ComponentSummary || !after.UpdatedAt.Equal(before.UpdatedAt) {
		t.Fatal("failed write changed version summary")
	}
	if _, err := database.Exec(`DROP TRIGGER fail_env`); err != nil {
		t.Fatal(err)
	}
	result, err = session.CallTool(ctx, input)
	if err != nil || result.IsError {
		t.Fatalf("want committed tool write, result=%+v err=%v", result, err)
	}
	components, err = repo.VersionComponentsByVersion(ctx, "project-1", "version-1")
	if err != nil {
		t.Fatal(err)
	}
	if len(components) != 1 || len(components[0].Env) != 1 || components[0].Env[0] != (model.VersionComponentEnv{Key: "NEW", Value: "new"}) {
		t.Fatalf("successful write did not replace env: %+v", components)
	}
}

type transactionProjectReader struct{ repository.ProjectReader }

func (transactionProjectReader) Project(context.Context, string) (model.Project, error) {
	return model.Project{Id: "project-1"}, nil
}
func (transactionProjectReader) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

package environmentsvc

import (
	"context"
	"database/sql"
	"runtime"
	"testing"

	_ "modernc.org/sqlite"

	credentialsvc "github.com/leoninew/pomelo-orbit/internal/application/credential/usecase"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	credentialrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/credential"
	environmentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
)

func TestEnvironmentForUserReturnsLocalSeedWithoutDeploymentSSHCredential(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}

	projectStore := projectrepo.NewRepository(database)
	credentialService := credentialsvc.New(projectStore, credentialrepo.NewRepository(database), "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=")
	platform := model.EnvironmentPlatformLinux
	workspaceRoot := "/tmp/orbit-workspace"
	if runtime.GOOS == "windows" {
		platform = model.EnvironmentPlatformWindows
		workspaceRoot = `C:\orbit-workspace`
	}
	service := New(environmentrepo.NewRepository(database), projectStore, credentialService, credentialService, nil, nil).
		WithLocalDisplay(environmentdto.LocalDisplaySnapshot{Platform: platform})

	targetType := model.EnvironmentTargetTypeLocal
	created, err := service.SaveInitialization(context.Background(), "01KKX2YNPF6VJ9N7QYCWG61KVK", "01KRRKK0K3T519ZQZES3M4QA9Z", environmentdto.UpdateInput{
		TargetType: &targetType,
		Local:      &environmentdto.LocalTargetInput{WorkspaceRoot: workspaceRoot},
	})
	if err != nil {
		t.Fatal(err)
	}
	item, err := service.EnvironmentForUser(context.Background(), "01KKX2YNPF6VJ9N7QYCWG61KVK", "01KRRKK0K3T519ZQZES3M4QA9Z")
	if err != nil {
		t.Fatal(err)
	}
	if item.TargetType != "local" || item.SSH != nil || item.Local == nil || created.Id != item.Id {
		t.Fatalf("initialized environment = %+v, want local target without SSH configuration", item)
	}

	var count int
	if err := database.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM credential WHERE project_id = ? AND type = ?`, item.ProjectId, "deployment_ssh_private_key").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("seed local environment has %d deployment SSH credentials", count)
	}
}

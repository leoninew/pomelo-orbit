package environmentsvc

import (
	"context"
	"database/sql"
	"testing"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func TestMatchingEnvironmentTargetsSaveWithWarning(t *testing.T) {
	for _, targetType := range []string{model.EnvironmentTargetTypeLocal, model.EnvironmentTargetTypeSSH} {
		t.Run(targetType, func(t *testing.T) {
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
			for _, projectId := range []string{"project-1", "project-2"} {
				if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES (?, ?, ?)`, projectId, projectId, projectId); err != nil {
					t.Fatal(err)
				}
				if _, err := database.ExecContext(ctx, `INSERT INTO project_member (project_id, user_id) VALUES (?, 'user-1')`, projectId); err != nil {
					t.Fatal(err)
				}
			}
			service := testConfigurationEnvironmentService(database)
			input := environmentdto.UpdateInput{TargetType: &targetType}
			if targetType == model.EnvironmentTargetTypeLocal {
				input.Local = &environmentdto.LocalTargetInput{WorkspaceRoot: "~/project-1"}
			} else {
				input.SSH = &environmentdto.SSHTargetInput{
					Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22,
					Username: "deploy", WorkspaceRoot: "/srv/project-1",
				}
			}
			first, err := service.SaveInitialization(ctx, "user-1", "project-1", input)
			if err != nil {
				t.Fatal(err)
			}
			if first.TargetMayBeShared {
				t.Fatal("first environment warned about its own target")
			}
			if input.Local != nil {
				input.Local.WorkspaceRoot = "~/project-2"
			} else {
				input.SSH.WorkspaceRoot = "/srv/project-2"
				input.SSH.Username = "other"
			}
			second, err := service.SaveInitialization(ctx, "user-1", "project-2", input)
			if err != nil {
				t.Fatalf("matching initialization target rejected: %v", err)
			}
			if !second.TargetMayBeShared {
				t.Fatal("matching initialization target omitted the warning")
			}
			stored, err := service.environmentForProject(ctx, "project-2")
			if err != nil || stored.Id != second.Id {
				t.Fatalf("second environment not persisted: %+v, %v", stored, err)
			}
			if input.Local != nil {
				input.Local.WorkspaceRoot = "~/edited-project-2"
			} else {
				input.SSH.WorkspaceRoot = "/srv/edited-project-2"
			}
			edited, err := service.UpdateForUser(ctx, "user-1", "project-2", input)
			if err != nil || !edited.TargetMayBeShared || edited.TargetRevision != second.TargetRevision+1 {
				t.Fatalf("matching environment edit = %+v, %v", edited, err)
			}
			stored, err = service.environmentForProject(ctx, "project-2")
			expectedWorkspaceRoot := "~/edited-project-2"
			if targetType == model.EnvironmentTargetTypeSSH {
				expectedWorkspaceRoot = "/srv/edited-project-2"
			}
			if err != nil || stored.WorkspaceRoot != expectedWorkspaceRoot {
				t.Fatalf("environment edit not persisted: %+v, %v", stored, err)
			}
			if targetType == model.EnvironmentTargetTypeSSH {
				prepared, publicKey, err := service.PrepareSSHEnvironment(ctx, "user-1", "project-2", input)
				if err != nil || publicKey == "" || !prepared.TargetMayBeShared {
					t.Fatalf("SSH command with matching target = %+v, %q, %v", prepared, publicKey, err)
				}
			}
			first, err = service.EnvironmentForUser(ctx, "user-1", "project-1")
			if err != nil || !first.TargetMayBeShared {
				t.Fatalf("existing environment warning = %+v, %v", first, err)
			}
			if _, err := database.ExecContext(ctx, `UPDATE project SET is_active = FALSE WHERE id = 'project-1'`); err != nil {
				t.Fatal(err)
			}
			second, err = service.EnvironmentForUser(ctx, "user-1", "project-2")
			if err != nil || second.TargetMayBeShared {
				t.Fatalf("inactive project still produces a warning: %+v, %v", second, err)
			}
		})
	}
}

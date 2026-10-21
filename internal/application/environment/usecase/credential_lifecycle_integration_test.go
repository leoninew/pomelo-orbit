//go:build !unit

package environmentsvc

import (
	"context"
	"database/sql"
	"errors"
	"reflect"
	"runtime"
	"testing"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/infrastructure/database/tx"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func credentialLifecycleService(t *testing.T) (*sql.DB, Service) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		"PRAGMA foreign_keys = ON",
		"INSERT INTO project (id, name, code) VALUES ('project', 'Project', 'project')",
		"INSERT INTO project_member (project_id, user_id) VALUES ('project', 'user')",
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return database, testConfigurationEnvironmentService(database)
}

func lifecycleSSHInput() environmentdto.UpdateInput {
	targetType := model.EnvironmentTargetTypeSSH
	return environmentdto.UpdateInput{TargetType: &targetType, SSH: &environmentdto.SSHTargetInput{
		Platform: "linux", Host: "host", Port: 22, Username: "orbit", WorkspaceRoot: "/srv/orbit",
	}}
}

func TestCredentialLifecycleTypeSwitchAndReplacement(t *testing.T) {
	database, service := credentialLifecycleService(t)
	ctx := context.Background()
	write := func(fn func(context.Context) error) {
		t.Helper()
		if err := tx.RunInTx(ctx, database, fn); err != nil {
			t.Fatal(err)
		}
	}
	input := lifecycleSSHInput()
	write(func(ctx context.Context) error {
		_, err := service.PrepareSSHEnvironment(ctx, "user", "project", input)
		return err
	})
	first, err := service.environments.EnvironmentByProject(ctx, "project")
	if err != nil {
		t.Fatal(err)
	}
	write(func(ctx context.Context) error {
		_, err := service.PrepareSSHEnvironment(ctx, "user", "project", input)
		return err
	})
	repeated, err := service.environments.EnvironmentByProject(ctx, "project")
	if err != nil || repeated.TargetRevision != first.TargetRevision || repeated.SSH.CredentialId != first.SSH.CredentialId {
		t.Fatalf("repeat changed identity: %+v, %v", repeated, err)
	}
	definition, err := service.TargetDefinitionForUser(ctx, "user", "project")
	if err != nil {
		t.Fatal(err)
	}
	grant := TerminalGrant{ActorId: "user", ProjectId: "project", Target: environmentport.Target{Environment: first}}
	if err := service.validateConnection(ctx, grant); err != nil {
		t.Fatal(err)
	}
	definition.Credential.PrivateKey = "replacement private key"
	write(func(ctx context.Context) error {
		_, err := service.SaveTargetDefinitionForHandover(ctx, "user", "project", definition)
		return err
	})
	replaced, err := service.environments.EnvironmentByProject(ctx, "project")
	if err != nil {
		t.Fatal(err)
	}
	if replaced.SSH.CredentialId == first.SSH.CredentialId || replaced.SSH.CredentialRevision != first.SSH.CredentialRevision {
		t.Fatal("replacement did not establish new credential identity at the same revision")
	}
	if err := service.validateConnection(ctx, grant); err == nil {
		t.Fatal("old connection accepted replacement credential")
	}
	localType, root := model.EnvironmentTargetTypeLocal, "/srv/local"
	if runtime.GOOS == "windows" {
		root = `C:\orbit`
	}
	write(func(ctx context.Context) error {
		_, err := service.UpdateForUser(ctx, "user", "project", environmentdto.UpdateInput{TargetType: &localType, Local: &environmentdto.LocalTargetInput{WorkspaceRoot: root}})
		return err
	})
	if _, err := service.environmentCredentials.EnvironmentCredentialByEnvironment(ctx, first.Id); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("local retained key: %v", err)
	}
	write(func(ctx context.Context) error {
		_, err := service.UpdateForUser(ctx, "user", "project", input)
		return err
	})
	if _, err := service.environmentCredentials.EnvironmentCredentialByEnvironment(ctx, first.Id); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("switch to SSH created key: %v", err)
	}
	write(func(ctx context.Context) error {
		_, err := service.PrepareSSHEnvironment(ctx, "user", "project", input)
		return err
	})
	newCredential, err := service.environmentCredentials.EnvironmentCredentialByEnvironment(ctx, first.Id)
	if err != nil || newCredential.Id == first.SSH.CredentialId || newCredential.Id == replaced.SSH.CredentialId {
		t.Fatalf("explicit initialization did not create new identity: %v", err)
	}
}

func TestCredentialWriteFailureRollsBackCallerTransaction(t *testing.T) {
	for _, scenario := range []string{"new environment", "existing uninitialized", "replace credential"} {
		t.Run(scenario, func(t *testing.T) {
			database, service := credentialLifecycleService(t)
			ctx := context.Background()
			input := lifecycleSSHInput()
			if scenario != "new environment" {
				if err := tx.RunInTx(ctx, database, func(ctx context.Context) error {
					if scenario == "replace credential" {
						_, err := service.PrepareSSHEnvironment(ctx, "user", "project", input)
						return err
					}
					_, err := service.SaveInitialization(ctx, "user", "project", input)
					return err
				}); err != nil {
					t.Fatal(err)
				}
			}
			before, beforeErr := service.environments.EnvironmentByProject(ctx, "project")
			var definition environmentdto.TargetDefinition
			var oldCredential model.EnvironmentCredential
			if scenario == "replace credential" {
				var err error
				definition, err = service.TargetDefinitionForUser(ctx, "user", "project")
				if err != nil {
					t.Fatal(err)
				}
				oldCredential, err = service.environmentCredentials.EnvironmentCredentialByEnvironment(ctx, before.Id)
				if err != nil {
					t.Fatal(err)
				}
				definition.SSH.Host = "changed-host"
			}
			if _, err := database.Exec(`CREATE TRIGGER fail_credential BEFORE INSERT ON environment_credential BEGIN SELECT RAISE(ABORT, 'credential write failed'); END`); err != nil {
				t.Fatal(err)
			}
			input.SSH.Host = "changed-host"
			err := tx.RunInTx(ctx, database, func(ctx context.Context) error {
				if scenario == "replace credential" {
					_, err := service.SaveTargetDefinitionForHandover(ctx, "user", "project", definition)
					return err
				}
				_, err := service.PrepareSSHEnvironment(ctx, "user", "project", input)
				return err
			})
			if err == nil {
				t.Fatal("injected credential failure succeeded")
			}
			after, afterErr := service.environments.EnvironmentByProject(ctx, "project")
			if scenario == "new environment" {
				if !errors.Is(beforeErr, repository.ErrNotFound) || !errors.Is(afterErr, repository.ErrNotFound) {
					t.Fatalf("parent survived rollback: %v", afterErr)
				}
			} else if afterErr != nil || !reflect.DeepEqual(before, after) {
				t.Fatalf("parent not restored: before=%+v after=%+v err=%v", before, after, afterErr)
			}
			if scenario == "replace credential" {
				current, err := service.environmentCredentials.EnvironmentCredentialByEnvironment(ctx, before.Id)
				if err != nil || !reflect.DeepEqual(oldCredential, current) {
					t.Fatalf("old credential not restored: %v", err)
				}
			} else {
				var count int
				if err := database.QueryRow("SELECT COUNT(*) FROM environment_credential").Scan(&count); err != nil || count != 0 {
					t.Fatalf("credential survived rollback: count=%d err=%v", count, err)
				}
			}
		})
	}
}

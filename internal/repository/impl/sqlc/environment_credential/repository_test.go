package environmentcredentialrepo

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
	environmentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment"
	_ "modernc.org/sqlite"
)

func TestCredentialBelongsToExistingEnvironment(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec("PRAGMA foreign_keys = ON"); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	credentials, environments := NewRepository(database), environmentrepo.NewRepository(database)
	credential := model.EnvironmentCredential{Id: "key", EnvironmentId: "env", PublicKey: "public", EncryptedPrivateKey: "encrypted", Revision: 2, CreatedAt: time.Now().UTC()}
	if err := credentials.CreateEnvironmentCredential(ctx, credential); err == nil {
		t.Fatal("created credential without parent")
	}
	environment := model.Environment{Id: "env", ProjectId: "project", Code: "project", TargetType: "ssh", WorkspaceRoot: "/srv/orbit", TargetRevision: 1,
		SSH: &model.EnvironmentSSHTarget{Platform: "linux", Host: "host", Port: 22, Username: "orbit", CredentialId: "ignored", CredentialRevision: 99}}
	if err := environments.CreateEnvironment(ctx, environment); err != nil {
		t.Fatal(err)
	}
	if err := credentials.CreateEnvironmentCredential(ctx, credential); err != nil {
		t.Fatal(err)
	}
	duplicate := credential
	duplicate.Id = "duplicate"
	if err := credentials.CreateEnvironmentCredential(ctx, duplicate); err == nil {
		t.Fatal("created two current credentials for one environment")
	}
	if _, err := credentials.EnvironmentCredentialByEnvironment(ctx, "other"); !errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("cross-environment read: %v", err)
	}
	if err := environments.UpdateEnvironment(ctx, environment); err != nil {
		t.Fatal(err)
	}
	loaded, err := environments.Environment(ctx, "env")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.SSH.CredentialId != credential.Id || loaded.SSH.CredentialRevision != 2 {
		t.Fatalf("environment write changed credential identity: %+v", loaded.SSH)
	}
	stored, err := credentials.EnvironmentCredentialByEnvironment(ctx, "env")
	if err != nil || stored.EnvironmentId != "env" || stored.EncryptedPrivateKey != "encrypted" {
		t.Fatalf("stored credential: %+v, %v", stored, err)
	}
	if _, err := database.Exec("DELETE FROM environment WHERE id = 'env'"); err == nil {
		t.Fatal("deleted parent with a current credential")
	}
	if err := credentials.DeleteEnvironmentCredential(ctx, "env"); err != nil {
		t.Fatal(err)
	}
	loaded, err = environments.Environment(ctx, "env")
	if err != nil || loaded.SSH.CredentialId != "" || loaded.SSH.CredentialRevision != 0 {
		t.Fatalf("deleted credential still projected: %+v, %v", loaded.SSH, err)
	}
}

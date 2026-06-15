package orbit

import (
	"context"
	"testing"

	"github.com/jmoiron/sqlx"
)

func TestCredentialStoreCRUD(t *testing.T) {
	database := openCredentialStoreDB(t)
	defer func() { _ = database.Close() }()
	store := NewStore(database, "sqlite")
	ctx := context.Background()
	projectId := "project-1"

	credential := Credential{Id: "credential-2", ProjectId: &projectId, Name: "Registry", Type: "registry_token", EncryptedData: "encrypted-2"}
	if err := store.CreateCredential(ctx, credential); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Credential(ctx, "credential-2")
	if err != nil {
		t.Fatal(err)
	}
	if loaded.Name != "Registry" || loaded.EncryptedData != "encrypted-2" {
		t.Fatalf("unexpected loaded credential: %+v", loaded)
	}

	page, err := store.ListCredentials(ctx, projectId, 1, 20, "reg")
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || len(page.Items) != 1 || page.Items[0].Id != "credential-2" {
		t.Fatalf("unexpected credential page: %+v", page)
	}

	byName, err := store.CredentialByName(ctx, projectId, "Registry")
	if err != nil {
		t.Fatal(err)
	}
	if byName.Id != "credential-2" {
		t.Fatalf("unexpected credential by name: %+v", byName)
	}

	loaded.Name = "Registry Updated"
	loaded.EncryptedData = "encrypted-updated"
	if err := store.UpdateCredential(ctx, loaded); err != nil {
		t.Fatal(err)
	}
	updated, err := store.Credential(ctx, loaded.Id)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Registry Updated" || updated.EncryptedData != "encrypted-updated" {
		t.Fatalf("unexpected updated credential: %+v", updated)
	}

	if err := store.DeleteCredential(ctx, loaded.Id); err != nil {
		t.Fatal(err)
	}
	page, err = store.ListCredentials(ctx, projectId, 1, 20, "")
	if err != nil {
		t.Fatal(err)
	}
	if page.Total != 1 || page.Items[0].Id != "credential-1" {
		t.Fatalf("unexpected credential page after delete: %+v", page)
	}
}

func TestCredentialReferencedByRepositories(t *testing.T) {
	database := openCredentialStoreDB(t)
	defer func() { _ = database.Close() }()
	store := NewStore(database, "sqlite")
	ctx := context.Background()

	referenced, err := store.CredentialReferencedByRepositories(ctx, "project-1", "credential-1")
	if err != nil {
		t.Fatal(err)
	}
	if !referenced {
		t.Fatal("expected credential to be referenced")
	}
	referenced, err = store.CredentialReferencedByRepositories(ctx, "project-2", "credential-1")
	if err != nil {
		t.Fatal(err)
	}
	if referenced {
		t.Fatal("expected credential not to be referenced by another project")
	}
}

func openCredentialStoreDB(t *testing.T) *sqlx.DB {
	t.Helper()
	database := openStoreDB(t)
	statements := []string{
		`CREATE TABLE credential (id TEXT PRIMARY KEY, name TEXT NOT NULL, type TEXT NOT NULL, encrypted_data TEXT NOT NULL, created_at DATETIME NOT NULL DEFAULT (datetime('now')), project_id TEXT)`,
		`INSERT INTO credential (id, project_id, name, type, encrypted_data) VALUES ('credential-1', 'project-1', 'GitHub', 'github_token', 'encrypted-1')`,
		`UPDATE repository SET project_id = 'project-1', git_credential_id = 'credential-1' WHERE id = 'repo-1'`,
	}
	for _, statement := range statements {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	return database
}

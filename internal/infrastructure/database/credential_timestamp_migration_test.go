package db

import (
	"testing"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestCredentialTimestampMigrationPreservesCredentialsAndRepositoryReferences(t *testing.T) {
	database := openMemoryDb(t)
	database.SetMaxOpenConns(1)
	if err := MigrateTo(database, config.DatabaseDriverSQLite, 50); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`PRAGMA foreign_keys = ON`,
		`INSERT INTO repository_credential (id, name, type, encrypted_data, revision, created_at) VALUES ('timestamp-credential', 'Timestamp credential', 'github_token', 'encrypted-token', 4, '2025-01-02 03:04:05')`,
		`INSERT INTO repository (id, name, code, repository_url, git_credential_id) VALUES ('timestamp-repository', 'Timestamp repository', 'timestamp-repository', 'https://github.com/example/repo.git', 'timestamp-credential')`,
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	var name, encryptedData string
	var revision int64
	var createdAt, updatedAt time.Time
	if err := database.QueryRow(`
		SELECT c.name, c.encrypted_data, c.revision, c.created_at, c.updated_at
		FROM repository_credential c
		JOIN repository r ON r.git_credential_id = c.id
		WHERE r.id = 'timestamp-repository'
	`).Scan(&name, &encryptedData, &revision, &createdAt, &updatedAt); err != nil {
		t.Fatal(err)
	}
	wantCreatedAt := time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC)
	if name != "Timestamp credential" || encryptedData != "encrypted-token" || revision != 4 || !createdAt.Equal(wantCreatedAt) || !updatedAt.Equal(createdAt) {
		t.Fatalf("unexpected migrated credential: name=%s data=%s revision=%d created=%v updated=%v", name, encryptedData, revision, createdAt, updatedAt)
	}
	var violations int
	if err := database.QueryRow(`SELECT COUNT(*) FROM pragma_foreign_key_check`).Scan(&violations); err != nil {
		t.Fatal(err)
	}
	if violations != 0 {
		t.Fatalf("expected valid foreign keys after migration, got %d violations", violations)
	}
}

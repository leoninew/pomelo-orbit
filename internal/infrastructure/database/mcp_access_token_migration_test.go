package db

import (
	"database/sql"
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestMCPAccessTokenEncryptionMigrationLeavesUnrecoverableValuesNull(t *testing.T) {
	database := openMemoryDb(t)
	database.SetMaxOpenConns(1)
	if err := MigrateTo(database, config.DatabaseDriverSQLite, 54); err != nil {
		t.Fatal(err)
	}
	result, err := database.Exec(`INSERT INTO mcp_access_token (id, user_id, name, token_hash)
		SELECT 'hash-only-token', id, 'Existing token', 'irreversible-hash' FROM user WHERE email = 'admin@lvh.me'`)
	if err != nil {
		t.Fatal(err)
	}
	if count, err := result.RowsAffected(); err != nil || count != 1 {
		t.Fatal("expected one existing token fixture")
	}
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	var name, hash string
	var encrypted sql.NullString
	if err := database.QueryRow(`SELECT name, token_hash, encrypted_token FROM mcp_access_token WHERE id = 'hash-only-token'`).Scan(&name, &hash, &encrypted); err != nil {
		t.Fatal(err)
	}
	if name != "Existing token" || hash != "irreversible-hash" || encrypted.Valid {
		t.Fatal("migration must preserve metadata and authentication hash, leaving the unrecoverable token NULL")
	}
}

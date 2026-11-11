package db

import (
	"testing"

	"github.com/leoninew/pomelo-orbit/internal/config"
)

func TestMigrateHostIdentityPolicyPreservesExistingPins(t *testing.T) {
	database := openMemoryDb(t)
	if err := MigrateTo(database, config.DatabaseDriverSQLite, 56); err != nil {
		t.Fatal(err)
	}
	for _, statement := range []string{
		`INSERT INTO project (id, name, code) VALUES ('project-local', 'Local', 'local'), ('project-ssh', 'SSH', 'ssh')`,
		`INSERT INTO environment (id, project_id, code, target_type, target_revision, host_key_fingerprint) VALUES ('local', 'project-local', 'local', 'local', 1, ''), ('ssh', 'project-ssh', 'ssh', 'ssh', 4, 'SHA256:existing')`,
	} {
		if _, err := database.Exec(statement); err != nil {
			t.Fatal(err)
		}
	}
	if err := MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	for _, id := range []string{"local", "ssh"} {
		var policy, pin string
		var revision int
		if err := database.QueryRow(`SELECT host_identity_policy, host_key_fingerprint, target_revision FROM environment WHERE id = ?`, id).Scan(&policy, &pin, &revision); err != nil {
			t.Fatal(err)
		}
		if policy != "negotiated" || (id == "ssh" && (pin != "SHA256:existing" || revision != 4)) || (id == "local" && pin != "") {
			t.Fatalf("migrated %s: policy=%q pin=%q revision=%d", id, policy, pin, revision)
		}
	}
	for _, statement := range []string{
		`UPDATE environment SET host_identity_policy = '' WHERE id = 'ssh'`,
		`UPDATE environment SET host_identity_policy = 'unknown' WHERE id = 'ssh'`,
		`UPDATE environment SET host_identity_policy = NULL WHERE id = 'ssh'`,
	} {
		if _, err := database.Exec(statement); err == nil {
			t.Fatalf("invalid policy accepted: %s", statement)
		}
	}
	if _, err := database.Exec(`UPDATE environment SET host_identity_policy = 'stable' WHERE id = 'ssh'`); err != nil {
		t.Fatal(err)
	}
	if err := MigrateTo(database, config.DatabaseDriverSQLite, 56); err != nil {
		t.Fatal(err)
	}
	var pin string
	if err := database.QueryRow(`SELECT host_key_fingerprint FROM environment WHERE id = 'ssh'`).Scan(&pin); err != nil || pin != "SHA256:existing" {
		t.Fatalf("downgrade pin=%q err=%v", pin, err)
	}
	var columns int
	if err := database.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('environment') WHERE name = 'host_identity_policy'`).Scan(&columns); err != nil || columns != 0 {
		t.Fatalf("downgrade policy columns=%d err=%v", columns, err)
	}
}

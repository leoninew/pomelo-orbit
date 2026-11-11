package environmentrepo

import (
	"context"
	"database/sql"
	"reflect"
	"testing"
	"time"

	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	_ "modernc.org/sqlite"
)

func TestRecordHostKeyBindsTargetAndLeavesProbeAndGatewayUntouched(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	revision, status, diagnostic, gateway := int64(3), model.EnvironmentProbeStatusFailed, "Docker unavailable", "gateway"
	probedAt := time.Now().UTC().Truncate(time.Second)
	environment := model.Environment{Id: "environment", ProjectId: "project", Code: "project", TargetType: model.EnvironmentTargetTypeSSH,
		WorkspaceRoot: "/srv/orbit", TargetRevision: revision, LastProbeRevision: &revision, LastProbeStatus: &status,
		LastProbeAt: &probedAt, LastProbeDiagnostic: &diagnostic, GatewayApplicationId: &gateway,
		SSH: &model.EnvironmentSSHTarget{Platform: "linux", Host: "host", Port: 22, Username: "orbit", CredentialId: "credential", CredentialRevision: 2, HostIdentityPolicy: model.EnvironmentHostIdentityPolicyNegotiated}}
	repo := NewRepository(database)
	if err := repo.CreateEnvironment(ctx, environment); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO environment_credential (id, environment_id, public_key, encrypted_private_key, revision) VALUES ('credential', 'environment', 'public', 'encrypted', 2)`); err != nil {
		t.Fatal(err)
	}
	before, err := repo.Environment(ctx, environment.Id)
	if err != nil {
		t.Fatal(err)
	}
	for _, scenario := range []string{"first", "same", "different", "target revision", "credential id", "credential revision"} {
		t.Run(scenario, func(t *testing.T) {
			candidate, fingerprint := environment, "SHA256:first"
			ssh := *environment.SSH
			candidate.SSH = &ssh
			switch scenario {
			case "different":
				fingerprint = "SHA256:other"
			case "target revision":
				candidate.TargetRevision--
			case "credential id":
				candidate.SSH.CredentialId = "other"
			case "credential revision":
				candidate.SSH.CredentialRevision++
			}
			recorded, err := repo.RecordHostKey(ctx, candidate, fingerprint)
			if err != nil || recorded != (scenario == "first" || scenario == "same") {
				t.Fatalf("recorded=%v err=%v", recorded, err)
			}
		})
	}
	after, err := repo.Environment(ctx, environment.Id)
	if err != nil {
		t.Fatal(err)
	}
	if after.SSH.HostKeyFingerprint != "SHA256:first" {
		t.Fatalf("recorded fingerprint = %q", after.SSH.HostKeyFingerprint)
	}
	after.SSH.HostKeyFingerprint, after.UpdatedAt = before.SSH.HostKeyFingerprint, before.UpdatedAt
	if !reflect.DeepEqual(before, after) {
		t.Fatalf("pin changed unrelated environment fields: before=%+v after=%+v", before, after)
	}
	after.SSH.HostIdentityPolicy = model.EnvironmentHostIdentityPolicyStable
	after.SSH.HostKeyFingerprint = ""
	after.TargetRevision++
	if err := repo.UpdateEnvironment(ctx, after); err != nil {
		t.Fatal(err)
	}
	if recorded, err := repo.RecordHostKey(ctx, before, "SHA256:stale"); err != nil || recorded {
		t.Fatalf("old policy observation recorded=%v err=%v", recorded, err)
	}
	if recorded, err := repo.RecordHostKey(ctx, after, "SHA256:stable"); err != nil || !recorded {
		t.Fatalf("stable policy observation recorded=%v err=%v", recorded, err)
	}
	stored, err := repo.EnvironmentByProject(ctx, after.ProjectId)
	if err != nil {
		t.Fatal(err)
	}
	if stored.SSH.HostIdentityPolicy != model.EnvironmentHostIdentityPolicyStable || stored.SSH.HostKeyFingerprint != "SHA256:stable" || stored.TargetRevision != after.TargetRevision {
		t.Fatalf("stored stable policy: %+v", stored)
	}
}

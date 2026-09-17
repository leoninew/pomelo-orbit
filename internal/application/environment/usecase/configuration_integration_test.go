package environmentsvc

import (
	"context"
	"database/sql"
	"runtime"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	environmentrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment"
	environmentcredentialrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/environment_credential"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
)

func TestSaveTargetDefinitionForUserPersistsKnownSSHTargetWithoutProbe(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	for _, row := range []struct{ id, name, code, userID string }{
		{id: "project-1", name: "Project One", code: "project-one", userID: "user-1"},
		{id: "project-2", name: "Project Two", code: "project-two", userID: "user-2"},
	} {
		if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES (?, ?, ?)`, row.id, row.name, row.code); err != nil {
			t.Fatal(err)
		}
		if _, err := database.ExecContext(ctx, `INSERT INTO project_member (project_id, user_id) VALUES (?, ?)`, row.id, row.userID); err != nil {
			t.Fatal(err)
		}
	}
	service := testConfigurationEnvironmentService(database)
	probedAt := time.Date(2026, time.September, 17, 6, 0, 0, 0, time.UTC)
	probeRevision := int64(3)
	probeStatus := model.EnvironmentProbeStatusSucceeded
	configuration := environmentdto.TargetDefinition{
		TargetType: model.EnvironmentTargetTypeSSH, WorkspaceRoot: "/srv/orbit", TargetRevision: 3,
		LastProbeRevision: &probeRevision, LastProbeStatus: &probeStatus, LastProbeAt: &probedAt,
		SSH: &environmentdto.SSHDefinition{
			Platform: model.EnvironmentPlatformLinux, Host: "10.0.0.10", Port: 22, Username: "deployer",
			CredentialRevision: 4, HostKeyFingerprint: "SHA256:AbCdEf0123456789+/=",
		},
		Credential: &environmentdto.SSHCredentialDefinition{PublicKey: "ssh-ed25519 AAAA source", PrivateKey: "PRIVATE KEY", Revision: 4},
	}
	saved, err := service.SaveTargetDefinitionForUser(ctx, "user-1", "project-1", configuration)
	if err != nil {
		t.Fatal(err)
	}
	if saved.TargetType != model.EnvironmentTargetTypeSSH || saved.SSH == nil || saved.Credential == nil {
		t.Fatalf("saved configuration = %+v", saved)
	}
	if saved.Credential.PrivateKey != "PRIVATE KEY" || saved.SSH.Host != "10.0.0.10" || saved.TargetRevision != 3 {
		t.Fatalf("saved configuration = %+v", saved)
	}
	if saved.LastProbeRevision == nil || *saved.LastProbeRevision != probeRevision || saved.LastProbeAt == nil || !saved.LastProbeAt.Equal(probedAt) {
		t.Fatalf("saved probe state = %+v", saved)
	}
	var encrypted string
	if err := database.QueryRowContext(ctx, `SELECT encrypted_private_key FROM environment_credential WHERE project_id = 'project-1'`).Scan(&encrypted); err != nil {
		t.Fatal(err)
	}
	if encrypted == "PRIVATE KEY" || encrypted == "" {
		t.Fatalf("credential storage is not encrypted: %q", encrypted)
	}
	var lastProbeStatus string
	if err := database.QueryRowContext(ctx, `SELECT last_probe_status FROM environment WHERE project_id = 'project-1'`).Scan(&lastProbeStatus); err != nil {
		t.Fatal(err)
	}
	if lastProbeStatus != model.EnvironmentProbeStatusSucceeded {
		t.Fatalf("last probe status = %q", lastProbeStatus)
	}

	_, err = service.SaveTargetDefinitionForUser(ctx, "user-2", "project-2", environmentdto.TargetDefinition{
		TargetType: model.EnvironmentTargetTypeSSH, WorkspaceRoot: "/srv/other", TargetRevision: 1,
		SSH:        &environmentdto.SSHDefinition{Platform: model.EnvironmentPlatformLinux, Host: "10.0.0.10", Port: 22, Username: "other", CredentialRevision: 1},
		Credential: &environmentdto.SSHCredentialDefinition{PublicKey: "ssh-ed25519 AAAA other", PrivateKey: "OTHER PRIVATE KEY", Revision: 1},
	})
	if err == nil {
		t.Fatal("SaveTargetDefinitionForUser() accepted a Docker target already bound to another project")
	}
}

func TestSaveTargetDefinitionForUserPersistsLocalEnvironmentWithoutCredential(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = database.Close() }()
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if _, err := database.ExecContext(ctx, `INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project')`); err != nil {
		t.Fatal(err)
	}
	if _, err := database.ExecContext(ctx, `INSERT INTO project_member (project_id, user_id) VALUES ('project-1', 'user-1')`); err != nil {
		t.Fatal(err)
	}
	service := testConfigurationEnvironmentService(database)
	workspaceRoot := "/srv/orbit"
	if runtime.GOOS == "windows" {
		workspaceRoot = `C:\orbit`
	}
	saved, err := service.SaveTargetDefinitionForUser(ctx, "user-1", "project-1", environmentdto.TargetDefinition{
		TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: workspaceRoot, TargetRevision: 1,
	})
	if err != nil {
		t.Fatal(err)
	}
	if saved.SSH != nil || saved.Credential != nil || saved.WorkspaceRoot != workspaceRoot {
		t.Fatalf("saved local configuration = %+v", saved)
	}
	var credentialCount int
	if err := database.QueryRowContext(ctx, `SELECT COUNT(*) FROM environment_credential WHERE project_id = 'project-1'`).Scan(&credentialCount); err != nil {
		t.Fatal(err)
	}
	if credentialCount != 0 {
		t.Fatalf("local environment created %d credentials", credentialCount)
	}
}

func testConfigurationEnvironmentService(database *sql.DB) Service {
	platform := model.EnvironmentPlatformLinux
	if runtime.GOOS == "windows" {
		platform = model.EnvironmentPlatformWindows
	}
	return New(
		environmentrepo.NewRepository(database),
		projectrepo.NewRepository(database),
		environmentcredentialrepo.NewRepository(database),
		"AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA=",
		nil,
		nil,
	).WithLocalDisplay(environmentdto.LocalDisplaySnapshot{Platform: platform})
}

package credentialsvc

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	credentialrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/credential"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	testseed "github.com/leoninew/pomelo-orbit/internal/testutil/seed"
)

const (
	ciTestUserId    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	ciTestProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"
	ciTestSecretKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
)

func TestCredentialServiceEncryptsExportsAndRejectsDuplicates(t *testing.T) {
	service, database := newCredentialIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	plainData := "-----BEGIN OPENSSH PRIVATE KEY-----\nsecret\n-----END OPENSSH PRIVATE KEY-----\n"

	created, err := service.CreateCredential(ctx, ciTestUserId, credentialdto.CredentialCreateInput{ProjectId: ciTestProjectId, Name: "GitHub Token", Type: "github_token", Data: plainData})
	if err != nil {
		t.Fatal(err)
	}
	if created.Id == "" || created.Name != "GitHub Token" || created.Type != "github_token" {
		t.Fatalf("unexpected credential: %+v", created)
	}
	if created.CreatedAt.IsZero() {
		t.Fatal("expected created credential timestamp to be set")
	}
	if created.EncryptedData == "" || created.EncryptedData == plainData {
		t.Fatalf("expected stored credential data to be encrypted")
	}
	decrypted, err := security.DecryptString(ciTestSecretKey, created.EncryptedData)
	if err != nil {
		t.Fatal(err)
	}
	if decrypted != plainData {
		t.Fatalf("unexpected decrypted credential data: %q", decrypted)
	}

	exported, err := service.ExportCredential(ctx, ciTestUserId, created.Id)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Version != credentialdto.CredentialExportVersion || exported.Name != "GitHub Token" || exported.Type != "github_token" || exported.Data != plainData {
		t.Fatalf("unexpected exported credential: %+v", exported)
	}

	if _, err := service.CreateCredential(ctx, ciTestUserId, credentialdto.CredentialCreateInput{ProjectId: ciTestProjectId, Name: "GitHub Token", Type: "github_token", Data: "other"}); err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("expected duplicate credential create conflict, got %v", err)
	}

	other, err := service.CreateCredential(ctx, ciTestUserId, credentialdto.CredentialCreateInput{ProjectId: ciTestProjectId, Name: "Other Token", Type: "github_token", Data: "other"})
	if err != nil {
		t.Fatal(err)
	}
	name := "GitHub Token"
	if _, err := service.UpdateCredential(ctx, ciTestUserId, other.Id, credentialdto.CredentialUpdateInput{Name: &name}); err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("expected duplicate credential update conflict, got %v", err)
	}

	if _, err := database.ExecContext(ctx, `UPDATE repository SET git_credential_id = ? WHERE id = ?`, created.Id, "01KNNRBH52BQJYT9487B2H8N62"); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteCredential(ctx, ciTestUserId, created.Id); err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected referenced credential delete validation error, got %v", err)
	}
}

func TestCreateCredentialAcceptsGiteaTokenAndRejectsUnknownType(t *testing.T) {
	service, database := newCredentialIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	created, err := service.CreateCredential(ctx, ciTestUserId, credentialdto.CredentialCreateInput{
		ProjectId: ciTestProjectId, Name: "Gitea Token", Type: "gitea_token", Data: "alice:gitea-access-token",
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.Type != "gitea_token" {
		t.Fatalf("created type=%q", created.Type)
	}
	exported, err := service.ExportCredential(ctx, ciTestUserId, created.Id)
	if err != nil {
		t.Fatal(err)
	}
	if exported.Type != "gitea_token" || exported.Data != "alice:gitea-access-token" {
		t.Fatalf("exported=%+v", exported)
	}

	_, err = service.CreateCredential(ctx, ciTestUserId, credentialdto.CredentialCreateInput{
		ProjectId: ciTestProjectId, Name: "Unknown Token", Type: "unknown_token", Data: "alice:token",
	})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected unknown credential type to be rejected, got %v", err)
	}
}

func TestDeploymentSSHCredentialPlaceholderRequiresReplacement(t *testing.T) {
	service, database := newCredentialIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	credentialID := "01M00000000000000000000001"

	if _, err := database.ExecContext(ctx, `
		INSERT INTO credential (id, project_id, name, type, encrypted_data, revision, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, credentialID, ciTestProjectId, "deployment-ssh-reconfiguration", model.CredentialTypeDeploymentSSHPrivateKey, model.DeploymentSSHCredentialReconfigurationPlaceholder, 1, time.Now().UTC()); err != nil {
		t.Fatal(err)
	}

	if _, _, err := service.DeploymentSSHCredential(ctx, credentialID); err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected placeholder credential validation error, got %v", err)
	}

	updated, err := service.EnsureGeneratedDeploymentSSHCredential(ctx, credentialID)
	if err != nil {
		t.Fatal(err)
	}
	if updated.Revision != 2 || updated.RequiresDeploymentSSHCredentialReconfiguration() {
		t.Fatalf("unexpected generated replacement: %+v", updated)
	}

	_, payload, err := service.DeploymentSSHCredential(ctx, credentialID)
	if err != nil {
		t.Fatal(err)
	}
	if payload.PrivateKey == "" || payload.PublicKey == "" || !strings.HasPrefix(payload.PublicKey, "ssh-ed25519 ") {
		t.Fatalf("unexpected generated payload: %+v", payload)
	}
}

func TestCreateDeploymentSSHCredentialReusesExistingManagedCredential(t *testing.T) {
	service, database := newCredentialIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()

	first, err := service.CreateDeploymentSSHCredential(ctx, ciTestProjectId, "")
	if err != nil {
		t.Fatal(err)
	}
	second, err := service.CreateDeploymentSSHCredential(ctx, ciTestProjectId, "")
	if err != nil {
		t.Fatal(err)
	}
	if first.Id == "" || second.Id != first.Id || second.Name != "deployment-ssh" {
		t.Fatalf("credentials were not reused: first=%+v second=%+v", first, second)
	}
}

func newCredentialIntegrationService(t *testing.T) (Service, *sql.DB) {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	testseed.ApplySQLitePipelineDemo(t, database)
	_ = slog.New(slog.NewTextHandler(io.Discard, nil))
	service := New(
		projectrepo.NewRepository(database),
		credentialrepo.NewRepository(database),
		ciTestSecretKey,
	)
	return service, database
}

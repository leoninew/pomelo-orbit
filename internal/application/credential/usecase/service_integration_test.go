package credentialsvc

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	_ "modernc.org/sqlite"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	credentialrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/credential"
	projectrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/project"
	vcsrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/repository"
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
	if !created.UpdatedAt.Equal(created.CreatedAt) {
		t.Fatal("expected newly created credential timestamps to match")
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

	exported, err := service.ExportCredential(ctx, ciTestUserId, ciTestProjectId, created.Id)
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
	if _, err := service.UpdateCredential(ctx, ciTestUserId, ciTestProjectId, other.Id, credentialdto.CredentialUpdateInput{Name: &name}); err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("expected duplicate credential update conflict, got %v", err)
	}

	if _, err := database.ExecContext(ctx, `UPDATE repository SET git_credential_id = ? WHERE id = ?`, created.Id, "01M327332NTE0VY4S5YRWJ2HZR"); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteCredential(ctx, ciTestUserId, ciTestProjectId, created.Id); err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("expected referenced credential delete validation error, got %v", err)
	}
}

func TestCredentialServiceTracksUpdatesWithoutChangingCreationTime(t *testing.T) {
	service, database := newCredentialIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	created, err := service.CreateCredential(ctx, ciTestUserId, credentialdto.CredentialCreateInput{
		ProjectId: ciTestProjectId, Name: "Tracked credential", Type: "github_token", Data: "original-token",
	})
	if err != nil {
		t.Fatal(err)
	}

	name := "Tracked renamed credential"
	data := "updated-token"
	previous := created
	for _, input := range []credentialdto.CredentialUpdateInput{{Name: &name}, {Data: &data}} {
		updated, err := service.UpdateCredential(ctx, ciTestUserId, ciTestProjectId, created.Id, input)
		if err != nil {
			t.Fatal(err)
		}
		if !updated.CreatedAt.Equal(created.CreatedAt) {
			t.Fatal("expected creation time to remain unchanged")
		}
		if !updated.UpdatedAt.After(previous.UpdatedAt) {
			t.Fatalf("expected update time to advance: previous=%v updated=%v", previous.UpdatedAt, updated.UpdatedAt)
		}
		previous = updated
	}
	items, err := service.ListCredentials(ctx, ciTestUserId, ciTestProjectId, 1, 10, "Tracked")
	if err != nil {
		t.Fatal(err)
	}
	if len(items.Items) != 1 || !items.Items[0].CreatedAt.Equal(created.CreatedAt) || !items.Items[0].UpdatedAt.Equal(previous.UpdatedAt) {
		t.Fatalf("unexpected listed credential timestamps: %+v", items.Items)
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
	exported, err := service.ExportCredential(ctx, ciTestUserId, ciTestProjectId, created.Id)
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

func TestNonRepositoryCredentialCannotBeReadOrBoundAsRepositoryCredential(t *testing.T) {
	service, database := newCredentialIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	const credentialId = "legacy-deployment-credential"
	if _, err := database.ExecContext(ctx, `INSERT INTO repository_credential (id, name, type, encrypted_data, revision) VALUES (?, ?, ?, ?, 1)`,
		credentialId, "Legacy deployment key", "deployment_ssh_private_key", "legacy-secret"); err != nil {
		t.Fatal(err)
	}

	if _, err := service.CredentialDetailForUser(ctx, ciTestUserId, ciTestProjectId, credentialId); !apperror.IsKind(err, apperror.KindNotFound) {
		t.Fatalf("expected non-repository credential detail to be unavailable, got %v", err)
	}
	if _, err := service.ExportCredential(ctx, ciTestUserId, ciTestProjectId, credentialId); !apperror.IsKind(err, apperror.KindNotFound) {
		t.Fatalf("expected non-repository credential export to be unavailable, got %v", err)
	}
	exists, err := credentialrepo.NewRepository(database).CredentialExists(ctx, credentialId)
	if err != nil || exists {
		t.Fatalf("expected non-repository credential to be unavailable for repository binding, exists=%t err=%v", exists, err)
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
	_ = slog.New(slog.NewTextHandler(io.Discard, nil))
	service := New(
		projectrepo.NewRepository(database),
		credentialrepo.NewRepository(database),
		vcsrepo.NewRepository(database),
		ciTestSecretKey,
	)
	return service, database
}

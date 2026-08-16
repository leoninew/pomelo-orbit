package credentialsvc

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"testing"

	_ "modernc.org/sqlite"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
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

	if _, err := service.CreateCredential(ctx, ciTestUserId, credentialdto.CredentialCreateInput{ProjectId: ciTestProjectId, Name: "GitHub Token", Type: "github_token", Data: "other"}); err == nil || apperror.StatusCode(err) != http.StatusConflict {
		t.Fatalf("expected duplicate credential create conflict, got %v", err)
	}

	other, err := service.CreateCredential(ctx, ciTestUserId, credentialdto.CredentialCreateInput{ProjectId: ciTestProjectId, Name: "Other Token", Type: "github_token", Data: "other"})
	if err != nil {
		t.Fatal(err)
	}
	name := "GitHub Token"
	if _, err := service.UpdateCredential(ctx, ciTestUserId, other.Id, credentialdto.CredentialUpdateInput{Name: &name}); err == nil || apperror.StatusCode(err) != http.StatusConflict {
		t.Fatalf("expected duplicate credential update conflict, got %v", err)
	}

	if _, err := database.ExecContext(ctx, `UPDATE repository SET git_credential_id = ? WHERE id = ?`, created.Id, "01KNNRBH52BQJYT9487B2H8N62"); err != nil {
		t.Fatal(err)
	}
	if err := service.DeleteCredential(ctx, ciTestUserId, created.Id); err == nil || apperror.StatusCode(err) != http.StatusBadRequest {
		t.Fatalf("expected referenced credential delete validation error, got %v", err)
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

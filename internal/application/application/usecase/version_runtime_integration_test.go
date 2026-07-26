package applicationsvc

import (
	"context"
	"testing"

	applicationdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/application/dto"
	security "gitee.com/leoninew/PomeloOrbit-go/internal/common/crypto"
	apperror "gitee.com/leoninew/PomeloOrbit-go/internal/common/errors"
	idutil "gitee.com/leoninew/PomeloOrbit-go/internal/common/util"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
	credentialrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/credential"
	projectrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/impl/sqlc/project"
)

const runtimeVersionTestSecretKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

func TestVersionPersistsRuntimeFieldsAndCredentialReferences(t *testing.T) {
	service, database, applicationStore := newVersionIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	app, _ := createPublishedVersion(t, ctx, applicationStore, "runtime-fields")

	encrypted, err := security.EncryptString(runtimeVersionTestSecretKey, `{"MYSQL_PASSWORD":"runtime-test-secret"}`)
	if err != nil {
		t.Fatal(err)
	}
	credentialId := idutil.NewId()
	projectId := versionTestProjectId
	credentialStore := credentialrepo.NewRepository(database)
	if err := credentialStore.CreateCredential(ctx, model.Credential{
		Id: credentialId, ProjectId: &projectId, Name: "runtime", Type: "runtime_env", EncryptedData: encrypted,
	}); err != nil {
		t.Fatal(err)
	}
	service = NewWithCredential(projectrepo.NewRepository(database), applicationStore, credentialStore, runtimeVersionTestSecretKey)

	restartPolicy := "unless-stopped"
	tmpfsJSON := `[{"target":"/tmp","size_bytes":1048576,"mode":"1777"}]`
	ulimitsJSON := `[{"name":"memlock","soft":-1,"hard":-1}]`
	view, err := service.CreateVersion(ctx, versionTestUserId, applicationdto.VersionCreateInput{
		ApplicationId: app.Id,
		Label:         "runtime",
		Components: []applicationdto.VersionComponentInput{{
			Name: "web", Image: "nginx", RestartPolicy: &restartPolicy, TmpfsJSON: &tmpfsJSON, UlimitsJSON: &ulimitsJSON,
			SecretEnvRefs: []applicationdto.VersionComponentSecretEnvRefInput{{
				EnvKey: "MYSQL_PASSWORD", CredentialId: credentialId, DataKey: "MYSQL_PASSWORD",
			}},
		}},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(view.Components) != 1 {
		t.Fatalf("unexpected component count: %d", len(view.Components))
	}
	component := view.Components[0]
	if component.RestartPolicy == nil || *component.RestartPolicy != restartPolicy || component.TmpfsJSON == nil || component.UlimitsJSON == nil {
		t.Fatalf("runtime fields were not persisted: %+v", component)
	}
	if len(component.SecretEnvRefs) != 1 || component.SecretEnvRefs[0].CredentialId != credentialId {
		t.Fatalf("secret env ref was not persisted: %+v", component.SecretEnvRefs)
	}
}

func TestVersionRejectsUnsupportedRestartPolicy(t *testing.T) {
	service, database, applicationStore := newVersionIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	app, _ := createPublishedVersion(t, ctx, applicationStore, "invalid-restart")
	restartPolicy := "always"
	_, err := service.CreateVersion(ctx, versionTestUserId, applicationdto.VersionCreateInput{
		ApplicationId: app.Id,
		Label:         "invalid",
		Components:    []applicationdto.VersionComponentInput{{Name: "web", Image: "nginx", RestartPolicy: &restartPolicy}},
	})
	if err == nil {
		t.Fatal("expected unsupported restart policy to be rejected")
	}
}

func TestVersionClassifiesUnreadableRuntimeEnvCredentialAsInternal(t *testing.T) {
	service, database, applicationStore := newVersionIntegrationService(t)
	defer func() { _ = database.Close() }()
	ctx := context.Background()
	app, _ := createPublishedVersion(t, ctx, applicationStore, "unreadable-runtime-credential")

	credentialId := idutil.NewId()
	projectId := versionTestProjectId
	credentialStore := credentialrepo.NewRepository(database)
	if err := credentialStore.CreateCredential(ctx, model.Credential{
		Id: credentialId, ProjectId: &projectId, Name: "unreadable", Type: "runtime_env", EncryptedData: "not-a-valid-ciphertext",
	}); err != nil {
		t.Fatal(err)
	}
	service = NewWithCredential(projectrepo.NewRepository(database), applicationStore, credentialStore, runtimeVersionTestSecretKey)

	_, err := service.CreateVersion(ctx, versionTestUserId, applicationdto.VersionCreateInput{
		ApplicationId: app.Id,
		Label:         "runtime",
		Components: []applicationdto.VersionComponentInput{{
			Name: "web", Image: "nginx",
			SecretEnvRefs: []applicationdto.VersionComponentSecretEnvRefInput{{
				EnvKey: "MYSQL_PASSWORD", CredentialId: credentialId, DataKey: "MYSQL_PASSWORD",
			}},
		}},
	})
	if err == nil {
		t.Fatal("expected unreadable runtime credential to fail")
	}
	classification := apperror.Classify(err)
	if classification.StatusCode != 500 || classification.Code != "internal_error" || classification.Message != "Internal server error." {
		t.Fatalf("unexpected unreadable credential classification: %+v", classification)
	}
}

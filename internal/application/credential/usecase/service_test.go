package credentialsvc

import (
	"context"
	"testing"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

const (
	ciTestUserId    = "01KKX2YNPF6VJ9N7QYCWG61KVK"
	ciTestProjectId = "01KRRKK0K3T519ZQZES3M4QA9Z"
	ciTestSecretKey = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="
)

func TestCreateCredentialAcceptsGiteaTokenAndRejectsUnknownType(t *testing.T) {
	store := &credentialCreateStore{}
	service := New(credentialProjectFacts{}, store, nil, ciTestSecretKey)
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
	if len(store.saved) != 1 {
		t.Fatalf("invalid credential was saved: writes=%d", len(store.saved))
	}
}

type credentialProjectFacts struct{ repository.ProjectReader }

func (credentialProjectFacts) Project(_ context.Context, id string) (model.Project, error) {
	return model.Project{Id: id}, nil
}
func (credentialProjectFacts) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type credentialCreateStore struct {
	repository.CredentialStore
	saved []model.Credential
}

func (*credentialCreateStore) CredentialByName(context.Context, string) (model.Credential, error) {
	return model.Credential{}, repository.ErrNotFound
}
func (s *credentialCreateStore) CreateCredential(_ context.Context, credential model.Credential) error {
	s.saved = append(s.saved, credential)
	return nil
}
func (s *credentialCreateStore) Credential(_ context.Context, id string) (model.Credential, error) {
	for _, credential := range s.saved {
		if credential.Id == id {
			return credential, nil
		}
	}
	return model.Credential{}, repository.ErrNotFound
}

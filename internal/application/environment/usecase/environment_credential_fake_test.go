package environmentsvc

import (
	"context"
	"testing"
	"time"

	security "github.com/leoninew/pomelo-orbit/internal/common/crypto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

const testCredentialSecret = "AAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAAA="

type memoryEnvironmentCredentials struct {
	items []model.EnvironmentCredential
}

func (m *memoryEnvironmentCredentials) EnvironmentCredential(_ context.Context, id string) (model.EnvironmentCredential, error) {
	for _, item := range m.items {
		if item.Id == id {
			return item, nil
		}
	}
	return model.EnvironmentCredential{}, repository.ErrNotFound
}

func (m *memoryEnvironmentCredentials) LatestEnvironmentCredentialByProject(_ context.Context, projectId string) (model.EnvironmentCredential, error) {
	var found *model.EnvironmentCredential
	for i := range m.items {
		if m.items[i].ProjectId != projectId {
			continue
		}
		item := m.items[i]
		if found == nil || item.CreatedAt.After(found.CreatedAt) || (item.CreatedAt.Equal(found.CreatedAt) && item.Id > found.Id) {
			copyItem := item
			found = &copyItem
		}
	}
	if found == nil {
		return model.EnvironmentCredential{}, repository.ErrNotFound
	}
	return *found, nil
}

func credentialsForEnvironment(t *testing.T, environment model.Environment, privateKey string) *memoryEnvironmentCredentials {
	t.Helper()
	encrypted, err := security.EncryptString(testCredentialSecret, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	revision := int64(1)
	id := "credential-1"
	projectId := environment.ProjectId
	if environment.SSH != nil {
		id = environment.SSH.CredentialId
		revision = environment.SSH.CredentialRevision
	}
	return &memoryEnvironmentCredentials{items: []model.EnvironmentCredential{{
		Id: id, ProjectId: projectId, PublicKey: "ssh-ed25519 test",
		EncryptedPrivateKey: encrypted, Revision: revision,
	}}}
}

func (m *memoryEnvironmentCredentials) CreateEnvironmentCredential(_ context.Context, credential model.EnvironmentCredential) error {
	if credential.CreatedAt.IsZero() {
		credential.CreatedAt = time.Now().UTC()
	}
	m.items = append(m.items, credential)
	return nil
}

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

func (m *memoryEnvironmentCredentials) EnvironmentCredentialByEnvironment(_ context.Context, environmentId string) (model.EnvironmentCredential, error) {
	for _, item := range m.items {
		if item.EnvironmentId == environmentId {
			return item, nil
		}
	}
	return model.EnvironmentCredential{}, repository.ErrNotFound
}

func credentialsForEnvironment(t *testing.T, environment model.Environment, privateKey string) *memoryEnvironmentCredentials {
	t.Helper()
	encrypted, err := security.EncryptString(testCredentialSecret, privateKey)
	if err != nil {
		t.Fatal(err)
	}
	revision := int64(1)
	id := "credential-1"
	environmentId := environment.Id
	if environment.SSH != nil {
		id = environment.SSH.CredentialId
		revision = environment.SSH.CredentialRevision
	}
	return &memoryEnvironmentCredentials{items: []model.EnvironmentCredential{{
		Id: id, EnvironmentId: environmentId, PublicKey: "ssh-ed25519 test",
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

func (m *memoryEnvironmentCredentials) DeleteEnvironmentCredential(_ context.Context, environmentId string) error {
	for index := range m.items {
		if m.items[index].EnvironmentId != environmentId {
			continue
		}
		m.items = append(m.items[:index], m.items[index+1:]...)
		return nil
	}
	return nil
}

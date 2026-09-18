package environmentsvc

import (
	"context"
	"testing"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestSaveInitializationPersistsSSHWithoutCreatingCredential(t *testing.T) {
	projectID := "project-1"
	store := &initializationEnvironmentStore{}
	credentials := &memoryEnvironmentCredentials{}
	targetType := model.EnvironmentTargetTypeSSH

	view, err := New(store, initializationProjectReader{}, credentials, testCredentialSecret, nil, nil).
		SaveInitialization(context.Background(), "user-1", projectID, environmentdto.UpdateInput{
			TargetType: &targetType,
			SSH: &environmentdto.SSHTargetInput{
				Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22,
				Username: "deploy", WorkspaceRoot: "/srv/orbit",
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(credentials.items) != 0 || store.environment.SSH == nil || store.environment.SSH.CredentialId != "" {
		t.Fatalf("save created a credential: environment=%#v credentials=%#v", store.environment, credentials.items)
	}
	if view.SSH == nil || view.SSH.Host != "192.0.2.10" {
		t.Fatalf("saved environment = %#v", view)
	}
}

func TestPrepareSSHEnvironmentCreatesAndReusesEnvironmentCredential(t *testing.T) {
	projectID := "project-1"
	store := &initializationEnvironmentStore{}
	credentials := &memoryEnvironmentCredentials{}
	targetType := model.EnvironmentTargetTypeSSH
	service := New(store, initializationProjectReader{}, credentials, testCredentialSecret, nil, nil)
	first, publicKey, err := service.PrepareSSHEnvironment(context.Background(), "user-1", projectID, environmentdto.UpdateInput{
		TargetType: &targetType,
		SSH: &environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22,
			Username: "deploy", WorkspaceRoot: "/srv/orbit",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if publicKey == "" || len(credentials.items) != 1 || first.SSH == nil || first.SSH.Platform != model.EnvironmentPlatformLinux {
		t.Fatalf("first SSH command = view=%#v key=%q credentials=%#v", first, publicKey, credentials.items)
	}
	credentialID, revision := store.environment.SSH.CredentialId, store.environment.SSH.CredentialRevision
	store.environment.SSH.HostKeyFingerprint = "SHA256:abcdefghijklmnopqrstuvwxyz0123456789abcde="

	second, repeatedKey, err := service.PrepareSSHEnvironment(context.Background(), "user-1", projectID, environmentdto.UpdateInput{
		TargetType: &targetType,
		SSH: &environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.20", Port: 22,
			Username: "deploy", WorkspaceRoot: "/srv/orbit-next",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if repeatedKey != publicKey || len(credentials.items) != 1 || store.environment.SSH.CredentialId != credentialID || store.environment.SSH.CredentialRevision != revision {
		t.Fatalf("credential was not reused: environment=%#v credentials=%#v", store.environment, credentials.items)
	}
	if second.TargetRevision != first.TargetRevision+1 || store.environment.SSH.HostKeyFingerprint != "" {
		t.Fatalf("target change did not reset target state: %#v", store.environment)
	}
}

func TestPrepareSSHEnvironmentAcceptsWindowsTarget(t *testing.T) {
	projectID := "project-1"
	store := &initializationEnvironmentStore{}
	credentials := &memoryEnvironmentCredentials{}
	targetType := model.EnvironmentTargetTypeSSH
	view, publicKey, err := New(store, initializationProjectReader{}, credentials, testCredentialSecret, nil, nil).
		PrepareSSHEnvironment(context.Background(), "user-1", projectID, environmentdto.UpdateInput{
			TargetType: &targetType,
			SSH: &environmentdto.SSHTargetInput{
				Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 2222,
				Username: "orbit", WorkspaceRoot: `C:\\orbit`,
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if publicKey == "" || view.SSH == nil || view.SSH.Platform != model.EnvironmentPlatformWindows || view.SSH.Port != 2222 {
		t.Fatalf("prepared Windows environment = %#v", view)
	}
}

type initializationProjectReader struct{ probeProjectReader }

func (initializationProjectReader) Project(_ context.Context, projectID string) (model.Project, error) {
	return model.Project{Id: projectID, Code: "demo"}, nil
}

type initializationEnvironmentStore struct {
	repository.EnvironmentStore
	environment model.Environment
	found       bool
	created     bool
	updated     bool
}

func (s *initializationEnvironmentStore) EnvironmentByProject(context.Context, string) (model.Environment, error) {
	if !s.found {
		return model.Environment{}, repository.ErrNotFound
	}
	return s.environment, nil
}

func (s *initializationEnvironmentStore) CreateEnvironment(_ context.Context, environment model.Environment) error {
	s.created = true
	s.found = true
	s.environment = environment
	return nil
}

func (s *initializationEnvironmentStore) UpdateEnvironment(_ context.Context, environment model.Environment) error {
	s.updated = true
	s.environment = environment
	return nil
}

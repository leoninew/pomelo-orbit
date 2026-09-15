package environmentsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestSaveInitializationCreatesLocalEnvironment(t *testing.T) {
	store := &initializationEnvironmentStore{}
	targetType := model.EnvironmentTargetTypeLocal
	created, err := New(store, initializationProjectReader{}, nil, "", nil, nil).
		WithLocalDisplay(environmentdto.LocalDisplaySnapshot{Platform: model.EnvironmentPlatformLinux}).
		SaveInitialization(context.Background(), "user-1", "project-1", environmentdto.UpdateInput{
			TargetType: &targetType,
			Local:      &environmentdto.LocalTargetInput{WorkspaceRoot: "/srv/orbit"},
		})
	if err != nil {
		t.Fatal(err)
	}
	if !store.created || store.environment.TargetType != model.EnvironmentTargetTypeLocal || store.environment.WorkspaceRoot != "/srv/orbit" {
		t.Fatalf("stored environment = %#v", store.environment)
	}
	if created.Local == nil || created.Local.WorkspaceRoot != "/srv/orbit" || created.SSH != nil {
		t.Fatalf("created view = %#v", created)
	}
}

func TestSaveInitializationKeepsHomeWorkspaceRoot(t *testing.T) {
	store := &initializationEnvironmentStore{}
	targetType := model.EnvironmentTargetTypeLocal
	created, err := New(store, initializationProjectReader{}, nil, "", nil, nil).
		WithLocalDisplay(environmentdto.LocalDisplaySnapshot{Platform: model.EnvironmentPlatformLinux}).
		SaveInitialization(context.Background(), "user-1", "project-1", environmentdto.UpdateInput{
			TargetType: &targetType,
			Local:      &environmentdto.LocalTargetInput{WorkspaceRoot: "~/.pomelo-orbit"},
		})
	if err != nil {
		t.Fatal(err)
	}
	if store.environment.WorkspaceRoot != "~/.pomelo-orbit" {
		t.Fatalf("stored workspace root = %q", store.environment.WorkspaceRoot)
	}
	if created.Local == nil || created.Local.WorkspaceRoot != "~/.pomelo-orbit" {
		t.Fatalf("created view = %#v", created)
	}
}

func TestSaveInitializationUpdatesUnprobedLegacyEnvironmentWithGatewayBinding(t *testing.T) {
	gatewayId := "gateway-1"
	store := &initializationEnvironmentStore{
		found: true,
		environment: model.Environment{
			Id: "environment-1", ProjectId: "project-1", Code: "demo",
			State: model.EnvironmentStateActive, TargetType: model.EnvironmentTargetTypeLocal,
			TargetRevision: 1, GatewayApplicationId: &gatewayId,
		},
	}
	targetType := model.EnvironmentTargetTypeLocal
	_, err := New(store, initializationProjectReader{}, nil, "", nil, nil).
		WithLocalDisplay(environmentdto.LocalDisplaySnapshot{Platform: model.EnvironmentPlatformLinux}).
		SaveInitialization(context.Background(), "user-1", "project-1", environmentdto.UpdateInput{
			TargetType: &targetType,
			Local:      &environmentdto.LocalTargetInput{WorkspaceRoot: "/srv/orbit"},
		})
	if err != nil {
		t.Fatal(err)
	}
	if !store.updated || store.environment.GatewayApplicationId == nil || *store.environment.GatewayApplicationId != gatewayId {
		t.Fatalf("legacy gateway binding was not preserved: %#v", store.environment)
	}
}

func TestSaveInitializationCreatesSSHEnvironmentWithDeploymentCredential(t *testing.T) {
	store := &initializationEnvironmentStore{}
	credentials := &memoryEnvironmentCredentials{}
	targetType := model.EnvironmentTargetTypeSSH
	created, err := New(store, initializationProjectReader{}, credentials, testCredentialSecret, &reachableSSHProber{}, nil).
		SaveInitialization(context.Background(), "user-1", "project-1", environmentdto.UpdateInput{
			TargetType: &targetType,
			SSH: &environmentdto.SSHTargetInput{
				Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22,
				Username: "deploy", WorkspaceRoot: "/srv/orbit",
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if len(credentials.items) != 1 {
		t.Fatalf("environment credentials = %#v", credentials.items)
	}
	if !store.created || store.environment.SSH == nil || store.environment.SSH.CredentialId != credentials.items[0].Id {
		t.Fatalf("stored ssh environment = %#v", store.environment)
	}
	if created.SSH == nil || created.SSH.Host != "192.0.2.10" || created.Local != nil {
		t.Fatalf("created ssh view = %#v", created)
	}
}

func TestSaveInitializationWorkspaceChangeKeepsSSHIdentity(t *testing.T) {
	projectId := "project-1"
	probeRevision := int64(4)
	probeStatus := model.EnvironmentProbeStatusSucceeded
	environment := model.Environment{
		Id: "environment-1", ProjectId: projectId, Code: "demo", State: model.EnvironmentStateActive,
		TargetType: model.EnvironmentTargetTypeSSH, WorkspaceRoot: "/srv/orbit/previous",
		TargetRevision: probeRevision, LastProbeRevision: &probeRevision, LastProbeStatus: &probeStatus,
		SSH: &model.EnvironmentSSHTarget{
			Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22, Username: "deploy",
			CredentialId: "credential-1", CredentialRevision: 2,
			HostKeyFingerprint: "SHA256:abcdefghijklmnopqrstuvwxyz0123456789abcde=",
		},
	}
	store := &initializationEnvironmentStore{found: true, environment: environment}
	credentials := &memoryEnvironmentCredentials{items: []model.EnvironmentCredential{{
		Id: "credential-1", ProjectId: projectId, PublicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOrbit", Revision: 2,
	}}}
	targetType := model.EnvironmentTargetTypeSSH
	_, err := New(store, initializationProjectReader{}, credentials, testCredentialSecret, &reachableSSHProber{}, nil).
		SaveInitialization(context.Background(), "user-1", projectId, environmentdto.UpdateInput{
			TargetType: &targetType,
			SSH: &environmentdto.SSHTargetInput{
				Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22,
				Username: "deploy", WorkspaceRoot: "/srv/orbit/next",
			},
		})
	if err != nil {
		t.Fatal(err)
	}
	if !store.updated || store.environment.TargetRevision != probeRevision+1 || store.environment.WorkspaceRoot != "/srv/orbit/next" {
		t.Fatalf("stored environment = %#v", store.environment)
	}
	if store.environment.SSH == nil || store.environment.SSH.CredentialId != "credential-1" || store.environment.SSH.CredentialRevision != 2 || store.environment.SSH.HostKeyFingerprint != environment.SSH.HostKeyFingerprint {
		t.Fatalf("SSH identity was not preserved: %#v", store.environment.SSH)
	}
}

func TestSaveInitializationRejectsUnreachableSSH(t *testing.T) {
	store := &initializationEnvironmentStore{}
	targetType := model.EnvironmentTargetTypeSSH
	_, err := New(store, initializationProjectReader{}, &memoryEnvironmentCredentials{}, testCredentialSecret, &reachableSSHProber{err: errors.New("Cannot connect to the configured SSH host.")}, nil).
		SaveInitialization(context.Background(), "user-1", "project-1", environmentdto.UpdateInput{
			TargetType: &targetType,
			SSH: &environmentdto.SSHTargetInput{
				Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 22,
				Username: "orbit", WorkspaceRoot: `C:\orbit`,
			},
		})
	if err == nil || !strings.Contains(err.Error(), "Cannot connect to the configured SSH host") {
		t.Fatalf("error = %v", err)
	}
	if store.created {
		t.Fatal("unreachable SSH target must not be saved")
	}
}

func TestPrepareWindowsEnvironmentCreatesEnvironmentAndCredentialWithoutSSHReachability(t *testing.T) {
	projectId := "project-1"
	store := &initializationEnvironmentStore{}
	credentials := &memoryEnvironmentCredentials{}
	view, publicKey, err := New(store, initializationProjectReader{}, credentials, testCredentialSecret, &reachableSSHProber{err: errors.New("unreachable")}, nil).
		PrepareWindowsEnvironment(context.Background(), "user-1", projectId, environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 2222,
			Username: "orbit", WorkspaceRoot: `C:\\orbit`,
		})
	if err != nil {
		t.Fatal(err)
	}
	if publicKey == "" || len(credentials.items) != 1 {
		t.Fatalf("public key=%q credentials=%#v", publicKey, credentials.items)
	}
	if !store.created || store.environment.SSH == nil || store.environment.SSH.CredentialId != credentials.items[0].Id {
		t.Fatalf("stored environment = %#v", store.environment)
	}
	if view.SSH == nil || view.SSH.Platform != model.EnvironmentPlatformWindows || view.SSH.Port != 2222 {
		t.Fatalf("prepared view = %#v", view)
	}
}

func TestPrepareWindowsEnvironmentOverwritesCompletePair(t *testing.T) {
	projectId := "project-1"
	store := &initializationEnvironmentStore{
		found: true,
		environment: model.Environment{
			Id: "environment-1", ProjectId: projectId, Code: "demo", State: model.EnvironmentStateActive,
			TargetType: model.EnvironmentTargetTypeSSH, WorkspaceRoot: `C:\\orbit`, TargetRevision: 1,
			SSH: &model.EnvironmentSSHTarget{
				Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 22, Username: "orbit",
				CredentialId: "credential-1", CredentialRevision: 1,
			},
		},
	}
	credentials := &memoryEnvironmentCredentials{items: []model.EnvironmentCredential{{
		Id: "credential-1", ProjectId: projectId, PublicKey: "ssh-ed25519 old", EncryptedPrivateKey: "encrypted", Revision: 1,
	}}}
	view, publicKey, err := New(store, initializationProjectReader{}, credentials, testCredentialSecret, nil, nil).
		PrepareWindowsEnvironment(context.Background(), "user-1", projectId, environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 2222,
			Username: "orbit", WorkspaceRoot: `C:\\orbit-next`,
		})
	if err != nil {
		t.Fatal(err)
	}
	if publicKey == "" || publicKey == "ssh-ed25519 old" || credentials.items[0].EncryptedPrivateKey == "encrypted" || credentials.items[0].Revision != 2 {
		t.Fatalf("public key=%q credential=%#v", publicKey, credentials.items[0])
	}
	if !store.updated || store.environment.TargetRevision != 2 || store.environment.SSH == nil || store.environment.SSH.CredentialId != "credential-1" || store.environment.SSH.CredentialRevision != 2 {
		t.Fatalf("stored environment = %#v", store.environment)
	}
	if view.SSH == nil || view.SSH.WorkspaceRoot != `C:\\orbit-next` {
		t.Fatalf("prepared view = %#v", view)
	}
}

func TestPrepareWindowsEnvironmentRepairsLegacyCredentialBinding(t *testing.T) {
	projectId := "project-1"
	store := &initializationEnvironmentStore{
		found: true,
		environment: model.Environment{
			Id: "environment-1", ProjectId: projectId, Code: "demo", State: model.EnvironmentStateActive,
			TargetType: model.EnvironmentTargetTypeSSH, WorkspaceRoot: `C:\\orbit`, TargetRevision: 1,
			SSH: &model.EnvironmentSSHTarget{
				Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 22, Username: "orbit",
				CredentialId: "legacy-repository-credential", CredentialRevision: 1,
			},
		},
	}
	credentials := &memoryEnvironmentCredentials{}
	view, publicKey, err := New(store, initializationProjectReader{}, credentials, testCredentialSecret, nil, nil).
		PrepareWindowsEnvironment(context.Background(), "user-1", projectId, environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 22,
			Username: "orbit", WorkspaceRoot: `C:\\orbit`,
		})
	if err != nil {
		t.Fatal(err)
	}
	if publicKey == "" || len(credentials.items) != 1 || store.environment.SSH == nil || store.environment.SSH.CredentialId != credentials.items[0].Id {
		t.Fatalf("legacy repair = environment=%#v credentials=%#v", store.environment, credentials.items)
	}
	if store.environment.SSH.CredentialId == "legacy-repository-credential" || view.SSH == nil {
		t.Fatalf("legacy binding was not replaced: %#v", store.environment.SSH)
	}
}

func TestEnvironmentForUserLeavesLegacyCredentialBindingForWindowsPreparation(t *testing.T) {
	store := &initializationEnvironmentStore{
		found: true,
		environment: model.Environment{
			Id: "environment-1", ProjectId: "project-1", Code: "demo", State: model.EnvironmentStateActive,
			TargetType: model.EnvironmentTargetTypeSSH, WorkspaceRoot: `C:\\orbit`, TargetRevision: 1,
			SSH: &model.EnvironmentSSHTarget{
				Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 22, Username: "orbit",
				CredentialId: "legacy-repository-credential", CredentialRevision: 1,
			},
		},
	}
	view, err := New(store, initializationProjectReader{}, &memoryEnvironmentCredentials{}, testCredentialSecret, nil, nil).
		EnvironmentForUser(context.Background(), "user-1", "project-1")
	if err != nil {
		t.Fatal(err)
	}
	if view.SSH == nil || view.SSH.Host != "192.0.2.10" || store.updated {
		t.Fatalf("legacy environment view = %#v", view)
	}
}

type reachableSSHProber struct{ err error }

func (p *reachableSSHProber) Probe(context.Context, model.Environment, environmentdto.DeploymentSSHPrivateKey) (string, error) {
	return "", p.err
}

func (p *reachableSSHProber) ProbeLocal(context.Context, model.Environment) error {
	return p.err
}

func (p *reachableSSHProber) TestSSH(context.Context, string, int, string) error {
	return p.err
}

type initializationProjectReader struct{ probeProjectReader }

func (initializationProjectReader) Project(_ context.Context, projectId string) (model.Project, error) {
	return model.Project{Id: projectId, Code: "demo"}, nil
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

package environmentsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestSaveInitializationCreatesLocalEnvironment(t *testing.T) {
	store := &initializationEnvironmentStore{}
	targetType := model.EnvironmentTargetTypeLocal
	created, err := New(store, initializationProjectReader{}, nil, nil, nil, nil).
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
	created, err := New(store, initializationProjectReader{}, nil, nil, nil, nil).
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
	gatewayID := "gateway-1"
	store := &initializationEnvironmentStore{
		found: true,
		environment: model.Environment{
			Id: "environment-1", ProjectId: "project-1", Code: "demo",
			State: model.EnvironmentStateActive, TargetType: model.EnvironmentTargetTypeLocal,
			TargetRevision: 1, GatewayApplicationId: &gatewayID,
		},
	}
	targetType := model.EnvironmentTargetTypeLocal
	_, err := New(store, initializationProjectReader{}, nil, nil, nil, nil).
		WithLocalDisplay(environmentdto.LocalDisplaySnapshot{Platform: model.EnvironmentPlatformLinux}).
		SaveInitialization(context.Background(), "user-1", "project-1", environmentdto.UpdateInput{
			TargetType: &targetType,
			Local:      &environmentdto.LocalTargetInput{WorkspaceRoot: "/srv/orbit"},
		})
	if err != nil {
		t.Fatal(err)
	}
	if !store.updated || store.environment.GatewayApplicationId == nil || *store.environment.GatewayApplicationId != gatewayID {
		t.Fatalf("legacy gateway binding was not preserved: %#v", store.environment)
	}
}

func TestSaveInitializationCreatesSSHEnvironmentWithDeploymentCredential(t *testing.T) {
	store := &initializationEnvironmentStore{}
	keyManager := &createInitializationKeyManager{}
	targetType := model.EnvironmentTargetTypeSSH
	created, err := New(store, initializationProjectReader{}, keyManager, nil, &reachableSSHProber{}, nil).
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
	if keyManager.createCalls != 1 {
		t.Fatalf("credential create calls = %d", keyManager.createCalls)
	}
	if !store.created || store.environment.SSH == nil || store.environment.SSH.CredentialId != "credential-new" {
		t.Fatalf("stored ssh environment = %#v", store.environment)
	}
	if created.SSH == nil || created.SSH.Host != "192.0.2.10" || created.Local != nil {
		t.Fatalf("created ssh view = %#v", created)
	}
}

func TestSaveInitializationRejectsUnreachableSSH(t *testing.T) {
	store := &initializationEnvironmentStore{}
	targetType := model.EnvironmentTargetTypeSSH
	_, err := New(store, initializationProjectReader{}, &createInitializationKeyManager{}, nil, &reachableSSHProber{err: errors.New("Cannot connect to the configured SSH host.")}, nil).
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

func TestDeploymentSSHPublicKeyForProjectReturnsBoundKey(t *testing.T) {
	projectID := "project-1"
	store := &initializationEnvironmentStore{
		found: true,
		environment: model.Environment{
			Id: "environment-1", ProjectId: projectID, Code: "demo", State: model.EnvironmentStateActive,
			TargetType: model.EnvironmentTargetTypeSSH, WorkspaceRoot: `C:\\orbit`,
			SSH: &model.EnvironmentSSHTarget{
				Platform: model.EnvironmentPlatformWindows, Host: "192.0.2.10", Port: 22, Username: "orbit",
				CredentialId: "credential-1", CredentialRevision: 1,
			},
		},
	}
	keyManager := &createInitializationKeyManager{publicKey: " ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOrbitDeploymentKey\n"}
	publicKey, err := New(store, initializationProjectReader{}, keyManager, nil, nil, nil).
		DeploymentSSHPublicKeyForProject(context.Background(), "user-1", projectID)
	if err != nil {
		t.Fatal(err)
	}
	if publicKey != "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOrbitDeploymentKey" {
		t.Fatalf("public key = %q", publicKey)
	}
}

func TestDeploymentSSHPublicKeyForProjectCreatesKeyBeforeEnvironmentSave(t *testing.T) {
	projectID := "project-1"
	store := &initializationEnvironmentStore{}
	keyManager := &createInitializationKeyManager{publicKey: "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOrbitDeploymentKey"}
	publicKey, err := New(store, initializationProjectReader{}, keyManager, nil, nil, nil).
		DeploymentSSHPublicKeyForProject(context.Background(), "user-1", projectID)
	if err != nil {
		t.Fatal(err)
	}
	if publicKey != keyManager.publicKey || keyManager.createCalls != 1 {
		t.Fatalf("public key=%q create calls=%d", publicKey, keyManager.createCalls)
	}
}

type reachableSSHProber struct{ err error }

func (p *reachableSSHProber) Probe(context.Context, model.Environment, credentialdto.DeploymentSSHPrivateKey) (string, error) {
	return "", p.err
}

func (p *reachableSSHProber) ProbeLocal(context.Context, model.Environment) error {
	return p.err
}

func (p *reachableSSHProber) TestSSH(context.Context, string, int, string) error {
	return p.err
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

type createInitializationKeyManager struct {
	createCalls int
	publicKey   string
}

func (m *createInitializationKeyManager) CreateDeploymentSSHCredential(_ context.Context, projectID string, _ string) (model.Credential, error) {
	m.createCalls++
	return model.Credential{
		Id: "credential-new", ProjectId: &projectID, Type: model.CredentialTypeDeploymentSSHPrivateKey, Revision: 1,
	}, nil
}

func (m *createInitializationKeyManager) EnsureGeneratedDeploymentSSHCredential(context.Context, string) (model.Credential, error) {
	return model.Credential{}, errors.New("existing credential must not be ensured on create")
}

func (m *createInitializationKeyManager) DeploymentSSHPublicKey(context.Context, string) (string, error) {
	if m.publicKey == "" {
		return "", errors.New("public key must not be read on save")
	}
	return m.publicKey, nil
}

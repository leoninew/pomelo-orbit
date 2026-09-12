package environmentsvc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestProbeForUserRecordsSuccessfulProbeForCurrentTargetRevision(t *testing.T) {
	projectID := "project-1"
	environment := testProbeEnvironment(projectID)
	store := &probeEnvironmentStore{environment: environment}
	privateKey := credentialdto.DeploymentSSHPrivateKey{PrivateKey: "private-key"}
	prober := &probeEnvironmentProber{}
	service := New(store, probeProjectReader{}, nil, probeCredentialReader{
		credential: testProbeCredential(projectID, environment),
		privateKey: privateKey,
	}, prober, nil)

	result, err := service.ProbeForUser(context.Background(), "user-1", projectID)
	if err != nil {
		t.Fatal(err)
	}
	if !prober.called || prober.environment.Id != environment.Id || prober.privateKey != privateKey {
		t.Fatalf("probe invocation = %#v", prober)
	}
	if !store.recorded || store.targetRevision != environment.TargetRevision || store.status != model.EnvironmentProbeStatusSucceeded {
		t.Fatalf("recorded probe = %#v", store)
	}
	if result.LastProbeStatus == nil || *result.LastProbeStatus != model.EnvironmentProbeStatusSucceeded || result.LastProbeRevision == nil || *result.LastProbeRevision != environment.TargetRevision {
		t.Fatalf("probe result = %#v", result)
	}
	if result.LastProbeDiagnostic == nil || *result.LastProbeDiagnostic != "Linux SSH and Docker Compose prerequisites are ready." {
		t.Fatalf("probe diagnostic = %#v", result.LastProbeDiagnostic)
	}
}

func TestProbeForUserRecordsSanitizedFailure(t *testing.T) {
	projectID := "project-1"
	environment := testProbeEnvironment(projectID)
	store := &probeEnvironmentStore{environment: environment}
	secret := "private-key-must-not-appear"
	service := New(store, probeProjectReader{}, nil, probeCredentialReader{
		credential: testProbeCredential(projectID, environment),
		privateKey: credentialdto.DeploymentSSHPrivateKey{PrivateKey: secret},
	}, &probeEnvironmentProber{err: errors.New(secret)}, nil)

	result, err := service.ProbeForUser(context.Background(), "user-1", projectID)
	if err != nil {
		t.Fatal(err)
	}
	if !store.recorded || store.status != model.EnvironmentProbeStatusFailed || store.diagnostic != probeFailureDiagnostic {
		t.Fatalf("recorded probe = %#v", store)
	}
	if result.LastProbeDiagnostic == nil || strings.Contains(*result.LastProbeDiagnostic, secret) {
		t.Fatalf("probe diagnostic leaks secret: %#v", result.LastProbeDiagnostic)
	}
}

func TestProbeForUserProbesLocalEnvironmentWithoutDeploymentCredential(t *testing.T) {
	projectID := "project-1"
	environment := model.Environment{
		Id: "environment-local", ProjectId: projectID, State: model.EnvironmentStateActive,
		TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: "/srv/pomelo-orbit", TargetRevision: 1,
	}
	store := &probeEnvironmentStore{environment: environment}
	prober := &localProbeEnvironmentProber{}
	service := New(store, probeProjectReader{}, nil, nil, prober, nil)

	result, err := service.ProbeForUser(context.Background(), "user-1", projectID)
	if err != nil {
		t.Fatal(err)
	}
	if !prober.localCalled || store.status != model.EnvironmentProbeStatusSucceeded {
		t.Fatalf("local probe = %#v store=%#v", prober, store)
	}
	if result.LastProbeDiagnostic == nil || *result.LastProbeDiagnostic != "Local Docker and Docker Compose prerequisites are ready." {
		t.Fatalf("local probe result = %#v", result)
	}
}

func TestProbeForUserRecordsSafeRunnerDiagnostic(t *testing.T) {
	projectID := "project-1"
	environment := testProbeEnvironment(projectID)
	store := &probeEnvironmentStore{environment: environment}
	service := New(store, probeProjectReader{}, nil, probeCredentialReader{
		credential: testProbeCredential(projectID, environment),
		privateKey: credentialdto.DeploymentSSHPrivateKey{PrivateKey: "private-key"},
	}, &probeEnvironmentProber{err: testProbeDiagnosticError("SSH key authentication failed for the configured user.")}, nil)

	result, err := service.ProbeForUser(context.Background(), "user-1", projectID)
	if err != nil {
		t.Fatal(err)
	}
	if result.LastProbeDiagnostic == nil || *result.LastProbeDiagnostic != "SSH key authentication failed for the configured user." {
		t.Fatalf("probe diagnostic = %#v", result.LastProbeDiagnostic)
	}
}

type testProbeDiagnosticError string

func (e testProbeDiagnosticError) Error() string {
	return string(e)
}

func (e testProbeDiagnosticError) ProbeDiagnostic() string {
	return string(e)
}

func TestProbeForUserRecordsHostKeyFingerprintOnFirstSuccess(t *testing.T) {
	projectID := "project-1"
	environment := testProbeEnvironment(projectID)
	environment.SSH.HostKeyFingerprint = ""
	store := &probeEnvironmentStore{environment: environment}
	fingerprint := "SHA256:recordedhostkeyfingerprintvalueabcdefghijk="
	service := New(store, probeProjectReader{}, nil, probeCredentialReader{
		credential: testProbeCredential(projectID, environment),
		privateKey: credentialdto.DeploymentSSHPrivateKey{PrivateKey: "private-key"},
	}, &probeEnvironmentProber{fingerprint: fingerprint}, nil)

	result, err := service.ProbeForUser(context.Background(), "user-1", projectID)
	if err != nil {
		t.Fatal(err)
	}
	if !store.updated || store.environment.SSH.HostKeyFingerprint != fingerprint {
		t.Fatalf("recorded fingerprint = %#v", store)
	}
	if result.SSH.HostKeyFingerprint != fingerprint || result.TargetRevision != environment.TargetRevision {
		t.Fatalf("probe result = %#v", result)
	}
	if !store.recorded || store.status != model.EnvironmentProbeStatusSucceeded {
		t.Fatalf("recorded probe = %#v", store)
	}
}

func TestUpdateForUserWorkspaceChangeKeepsSSHIdentity(t *testing.T) {
	projectID := "project-1"
	environment := testProbeEnvironment(projectID)
	environment.Code = "demo"
	store := &updateEnvironmentStore{environment: environment}
	credential := testProbeCredential(projectID, environment)
	keyManager := &updateDeploymentKeyManager{credential: credential}
	targetType := model.EnvironmentTargetTypeSSH
	service := New(store, probeProjectReader{}, keyManager, probeCredentialReader{credential: credential}, nil, nil)

	updated, err := service.UpdateForUser(context.Background(), "user-1", projectID, environmentdto.UpdateInput{
		TargetType: &targetType,
		SSH: &environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformLinux, Host: environment.SSH.Host, Port: environment.SSH.Port,
			Username: environment.SSH.Username, WorkspaceRoot: "/srv/pomelo-orbit/next",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if !store.updated || store.environment.TargetRevision != environment.TargetRevision+1 || store.environment.WorkspaceRoot != "/srv/pomelo-orbit/next" {
		t.Fatalf("stored environment = %#v", store.environment)
	}
	if store.environment.SSH == nil || store.environment.SSH.HostKeyFingerprint != environment.SSH.HostKeyFingerprint || store.environment.SSH.CredentialId != environment.SSH.CredentialId {
		t.Fatalf("SSH identity was not preserved: %#v", store.environment.SSH)
	}
	if updated.SSH == nil || updated.SSH.HostKeyFingerprint != environment.SSH.HostKeyFingerprint {
		t.Fatalf("updated view = %#v", updated)
	}
}

func TestProbeForUserRejectsDisabledEnvironmentWithoutSSH(t *testing.T) {
	environment := testProbeEnvironment("project-1")
	environment.State = model.EnvironmentStateDisabled
	store := &probeEnvironmentStore{environment: environment}
	prober := &probeEnvironmentProber{}
	service := New(store, probeProjectReader{}, nil, probeCredentialReader{
		credential: testProbeCredential(environment.ProjectId, environment),
	}, prober, nil)

	_, err := service.ProbeForUser(context.Background(), "user-1", environment.ProjectId)
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) {
		t.Fatalf("ProbeForUser error = %v", err)
	}
	if prober.called || store.recorded {
		t.Fatalf("disabled environment performed probe: prober=%#v store=%#v", prober, store)
	}
}

func TestUpdateForUserRejectsActivationWithoutConfiguredDeploymentCredential(t *testing.T) {
	projectID := "project-1"
	environment := testProbeEnvironment(projectID)
	environment.State = model.EnvironmentStateDisabled
	store := &updateEnvironmentStore{environment: environment}
	active := model.EnvironmentStateActive
	service := New(store, probeProjectReader{}, nil, probeCredentialReader{err: errors.New("credential requires reconfiguration")}, nil, nil)

	_, err := service.UpdateForUser(context.Background(), "user-1", projectID, environmentdto.UpdateInput{State: &active})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) || !strings.Contains(err.Error(), "must be configured") {
		t.Fatalf("UpdateForUser error = %v", err)
	}
	if store.updated {
		t.Fatal("environment update must not be persisted without a configured deployment SSH credential")
	}
}

func TestUpdateForUserKeepsDeploymentCredentialForSSHTargetChange(t *testing.T) {
	projectID := "project-1"
	environment := testProbeEnvironment(projectID)
	environment.Code = "environment"
	store := &updateEnvironmentStore{environment: environment}
	credential := testProbeCredential(projectID, environment)
	keyManager := &updateDeploymentKeyManager{credential: credential}
	targetType := model.EnvironmentTargetTypeSSH
	service := New(
		store,
		probeProjectReader{},
		keyManager,
		probeCredentialReader{credential: credential},
		nil,
		nil,
	)

	updated, err := service.UpdateForUser(context.Background(), "user-1", projectID, environmentdto.UpdateInput{
		TargetType: &targetType,
		SSH: &environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformLinux, Host: "198.51.100.10", Port: 22,
			Username: "deploy", WorkspaceRoot: "/srv/pomelo-orbit",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if keyManager.createCalls != 0 || keyManager.ensuredCredentialID != environment.SSH.CredentialId {
		t.Fatalf("credential calls = %#v", keyManager)
	}
	if !store.updated || updated.SSH == nil || updated.SSH.Host != "198.51.100.10" {
		t.Fatalf("updated environment = %#v", updated)
	}
	if store.environment.SSH == nil || store.environment.SSH.CredentialId != environment.SSH.CredentialId || store.environment.SSH.CredentialRevision != environment.SSH.CredentialRevision {
		t.Fatalf("stored credential binding = %#v", store.environment.SSH)
	}
	if store.environment.SSH.HostKeyFingerprint != "" {
		t.Fatalf("SSH target change retained host key fingerprint: %#v", store.environment.SSH)
	}
}

func TestUpdateForUserChangesLocalWorkspaceAndInvalidatesProbeFreshness(t *testing.T) {
	projectID := "project-1"
	revision := int64(5)
	probeStatus := model.EnvironmentProbeStatusSucceeded
	environment := model.Environment{
		Id: "environment-local", ProjectId: projectID, Code: "project", State: model.EnvironmentStateActive,
		TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: "/srv/orbit/previous", TargetRevision: revision,
		LastProbeRevision: &revision, LastProbeStatus: &probeStatus,
	}
	store := &updateEnvironmentStore{environment: environment}
	targetType := model.EnvironmentTargetTypeLocal
	updated, err := New(store, probeProjectReader{}, nil, nil, nil, nil).
		WithLocalDisplay(environmentdto.LocalDisplaySnapshot{Platform: model.EnvironmentPlatformLinux}).
		UpdateForUser(context.Background(), "user-1", projectID, environmentdto.UpdateInput{
			TargetType: &targetType,
			Local:      &environmentdto.LocalTargetInput{WorkspaceRoot: "/srv/orbit/next"},
		})
	if err != nil {
		t.Fatal(err)
	}
	if !store.updated || store.environment.WorkspaceRoot != "/srv/orbit/next" || store.environment.TargetRevision != revision+1 {
		t.Fatalf("stored environment = %#v", store.environment)
	}
	if store.environment.HasFreshSuccessfulProbe() {
		t.Fatalf("workspace update retained fresh probe: %#v", store.environment)
	}
	if updated.Local == nil || updated.Local.WorkspaceRoot != "/srv/orbit/next" {
		t.Fatalf("updated view = %#v", updated)
	}
}

func TestProbeForUserRejectsStaleResult(t *testing.T) {
	environment := testProbeEnvironment("project-1")
	store := &probeEnvironmentStore{environment: environment, stale: true}
	service := New(store, probeProjectReader{}, nil, probeCredentialReader{
		credential: testProbeCredential(environment.ProjectId, environment),
	}, &probeEnvironmentProber{}, nil)

	_, err := service.ProbeForUser(context.Background(), "user-1", environment.ProjectId)
	if err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("ProbeForUser error = %v", err)
	}
}

func TestInitializeForUserBootstrapsWithEphemeralPasswordAndProbesGeneratedKey(t *testing.T) {
	projectID := "project-1"
	environment := testProbeEnvironment(projectID)
	environment.SSH.HostKeyFingerprint = ""
	store := &probeEnvironmentStore{environment: environment}
	credential := testProbeCredential(projectID, environment)
	keyManager := &updateDeploymentKeyManager{credential: credential}
	prober := &probeEnvironmentProber{}
	bootstrapper := &testEnvironmentBootstrapper{}
	service := New(
		store,
		probeProjectReader{},
		keyManager,
		probeCredentialReader{credential: credential, privateKey: credentialdto.DeploymentSSHPrivateKey{PrivateKey: "generated-private-key"}},
		prober,
		bootstrapper,
	)

	result, err := service.InitializeForUser(context.Background(), "user-1", projectID, environmentdto.InitializeInput{
		Username: "opc", Password: "bootstrap-password",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bootstrapper.called || bootstrapper.environment.Id != environment.Id || bootstrapper.publicKey == "" || bootstrapper.auth.Username != "opc" || bootstrapper.auth.Password != "bootstrap-password" || bootstrapper.auth.PrivateKey != "" {
		t.Fatalf("bootstrap invocation = %#v", bootstrapper)
	}
	if !prober.called || !store.recorded || result.LastProbeStatus == nil || *result.LastProbeStatus != model.EnvironmentProbeStatusSucceeded {
		t.Fatalf("initialization result = %#v probe=%#v store=%#v", result, prober, store)
	}
}

func TestInitializeForUserRejectsUnsupportedTargetAndAmbiguousCredentials(t *testing.T) {
	projectID := "project-1"
	environment := testProbeEnvironment(projectID)
	environment.SSH.Platform = model.EnvironmentPlatformWindows
	bootstrapper := &testEnvironmentBootstrapper{}
	service := New(
		&probeEnvironmentStore{environment: environment},
		probeProjectReader{},
		&updateDeploymentKeyManager{credential: testProbeCredential(projectID, environment)},
		probeCredentialReader{credential: testProbeCredential(projectID, environment)},
		&probeEnvironmentProber{},
		bootstrapper,
	)
	_, err := service.InitializeForUser(context.Background(), "user-1", projectID, environmentdto.InitializeInput{
		Username: "administrator", Password: "password", PrivateKey: "private-key",
	})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) || bootstrapper.called {
		t.Fatalf("InitializeForUser() error = %v bootstrap=%#v", err, bootstrapper)
	}

	environment.SSH.Platform = model.EnvironmentPlatformLinux
	service = New(
		&probeEnvironmentStore{environment: environment},
		probeProjectReader{},
		&updateDeploymentKeyManager{credential: testProbeCredential(projectID, environment)},
		probeCredentialReader{credential: testProbeCredential(projectID, environment)},
		&probeEnvironmentProber{},
		bootstrapper,
	)
	_, err = service.InitializeForUser(context.Background(), "user-1", projectID, environmentdto.InitializeInput{
		Username: "opc", Password: "password", PrivateKey: "private-key",
	})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) || bootstrapper.called {
		t.Fatalf("InitializeForUser() ambiguous auth error = %v bootstrap=%#v", err, bootstrapper)
	}
}

func TestInitializeForUserDoesNotExposeBootstrapSecret(t *testing.T) {
	projectID := "project-1"
	environment := testProbeEnvironment(projectID)
	secret := "bootstrap-private-key-must-not-appear"
	service := New(
		&probeEnvironmentStore{environment: environment},
		probeProjectReader{},
		&updateDeploymentKeyManager{credential: testProbeCredential(projectID, environment)},
		probeCredentialReader{credential: testProbeCredential(projectID, environment)},
		&probeEnvironmentProber{},
		&testEnvironmentBootstrapper{err: errors.New(secret)},
	)
	_, err := service.InitializeForUser(context.Background(), "user-1", projectID, environmentdto.InitializeInput{
		Username: "opc", PrivateKey: secret,
	})
	if err == nil || strings.Contains(err.Error(), secret) {
		t.Fatalf("bootstrap error leaks secret: %v", err)
	}
}

type updateEnvironmentStore struct {
	repository.EnvironmentStore
	environment model.Environment
	updated     bool
}

func (s *updateEnvironmentStore) EnvironmentByProject(context.Context, string) (model.Environment, error) {
	return s.environment, nil
}

func (s *updateEnvironmentStore) UpdateEnvironment(_ context.Context, environment model.Environment) error {
	s.updated = true
	s.environment = environment
	return nil
}

type updateDeploymentKeyManager struct {
	credential          model.Credential
	createCalls         int
	ensuredCredentialID string
}

func (m *updateDeploymentKeyManager) CreateDeploymentSSHCredential(context.Context, string, string) (model.Credential, error) {
	m.createCalls++
	return model.Credential{}, errors.New("deployment credential creation must not be called")
}

func (m *updateDeploymentKeyManager) EnsureGeneratedDeploymentSSHCredential(_ context.Context, credentialID string) (model.Credential, error) {
	m.ensuredCredentialID = credentialID
	return m.credential, nil
}

func (m *updateDeploymentKeyManager) DeploymentSSHPublicKey(context.Context, string) (string, error) {
	return "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIExample comment", nil
}

type probeEnvironmentStore struct {
	repository.EnvironmentStore
	environment    model.Environment
	recorded       bool
	updated        bool
	stale          bool
	targetRevision int64
	status         string
	diagnostic     string
}

func (s *probeEnvironmentStore) EnvironmentByProject(context.Context, string) (model.Environment, error) {
	return s.environment, nil
}

func (s *probeEnvironmentStore) UpdateEnvironment(_ context.Context, environment model.Environment) error {
	s.updated = true
	s.environment = environment
	return nil
}

func (s *probeEnvironmentStore) RecordProbe(_ context.Context, _ string, targetRevision int64, status string, _ time.Time, diagnostic string) (bool, error) {
	s.recorded = true
	s.targetRevision = targetRevision
	s.status = status
	s.diagnostic = diagnostic
	return !s.stale, nil
}

type probeProjectReader struct{ repository.ProjectReader }

func (probeProjectReader) Project(_ context.Context, projectID string) (model.Project, error) {
	return model.Project{Id: projectID}, nil
}

func (probeProjectReader) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type probeCredentialReader struct {
	credential model.Credential
	privateKey credentialdto.DeploymentSSHPrivateKey
	err        error
}

func (s probeCredentialReader) DeploymentSSHCredential(context.Context, string) (model.Credential, credentialdto.DeploymentSSHPrivateKey, error) {
	return s.credential, s.privateKey, s.err
}

type probeEnvironmentProber struct {
	called      bool
	environment model.Environment
	privateKey  credentialdto.DeploymentSSHPrivateKey
	fingerprint string
	err         error
}

type testEnvironmentBootstrapper struct {
	called      bool
	environment model.Environment
	publicKey   string
	auth        environmentport.BootstrapAuth
	err         error
}

func (b *testEnvironmentBootstrapper) Bootstrap(_ context.Context, environment model.Environment, publicKey string, auth environmentport.BootstrapAuth) error {
	b.called = true
	b.environment = environment
	b.publicKey = publicKey
	b.auth = auth
	return b.err
}

type localProbeEnvironmentProber struct {
	localCalled bool
	err         error
}

func (p *localProbeEnvironmentProber) Probe(context.Context, model.Environment, credentialdto.DeploymentSSHPrivateKey) (string, error) {
	return "", p.err
}

func (p *localProbeEnvironmentProber) ProbeLocal(context.Context, model.Environment) error {
	p.localCalled = true
	return p.err
}

func (p *localProbeEnvironmentProber) TestSSH(context.Context, string, int, string) error {
	return p.err
}

func (p *probeEnvironmentProber) ProbeLocal(context.Context, model.Environment) error {
	return p.err
}

func (p *probeEnvironmentProber) TestSSH(context.Context, string, int, string) error {
	return p.err
}

func (p *probeEnvironmentProber) Probe(_ context.Context, environment model.Environment, privateKey credentialdto.DeploymentSSHPrivateKey) (string, error) {
	p.called = true
	p.environment = environment
	p.privateKey = privateKey
	if p.err != nil {
		return "", p.err
	}
	if p.fingerprint != "" {
		return p.fingerprint, nil
	}
	return "SHA256:abcdefghijklmnopqrstuvwxyz0123456789abcde=", nil
}

func testProbeEnvironment(projectID string) model.Environment {
	return model.Environment{
		Id:             "environment-1",
		ProjectId:      projectID,
		State:          model.EnvironmentStateActive,
		TargetType:     model.EnvironmentTargetTypeSSH,
		WorkspaceRoot:  "/srv/pomelo-orbit",
		TargetRevision: 7,
		SSH: &model.EnvironmentSSHTarget{
			Platform: model.EnvironmentPlatformLinux, Host: "192.0.2.10", Port: 22,
			Username:     "deploy",
			CredentialId: "credential-1", CredentialRevision: 3,
			HostKeyFingerprint: "SHA256:abcdefghijklmnopqrstuvwxyz0123456789abcde=",
		},
	}
}

func testProbeCredential(projectID string, environment model.Environment) model.Credential {
	return model.Credential{
		Id:        environment.SSH.CredentialId,
		ProjectId: &projectID,
		Type:      model.CredentialTypeDeploymentSSHPrivateKey,
		Revision:  environment.SSH.CredentialRevision,
	}
}

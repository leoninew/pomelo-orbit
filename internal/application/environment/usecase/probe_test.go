package environmentsvc

import (
	"bytes"
	"context"
	"errors"
	"log/slog"
	"strings"
	"testing"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestProbeForUserRecordsSuccessfulProbeForCurrentTargetRevision(t *testing.T) {
	projectId := "project-1"
	environment := testProbeEnvironment(projectId)
	store := &probeEnvironmentStore{environment: environment}
	privateKey := environmentdto.DeploymentSSHPrivateKey{PrivateKey: "private-key"}
	prober := &probeEnvironmentProber{}
	service := New(store, probeProjectReader{}, credentialsForEnvironment(t, environment, privateKey.PrivateKey), testCredentialSecret, prober, nil, nil)

	result, err := service.ProbeForUser(context.Background(), "user-1", projectId)
	if err != nil {
		t.Fatal(err)
	}
	if !prober.called || prober.environment.Id != environment.Id || prober.privateKey.PrivateKey != privateKey.PrivateKey {
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
	projectId := "project-1"
	environment := testProbeEnvironment(projectId)
	store := &probeEnvironmentStore{environment: environment}
	secret := "private-key-must-not-appear"
	service := New(store, probeProjectReader{}, credentialsForEnvironment(t, environment, secret), testCredentialSecret, &probeEnvironmentProber{err: errors.New(secret)}, nil, nil)

	result, err := service.ProbeForUser(context.Background(), "user-1", projectId)
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
	projectId := "project-1"
	environment := model.Environment{
		Id: "environment-local", ProjectId: projectId,
		TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: "/srv/pomelo-orbit", TargetRevision: 1,
	}
	store := &probeEnvironmentStore{environment: environment}
	prober := &localProbeEnvironmentProber{}
	service := New(store, probeProjectReader{}, nil, "", prober, nil, nil)

	result, err := service.ProbeForUser(context.Background(), "user-1", projectId)
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

func TestProbeForUserRecordsAndLogsRunnerDiagnostic(t *testing.T) {
	projectId := "project-1"
	environment := testProbeEnvironment(projectId)
	store := &probeEnvironmentStore{environment: environment}
	diagnostic := "SSH key authentication failed for the configured user: ssh: handshake failed: ssh: unable to authenticate, attempted methods [none publickey], no supported methods remain"
	var logBuffer bytes.Buffer
	logger := slog.New(slog.NewTextHandler(&logBuffer, nil))
	service := New(store, probeProjectReader{}, credentialsForEnvironment(t, environment, "private-key"), testCredentialSecret, &probeEnvironmentProber{err: testProbeDiagnosticError(diagnostic)}, nil, logger)

	result, err := service.ProbeForUser(context.Background(), "user-1", projectId)
	if err != nil {
		t.Fatal(err)
	}
	if result.LastProbeDiagnostic == nil || *result.LastProbeDiagnostic != diagnostic {
		t.Fatalf("probe diagnostic = %#v", result.LastProbeDiagnostic)
	}
	logged := logBuffer.String()
	if !strings.Contains(logged, "project environment probe failed") || !strings.Contains(logged, diagnostic) || !strings.Contains(logged, "project_id=project-1") || !strings.Contains(logged, "environment_id=environment-1") {
		t.Fatalf("probe failure log = %q", logged)
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
	projectId := "project-1"
	environment := testProbeEnvironment(projectId)
	environment.SSH.HostKeyFingerprint = ""
	store := &probeEnvironmentStore{environment: environment}
	fingerprint := "SHA256:recordedhostkeyfingerprintvalueabcdefghijk="
	service := New(store, probeProjectReader{}, credentialsForEnvironment(t, environment, "private-key"), testCredentialSecret, &probeEnvironmentProber{fingerprint: fingerprint}, nil, nil)

	result, err := service.ProbeForUser(context.Background(), "user-1", projectId)
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
	projectId := "project-1"
	environment := testProbeEnvironment(projectId)
	environment.Code = "demo"
	store := &updateEnvironmentStore{environment: environment}
	targetType := model.EnvironmentTargetTypeSSH
	service := New(store, probeProjectReader{}, credentialsForEnvironment(t, environment, "private-key"), testCredentialSecret, nil, nil, nil)

	updated, err := service.UpdateForUser(context.Background(), "user-1", projectId, environmentdto.UpdateInput{
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

func TestUpdateForUserKeepsDeploymentCredentialForSSHTargetChange(t *testing.T) {
	projectId := "project-1"
	environment := testProbeEnvironment(projectId)
	environment.Code = "environment"
	store := &updateEnvironmentStore{environment: environment}
	targetType := model.EnvironmentTargetTypeSSH
	service := New(
		store,
		probeProjectReader{},
		credentialsForEnvironment(t, environment, "private-key"),
		testCredentialSecret,
		nil,
		nil,
		nil,
	)

	updated, err := service.UpdateForUser(context.Background(), "user-1", projectId, environmentdto.UpdateInput{
		TargetType: &targetType,
		SSH: &environmentdto.SSHTargetInput{
			Platform: model.EnvironmentPlatformLinux, Host: "198.51.100.10", Port: 22,
			Username: "deploy", WorkspaceRoot: "/srv/pomelo-orbit",
		},
	})
	if err != nil {
		t.Fatal(err)
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
	projectId := "project-1"
	revision := int64(5)
	probeStatus := model.EnvironmentProbeStatusSucceeded
	environment := model.Environment{
		Id: "environment-local", ProjectId: projectId, Code: "project",
		TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: "/srv/orbit/previous", TargetRevision: revision,
		LastProbeRevision: &revision, LastProbeStatus: &probeStatus,
	}
	store := &updateEnvironmentStore{environment: environment}
	targetType := model.EnvironmentTargetTypeLocal
	updated, err := New(store, probeProjectReader{}, nil, "", nil, nil, nil).
		WithLocalDisplay(environmentdto.LocalDisplaySnapshot{Platform: model.EnvironmentPlatformLinux}).
		UpdateForUser(context.Background(), "user-1", projectId, environmentdto.UpdateInput{
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
	service := New(store, probeProjectReader{}, credentialsForEnvironment(t, environment, "private-key"), testCredentialSecret, &probeEnvironmentProber{}, nil, nil)

	_, err := service.ProbeForUser(context.Background(), "user-1", environment.ProjectId)
	if err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("ProbeForUser error = %v", err)
	}
}

func TestInitializeForUserBootstrapsWithEphemeralPasswordAndProbesGeneratedKey(t *testing.T) {
	projectId := "project-1"
	environment := testProbeEnvironment(projectId)
	environment.SSH.HostKeyFingerprint = ""
	store := &probeEnvironmentStore{environment: environment}
	prober := &probeEnvironmentProber{}
	bootstrapper := &testEnvironmentBootstrapper{}
	service := New(
		store,
		probeProjectReader{},
		credentialsForEnvironment(t, environment, "generated-private-key"),
		testCredentialSecret,
		prober,
		bootstrapper,
		nil,
	)

	result, err := service.InitializeForUser(context.Background(), "user-1", projectId, environmentdto.InitializeInput{
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
	projectId := "project-1"
	environment := testProbeEnvironment(projectId)
	environment.SSH.Platform = model.EnvironmentPlatformWindows
	bootstrapper := &testEnvironmentBootstrapper{}
	service := New(
		&probeEnvironmentStore{environment: environment},
		probeProjectReader{},
		credentialsForEnvironment(t, environment, "private-key"),
		testCredentialSecret,
		&probeEnvironmentProber{},
		bootstrapper,
		nil,
	)
	_, err := service.InitializeForUser(context.Background(), "user-1", projectId, environmentdto.InitializeInput{
		Username: "administrator", Password: "password", PrivateKey: "private-key",
	})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) || bootstrapper.called {
		t.Fatalf("InitializeForUser() error = %v bootstrap=%#v", err, bootstrapper)
	}

	environment.SSH.Platform = model.EnvironmentPlatformLinux
	service = New(
		&probeEnvironmentStore{environment: environment},
		probeProjectReader{},
		credentialsForEnvironment(t, environment, "private-key"),
		testCredentialSecret,
		&probeEnvironmentProber{},
		bootstrapper,
		nil,
	)
	_, err = service.InitializeForUser(context.Background(), "user-1", projectId, environmentdto.InitializeInput{
		Username: "opc", Password: "password", PrivateKey: "private-key",
	})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) || bootstrapper.called {
		t.Fatalf("InitializeForUser() ambiguous auth error = %v bootstrap=%#v", err, bootstrapper)
	}
}

func TestInitializeForUserDoesNotExposeBootstrapSecret(t *testing.T) {
	projectId := "project-1"
	environment := testProbeEnvironment(projectId)
	secret := "bootstrap-private-key-must-not-appear"
	service := New(
		&probeEnvironmentStore{environment: environment},
		probeProjectReader{},
		credentialsForEnvironment(t, environment, "private-key"),
		testCredentialSecret,
		&probeEnvironmentProber{},
		&testEnvironmentBootstrapper{err: errors.New(secret)},
		nil,
	)
	_, err := service.InitializeForUser(context.Background(), "user-1", projectId, environmentdto.InitializeInput{
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

func (probeProjectReader) Project(_ context.Context, projectId string) (model.Project, error) {
	return model.Project{Id: projectId}, nil
}

func (probeProjectReader) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type probeEnvironmentProber struct {
	called      bool
	environment model.Environment
	privateKey  environmentdto.DeploymentSSHPrivateKey
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

func (p *localProbeEnvironmentProber) Probe(context.Context, model.Environment, environmentdto.DeploymentSSHPrivateKey) (string, error) {
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

func (p *probeEnvironmentProber) Probe(_ context.Context, environment model.Environment, privateKey environmentdto.DeploymentSSHPrivateKey) (string, error) {
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

func testProbeEnvironment(projectId string) model.Environment {
	return model.Environment{
		Id:             "environment-1",
		ProjectId:      projectId,
		Code:           "environment",
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

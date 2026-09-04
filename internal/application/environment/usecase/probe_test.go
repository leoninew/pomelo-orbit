package environmentsvc

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
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
	}, prober)

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
	}, &probeEnvironmentProber{err: errors.New(secret)})

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

func TestProbeForUserRejectsDisabledEnvironmentWithoutSSH(t *testing.T) {
	environment := testProbeEnvironment("project-1")
	environment.State = model.EnvironmentStateDisabled
	store := &probeEnvironmentStore{environment: environment}
	prober := &probeEnvironmentProber{}
	service := New(store, probeProjectReader{}, nil, probeCredentialReader{
		credential: testProbeCredential(environment.ProjectId, environment),
	}, prober)

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
	service := New(store, probeProjectReader{}, nil, probeCredentialReader{err: errors.New("credential requires reconfiguration")}, nil)

	_, err := service.UpdateForUser(context.Background(), "user-1", projectID, environmentdto.UpdateInput{State: &active})
	if err == nil || !apperror.IsKind(err, apperror.KindValidation) || !strings.Contains(err.Error(), "must be configured") {
		t.Fatalf("UpdateForUser error = %v", err)
	}
	if store.updated {
		t.Fatal("environment update must not be persisted without a configured deployment SSH credential")
	}
}
func TestProbeForUserRejectsStaleResult(t *testing.T) {
	environment := testProbeEnvironment("project-1")
	store := &probeEnvironmentStore{environment: environment, stale: true}
	service := New(store, probeProjectReader{}, nil, probeCredentialReader{
		credential: testProbeCredential(environment.ProjectId, environment),
	}, &probeEnvironmentProber{})

	_, err := service.ProbeForUser(context.Background(), "user-1", environment.ProjectId)
	if err == nil || !apperror.IsKind(err, apperror.KindConflict) {
		t.Fatalf("ProbeForUser error = %v", err)
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

func (s *updateEnvironmentStore) UpdateEnvironment(context.Context, model.Environment) error {
	s.updated = true
	return nil
}

type probeEnvironmentStore struct {
	repository.EnvironmentStore
	environment    model.Environment
	recorded       bool
	stale          bool
	targetRevision int64
	status         string
	diagnostic     string
}

func (s *probeEnvironmentStore) EnvironmentByProject(context.Context, string) (model.Environment, error) {
	return s.environment, nil
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
	err         error
}

func (p *probeEnvironmentProber) Probe(_ context.Context, environment model.Environment, privateKey credentialdto.DeploymentSSHPrivateKey) error {
	p.called = true
	p.environment = environment
	p.privateKey = privateKey
	return p.err
}

func testProbeEnvironment(projectID string) model.Environment {
	return model.Environment{
		Id:                    "environment-1",
		ProjectId:             projectID,
		State:                 model.EnvironmentStateActive,
		Platform:              model.EnvironmentPlatformLinux,
		Host:                  "192.0.2.10",
		Port:                  22,
		Username:              "deploy",
		WorkspaceRoot:         "/srv/pomelo-orbit",
		SSHCredentialId:       "credential-1",
		SSHCredentialRevision: 3,
		TargetRevision:        7,
	}
}

func testProbeCredential(projectID string, environment model.Environment) model.Credential {
	return model.Credential{
		Id:        environment.SSHCredentialId,
		ProjectId: &projectID,
		Type:      model.CredentialTypeDeploymentSSHPrivateKey,
		Revision:  environment.SSHCredentialRevision,
	}
}

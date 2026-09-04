package environmentsvc

import (
	"context"
	"errors"
	"strings"
	"testing"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

type targetEnvironmentStore struct {
	repository.EnvironmentStore
	environment model.Environment
	err         error
}

func (s targetEnvironmentStore) EnvironmentByProject(context.Context, string) (model.Environment, error) {
	return s.environment, s.err
}

type targetCredentialReader struct {
	credential model.Credential
	privateKey credentialdto.DeploymentSSHPrivateKey
	err        error
}

func (r targetCredentialReader) DeploymentSSHCredential(context.Context, string) (model.Credential, credentialdto.DeploymentSSHPrivateKey, error) {
	return r.credential, r.privateKey, r.err
}

func TestTargetResolverRejectsUnavailableTargets(t *testing.T) {
	base := readyTargetEnvironment()
	cases := []struct {
		name        string
		environment model.Environment
		credential  model.Credential
		want        string
	}{
		{name: "disabled environment", environment: withEnvironmentState(base, model.EnvironmentStateDisabled), credential: matchingTargetCredential(base), want: "disabled"},
		{name: "stale probe", environment: withProbeRevision(base, 1), credential: matchingTargetCredential(base), want: "must pass probe"},
		{name: "credential revision mismatch", environment: base, credential: withCredentialRevision(matchingTargetCredential(base), base.SSHCredentialRevision+1), want: "credential binding is invalid"},
	}
	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			resolver := NewTargetResolver(targetEnvironmentStore{environment: tt.environment}, targetCredentialReader{credential: tt.credential})
			_, err := resolver.ResolveProjectTarget(context.Background(), base.ProjectId)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("ResolveProjectTarget error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestTargetResolverReturnsPinnedPrivateKey(t *testing.T) {
	environment := readyTargetEnvironment()
	privateKey := credentialdto.DeploymentSSHPrivateKey{PrivateKey: "secret-key", Passphrase: "secret-passphrase"}
	resolver := NewTargetResolver(
		targetEnvironmentStore{environment: environment},
		targetCredentialReader{credential: matchingTargetCredential(environment), privateKey: privateKey},
	)

	target, err := resolver.ResolveProjectTarget(context.Background(), environment.ProjectId)
	if err != nil {
		t.Fatalf("ResolveProjectTarget returned error: %v", err)
	}
	if target.Environment.Id != environment.Id || target.PrivateKey != privateKey {
		t.Fatalf("resolved target = %#v", target)
	}
}

func TestTargetResolverMapsMissingEnvironmentToValidation(t *testing.T) {
	resolver := NewTargetResolver(targetEnvironmentStore{err: repository.ErrNotFound}, targetCredentialReader{})
	_, err := resolver.ResolveProjectTarget(context.Background(), "project-1")
	if err == nil || !strings.Contains(err.Error(), "not configured") || errors.Is(err, repository.ErrNotFound) {
		t.Fatalf("ResolveProjectTarget error = %v", err)
	}
}

func readyTargetEnvironment() model.Environment {
	revision := int64(2)
	status := model.EnvironmentProbeStatusSucceeded
	return model.Environment{
		Id: "environment-1", ProjectId: "project-1", State: model.EnvironmentStateActive,
		SSHCredentialId: "credential-1", SSHCredentialRevision: revision, TargetRevision: revision,
		LastProbeRevision: &revision, LastProbeStatus: &status,
	}
}

func matchingTargetCredential(environment model.Environment) model.Credential {
	projectID := environment.ProjectId
	return model.Credential{
		Id: environment.SSHCredentialId, ProjectId: &projectID,
		Type: model.CredentialTypeDeploymentSSHPrivateKey, Revision: environment.SSHCredentialRevision,
	}
}

func withEnvironmentState(environment model.Environment, state string) model.Environment {
	environment.State = state
	return environment
}

func withProbeRevision(environment model.Environment, revision int64) model.Environment {
	environment.LastProbeRevision = &revision
	return environment
}

func withCredentialRevision(credential model.Credential, revision int64) model.Credential {
	credential.Revision = revision
	return credential
}

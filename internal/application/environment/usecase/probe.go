package environmentsvc

import (
	"context"
	"strings"
	"time"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	probeFailureDiagnostic           = "SSH connection, host key verification, key authentication, or Docker prerequisites failed."
	probeCredentialFailureDiagnostic = "The deployment SSH credential binding is invalid."
	probeUnavailableDiagnostic       = "SSH environment probing is not configured."
)

type deploymentCredentialReader interface {
	DeploymentSSHCredential(context.Context, string) (model.Credential, credentialdto.DeploymentSSHPrivateKey, error)
}

type environmentProber interface {
	Probe(context.Context, model.Environment, credentialdto.DeploymentSSHPrivateKey) error
}

// ProbeForUser verifies the configured SSH target outside a request transaction.
// Its single conditional update records a result only for the target revision
// that was actually probed, preventing stale observations after an edit.
func (s Service) ProbeForUser(ctx context.Context, userID string, projectID string) (model.Environment, error) {
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return model.Environment{}, err
	}
	item, err := s.environmentForProject(ctx, projectID)
	if err != nil {
		return model.Environment{}, err
	}
	if !item.IsActive() {
		return model.Environment{}, apperror.New(apperror.KindValidation, "Environment must be active before it can be probed")
	}

	statusValue, diagnostic := s.probeOutcome(ctx, item)
	probedAt := time.Now().UTC()
	recorded, err := s.environments.RecordProbe(ctx, item.Id, item.TargetRevision, statusValue, probedAt, diagnostic)
	if err != nil {
		return model.Environment{}, apperror.Wrap(apperror.KindInternal, "Failed to record environment probe", err)
	}
	if !recorded {
		return model.Environment{}, apperror.New(apperror.KindConflict, "Environment changed during probe; probe it again")
	}

	revision := item.TargetRevision
	item.LastProbeRevision = &revision
	item.LastProbeStatus = stringPointer(statusValue)
	item.LastProbeAt = &probedAt
	item.LastProbeDiagnostic = stringPointer(diagnostic)
	return item, nil
}

func (s Service) probeOutcome(ctx context.Context, item model.Environment) (string, string) {
	if s.deploymentCredential == nil {
		return model.EnvironmentProbeStatusFailed, probeUnavailableDiagnostic
	}
	credential, privateKey, err := s.deploymentCredential.DeploymentSSHCredential(ctx, item.SSHCredentialId)
	if err != nil || !matchesEnvironmentCredential(item, credential) {
		return model.EnvironmentProbeStatusFailed, probeCredentialFailureDiagnostic
	}
	if s.prober == nil {
		return model.EnvironmentProbeStatusFailed, probeUnavailableDiagnostic
	}
	if err := s.prober.Probe(ctx, item, privateKey); err != nil {
		return model.EnvironmentProbeStatusFailed, probeFailureDiagnostic
	}
	return model.EnvironmentProbeStatusSucceeded, probeSuccessDiagnostic(item.Platform)
}

func matchesEnvironmentCredential(environment model.Environment, credential model.Credential) bool {
	return credential.Id == environment.SSHCredentialId &&
		credential.ProjectId != nil &&
		strings.TrimSpace(*credential.ProjectId) == environment.ProjectId &&
		credential.Revision == environment.SSHCredentialRevision &&
		credential.IsDeploymentSSHPrivateKey()
}

func probeSuccessDiagnostic(platform string) string {
	switch platform {
	case model.EnvironmentPlatformLinux:
		return "Linux SSH and Docker Compose prerequisites are ready."
	case model.EnvironmentPlatformWindows:
		return "Windows OpenSSH, WSL2, and Docker Desktop prerequisites are ready."
	default:
		return "SSH deployment prerequisites are ready."
	}
}

func stringPointer(value string) *string {
	return &value
}

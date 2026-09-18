package environmentsvc

import (
	"context"
	"errors"
	"strings"
	"time"

	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	probeFailureDiagnostic           = "SSH connection, host key verification, key authentication, or Docker prerequisites failed."
	probeInitializationDiagnostic    = "Generate and run the SSH initialization command before probing this target."
	probeCredentialFailureDiagnostic = "The deployment SSH credential binding is invalid."
	probeUnavailableDiagnostic       = "SSH environment probing is not configured."
	localProbeFailureDiagnostic      = "Local Docker and Docker Compose prerequisites failed."
	localProbeUnavailableDiagnostic  = "Local environment probing is not configured."
)

type probeDiagnosticError interface {
	ProbeDiagnostic() string
}

// ProbeForUser verifies the configured SSH target outside a request transaction.
// Its single conditional update records a result only for the target revision
// that was actually probed, preventing stale observations after an edit.
func (s Service) ProbeForUser(ctx context.Context, userId string, projectId string) (environmentdto.View, error) {
	if err := s.ensureProjectMembership(ctx, projectId, userId); err != nil {
		return environmentdto.View{}, err
	}
	item, err := s.environmentForProject(ctx, projectId)
	if err != nil {
		return environmentdto.View{}, err
	}
	if strings.TrimSpace(item.WorkspaceRoot) == "" {
		return environmentdto.View{}, apperror.New(apperror.KindValidation, "Environment workspace_root must be configured before it can be probed")
	}

	statusValue, diagnostic, observedFingerprint := model.EnvironmentProbeStatusFailed, probeInitializationDiagnostic, ""
	if !item.IsSSH() || hasSSHCredentialBinding(item) {
		statusValue, diagnostic, observedFingerprint = s.probeOutcome(ctx, item)
	}
	if item.IsSSH() && statusValue == model.EnvironmentProbeStatusSucceeded && strings.TrimSpace(item.SSH.HostKeyFingerprint) == "" {
		if !hostKeyFingerprintPattern.MatchString(observedFingerprint) {
			statusValue = model.EnvironmentProbeStatusFailed
			diagnostic = probeFailureDiagnostic
		} else {
			item.SSH.HostKeyFingerprint = observedFingerprint
			if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
				return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to record host key fingerprint", err)
			}
		}
	}

	probedAt := time.Now().UTC()
	recorded, err := s.environments.RecordProbe(ctx, item.Id, item.TargetRevision, statusValue, probedAt, diagnostic)
	if err != nil {
		return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to record environment probe", err)
	}
	if !recorded {
		return environmentdto.View{}, apperror.New(apperror.KindConflict, "Environment changed during probe; probe it again")
	}

	revision := item.TargetRevision
	item.LastProbeRevision = &revision
	item.LastProbeStatus = stringPointer(statusValue)
	item.LastProbeAt = &probedAt
	item.LastProbeDiagnostic = stringPointer(diagnostic)
	return s.toView(item), nil
}

func (s Service) probeOutcome(ctx context.Context, item model.Environment) (string, string, string) {
	if item.IsLocal() {
		if s.prober == nil {
			return model.EnvironmentProbeStatusFailed, localProbeUnavailableDiagnostic, ""
		}
		if err := s.prober.ProbeLocal(ctx, item); err != nil {
			return model.EnvironmentProbeStatusFailed, localProbeDiagnostic(err), ""
		}
		return model.EnvironmentProbeStatusSucceeded, "Local Docker and Docker Compose prerequisites are ready.", ""
	}
	if !item.IsSSH() {
		return model.EnvironmentProbeStatusFailed, probeUnavailableDiagnostic, ""
	}
	credential, err := s.environmentCredential(ctx, item.SSH.CredentialId)
	if err != nil || !matchesEnvironmentCredential(item, credential) {
		return model.EnvironmentProbeStatusFailed, probeCredentialFailureDiagnostic, ""
	}
	privateKey, err := decryptEnvironmentPrivateKey(s.secretKey, credential)
	if err != nil {
		return model.EnvironmentProbeStatusFailed, probeCredentialFailureDiagnostic, ""
	}
	if s.prober == nil {
		return model.EnvironmentProbeStatusFailed, probeUnavailableDiagnostic, ""
	}
	observedFingerprint, err := s.prober.Probe(ctx, item, privateKey)
	if err != nil {
		s.logEnvironmentFailure("project environment probe failed", item, err)
		return model.EnvironmentProbeStatusFailed, safeProbeDiagnostic(err), ""
	}
	return model.EnvironmentProbeStatusSucceeded, probeSuccessDiagnostic(item.SSH.Platform), observedFingerprint
}

func localProbeDiagnostic(err error) string {
	if diagnostic, ok := err.(probeDiagnosticError); ok && strings.TrimSpace(diagnostic.ProbeDiagnostic()) != "" {
		return diagnostic.ProbeDiagnostic()
	}
	return localProbeFailureDiagnostic
}

func safeProbeDiagnostic(err error) string {
	var diagnosticError probeDiagnosticError
	if errors.As(err, &diagnosticError) {
		if diagnostic := strings.TrimSpace(diagnosticError.ProbeDiagnostic()); diagnostic != "" {
			return diagnostic
		}
	}
	return probeFailureDiagnostic
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

func (s Service) logEnvironmentFailure(message string, item model.Environment, err error, extra ...any) {
	if s.logger == nil {
		return
	}
	attributes := []any{
		"project_id", item.ProjectId,
		"environment_id", item.Id,
		"target_type", item.TargetType,
	}
	if item.SSH != nil {
		attributes = append(attributes,
			"ssh_host", item.SSH.Host,
			"ssh_port", item.SSH.Port,
			"ssh_username", item.SSH.Username,
		)
	}
	attributes = append(attributes, extra...)
	attributes = append(attributes, "error", err)
	s.logger.Error(message, attributes...)
}

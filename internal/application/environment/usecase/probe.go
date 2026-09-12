package environmentsvc

import (
	"context"
	"errors"
	"strings"
	"time"

	credentialdto "github.com/leoninew/pomelo-orbit/internal/application/credential/dto"
	environmentdto "github.com/leoninew/pomelo-orbit/internal/application/environment/dto"
	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const (
	probeFailureDiagnostic           = "SSH connection, host key verification, key authentication, or Docker prerequisites failed."
	probeCredentialFailureDiagnostic = "The deployment SSH credential binding is invalid."
	probeUnavailableDiagnostic       = "SSH environment probing is not configured."
	localProbeFailureDiagnostic      = "Local Docker and Docker Compose prerequisites failed."
	localProbeUnavailableDiagnostic  = "Local environment probing is not configured."
)

type deploymentCredentialReader interface {
	DeploymentSSHCredential(context.Context, string) (model.Credential, credentialdto.DeploymentSSHPrivateKey, error)
}

type probeDiagnosticError interface {
	ProbeDiagnostic() string
}

// ProbeForUser verifies the configured SSH target outside a request transaction.
// Its single conditional update records a result only for the target revision
// that was actually probed, preventing stale observations after an edit.
func (s Service) ProbeForUser(ctx context.Context, userID string, projectID string) (environmentdto.View, error) {
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return environmentdto.View{}, err
	}
	item, err := s.environmentForProject(ctx, projectID)
	if err != nil {
		return environmentdto.View{}, err
	}
	if !item.IsActive() {
		return environmentdto.View{}, apperror.New(apperror.KindValidation, "Environment must be active before it can be probed")
	}
	if strings.TrimSpace(item.WorkspaceRoot) == "" {
		return environmentdto.View{}, apperror.New(apperror.KindValidation, "Environment workspace_root must be configured before it can be probed")
	}

	if item.IsSSH() {
		previous := item
		item, err = s.ensureGeneratedCredential(ctx, item)
		if err != nil {
			return environmentdto.View{}, err
		}
		if environmentTargetChanged(previous, item) {
			if environmentIdentityChanged(previous, item) {
				item.SSH.HostKeyFingerprint = ""
			}
			item.TargetRevision++
			if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
				return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
			}
		}
	}

	statusValue, diagnostic, observedFingerprint := s.probeOutcome(ctx, item)
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

type bootstrapDiagnosticError interface {
	BootstrapDiagnostic() string
}

// InitializeForUser uses temporary SSH credentials to install the generated
// deployment public key on a configured Linux target. The temporary
// credentials are never stored and the target is probed with the generated
// key immediately after bootstrap.
func (s Service) InitializeForUser(ctx context.Context, userID string, projectID string, input environmentdto.InitializeInput) (environmentdto.View, error) {
	if err := s.ensureProjectMembership(ctx, projectID, userID); err != nil {
		return environmentdto.View{}, err
	}
	item, err := s.environmentForProject(ctx, projectID)
	if err != nil {
		return environmentdto.View{}, err
	}
	if !item.IsActive() {
		return environmentdto.View{}, apperror.New(apperror.KindValidation, "Environment must be active before it can be initialized")
	}
	if !item.IsSSH() {
		return environmentdto.View{}, apperror.New(apperror.KindValidation, "Only an SSH environment can be initialized")
	}
	if strings.TrimSpace(item.WorkspaceRoot) == "" {
		return environmentdto.View{}, apperror.New(apperror.KindValidation, "Environment workspace_root must be configured before it can be initialized")
	}
	if item.SSH.Platform != model.EnvironmentPlatformLinux {
		return environmentdto.View{}, apperror.New(apperror.KindValidation, "Automatic SSH initialization is available only for Linux environments")
	}
	if s.bootstrapper == nil {
		return environmentdto.View{}, apperror.New(apperror.KindInternal, "SSH environment bootstrap is not configured")
	}
	if s.deploymentKey == nil {
		return environmentdto.View{}, apperror.New(apperror.KindInternal, "deployment SSH credential manager is not configured")
	}
	auth, err := bootstrapAuth(input)
	if err != nil {
		return environmentdto.View{}, err
	}

	previous := item
	item, err = s.ensureGeneratedCredential(ctx, item)
	if err != nil {
		return environmentdto.View{}, err
	}
	if environmentTargetChanged(previous, item) {
		if environmentIdentityChanged(previous, item) {
			item.SSH.HostKeyFingerprint = ""
		}
		item.TargetRevision++
		if err := s.environments.UpdateEnvironment(ctx, item); err != nil {
			return environmentdto.View{}, apperror.Wrap(apperror.KindInternal, "Failed to update project environment", err)
		}
	}
	publicKey, err := s.deploymentKey.DeploymentSSHPublicKey(ctx, item.SSH.CredentialId)
	if err != nil {
		return environmentdto.View{}, err
	}
	if err := s.bootstrapper.Bootstrap(ctx, item, publicKey, auth); err != nil {
		return environmentdto.View{}, apperror.New(apperror.KindValidation, safeBootstrapDiagnostic(err))
	}
	return s.ProbeForUser(ctx, userID, projectID)
}

func bootstrapAuth(input environmentdto.InitializeInput) (environmentport.BootstrapAuth, error) {
	auth := environmentport.BootstrapAuth{
		Username:             strings.TrimSpace(input.Username),
		Password:             input.Password,
		PrivateKey:           strings.TrimSpace(input.PrivateKey),
		PrivateKeyPassphrase: input.PrivateKeyPassphrase,
	}
	if auth.Username == "" || strings.ContainsAny(auth.Username, "\r\n") {
		return environmentport.BootstrapAuth{}, apperror.New(apperror.KindValidation, "Bootstrap SSH username is required")
	}
	hasPassword := auth.Password != ""
	hasPrivateKey := auth.PrivateKey != ""
	if hasPassword == hasPrivateKey {
		return environmentport.BootstrapAuth{}, apperror.New(apperror.KindValidation, "Provide exactly one SSH password or private key")
	}
	if !hasPrivateKey && auth.PrivateKeyPassphrase != "" {
		return environmentport.BootstrapAuth{}, apperror.New(apperror.KindValidation, "A private key passphrase requires a private key")
	}
	if len(auth.Password) > 4096 || len(auth.PrivateKey) > 64*1024 || len(auth.PrivateKeyPassphrase) > 4096 {
		return environmentport.BootstrapAuth{}, apperror.New(apperror.KindValidation, "Bootstrap SSH credentials are too large")
	}
	return auth, nil
}

func safeBootstrapDiagnostic(err error) string {
	var diagnosticError bootstrapDiagnosticError
	if errors.As(err, &diagnosticError) {
		if diagnostic := strings.TrimSpace(diagnosticError.BootstrapDiagnostic()); diagnostic != "" {
			return diagnostic
		}
	}
	return "SSH initialization failed. Verify the temporary SSH credential and target prerequisites."
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
	if s.deploymentCredential == nil {
		return model.EnvironmentProbeStatusFailed, probeUnavailableDiagnostic, ""
	}
	credential, privateKey, err := s.deploymentCredential.DeploymentSSHCredential(ctx, item.SSH.CredentialId)
	if err != nil || !matchesEnvironmentCredential(item, credential) {
		return model.EnvironmentProbeStatusFailed, probeCredentialFailureDiagnostic, ""
	}
	if s.prober == nil {
		return model.EnvironmentProbeStatusFailed, probeUnavailableDiagnostic, ""
	}
	observedFingerprint, err := s.prober.Probe(ctx, item, privateKey)
	if err != nil {
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

func matchesEnvironmentCredential(environment model.Environment, credential model.Credential) bool {
	return environment.IsSSH() &&
		credential.Id == environment.SSH.CredentialId &&
		credential.ProjectId != nil &&
		strings.TrimSpace(*credential.ProjectId) == environment.ProjectId &&
		credential.Revision == environment.SSH.CredentialRevision &&
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

package model

import (
	"strings"
	"time"
)

const (
	EnvironmentStateActive   = "active"
	EnvironmentStateDisabled = "disabled"

	EnvironmentTargetTypeLocal = "local"
	EnvironmentTargetTypeSSH   = "ssh"

	EnvironmentPlatformLinux   = "linux"
	EnvironmentPlatformWindows = "windows"

	EnvironmentProbeStatusSucceeded = "succeeded"
	EnvironmentProbeStatusFailed    = "failed"
)

// Environment is the unique Docker Compose target of one Project.
// project_id and gateway_application_id are logical references by design.
type Environment struct {
	Id                   string                `db:"id"`
	ProjectId            string                `db:"project_id"`
	Code                 string                `db:"code"`
	State                string                `db:"state"`
	TargetType           string                `db:"target_type"`
	WorkspaceRoot        string                `db:"workspace_root"`
	SSH                  *EnvironmentSSHTarget `db:"-"`
	TargetRevision       int64                 `db:"target_revision"`
	LastProbeRevision    *int64                `db:"last_probe_revision"`
	LastProbeStatus      *string               `db:"last_probe_status"`
	LastProbeAt          *time.Time            `db:"last_probe_at"`
	LastProbeDiagnostic  *string               `db:"last_probe_diagnostic"`
	GatewayApplicationId *string               `db:"gateway_application_id"`
	CreatedAt            time.Time             `db:"created_at"`
	UpdatedAt            time.Time             `db:"updated_at"`
}

// EnvironmentSSHTarget contains data that only exists when target_type is ssh.
type EnvironmentSSHTarget struct {
	Platform           string
	Host               string
	Port               int
	Username           string
	CredentialId       string
	CredentialRevision int64
	HostKeyFingerprint string
}

func (e Environment) IsActive() bool {
	return e.State == EnvironmentStateActive
}

func (e Environment) IsLocal() bool {
	return e.TargetType == EnvironmentTargetTypeLocal
}

func (e Environment) IsSSH() bool {
	return e.TargetType == EnvironmentTargetTypeSSH && e.SSH != nil
}

func (e Environment) HasHostKeyFingerprint() bool {
	if !e.IsSSH() {
		return false
	}
	fingerprint := strings.TrimSpace(e.SSH.HostKeyFingerprint)
	return strings.HasPrefix(fingerprint, "SHA256:") && len(fingerprint) > len("SHA256:")
}

func (e Environment) HasFreshSuccessfulProbe() bool {
	if e.LastProbeRevision == nil || *e.LastProbeRevision != e.TargetRevision || e.LastProbeStatus == nil || *e.LastProbeStatus != EnvironmentProbeStatusSucceeded {
		return false
	}
	return e.IsLocal() || e.HasHostKeyFingerprint()
}

package model

import "time"

const (
	EnvironmentStateActive   = "active"
	EnvironmentStateDisabled = "disabled"

	EnvironmentPlatformLinux   = "linux"
	EnvironmentPlatformWindows = "windows"

	EnvironmentProbeStatusSucceeded = "succeeded"
	EnvironmentProbeStatusFailed    = "failed"
)

// Environment is the unique SSH-managed Docker Compose target of one Project.
// project_id and gateway_application_id are logical references by design.
type Environment struct {
	Id                    string     `db:"id"`
	ProjectId             string     `db:"project_id"`
	Code                  string     `db:"code"`
	State                 string     `db:"state"`
	Platform              string     `db:"platform"`
	Host                  string     `db:"host"`
	Port                  int        `db:"port"`
	Username              string     `db:"username"`
	WorkspaceRoot         string     `db:"workspace_root"`
	SSHCredentialId       string     `db:"ssh_credential_id"`
	SSHCredentialRevision int64      `db:"ssh_credential_revision"`
	HostKeyFingerprint    string     `db:"host_key_fingerprint"`
	TargetRevision        int64      `db:"target_revision"`
	LastProbeRevision     *int64     `db:"last_probe_revision"`
	LastProbeStatus       *string    `db:"last_probe_status"`
	LastProbeAt           *time.Time `db:"last_probe_at"`
	LastProbeDiagnostic   *string    `db:"last_probe_diagnostic"`
	GatewayApplicationId  *string    `db:"gateway_application_id"`
	CreatedAt             time.Time  `db:"created_at"`
	UpdatedAt             time.Time  `db:"updated_at"`
}

func (e Environment) IsActive() bool {
	return e.State == EnvironmentStateActive
}

func (e Environment) HasFreshSuccessfulProbe() bool {
	return e.LastProbeRevision != nil && *e.LastProbeRevision == e.TargetRevision &&
		e.LastProbeStatus != nil && *e.LastProbeStatus == EnvironmentProbeStatusSucceeded
}

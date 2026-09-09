package dto

import "time"

type UpdateInput struct {
	State      *string
	TargetType *string
	SSH        *SSHTargetInput
}

type SSHTargetInput struct {
	Platform      string
	Host          string
	Port          int
	Username      string
	WorkspaceRoot string
}

type View struct {
	Id                   string
	ProjectId            string
	Code                 string
	State                string
	TargetType           string
	TargetRevision       int64
	LastProbeRevision    *int64
	LastProbeStatus      *string
	LastProbeAt          *time.Time
	LastProbeDiagnostic  *string
	GatewayApplicationId *string
	CreatedAt            time.Time
	UpdatedAt            time.Time
	Local                *LocalTargetView
	SSH                  *SSHTargetView
}

type LocalTargetView struct {
	WorkspaceRoot string
	Platform      string
	Host          string
	Username      string
}

type SSHTargetView struct {
	Platform           string
	Host               string
	Port               int
	Username           string
	WorkspaceRoot      string
	HostKeyFingerprint string
}

type LocalDisplaySnapshot struct {
	Platform string
	Host     string
	Username string
}

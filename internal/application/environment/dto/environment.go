package dto

import "time"

type UpdateInput struct {
	TargetType *string
	Local      *LocalTargetInput
	SSH        *SSHTargetInput
}

// TargetDefinition is the complete persisted Environment target state used by
// application workflows that operate on an already-known deployment target.
// It deliberately has no Project or storage identity.
type TargetDefinition struct {
	TargetType          string
	WorkspaceRoot       string
	TargetRevision      int64
	LastProbeRevision   *int64
	LastProbeStatus     *string
	LastProbeAt         *time.Time
	LastProbeDiagnostic *string
	SSH                 *SSHDefinition
	Credential          *SSHCredentialDefinition
}

type SSHDefinition struct {
	Platform           string
	Host               string
	Port               int
	Username           string
	CredentialRevision int64
	HostKeyFingerprint string
}

// SSHCredentialDefinition carries the logical key pair, not its encrypted
// persistence representation.
type SSHCredentialDefinition struct {
	PublicKey  string
	PrivateKey string
	Revision   int64
}

type LocalTargetInput struct {
	WorkspaceRoot string
}

type SSHTargetInput struct {
	Platform      string
	Host          string
	Port          int
	Username      string
	WorkspaceRoot string
}

// DeploymentSSHPrivateKey is decrypted only for an SSH probe or deployment.
// It must never cross an HTTP, MCP, log, or persistence boundary.
type DeploymentSSHPrivateKey struct {
	PrivateKey string
	Passphrase string
	PublicKey  string
}

type View struct {
	Id                   string
	ProjectId            string
	Code                 string
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

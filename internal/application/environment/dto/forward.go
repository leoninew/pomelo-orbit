package dto

import "time"

type ForwardInput struct {
	ProjectCode string `json:"project_code"`
	LocalPort   int    `json:"local_port"`
	RemotePort  int    `json:"remote_port"`
}

// ForwardSpec contains connection identity, never decrypted credentials.
type ForwardSpec struct {
	ForwardInput
	ActorId            string `json:"actor_id"`
	ProjectId          string `json:"project_id"`
	EnvironmentId      string `json:"environment_id"`
	TargetRevision     int64  `json:"target_revision"`
	CredentialRevision int64  `json:"credential_revision"`
}

type ForwardStatus struct {
	ForwardSpec
	Pid       int       `json:"pid"`
	StartedAt time.Time `json:"started_at"`
	State     string    `json:"state"`
	Reason    string    `json:"reason,omitempty"`
}

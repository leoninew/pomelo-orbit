package dto

type DeploymentListInput struct {
	ProjectId     string
	ApplicationId string
	Status        string
	Search        string
	DateFrom      string
	DateTo        string
	Page          int
	PerPage       int
}

// DeployInput describes a user-requested deployment of an application version.
type DeployInput struct {
	VersionId     string
	InstanceKey   string
	ForceRecreate bool
	RuntimeConfig map[string]string
}

// ServiceTargetInput identifies the runtime service affected by a restart or stop.
type ServiceTargetInput struct {
	InstanceKey   string
	ServiceId     string
	RemoveVolumes bool
}

// DeployOptionsJSON is persisted with a deployment and consumed by the worker.
type DeployOptionsJSON struct {
	ForceRecreate bool              `json:"force_recreate"`
	InstanceKey   string            `json:"instance_key"`
	RemoveVolumes bool              `json:"remove_volumes"`
	RuntimeConfig map[string]string `json:"runtime_config,omitempty"`
}

type DeployDispatchInput struct {
	ApplicationID string
	DeploymentID  string
	ForceRecreate bool
}

type RestartDispatchInput struct {
	ApplicationID string
	DeploymentID  string
}

type StopDispatchInput struct {
	ApplicationID string
	DeploymentID  string
	RemoveVolumes bool
}

type DeploymentLog struct {
	Logs       string
	Offset     int
	IsComplete bool
	Status     string
}

type DeploymentContainerLog struct {
	Logs                string
	Source              string
	IsRealtimeSupported bool
}

// RuntimeContainer is a normalized docker compose ps record for one container.
type RuntimeContainer struct {
	ID      string
	Name    string
	Service string
	State   string
	Status  string
	Health  string
	Image   string
}

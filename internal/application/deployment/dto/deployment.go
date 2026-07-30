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

// DeployServiceInput describes a user-requested deployment of saved Service configuration.
type DeployServiceInput struct {
	ForceRecreate bool
}

type DeployServiceResult struct {
	DeploymentId string
	Warnings     []string
}

// ServiceTargetInput identifies the runtime service affected by an operation.
type ServiceTargetInput struct {
	ServiceId     string
	RemoveVolumes bool
}

// DeployOptionsJSON is persisted with a deployment and consumed by the worker.
type DeployOptionsJSON struct {
	ForceRecreate bool              `json:"force_recreate"`
	InstanceKey   string            `json:"instance_key"`
	RemoveVolumes bool              `json:"remove_volumes"`
	RuntimeConfig map[string]string `json:"runtime_config"`
}

type DeployDispatchInput struct {
	ApplicationId string
	DeploymentId  string
	ForceRecreate bool
}

type RestartDispatchInput struct {
	ApplicationId string
	DeploymentId  string
}

type StopDispatchInput struct {
	ApplicationId string
	DeploymentId  string
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
	Id           string
	Name         string
	Service      string
	State        string
	Status       string
	Health       string
	Image        string
	VersionId    string
	VersionLabel string
	ComponentId  string
}

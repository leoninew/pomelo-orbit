package dto

import (
	"time"

	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

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
	ForceRecreate bool   `json:"force_recreate"`
	InstanceKey   string `json:"instance_key"`
	RemoveVolumes bool   `json:"remove_volumes"`
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

// DeploymentWaitResult records a bounded wait over the persisted deployment
// state. TimedOut is distinct from a terminal deployment failure.
type DeploymentWaitResult struct {
	Deployment model.Deployment
	TimedOut   bool
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

// RuntimeNetwork is a small, stable projection of a Docker network.
type RuntimeNetwork struct {
	Id     string
	Name   string
	Driver string
}

// RuntimeTarget identifies one managed Compose workspace after the
// application use case has checked ownership and application scope.
type RuntimeTarget struct {
	ApplicationId    string
	ServiceId        string
	InstanceKey      string
	ServiceCode      string
	WorkingDirectory string
	ComposeProject   string
}

type RuntimeComposePSResult struct {
	Target     RuntimeTarget
	Containers []RuntimeContainer
	Raw        any
}

type RuntimeTextResult struct {
	Target RuntimeTarget
	Text   string
}

type RuntimeInspectResult struct {
	Target RuntimeTarget
	Data   any
}

type DeploymentVerificationInput struct {
	StabilityWindow time.Duration
	StabilityPoll   time.Duration
}

type DeploymentVerificationResult struct {
	Conclusion  string
	Differences []string
	Evidence    map[string]any
	Summary     map[string]any
}

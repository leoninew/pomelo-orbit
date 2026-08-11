package status

const (
	TaskPending   = "pending"
	TaskRunning   = "running"
	TaskSucceeded = "succeeded"
	TaskFailed    = "failed"
	TaskCanceled  = "canceled"
)

const (
	TaskTypePipelineRunExecute = "pipeline_run.execute"
	TaskTypeDeploymentDeploy   = "deployment.deploy"
	TaskTypeDeploymentRestart  = "deployment.restart"
	TaskTypeDeploymentStop     = "deployment.stop"
)

const (
	WorkStatusWaitingToRun    = "waiting_to_run"
	WorkStatusRunning         = "running"
	WorkStatusRanToCompletion = "ran_to_completion"
	WorkStatusFaulted         = "faulted"
	WorkStatusCanceled        = "canceled"
)

// WorkStatusIsComplete is the single definition of WorkStatus terminal / API is_complete:
// ran_to_completion | faulted | canceled.
func WorkStatusIsComplete(value string) bool {
	return value == WorkStatusRanToCompletion || value == WorkStatusFaulted || value == WorkStatusCanceled
}

// ServiceStatus is the runtime binding status for model.Service (not Application).
const (
	ServiceStatusRunning = "running"
	ServiceStatusStopped = "stopped"
	ServiceStatusFaulted = "faulted"
)

// ApplicationKind is the render strategy for an application.
const (
	ApplicationKindStandard = "standard"
	ApplicationKindGateway  = "gateway"
)

// VersionStatus is the lifecycle of model.Version.
const (
	VersionStatusUnpublished = "unpublished"
	VersionStatusPublished   = "published"
)

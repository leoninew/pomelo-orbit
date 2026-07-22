package status

const (
	TaskPending   = "pending"
	TaskRunning   = "running"
	TaskSucceeded = "succeeded"
	TaskFailed    = "failed"
	TaskCanceled  = "canceled"
)

const (
	TaskTypeCIPipelineRunExecute = "ci.pipeline_run.execute"
	TaskTypeCDApplicationDeploy  = "cd.application.deploy"
	TaskTypeCDApplicationRestart = "cd.application.restart"
	TaskTypeCDApplicationStop    = "cd.application.stop"
)

const (
	WorkStatusWaitingToRun    = "waiting_to_run"
	WorkStatusRunning         = "running"
	WorkStatusRanToCompletion = "ran_to_completion"
	WorkStatusFaulted         = "faulted"
	WorkStatusCanceled        = "canceled"
)

// ServiceStatus is the runtime binding status for model.Service (not Application).
const (
	ServiceStatusDeploying = "deploying"
	ServiceStatusRunning   = "running"
	ServiceStatusStopped   = "stopped"
	ServiceStatusFaulted   = "faulted"
)

// VersionStatus is the lifecycle of model.Version.
const (
	VersionStatusUnpublished = "unpublished"
	VersionStatusPublished   = "published"
)

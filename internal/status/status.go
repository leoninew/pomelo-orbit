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

const (
	ApplicationStatusDeployed     = "deployed"
	ApplicationStatusDeploying    = "deploying"
	ApplicationStatusUndeployed   = "undeployed"
	ApplicationStatusDeployFailed = "deploy_failed"
)

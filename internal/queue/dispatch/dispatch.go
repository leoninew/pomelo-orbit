package dispatch

import (
	"context"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
)

// PipelineRunDispatcher enqueues pipeline_run execution tasks.
type PipelineRunDispatcher struct {
	tasks tasksvc.Service
}

func NewPipelineRunDispatcher(tasks tasksvc.Service) PipelineRunDispatcher {
	return PipelineRunDispatcher{tasks: tasks}
}

func (d PipelineRunDispatcher) DispatchPipelineRun(ctx context.Context, input pipelinerundto.PipelineRunDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypePipelineRunExecute, pipelineRunExecutePayload{PipelineRunID: input.PipelineRunID})
	return err
}

// DeploymentDispatcher enqueues deployment command tasks.
type DeploymentDispatcher struct {
	tasks tasksvc.Service
}

func NewDeploymentDispatcher(tasks tasksvc.Service) DeploymentDispatcher {
	return DeploymentDispatcher{tasks: tasks}
}

func (d DeploymentDispatcher) DispatchDeploy(ctx context.Context, input deploymentdto.DeployDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypeDeploymentDeploy, deploymentDeployPayload{
		ApplicationID: input.ApplicationID,
		DeploymentID:  input.DeploymentID,
		ForceRecreate: input.ForceRecreate,
	})
	return err
}

func (d DeploymentDispatcher) DispatchRestart(ctx context.Context, input deploymentdto.RestartDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypeDeploymentRestart, deploymentRestartPayload{
		ApplicationID: input.ApplicationID,
		DeploymentID:  input.DeploymentID,
	})
	return err
}

func (d DeploymentDispatcher) DispatchStop(ctx context.Context, input deploymentdto.StopDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypeDeploymentStop, deploymentStopPayload{
		ApplicationID: input.ApplicationID,
		DeploymentID:  input.DeploymentID,
		RemoveVolumes: input.RemoveVolumes,
	})
	return err
}

type pipelineRunExecutePayload struct {
	PipelineRunID string `json:"pipeline_run_id"`
}

type deploymentDeployPayload struct {
	ApplicationID string `json:"application_id"`
	DeploymentID  string `json:"deployment_id"`
	ForceRecreate bool   `json:"force_recreate"`
}

type deploymentRestartPayload struct {
	ApplicationID string `json:"application_id"`
	DeploymentID  string `json:"deployment_id"`
}

type deploymentStopPayload struct {
	ApplicationID string `json:"application_id"`
	DeploymentID  string `json:"deployment_id"`
	RemoveVolumes bool   `json:"remove_volumes"`
}

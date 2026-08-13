package dispatch

import (
	"context"

	deploymentdto "github.com/leoninew/pomelo-orbit/internal/application/deployment/dto"
	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	tasksvc "github.com/leoninew/pomelo-orbit/internal/queue/task"
)

// PipelineRunDispatcher enqueues pipeline_run execution tasks.
type PipelineRunDispatcher struct {
	tasks tasksvc.Service
}

func NewPipelineRunDispatcher(tasks tasksvc.Service) PipelineRunDispatcher {
	return PipelineRunDispatcher{tasks: tasks}
}

func (d PipelineRunDispatcher) DispatchPipelineRun(ctx context.Context, input pipelinerundto.PipelineRunDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypePipelineRunExecute, pipelineRunExecutePayload{PipelineRunId: input.PipelineRunId})
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
		ApplicationId: input.ApplicationId,
		DeploymentId:  input.DeploymentId,
		ForceRecreate: input.ForceRecreate,
	})
	return err
}

func (d DeploymentDispatcher) DispatchRestart(ctx context.Context, input deploymentdto.RestartDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypeDeploymentRestart, deploymentRestartPayload{
		ApplicationId: input.ApplicationId,
		DeploymentId:  input.DeploymentId,
	})
	return err
}

func (d DeploymentDispatcher) DispatchStop(ctx context.Context, input deploymentdto.StopDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypeDeploymentStop, deploymentStopPayload{
		ApplicationId: input.ApplicationId,
		DeploymentId:  input.DeploymentId,
		RemoveVolumes: input.RemoveVolumes,
	})
	return err
}

type pipelineRunExecutePayload struct {
	PipelineRunId string `json:"pipeline_run_id"`
}

type deploymentDeployPayload struct {
	ApplicationId string `json:"application_id"`
	DeploymentId  string `json:"deployment_id"`
	ForceRecreate bool   `json:"force_recreate"`
}

type deploymentRestartPayload struct {
	ApplicationId string `json:"application_id"`
	DeploymentId  string `json:"deployment_id"`
}

type deploymentStopPayload struct {
	ApplicationId string `json:"application_id"`
	DeploymentId  string `json:"deployment_id"`
	RemoveVolumes bool   `json:"remove_volumes"`
}

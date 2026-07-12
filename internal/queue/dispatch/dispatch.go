package dispatch

import (
	"context"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
)

type CIDispatcher struct {
	tasks tasksvc.Service
}

func NewCIDispatcher(tasks tasksvc.Service) CIDispatcher {
	return CIDispatcher{tasks: tasks}
}

func (d CIDispatcher) DispatchPipelineRun(ctx context.Context, input cidto.PipelineRunDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypeCIPipelineRunExecute, pipelineRunExecutePayload{PipelineRunID: input.PipelineRunID})
	return err
}

type CDDispatcher struct {
	tasks tasksvc.Service
}

func NewCDDispatcher(tasks tasksvc.Service) CDDispatcher {
	return CDDispatcher{tasks: tasks}
}

func (d CDDispatcher) DispatchApplicationDeploy(ctx context.Context, input cdto.ApplicationDeployDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypeCDApplicationDeploy, applicationDeployPayload{
		ApplicationID: input.ApplicationID,
		DeploymentID:  input.DeploymentID,
		ForceRecreate: input.ForceRecreate,
	})
	return err
}

func (d CDDispatcher) DispatchApplicationRestart(ctx context.Context, input cdto.ApplicationRestartDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypeCDApplicationRestart, applicationRestartPayload{
		ApplicationID: input.ApplicationID,
		DeploymentID:  input.DeploymentID,
	})
	return err
}

func (d CDDispatcher) DispatchApplicationStop(ctx context.Context, input cdto.ApplicationStopDispatchInput) error {
	_, err := d.tasks.EnqueueTyped(ctx, status.TaskTypeCDApplicationStop, applicationStopPayload{
		ApplicationID: input.ApplicationID,
		DeploymentID:  input.DeploymentID,
		RemoveVolumes: input.RemoveVolumes,
	})
	return err
}

type pipelineRunExecutePayload struct {
	PipelineRunID string `json:"pipeline_run_id"`
}

type applicationDeployPayload struct {
	ApplicationID string `json:"application_id"`
	DeploymentID  string `json:"deployment_id"`
	ForceRecreate bool   `json:"force_recreate"`
}

type applicationRestartPayload struct {
	ApplicationID string `json:"application_id"`
	DeploymentID  string `json:"deployment_id"`
}

type applicationStopPayload struct {
	ApplicationID string `json:"application_id"`
	DeploymentID  string `json:"deployment_id"`
	RemoveVolumes bool   `json:"remove_volumes"`
}

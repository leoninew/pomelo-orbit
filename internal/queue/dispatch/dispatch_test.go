package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	deploymentdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/deployment/dto"
	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
)

func TestPipelineRunDispatcherPreservesTaskContract(t *testing.T) {
	repo := &recordingTaskRepository{}
	dispatcher := NewPipelineRunDispatcher(tasksvc.New(repo, 3))

	if err := dispatcher.DispatchPipelineRun(context.Background(), pipelinerundto.PipelineRunDispatchInput{PipelineRunId: "run-1"}); err != nil {
		t.Fatal(err)
	}
	if repo.taskType != status.TaskTypePipelineRunExecute {
		t.Fatalf("unexpected task type: %q", repo.taskType)
	}
	assertPayloadJSON(t, repo.payloadJSON, `{"pipeline_run_id":"run-1"}`)
}

func TestDispatchersPropagateEnqueueError(t *testing.T) {
	want := errors.New("enqueue unavailable")
	tests := []struct {
		name     string
		dispatch func(PipelineRunDispatcher, DeploymentDispatcher) error
	}{
		{
			name: "pipeline_run execute",
			dispatch: func(pipelineRun PipelineRunDispatcher, _ DeploymentDispatcher) error {
				return pipelineRun.DispatchPipelineRun(context.Background(), pipelinerundto.PipelineRunDispatchInput{PipelineRunId: "run-1"})
			},
		},
		{
			name: "deployment deploy",
			dispatch: func(_ PipelineRunDispatcher, deployment DeploymentDispatcher) error {
				return deployment.DispatchDeploy(context.Background(), deploymentdto.DeployDispatchInput{ApplicationId: "app-1", DeploymentId: "deploy-1"})
			},
		},
		{
			name: "deployment restart",
			dispatch: func(_ PipelineRunDispatcher, deployment DeploymentDispatcher) error {
				return deployment.DispatchRestart(context.Background(), deploymentdto.RestartDispatchInput{ApplicationId: "app-1", DeploymentId: "restart-1"})
			},
		},
		{
			name: "deployment stop",
			dispatch: func(_ PipelineRunDispatcher, deployment DeploymentDispatcher) error {
				return deployment.DispatchStop(context.Background(), deploymentdto.StopDispatchInput{ApplicationId: "app-1", DeploymentId: "stop-1"})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &recordingTaskRepository{enqueueErr: want}
			err := test.dispatch(NewPipelineRunDispatcher(tasksvc.New(repo, 3)), NewDeploymentDispatcher(tasksvc.New(repo, 3)))
			if !errors.Is(err, want) {
				t.Fatalf("expected enqueue error %v, got %v", want, err)
			}
		})
	}
}

func TestDeploymentDispatcherPreservesTaskContracts(t *testing.T) {
	tests := []struct {
		name     string
		dispatch func(DeploymentDispatcher) error
		taskType string
		payload  string
	}{
		{
			name: "deploy",
			dispatch: func(d DeploymentDispatcher) error {
				return d.DispatchDeploy(context.Background(), deploymentdto.DeployDispatchInput{ApplicationId: "app-1", DeploymentId: "deploy-1", ForceRecreate: true})
			},
			taskType: status.TaskTypeDeploymentDeploy,
			payload:  `{"application_id":"app-1","deployment_id":"deploy-1","force_recreate":true}`,
		},
		{
			name: "restart",
			dispatch: func(d DeploymentDispatcher) error {
				return d.DispatchRestart(context.Background(), deploymentdto.RestartDispatchInput{ApplicationId: "app-1", DeploymentId: "restart-1"})
			},
			taskType: status.TaskTypeDeploymentRestart,
			payload:  `{"application_id":"app-1","deployment_id":"restart-1"}`,
		},
		{
			name: "stop",
			dispatch: func(d DeploymentDispatcher) error {
				return d.DispatchStop(context.Background(), deploymentdto.StopDispatchInput{ApplicationId: "app-1", DeploymentId: "stop-1", RemoveVolumes: true})
			},
			taskType: status.TaskTypeDeploymentStop,
			payload:  `{"application_id":"app-1","deployment_id":"stop-1","remove_volumes":true}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &recordingTaskRepository{}
			if err := test.dispatch(NewDeploymentDispatcher(tasksvc.New(repo, 3))); err != nil {
				t.Fatal(err)
			}
			if repo.taskType != test.taskType {
				t.Fatalf("unexpected task type: got %q want %q", repo.taskType, test.taskType)
			}
			assertPayloadJSON(t, repo.payloadJSON, test.payload)
		})
	}
}

func assertPayloadJSON(t *testing.T, got string, want string) {
	t.Helper()
	var gotValue any
	if err := json.Unmarshal([]byte(got), &gotValue); err != nil {
		t.Fatalf("decode dispatched payload: %v", err)
	}
	var wantValue any
	if err := json.Unmarshal([]byte(want), &wantValue); err != nil {
		t.Fatalf("decode expected payload: %v", err)
	}
	gotJSON, err := json.Marshal(gotValue)
	if err != nil {
		t.Fatal(err)
	}
	wantJSON, err := json.Marshal(wantValue)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotJSON) != string(wantJSON) {
		t.Fatalf("unexpected payload: got %s want %s", gotJSON, wantJSON)
	}
}

type recordingTaskRepository struct {
	id          string
	taskType    string
	payloadJSON string
	enqueueErr  error
}

func (r *recordingTaskRepository) Enqueue(_ context.Context, id string, taskType string, payloadJSON string, _ int) error {
	r.id = id
	r.taskType = taskType
	r.payloadJSON = payloadJSON
	return r.enqueueErr
}

func (r *recordingTaskRepository) FindById(_ context.Context, id string) (*tasksvc.Task, error) {
	return &tasksvc.Task{Id: id, TaskType: r.taskType, PayloadJSON: r.payloadJSON}, nil
}

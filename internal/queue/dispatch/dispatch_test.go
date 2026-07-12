package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	cdto "gitee.com/leoninew/PomeloOrbit-go/internal/application/cd/dto"
	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	tasksvc "gitee.com/leoninew/PomeloOrbit-go/internal/queue/task"
)

func TestCIDispatcherPreservesTaskContract(t *testing.T) {
	repo := &recordingTaskRepository{}
	dispatcher := NewCIDispatcher(tasksvc.New(repo, 3))

	if err := dispatcher.DispatchPipelineRun(context.Background(), cidto.PipelineRunDispatchInput{PipelineRunID: "run-1"}); err != nil {
		t.Fatal(err)
	}
	if repo.taskType != status.TaskTypeCIPipelineRunExecute {
		t.Fatalf("unexpected task type: %q", repo.taskType)
	}
	assertPayloadJSON(t, repo.payloadJSON, `{"pipeline_run_id":"run-1"}`)
}

func TestDispatchersPropagateEnqueueError(t *testing.T) {
	want := errors.New("enqueue unavailable")
	tests := []struct {
		name     string
		dispatch func(CIDispatcher, CDDispatcher) error
	}{
		{
			name: "ci pipeline run",
			dispatch: func(ci CIDispatcher, _ CDDispatcher) error {
				return ci.DispatchPipelineRun(context.Background(), cidto.PipelineRunDispatchInput{PipelineRunID: "run-1"})
			},
		},
		{
			name: "cd deploy",
			dispatch: func(_ CIDispatcher, cd CDDispatcher) error {
				return cd.DispatchApplicationDeploy(context.Background(), cdto.ApplicationDeployDispatchInput{ApplicationID: "app-1", DeploymentID: "deploy-1"})
			},
		},
		{
			name: "cd restart",
			dispatch: func(_ CIDispatcher, cd CDDispatcher) error {
				return cd.DispatchApplicationRestart(context.Background(), cdto.ApplicationRestartDispatchInput{ApplicationID: "app-1", DeploymentID: "restart-1"})
			},
		},
		{
			name: "cd stop",
			dispatch: func(_ CIDispatcher, cd CDDispatcher) error {
				return cd.DispatchApplicationStop(context.Background(), cdto.ApplicationStopDispatchInput{ApplicationID: "app-1", DeploymentID: "stop-1"})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &recordingTaskRepository{enqueueErr: want}
			err := test.dispatch(NewCIDispatcher(tasksvc.New(repo, 3)), NewCDDispatcher(tasksvc.New(repo, 3)))
			if !errors.Is(err, want) {
				t.Fatalf("expected enqueue error %v, got %v", want, err)
			}
		})
	}
}

func TestCDDispatcherPreservesTaskContracts(t *testing.T) {
	tests := []struct {
		name     string
		dispatch func(CDDispatcher) error
		taskType string
		payload  string
	}{
		{
			name: "deploy",
			dispatch: func(d CDDispatcher) error {
				return d.DispatchApplicationDeploy(context.Background(), cdto.ApplicationDeployDispatchInput{ApplicationID: "app-1", DeploymentID: "deploy-1", ForceRecreate: true})
			},
			taskType: status.TaskTypeCDApplicationDeploy,
			payload:  `{"application_id":"app-1","deployment_id":"deploy-1","force_recreate":true}`,
		},
		{
			name: "restart",
			dispatch: func(d CDDispatcher) error {
				return d.DispatchApplicationRestart(context.Background(), cdto.ApplicationRestartDispatchInput{ApplicationID: "app-1", DeploymentID: "restart-1"})
			},
			taskType: status.TaskTypeCDApplicationRestart,
			payload:  `{"application_id":"app-1","deployment_id":"restart-1"}`,
		},
		{
			name: "stop",
			dispatch: func(d CDDispatcher) error {
				return d.DispatchApplicationStop(context.Background(), cdto.ApplicationStopDispatchInput{ApplicationID: "app-1", DeploymentID: "stop-1", RemoveVolumes: true})
			},
			taskType: status.TaskTypeCDApplicationStop,
			payload:  `{"application_id":"app-1","deployment_id":"stop-1","remove_volumes":true}`,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repo := &recordingTaskRepository{}
			if err := test.dispatch(NewCDDispatcher(tasksvc.New(repo, 3))); err != nil {
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

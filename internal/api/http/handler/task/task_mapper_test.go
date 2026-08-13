package taskhandler

import (
	"testing"

	taskv1 "github.com/leoninew/pomelo-orbit/internal/gen/proto/orbit/v1/task"
	tasksvc "github.com/leoninew/pomelo-orbit/internal/queue/task"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestTaskCreateInputPreservesPayloadSemantics(t *testing.T) {
	payload, err := structpb.NewValue(map[string]any{"run": "run-1"})
	if err != nil {
		t.Fatalf("create payload: %v", err)
	}
	input := taskCreateInput(&taskv1.CreateTaskReq{Id: "task-1", TaskType: "pipeline.run", Payload: payload, PayloadJson: `{"ignored":true}`})
	if input.Id != "task-1" || input.TaskType != "pipeline.run" || string(input.Payload) != `{"run":"run-1"}` || input.PayloadJSON != `{"ignored":true}` {
		t.Fatalf("unexpected task input: %+v", input)
	}
	if rawPayload(nil) != nil {
		t.Fatal("nil protocol payload must remain nil")
	}
}

func TestTypedTaskPayloadsUseExpectedKeys(t *testing.T) {
	if payload := pipelineRunExecutePayload("run-1"); len(payload) != 1 || payload["pipeline_run_id"] != "run-1" {
		t.Fatalf("unexpected pipeline payload: %#v", payload)
	}
	if payload := applicationDeploymentPayload("app-1", "deploy-1"); len(payload) != 2 || payload["application_id"] != "app-1" || payload["deployment_id"] != "deploy-1" {
		t.Fatalf("unexpected deployment payload: %#v", payload)
	}
}

func TestTaskResponseExposesFrozenMaxAttempts(t *testing.T) {
	response := taskResponse(&tasksvc.Task{Id: "task-1", Attempts: 1, MaxAttempts: 2})
	if response.GetId() != "task-1" || response.GetAttempts() != 1 || response.GetMaxAttempts() != 2 {
		t.Fatalf("unexpected task response fields: id=%q attempts=%d max_attempts=%d", response.GetId(), response.GetAttempts(), response.GetMaxAttempts())
	}
}

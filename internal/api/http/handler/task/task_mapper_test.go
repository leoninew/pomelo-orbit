package taskhandler

import (
	"testing"

	taskv1 "gitee.com/leoninew/PomeloOrbit-go/internal/gen/proto/orbit/v1/task"
	"google.golang.org/protobuf/types/known/structpb"
)

func TestTaskCreateInputPreservesPayloadSemantics(t *testing.T) {
	payload, err := structpb.NewValue(map[string]any{"run": "run-1"})
	if err != nil {
		t.Fatalf("create payload: %v", err)
	}
	input := taskCreateInput(&taskv1.CreateTaskReq{Id: "task-1", TaskType: "pipeline.run", Payload: payload, PayloadJson: `{"ignored":true}`, MaxAttempts: 3})
	if input.Id != "task-1" || input.TaskType != "pipeline.run" || input.MaxAttempts != 3 || string(input.Payload) != `{"run":"run-1"}` || input.PayloadJSON != `{"ignored":true}` {
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

package pipelinerunhandler

import (
	"context"
	"strings"
	"testing"

	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	tasksvc "github.com/leoninew/pomelo-orbit/internal/queue/task"
)

func TestHandleRejectsInvalidPayload(t *testing.T) {
	handler := NewHandler(&fakeExecutor{})
	err := handler.Handle(context.Background(), tasksvc.Task{PayloadJSON: `{invalid`})
	if err == nil || !strings.Contains(err.Error(), "parse pipeline_run task payload") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestHandleRequiresPipelineRunId(t *testing.T) {
	handler := NewHandler(&fakeExecutor{})
	err := handler.Handle(context.Background(), tasksvc.Task{PayloadJSON: `{}`})
	if err == nil || !strings.Contains(err.Error(), "pipeline_run_id is required") {
		t.Fatalf("expected required field error, got %v", err)
	}
}

func TestHandleExecutesPipelineRun(t *testing.T) {
	executor := &fakeExecutor{}
	handler := NewHandler(executor)

	err := handler.Handle(context.Background(), tasksvc.Task{PayloadJSON: `{"pipeline_run_id":"run-1","variables":{"IMAGE":"demo"}}`})
	if err != nil {
		t.Fatalf("Handle returned error: %v", err)
	}
	if executor.input.PipelineRunId != "run-1" {
		t.Fatalf("unexpected pipeline run id: %s", executor.input.PipelineRunId)
	}
	if executor.input.Variables["IMAGE"] != "demo" {
		t.Fatalf("unexpected variables: %+v", executor.input.Variables)
	}
}

type fakeExecutor struct {
	input pipelinerundto.ExecutePipelineRunInput
}

func (e *fakeExecutor) ExecutePipelineRun(ctx context.Context, input pipelinerundto.ExecutePipelineRunInput) error {
	e.input = input
	return nil
}

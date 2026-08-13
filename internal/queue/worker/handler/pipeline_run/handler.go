package pipelinerunhandler

import (
	"context"
	"encoding/json"
	"fmt"

	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	tasksvc "github.com/leoninew/pomelo-orbit/internal/queue/task"
)

type ExecutePayload struct {
	PipelineRunId string         `json:"pipeline_run_id"`
	Variables     map[string]any `json:"variables"`
}

type PipelineExecutor interface {
	ExecutePipelineRun(ctx context.Context, input pipelinerundto.ExecutePipelineRunInput) error
}

type Handler struct {
	executor PipelineExecutor
}

func NewHandler(executor PipelineExecutor) Handler {
	return Handler{executor: executor}
}

func (h Handler) Handle(ctx context.Context, item tasksvc.Task) error {
	var payload ExecutePayload
	if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("parse pipeline_run task payload: %w", err)
	}
	if payload.PipelineRunId == "" {
		return fmt.Errorf("pipeline_run_id is required")
	}
	return h.executor.ExecutePipelineRun(ctx, pipelinerundto.ExecutePipelineRunInput{PipelineRunId: payload.PipelineRunId, Variables: payload.Variables})
}

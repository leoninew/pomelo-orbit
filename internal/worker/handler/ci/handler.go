package ci

import (
	"context"
	"encoding/json"
	"fmt"

	taskrepo "gitee.com/leoninew/PomeloOrbit-go/internal/repository/task"
	cisvc "gitee.com/leoninew/PomeloOrbit-go/internal/service/ci"
)

type ExecutePayload struct {
	PipelineRunId string         `json:"pipeline_run_id"`
	Variables     map[string]any `json:"variables"`
}

type PipelineExecutor interface {
	ExecutePipelineRun(ctx context.Context, input cisvc.ExecutePipelineRunInput) error
}

type Handler struct {
	executor PipelineExecutor
}

func NewHandler(executor PipelineExecutor) Handler {
	return Handler{executor: executor}
}

func (h Handler) Handle(ctx context.Context, item taskrepo.Task) error {
	var payload ExecutePayload
	if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("parse ci task payload: %w", err)
	}
	if payload.PipelineRunId == "" {
		return fmt.Errorf("pipeline_run_id is required")
	}
	return h.executor.ExecutePipelineRun(ctx, cisvc.ExecutePipelineRunInput{PipelineRunId: payload.PipelineRunId, Variables: payload.Variables})
}

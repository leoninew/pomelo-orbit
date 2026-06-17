package ci

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	ciengine "backend/internal/ci"
	"backend/internal/config"
	taskrepo "backend/internal/repository/task"
)

type ExecutePayload struct {
	PipelineRunId string         `json:"pipeline_run_id"`
	Variables     map[string]any `json:"variables"`
}

type Handler struct {
	engine ciengine.Engine
}

func NewHandler(store ciengine.Store, cfg config.Config, logger *slog.Logger) Handler {
	return Handler{engine: ciengine.NewEngine(store, cfg, logger)}
}

func (h Handler) Handle(ctx context.Context, item taskrepo.Task) error {
	var payload ExecutePayload
	if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("parse ci task payload: %w", err)
	}
	if payload.PipelineRunId == "" {
		return fmt.Errorf("pipeline_run_id is required")
	}
	return h.engine.Execute(ctx, ciengine.ExecuteInput{PipelineRunId: payload.PipelineRunId, Variables: payload.Variables})
}

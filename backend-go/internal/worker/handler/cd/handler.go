package cd

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	cdengine "backend/internal/cd"
	"backend/internal/config"
	taskrepo "backend/internal/repository/task"
)

type Payload struct {
	ApplicationId string `json:"application_id"`
	DeploymentId  string `json:"deployment_id"`
}

type Handler struct {
	engine  cdengine.Engine
	restart bool
}

func NewDeployHandler(store cdengine.Store, cfg config.Config, logger *slog.Logger) Handler {
	return Handler{engine: cdengine.NewEngine(store, cfg, logger)}
}

func NewRestartHandler(store cdengine.Store, cfg config.Config, logger *slog.Logger) Handler {
	return Handler{engine: cdengine.NewEngine(store, cfg, logger), restart: true}
}

func (h Handler) Handle(ctx context.Context, item taskrepo.Task) error {
	var payload Payload
	if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("parse cd task payload: %w", err)
	}
	if payload.ApplicationId == "" || payload.DeploymentId == "" {
		return fmt.Errorf("application_id and deployment_id are required")
	}
	if h.restart {
		return h.engine.Restart(ctx, payload.ApplicationId, payload.DeploymentId)
	}
	return h.engine.Deploy(ctx, payload.ApplicationId, payload.DeploymentId)
}

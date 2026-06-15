package ci

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"backend/internal/config"
	"backend/internal/orbit"
	"backend/internal/task"
	"backend/internal/templatex"
)

type Store interface {
	PipelineRun(ctx context.Context, id string) (orbit.PipelineRun, error)
	Repository(ctx context.Context, id string) (orbit.Repository, error)
	PipelineSnapshot(ctx context.Context, id string) (orbit.PipelineSnapshot, error)
	MarkPipelineRunRunning(ctx context.Context, id string) error
	CompletePipelineRun(ctx context.Context, id string, status string, message string) error
	InsertStageRun(ctx context.Context, stage orbit.StageRun) error
	UpdateStageRun(ctx context.Context, stage orbit.StageRun) error
	InsertArtifact(ctx context.Context, projectId *string, run orbit.PipelineRun, stageName string, artifact orbit.ArtifactConfig, path string) error
}

type ExecutePayload struct {
	PipelineRunId string         `json:"pipeline_run_id"`
	Variables     map[string]any `json:"variables"`
}

type Handler struct {
	store  Store
	cfg    config.Config
	logger *slog.Logger
	runner ContainerRunner
}

func NewHandler(store Store, cfg config.Config, logger *slog.Logger) Handler {
	return Handler{store: store, cfg: cfg, logger: logger, runner: DockerRunner{}}
}

func (h Handler) Handle(ctx context.Context, item task.Task) error {
	var payload ExecutePayload
	if err := json.Unmarshal([]byte(item.PayloadJSON), &payload); err != nil {
		return fmt.Errorf("parse ci task payload: %w", err)
	}
	if payload.PipelineRunId == "" {
		return fmt.Errorf("pipeline_run_id is required")
	}
	return h.Execute(ctx, payload)
}

func (h Handler) Execute(ctx context.Context, payload ExecutePayload) error {
	run, err := h.store.PipelineRun(ctx, payload.PipelineRunId)
	if err != nil {
		return err
	}
	repo, err := h.store.Repository(ctx, run.RepositoryId)
	if err != nil {
		return err
	}
	snapshot, err := h.store.PipelineSnapshot(ctx, run.SnapshotId)
	if err != nil {
		return err
	}

	var stages []orbit.StageDefinition
	if err := json.Unmarshal([]byte(snapshot.StagesSnapshot), &stages); err != nil {
		return h.failRun(ctx, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}
	variables := payload.Variables
	if variables == nil {
		variables = pipelineRunVariables(run.VariablesSnapshot)
	}

	stages, err = resolveStages(stages, variables)
	if err != nil {
		return h.failRun(ctx, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}

	if err := createWorkspace(h.cfg.DataRoot(), repo.Code, run.Id); err != nil {
		return err
	}
	if err := h.store.MarkPipelineRunRunning(ctx, run.Id); err != nil {
		return err
	}

	executor := Executor(h)
	ok, message := executor.Execute(ctx, run, repo, variables, stages)
	current, err := h.store.PipelineRun(ctx, run.Id)
	if err != nil {
		return err
	}
	if current.Status == orbit.WorkStatusCanceled {
		return nil
	}
	if ok {
		return h.store.CompletePipelineRun(ctx, run.Id, orbit.WorkStatusRanToCompletion, "")
	}
	return h.failRun(ctx, run.Id, message)
}

func (h Handler) failRun(ctx context.Context, runId string, message string) error {
	if err := h.store.CompletePipelineRun(ctx, runId, orbit.WorkStatusFaulted, message); err != nil {
		return err
	}
	return nil
}

func resolveStages(stages []orbit.StageDefinition, variables map[string]any) ([]orbit.StageDefinition, error) {
	resolved := make([]orbit.StageDefinition, len(stages))
	copy(resolved, stages)
	for i := range resolved {
		script, err := templatex.Render(resolved[i].Script, variables)
		if err != nil {
			return nil, err
		}
		resolved[i].Script = script
		for j := range resolved[i].Artifacts {
			path, err := templatex.Render(resolved[i].Artifacts[j].Path, variables)
			if err != nil {
				return nil, err
			}
			name, err := templatex.Render(resolved[i].Artifacts[j].Name, variables)
			if err != nil {
				return nil, err
			}
			resolved[i].Artifacts[j].Path = path
			resolved[i].Artifacts[j].Name = name
		}
	}
	return resolved, nil
}

func pipelineRunVariables(value string) map[string]any {
	variables := map[string]any{}
	if strings.TrimSpace(value) == "" {
		return variables
	}
	var declarations []orbit.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &declarations); err == nil {
		for _, declaration := range declarations {
			variables[declaration.Name] = declaration.Value
		}
		return variables
	}
	_ = json.Unmarshal([]byte(value), &variables)
	return variables
}

func createWorkspace(dataRoot string, projectCode string, runId string) error {
	paths := []string{
		filepath.Join(dataRoot, "ci", projectCode, "workspace"),
		filepath.Join(dataRoot, "ci", "runs", runId, "artifacts"),
	}
	for _, path := range paths {
		if err := os.MkdirAll(path, 0o755); err != nil {
			return fmt.Errorf("create workspace path %s: %w", path, err)
		}
	}
	return nil
}

func envMap(variables map[string]any) []string {
	env := make([]string, 0, len(variables))
	for key, value := range variables {
		env = append(env, fmt.Sprintf("%s=%v", key, value))
	}
	return env
}

func commandLines(script string) string {
	lines := make([]string, 0)
	for _, line := range strings.Split(script, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

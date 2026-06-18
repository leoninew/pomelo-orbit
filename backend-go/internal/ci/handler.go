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
	"backend/internal/repository/model"
	"backend/internal/status"
	"backend/internal/templatex"
)

type Store interface {
	PipelineRun(ctx context.Context, id string) (model.PipelineRun, error)
	Repository(ctx context.Context, id string) (model.Repository, error)
	PipelineSnapshot(ctx context.Context, id string) (model.PipelineSnapshot, error)
	MarkPipelineRunRunning(ctx context.Context, id string) error
	CompletePipelineRun(ctx context.Context, id string, status string, message string) error
	InsertStageRun(ctx context.Context, stage model.StageRun) error
	UpdateStageRun(ctx context.Context, stage model.StageRun) error
	InsertArtifact(ctx context.Context, projectId *string, run model.PipelineRun, stageName string, artifact model.ArtifactConfig, path string) error
}

type ExecuteInput struct {
	PipelineRunId string
	Variables     map[string]any
}

type Engine struct {
	store  Store
	cfg    config.Config
	logger *slog.Logger
	runner ContainerRunner
}

func NewEngine(store Store, cfg config.Config, logger *slog.Logger) Engine {
	return Engine{store: store, cfg: cfg, logger: logger, runner: DockerRunner{}}
}

func (e Engine) Execute(ctx context.Context, input ExecuteInput) error {
	run, err := e.store.PipelineRun(ctx, input.PipelineRunId)
	if err != nil {
		return err
	}
	repo, err := e.store.Repository(ctx, run.RepositoryId)
	if err != nil {
		return err
	}
	snapshot, err := e.store.PipelineSnapshot(ctx, run.SnapshotId)
	if err != nil {
		return err
	}

	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(snapshot.StagesSnapshot), &stages); err != nil {
		return e.failRun(ctx, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}
	variables := input.Variables
	if variables == nil {
		variables = pipelineRunVariables(run.VariablesSnapshot)
	}

	stages, err = resolveStages(stages, variables)
	if err != nil {
		return e.failRun(ctx, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}

	if err := createWorkspace(e.cfg.DataRoot(), repo.Code, run.Id); err != nil {
		return err
	}
	if err := e.store.MarkPipelineRunRunning(ctx, run.Id); err != nil {
		return err
	}

	stageExecutor := Executor(e)
	ok, message := stageExecutor.Execute(ctx, run, repo, variables, stages)
	current, err := e.store.PipelineRun(ctx, run.Id)
	if err != nil {
		return err
	}
	if current.Status == status.WorkStatusCanceled {
		return nil
	}
	if ok {
		return e.store.CompletePipelineRun(ctx, run.Id, status.WorkStatusRanToCompletion, "")
	}
	return e.failRun(ctx, run.Id, message)
}

func (e Engine) failRun(ctx context.Context, runId string, message string) error {
	if err := e.store.CompletePipelineRun(ctx, runId, status.WorkStatusFaulted, message); err != nil {
		return err
	}
	return nil
}

func resolveStages(stages []model.StageDefinition, variables map[string]any) ([]model.StageDefinition, error) {
	resolved := make([]model.StageDefinition, len(stages))
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
	var declarations []model.VariableDeclaration
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

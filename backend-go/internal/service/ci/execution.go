package cisvc

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"backend/internal/repository/model"
	"backend/internal/status"
	"backend/internal/templatex"
)

type ExecutePipelineRunInput struct {
	PipelineRunId string
	Variables     map[string]any
}

func (s Service) ExecutePipelineRun(ctx context.Context, input ExecutePipelineRunInput) error {
	run, err := s.executionStore.PipelineRun(ctx, input.PipelineRunId)
	if err != nil {
		return err
	}
	repo, err := s.executionStore.Repository(ctx, run.RepositoryId)
	if err != nil {
		return err
	}
	snapshot, err := s.executionStore.PipelineSnapshot(ctx, run.SnapshotId)
	if err != nil {
		return err
	}

	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(snapshot.StagesSnapshot), &stages); err != nil {
		return s.failRun(ctx, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}
	variables := input.Variables
	if variables == nil {
		variables = pipelineRunExecutionVariables(run.VariablesSnapshot)
	}

	stages, err = resolveStages(stages, variables)
	if err != nil {
		return s.failRun(ctx, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}

	if err := createWorkspace(s.dataRoot, repo.Code, run.Id); err != nil {
		return err
	}
	if err := s.executionStore.MarkPipelineRunRunning(ctx, run.Id); err != nil {
		return err
	}

	stageExecutor := Executor{store: s.executionStore, dataRoot: s.dataRoot, logger: s.logger, runner: s.runner}
	ok, message := stageExecutor.Execute(ctx, run, repo, variables, stages)
	current, err := s.executionStore.PipelineRun(ctx, run.Id)
	if err != nil {
		return err
	}
	if current.Status == status.WorkStatusCanceled {
		return nil
	}
	if ok {
		return s.executionStore.CompletePipelineRun(ctx, run.Id, status.WorkStatusRanToCompletion, "")
	}
	return s.failRun(ctx, run.Id, message)
}

func (s Service) failRun(ctx context.Context, runId string, message string) error {
	if err := s.executionStore.CompletePipelineRun(ctx, runId, status.WorkStatusFaulted, message); err != nil {
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

func pipelineRunExecutionVariables(value string) map[string]any {
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

package cisvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	cidto "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/dto"
	civariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/ci/rule/civariable"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (s Service) ExecutePipelineRun(ctx context.Context, input cidto.ExecutePipelineRunInput) error {
	run, err := s.executionStore.PipelineRun(ctx, input.PipelineRunID)
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
	template, err := s.executionStore.PipelineTemplate(ctx, run.TemplateId)
	if err != nil {
		return err
	}
	variables, err := s.pipelineRunExecutionVariables(repo, template, snapshot, run)
	if err != nil {
		return s.failRun(ctx, run.Id, fmt.Sprintf("Variable resolution failed: %v", err))
	}

	stages, err = resolveStages(stages, variables)
	if err != nil {
		return s.failRun(ctx, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}

	if err := s.workspace.CreateRunDirectories(repo.Code, run.Id); err != nil {
		return err
	}
	if err := s.executionStore.MarkPipelineRunRunning(ctx, run.Id); err != nil {
		return err
	}

	stageExecutor := Executor{store: s.executionStore, workspace: s.workspace, logStore: s.executionLogStore, secretKey: s.secretKey, logger: s.logger, runner: s.runner}
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

func (s Service) pipelineRunExecutionVariables(repo model.Repository, template model.PipelineTemplate, snapshot model.PipelineSnapshot, run model.PipelineRun) (map[string]any, error) {
	declarations, err := civariable.CompleteSnapshotVariableDeclarations(snapshot, template)
	if err != nil {
		return nil, err
	}
	return civariable.BuildRuntimeVariables(repo, template, run.TriggerRef, pipelineRunRuntimeOverrides(run.VariablesSnapshot), declarations)
}

func pipelineRunRuntimeOverrides(value string) map[string]string {
	overrides := map[string]string{}
	if strings.TrimSpace(value) == "" {
		return overrides
	}
	var declarations []model.VariableDeclaration
	if err := json.Unmarshal([]byte(value), &declarations); err == nil {
		for _, declaration := range declarations {
			if civariable.IsPipelineTemplateBuiltinVariable(declaration.Name) || !civariable.HasRuntimeValue(declaration.Value) {
				continue
			}
			overrides[declaration.Name] = fmt.Sprint(declaration.Value)
		}
		return overrides
	}
	var legacy map[string]any
	if err := json.Unmarshal([]byte(value), &legacy); err == nil {
		for name, value := range legacy {
			overrides[name] = fmt.Sprint(value)
		}
	}
	return overrides
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
	for line := range strings.SplitSeq(script, "\n") {
		if strings.TrimSpace(line) != "" {
			lines = append(lines, line)
		}
	}
	return strings.Join(lines, "\n")
}

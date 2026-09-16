package pipelinerunsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

func (s Service) ExecutePipelineRun(ctx context.Context, input pipelinerundto.ExecutePipelineRunInput) error {
	projectId := strings.TrimSpace(input.ProjectId)
	runId := strings.TrimSpace(input.PipelineRunId)
	if projectId == "" || runId == "" {
		return fmt.Errorf("project_id and pipeline_run_id are required")
	}
	begun, err := s.executionStore.BeginPipelineRun(ctx, projectId, runId)
	if err != nil {
		return err
	}
	if !begun {
		return nil
	}
	run, err := s.executionStore.PipelineRun(ctx, projectId, runId)
	if err != nil {
		return err
	}
	repo, err := s.executionStore.Repository(ctx, projectId, run.RepositoryId)
	if err != nil {
		return err
	}
	workspace, err := s.workspaceForProject(ctx, projectId)
	if err != nil {
		return s.failRun(ctx, projectId, run.Id, err.Error())
	}
	snapshot, err := s.executionStore.PipelineSnapshot(ctx, projectId, run.SnapshotId)
	if err != nil {
		return err
	}

	var stages []model.StageDefinition
	if err := json.Unmarshal([]byte(snapshot.StagesSnapshot), &stages); err != nil {
		return s.failRun(ctx, projectId, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}
	variables, err := s.pipelineRunExecutionVariables(run)
	if err != nil {
		return s.failRun(ctx, projectId, run.Id, fmt.Sprintf("Variable resolution failed: %v", err))
	}

	stages, err = resolveStages(stages, variables)
	if err != nil {
		return s.failRun(ctx, projectId, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}

	if err := workspace.CreateRunDirectories(repo.Code, run.Id); err != nil {
		return s.failRun(ctx, projectId, run.Id, err.Error())
	}
	stageRuns, err := s.executionStore.ListPipelineStageRuns(ctx, projectId, run.Id)
	if err != nil {
		return s.failRun(ctx, projectId, run.Id, fmt.Sprintf("Load stage runs failed: %v", err))
	}

	executionCtx, cancel := s.pipelineExecutionContext(ctx, projectId, run.Id)
	defer cancel()
	stageExecutor := Executor{store: s.executionStore, versionForker: s.versionForker, transactionRunner: s.transactionRunner, workspace: workspace, logStore: s.executionLogStore, secretKey: s.secretKey, logger: s.logger, executionTimeout: s.executionTimeout, runner: s.runner, localSource: s.localSource}
	ok, message := stageExecutor.Execute(ctx, executionCtx, projectId, run, repo, variables, stages, stageRunByStageId(stageRuns))
	current, err := s.executionStore.PipelineRun(ctx, projectId, run.Id)
	if err != nil {
		return err
	}
	if current.Status == status.WorkStatusCanceled {
		return nil
	}
	if ok {
		return s.completeRun(ctx, projectId, run.Id, status.WorkStatusRanToCompletion, "")
	}
	return s.failRun(ctx, projectId, run.Id, message)
}

func (s Service) failRun(ctx context.Context, projectId, runId string, message string) error {
	return s.completeRun(ctx, projectId, runId, status.WorkStatusFaulted, message)
}

func (s Service) completeRun(ctx context.Context, projectId, runId, statusValue, message string) error {
	_, err := s.executionStore.CompletePipelineRun(ctx, projectId, runId, statusValue, message)
	return err
}

func (s Service) pipelineExecutionContext(ctx context.Context, projectId string, runId string) (context.Context, context.CancelFunc) {
	timedCtx, cancelTimeout := context.WithTimeout(ctx, s.executionTimeout)
	monitoredCtx, cancelMonitored := context.WithCancel(timedCtx)
	done := make(chan struct{})
	interval := s.pollInterval
	if interval <= 0 {
		interval = time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-timedCtx.Done():
				return
			case <-ticker.C:
				run, err := s.executionStore.PipelineRun(ctx, projectId, runId)
				if err == nil && run.Status == status.WorkStatusCanceled {
					cancelMonitored()
					return
				}
			}
		}
	}()
	return monitoredCtx, func() {
		close(done)
		cancelMonitored()
		cancelTimeout()
	}
}

func stageRunByStageId(stageRuns []model.PipelineStageRun) map[string]model.PipelineStageRun {
	byStageId := make(map[string]model.PipelineStageRun, len(stageRuns))
	for _, stageRun := range stageRuns {
		byStageId[stageRun.StageId] = stageRun
	}
	return byStageId
}

func (s Service) pipelineRunExecutionVariables(run model.PipelineRun) (pipelinevariable.RuntimeVariables, error) {
	_, variables, err := pipelinevariable.UnmarshalRuntimeVariableSnapshot(run.VariablesSnapshot)
	if err != nil {
		return pipelinevariable.RuntimeVariables{}, err
	}
	if ref, ok := variables.Global["repository_ref"].(string); !ok || ref != run.RepositoryRef {
		return pipelinevariable.RuntimeVariables{}, fmt.Errorf("repository_ref does not match pipeline run")
	}
	return variables, nil
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

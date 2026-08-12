package pipelinerunsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	pipelinevariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/rule/pipelinevariable"
	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (s Service) ExecutePipelineRun(ctx context.Context, input pipelinerundto.ExecutePipelineRunInput) error {
	begun, err := s.executionStore.BeginPipelineRun(ctx, input.PipelineRunId)
	if err != nil {
		return err
	}
	if !begun {
		return nil
	}
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
	variables, err := s.pipelineRunExecutionVariables(run)
	if err != nil {
		return s.failRun(ctx, run.Id, fmt.Sprintf("Variable resolution failed: %v", err))
	}

	stages, err = resolveStages(stages, variables)
	if err != nil {
		return s.failRun(ctx, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}

	if err := s.workspace.CreateRunDirectories(repo.Code, run.Id); err != nil {
		return s.failRun(ctx, run.Id, err.Error())
	}
	stageRuns, err := s.executionStore.ListPipelineStageRuns(ctx, run.Id)
	if err != nil {
		return s.failRun(ctx, run.Id, fmt.Sprintf("Load stage runs failed: %v", err))
	}

	executionCtx, cancel := s.pipelineExecutionContext(ctx, run.Id)
	defer cancel()
	stageExecutor := Executor{store: s.executionStore, versionForker: s.versionForker, transactionRunner: s.transactionRunner, workspace: s.workspace, logStore: s.executionLogStore, secretKey: s.secretKey, logger: s.logger, executionTimeout: s.executionTimeout, runner: s.runner, localSource: s.localSource}
	ok, message := stageExecutor.Execute(ctx, executionCtx, run, repo, variables, stages, stageRunByStageID(stageRuns))
	current, err := s.executionStore.PipelineRun(ctx, run.Id)
	if err != nil {
		return err
	}
	if current.Status == status.WorkStatusCanceled {
		return nil
	}
	if ok {
		return s.completeRun(ctx, run.Id, status.WorkStatusRanToCompletion, "")
	}
	return s.failRun(ctx, run.Id, message)
}

func (s Service) failRun(ctx context.Context, runId string, message string) error {
	return s.completeRun(ctx, runId, status.WorkStatusFaulted, message)
}

func (s Service) completeRun(ctx context.Context, runID, statusValue, message string) error {
	_, err := s.executionStore.CompletePipelineRun(ctx, runID, statusValue, message)
	return err
}

func (s Service) pipelineExecutionContext(ctx context.Context, runID string) (context.Context, context.CancelFunc) {
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
				run, err := s.executionStore.PipelineRun(ctx, runID)
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

func stageRunByStageID(stageRuns []model.PipelineStageRun) map[string]model.PipelineStageRun {
	byStageID := make(map[string]model.PipelineStageRun, len(stageRuns))
	for _, stageRun := range stageRuns {
		byStageID[stageRun.StageId] = stageRun
	}
	return byStageID
}

func (s Service) pipelineRunExecutionVariables(run model.PipelineRun) (map[string]any, error) {
	_, variables, err := pipelinevariable.UnmarshalRuntimeVariableSnapshot(run.VariablesSnapshot)
	if err != nil {
		return nil, err
	}
	if ref, ok := variables["repository_ref"].(string); !ok || ref != run.RepositoryRef {
		return nil, fmt.Errorf("repository_ref does not match pipeline run")
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

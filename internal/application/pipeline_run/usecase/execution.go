package pipelinerunsvc

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"strings"
	"time"

	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
)

const pipelineRunCompletionTimeout = 5 * time.Second

func (s ExecutionService) ExecutePipelineRun(ctx context.Context, input pipelinerundto.ExecutePipelineRunInput) (executionErr error) {
	projectId := input.ProjectId
	runId := input.PipelineRunId
	if projectId == "" || runId == "" {
		return fmt.Errorf("project_id and pipeline_run_id are required")
	}
	if s.executionTimeout <= 0 {
		return errors.New("pipeline execution timeout must be configured")
	}
	// The worker lease exceeds this budget, including target and workspace setup.
	ctx, stop := context.WithTimeout(ctx, s.executionTimeout)
	defer stop()
	begun, err := s.executionStore.BeginPipelineRun(ctx, projectId, runId)
	if err != nil {
		return err
	}
	if !begun {
		run, err := s.executionStore.PipelineRun(ctx, projectId, runId)
		if err != nil {
			return err
		}
		if status.WorkStatusIsComplete(run.Status) {
			return nil
		}
		if run.Status == status.WorkStatusRunning && input.Attempt > 1 {
			// A reclaimed task cannot safely replay external stage side effects.
			return s.failRun(ctx, projectId, runId, "Previous pipeline execution was interrupted; retry the pipeline run")
		}
		return fmt.Errorf("pipeline run %s cannot begin with status %s", runId, run.Status)
	}
	defer func() {
		if executionErr != nil {
			executionErr = errors.Join(executionErr, s.failRun(ctx, projectId, runId, executionErr.Error()))
		}
	}()
	run, err := s.executionStore.PipelineRun(ctx, projectId, runId)
	if err != nil {
		return err
	}
	target, err := s.resolveQueuedTarget(ctx, projectId, run)
	if err != nil {
		return s.failRun(ctx, projectId, run.Id, err.Error())
	}
	repo, err := s.executionStore.Repository(ctx, run.RepositoryId)
	if err != nil {
		return s.failRun(ctx, projectId, run.Id, fmt.Sprintf("Load repository failed: %v", err))
	}
	if target.Environment.IsSSH() {
		if repo.RepositoryType == model.RepositoryTypeLocalDirectory {
			return s.failRun(ctx, projectId, run.Id, "Local directory repositories cannot run on SSH environments; retry with an HTTPS Git repository")
		}
		remoteURL, parseErr := url.Parse(repo.RepositoryUrl)
		if repo.RepositoryType != model.RepositoryTypeRemoteGit || parseErr != nil || remoteURL.Scheme != "https" || remoteURL.Hostname() == "" {
			return s.failRun(ctx, projectId, run.Id, "SSH pipeline runs require an HTTPS Git repository URL")
		}
	}
	runtime, err := s.runtimeForTarget(ctx, target)
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
	variables, err := pipelineRunExecutionVariables(run)
	if err != nil {
		return s.failRun(ctx, projectId, run.Id, fmt.Sprintf("Variable resolution failed: %v", err))
	}

	stages, err = resolveStages(stages, variables)
	if err != nil {
		return s.failRun(ctx, projectId, run.Id, fmt.Sprintf("Stage resolution failed: %v", err))
	}

	if err := runtime.workspace.CreateRunDirectories(repo.Code, run.Id); err != nil {
		return s.failRun(ctx, projectId, run.Id, err.Error())
	}
	stageRuns, err := s.executionStore.ListPipelineStageRuns(ctx, projectId, run.Id)
	if err != nil {
		return s.failRun(ctx, projectId, run.Id, fmt.Sprintf("Load stage runs failed: %v", err))
	}

	executionCtx, cancel := s.pipelineExecutionContext(ctx, projectId, run.Id)
	defer cancel()
	stageExecutor := Executor{store: s.executionStore, versionForker: s.versionForker, transactionRunner: s.transactionRunner, workspace: runtime.workspace, logStore: runtime.writer, secretKey: s.secretKey, logger: s.logger, executionTimeout: s.executionTimeout, runner: runtime.runner, localSource: s.localSource}
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

func (s ExecutionService) failRun(ctx context.Context, projectId, runId string, message string) error {
	return s.completeRun(ctx, projectId, runId, status.WorkStatusFaulted, message)
}

func (s ExecutionService) completeRun(ctx context.Context, projectId, runId, statusValue, message string) error {
	cleanupCtx, stop := context.WithTimeout(context.WithoutCancel(ctx), pipelineRunCompletionTimeout)
	defer stop()
	_, err := s.executionStore.CompletePipelineRun(cleanupCtx, projectId, runId, statusValue, message)
	return err
}

func (s ExecutionService) pipelineExecutionContext(ctx context.Context, projectId string, runId string) (context.Context, context.CancelFunc) {
	monitoredCtx, cancelMonitored := context.WithCancel(ctx)
	done := make(chan struct{})
	interval := s.pollInterval
	if interval <= 0 || interval > time.Second {
		interval = time.Second
	}
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-done:
				return
			case <-ctx.Done():
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
	}
}

func stageRunByStageId(stageRuns []model.PipelineStageRun) map[string]model.PipelineStageRun {
	byStageId := make(map[string]model.PipelineStageRun, len(stageRuns))
	for _, stageRun := range stageRuns {
		byStageId[stageRun.StageId] = stageRun
	}
	return byStageId
}

func pipelineRunExecutionVariables(run model.PipelineRun) (pipelinevariable.RuntimeVariables, error) {
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

package pipelinerunsvc

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	pipelinevariable "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline/rule/pipelinevariable"
	pipelinerundto "gitee.com/leoninew/PomeloOrbit-go/internal/application/pipeline_run/dto"
	status "gitee.com/leoninew/PomeloOrbit-go/internal/common/constant"
	"gitee.com/leoninew/PomeloOrbit-go/internal/model"
)

func (s Service) ExecutePipelineRun(ctx context.Context, input pipelinerundto.ExecutePipelineRunInput) error {
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
	pipeline := model.Pipeline{
		Id: snapshot.PipelineId, ProjectId: snapshot.ProjectId, Kind: model.PipelineKindApplication,
		Name: snapshot.PipelineName, Version: snapshot.PipelineVersion,
		VariableDeclarations: snapshot.VariablesSnapshot,
	}
	variables, err := s.pipelineRunExecutionVariables(repo, pipeline, snapshot, run)
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

	executionCtx, cancel := context.WithTimeout(ctx, s.executionTimeout)
	defer cancel()
	stageExecutor := Executor{store: s.executionStore, versionForker: s.versionForker, transactionRunner: s.transactionRunner, workspace: s.workspace, logStore: s.executionLogStore, secretKey: s.secretKey, logger: s.logger, executionTimeout: s.executionTimeout, runner: s.runner, localSource: s.localSource}
	ok, message := stageExecutor.Execute(ctx, executionCtx, run, repo, variables, stages)
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

func (s Service) pipelineRunExecutionVariables(repo model.Repository, pipeline model.Pipeline, snapshot model.PipelineSnapshot, run model.PipelineRun) (map[string]any, error) {
	declarations, err := pipelinevariable.CompleteSnapshotVariableDeclarations(snapshot, pipeline)
	if err != nil {
		return nil, err
	}
	overrides, err := pipelineRunRuntimeOverrides(run.VariablesSnapshot)
	if err != nil {
		return nil, err
	}
	sourceRepo := repo
	if sourceRepo.RepositoryType == model.RepositoryTypeLocalDirectory {
		sourceRepo.RepositoryUrl = "file:///source"
	}
	variables, err := pipelinevariable.BuildRuntimeVariables(sourceRepo, pipeline, run.TriggerRef, overrides, declarations)
	if err != nil {
		return nil, err
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

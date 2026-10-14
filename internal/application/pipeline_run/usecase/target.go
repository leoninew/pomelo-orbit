package pipelinerunsvc

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	environmentport "github.com/leoninew/pomelo-orbit/internal/application/environment/port"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func applyRunTargetSnapshot(run *model.PipelineRun, target environmentport.Target) {
	environment := target.Environment
	run.EnvironmentId = &environment.Id
	run.EnvironmentTargetType = &environment.TargetType
	run.EnvironmentTargetRevision = &environment.TargetRevision
	if environment.IsSSH() {
		run.SSHCredentialId = &environment.SSH.CredentialId
		run.SSHCredentialRevision = &environment.SSH.CredentialRevision
	}
}

func verifyRunTargetSnapshot(run model.PipelineRun, environment model.Environment) error {
	if run.EnvironmentId == nil || *run.EnvironmentId == "" ||
		run.EnvironmentTargetType == nil || run.EnvironmentTargetRevision == nil || *run.EnvironmentTargetRevision < 1 {
		return errors.New("pipeline run has no environment target snapshot; retry run")
	}
	if *run.EnvironmentId != environment.Id || *run.EnvironmentTargetType != environment.TargetType ||
		*run.EnvironmentTargetRevision != environment.TargetRevision ||
		(environment.IsLocal() && (run.SSHCredentialId != nil || run.SSHCredentialRevision != nil)) ||
		(environment.IsSSH() && (run.SSHCredentialId == nil || run.SSHCredentialRevision == nil ||
			*run.SSHCredentialId != environment.SSH.CredentialId || *run.SSHCredentialRevision != environment.SSH.CredentialRevision)) ||
		(!environment.IsLocal() && !environment.IsSSH()) {
		return errors.New("project environment changed after pipeline run was queued; retry run")
	}
	return nil
}

func resolveProjectTarget(ctx context.Context, resolver environmentport.TargetResolver, projectId string) (environmentport.Target, error) {
	if resolver == nil {
		return environmentport.Target{}, apperror.New(apperror.KindInternal, "pipeline target resolver is not configured")
	}
	return resolver.ResolveProjectTarget(ctx, projectId)
}

func runEnvironment(ctx context.Context, environments repository.EnvironmentStore, projectId string, run model.PipelineRun) (model.Environment, error) {
	if environments == nil {
		return model.Environment{}, errors.New("pipeline environment store is not configured")
	}
	environment, err := environments.EnvironmentByProject(ctx, projectId)
	if errors.Is(err, repository.ErrNotFound) {
		return model.Environment{}, errors.New("project environment no longer exists; pipeline run files cannot be located")
	}
	if err != nil {
		return model.Environment{}, fmt.Errorf("load pipeline run environment: %w", err)
	}
	if run.EnvironmentId == nil && run.EnvironmentTargetType == nil && run.EnvironmentTargetRevision == nil &&
		run.SSHCredentialId == nil && run.SSHCredentialRevision == nil {
		if !environment.IsLocal() || environment.CreatedAt.After(run.CreatedAt) ||
			environment.UpdatedAt.IsZero() || !environment.UpdatedAt.Add(time.Second).Before(run.CreatedAt) {
			return model.Environment{}, errors.New("historical pipeline run target cannot be verified; files cannot be located safely")
		}
		return environment, nil
	}
	if err := verifyRunTargetSnapshot(run, environment); err != nil {
		return model.Environment{}, err
	}
	return environment, nil
}

func (s ExecutionService) resolveQueuedTarget(ctx context.Context, projectId string, run model.PipelineRun) (environmentport.Target, error) {
	environment, err := runEnvironment(ctx, s.environments, projectId, run)
	if err != nil {
		return environmentport.Target{}, err
	}
	if run.EnvironmentId == nil {
		return environmentport.Target{}, errors.New("historical pipeline run cannot execute without a target snapshot; retry run")
	}
	target, err := resolveProjectTarget(ctx, s.targetResolver, projectId)
	if err != nil {
		return environmentport.Target{}, fmt.Errorf("project environment is not ready for queued pipeline run: %w; retry run", err)
	}
	if err := verifyRunTargetSnapshot(run, target.Environment); err != nil {
		return environmentport.Target{}, err
	}
	if environment.Id != target.Environment.Id {
		return environmentport.Target{}, errors.New("project environment changed after pipeline run was queued; retry run")
	}
	return target, nil
}

func workspaceForEnvironment(ctx context.Context, workspace pipelinerunport.Workspace, environment model.Environment) (pipelinerunport.Workspace, error) {
	if !environment.IsLocal() {
		return nil, errors.New("remote pipeline workspace is not available until SSH CI execution is installed")
	}
	if strings.TrimSpace(environment.WorkspaceRoot) == "" {
		return nil, errors.New("project environment workspace root is required")
	}
	resolver, ok := workspace.(pipelinerunport.TargetWorkspaceResolver)
	if !ok {
		return nil, errors.New("target-aware pipeline workspace is not configured")
	}
	return resolver.WorkspaceForTarget(ctx, environment)
}

type runRuntime struct {
	workspace pipelinerunport.Workspace
	runner    pipelinerunport.ContainerRunner
	writer    pipelinerunport.ExecutionLogStore
}

func (s ExecutionService) runtimeForTarget(ctx context.Context, target environmentport.Target) (runRuntime, error) {
	if target.Environment.IsSSH() {
		if s.remoteRuntime == nil {
			return runRuntime{}, errors.New("SSH pipeline runtime is not configured")
		}
		workspace, runner, logs, err := s.remoteRuntime.RuntimeForTarget(ctx, target)
		if err != nil {
			return runRuntime{}, err
		}
		return runRuntime{workspace: workspace, runner: runner, writer: logs}, nil
	}
	workspace, err := workspaceForEnvironment(ctx, s.workspace, target.Environment)
	return runRuntime{workspace: workspace, runner: s.runner, writer: s.executionLogStore}, err
}

func (s Service) runtimeForRun(ctx context.Context, projectId string, run model.PipelineRun) (runFiles, error) {
	environment, err := runEnvironment(ctx, s.environments, projectId, run)
	if err != nil {
		return runFiles{}, err
	}
	if environment.IsSSH() {
		target, err := resolveProjectTarget(ctx, s.targetResolver, projectId)
		if err != nil {
			return runFiles{}, err
		}
		if err := verifyRunTargetSnapshot(run, target.Environment); err != nil {
			return runFiles{}, err
		}
		if s.remoteRuntime == nil {
			return runFiles{}, errors.New("SSH pipeline runtime is not configured")
		}
		workspace, _, logs, err := s.remoteRuntime.RuntimeForTarget(ctx, target)
		return runFiles{workspace: workspace, reader: logs}, err
	}
	workspace, err := workspaceForEnvironment(ctx, s.workspace, environment)
	return runFiles{workspace: workspace, reader: s.logStore}, err
}

func (s Service) workspaceForRun(ctx context.Context, projectId string, run model.PipelineRun) (pipelinerunport.Workspace, error) {
	runtime, err := s.runtimeForRun(ctx, projectId, run)
	return runtime.workspace, err
}

// runFiles is the read/cleanup resource view; it cannot execute a stage.
type runFiles struct {
	workspace pipelinerunport.Workspace
	reader    pipelinerunport.LogReader
}

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

func (s Service) resolveProjectTarget(ctx context.Context, projectId string) (environmentport.Target, error) {
	if s.targetResolver == nil {
		return environmentport.Target{}, apperror.New(apperror.KindInternal, "pipeline target resolver is not configured")
	}
	return s.targetResolver.ResolveProjectTarget(ctx, projectId)
}

func (s Service) runEnvironment(ctx context.Context, projectId string, run model.PipelineRun) (model.Environment, error) {
	if s.environments == nil {
		return model.Environment{}, errors.New("pipeline environment store is not configured")
	}
	environment, err := s.environments.EnvironmentByProject(ctx, projectId)
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

func (s Service) resolveQueuedTarget(ctx context.Context, projectId string, run model.PipelineRun) (environmentport.Target, error) {
	environment, err := s.runEnvironment(ctx, projectId, run)
	if err != nil {
		return environmentport.Target{}, err
	}
	if run.EnvironmentId == nil {
		return environmentport.Target{}, errors.New("historical pipeline run cannot execute without a target snapshot; retry run")
	}
	target, err := s.resolveProjectTarget(ctx, projectId)
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

func (s Service) workspaceForEnvironment(ctx context.Context, environment model.Environment) (pipelinerunport.Workspace, error) {
	if !environment.IsLocal() {
		return nil, errors.New("remote pipeline workspace is not available until SSH CI execution is installed")
	}
	if strings.TrimSpace(environment.WorkspaceRoot) == "" {
		return nil, errors.New("project environment workspace root is required")
	}
	resolver, ok := s.workspace.(pipelinerunport.TargetWorkspaceResolver)
	if !ok {
		return nil, errors.New("target-aware pipeline workspace is not configured")
	}
	return resolver.WorkspaceForTarget(ctx, environment)
}

type runRuntime struct {
	workspace pipelinerunport.Workspace
	runner    pipelinerunport.ContainerRunner
	reader    pipelinerunport.LogReader
	writer    pipelinerunport.ExecutionLogStore
}

func (s Service) runtimeForTarget(ctx context.Context, target environmentport.Target) (runRuntime, error) {
	if target.Environment.IsSSH() {
		if s.remoteRuntime == nil {
			return runRuntime{}, errors.New("SSH pipeline runtime is not configured")
		}
		workspace, runner, logs, err := s.remoteRuntime.RuntimeForTarget(ctx, target)
		if err != nil {
			return runRuntime{}, err
		}
		return runRuntime{workspace: workspace, runner: runner, reader: logs, writer: logs}, nil
	}
	return s.runtimeForEnvironment(ctx, target.Environment)
}

func (s Service) runtimeForEnvironment(ctx context.Context, environment model.Environment) (runRuntime, error) {
	workspace, err := s.workspaceForEnvironment(ctx, environment)
	if err != nil {
		return runRuntime{}, err
	}
	return runRuntime{workspace: workspace, runner: s.runner, reader: s.logStore, writer: s.executionLogStore}, nil
}

func (s Service) runtimeForRun(ctx context.Context, projectId string, run model.PipelineRun) (runRuntime, error) {
	environment, err := s.runEnvironment(ctx, projectId, run)
	if err != nil {
		return runRuntime{}, err
	}
	if environment.IsSSH() {
		target, err := s.resolveProjectTarget(ctx, projectId)
		if err != nil {
			return runRuntime{}, err
		}
		if err := verifyRunTargetSnapshot(run, target.Environment); err != nil {
			return runRuntime{}, err
		}
		return s.runtimeForTarget(ctx, target)
	}
	return s.runtimeForEnvironment(ctx, environment)
}

func (s Service) workspaceForRun(ctx context.Context, projectId string, run model.PipelineRun) (pipelinerunport.Workspace, error) {
	runtime, err := s.runtimeForRun(ctx, projectId, run)
	return runtime.workspace, err
}

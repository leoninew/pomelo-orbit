package pipelinerunsvc

import (
	"context"
	"fmt"

	logstream "github.com/leoninew/pomelo-orbit/internal/application/logstream/usecase"
	apperror "github.com/leoninew/pomelo-orbit/internal/common/errors"
)

func (s Service) OpenStageLogStream(ctx context.Context, userId, projectId, runId, stageId, cursor string) (logstream.Subscription, error) {
	run, err := s.loadPipelineRunForUser(ctx, userId, projectId, runId)
	if err != nil {
		return logstream.Subscription{}, err
	}
	stage, err := s.store.PipelineStageRun(ctx, projectId, stageId)
	if err != nil || stage.PipelineRunId != run.Id {
		return logstream.Subscription{}, apperror.New(apperror.KindNotFound, "Pipeline stage not found")
	}
	environment, err := s.runEnvironment(ctx, projectId, run)
	if err != nil {
		return logstream.Subscription{}, apperror.Wrap(apperror.KindConflict, "Pipeline log target changed", err)
	}
	runtime, err := s.runtimeForRun(ctx, projectId, run)
	if err != nil {
		return logstream.Subscription{}, err
	}
	source := logstream.SourceId(fmt.Sprintf("ci:%s:%s:%s:%s:%d:%s", projectId, run.Id, stage.Id, environment.Id, environment.TargetRevision, environment.WorkspaceRoot))
	reader, err := runtime.reader.OpenReader(ctx, runtime.workspace.StageLogPath(run.Id, stage.Id))
	if err != nil {
		return logstream.Subscription{}, apperror.Wrap(apperror.KindUnavailable, "Failed to open pipeline logs", err)
	}
	return logstream.File(source, cursor, stage.Status, reader, func(ctx context.Context) (string, error) {
		currentRun, err := s.loadPipelineRunForUser(ctx, userId, projectId, runId)
		if err != nil {
			return "", err
		}
		currentEnvironment, err := s.runEnvironment(ctx, projectId, currentRun)
		if err != nil || currentEnvironment.Id != environment.Id || currentEnvironment.TargetRevision != environment.TargetRevision {
			return "", apperror.New(apperror.KindConflict, "Pipeline log target changed")
		}
		if environment.IsSSH() {
			target, err := s.resolveProjectTarget(ctx, projectId)
			if err != nil {
				return "", err
			}
			if err := verifyRunTargetSnapshot(currentRun, target.Environment); err != nil {
				return "", apperror.New(apperror.KindConflict, "Pipeline log target changed")
			}
		}
		currentStage, err := s.store.PipelineStageRun(ctx, projectId, stageId)
		if err != nil || currentStage.PipelineRunId != run.Id {
			return "", apperror.New(apperror.KindNotFound, "Pipeline stage not found")
		}
		return currentStage.Status, nil
	})
}

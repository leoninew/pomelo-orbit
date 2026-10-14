package pipelinerunsvc

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"strings"
	"testing"
	"time"

	pipelinevariable "github.com/leoninew/pomelo-orbit/internal/application/pipeline/rule/pipelinevariable"
	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestExecutionReadFailuresCompleteRunWithBoundedCleanup(t *testing.T) {
	readFailure := errors.New("storage read failed")
	cleanupFailure := errors.New("storage completion failed")
	for _, test := range []struct {
		name        string
		failRead    int
		snapshotErr error
		cancel      bool
		completeErr error
	}{
		{name: "initial run read", failRead: 1},
		{name: "snapshot read", snapshotErr: readFailure},
		{name: "final run read", failRead: 2},
		{name: "worker cancellation", failRead: 1, cancel: true},
		{name: "cleanup failure propagates", failRead: 1, completeErr: cleanupFailure},
	} {
		t.Run(test.name, func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()
			environment := readyLocalTarget()
			variables, err := pipelinevariable.MarshalRuntimeVariableSnapshot(pipelinevariable.RuntimeVariables{Global: map[string]any{"repository_ref": "main"}}, []model.VariableDeclaration{{Name: "repository_ref", Value: "main", Source: "runtime", Editable: true}})
			if err != nil {
				t.Fatal(err)
			}
			store := &executionFailureStore{failRead: test.failRead, readErr: readFailure, snapshotErr: test.snapshotErr, completeErr: test.completeErr,
				run: model.PipelineRun{Id: "run-1", RepositoryId: "repo-1", RepositoryRef: "main", VariablesSnapshot: variables, Status: status.WorkStatusRunning}}
			applyRunTargetSnapshot(&store.run, structTarget(environment))
			if test.cancel {
				store.beforeRead = cancel
			}
			service := ExecutionService{executionStore: store, environments: &runTargetEnvironmentStore{environment: environment}, targetResolver: directTriggerTargetResolver{environment: environment}, workspace: executionWorkspace{}, executionTimeout: time.Minute, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
			err = service.ExecutePipelineRun(ctx, pipelinerundto.ExecutePipelineRunInput{ProjectId: "project-1", PipelineRunId: "run-1", Attempt: 1})
			if !errors.Is(err, readFailure) {
				t.Fatalf("execution error=%v", err)
			}
			if test.completeErr != nil && !errors.Is(err, cleanupFailure) {
				t.Fatalf("cleanup failure lost: %v", err)
			}
			if store.completedStatus != status.WorkStatusFaulted || !strings.Contains(store.completedMessage, readFailure.Error()) {
				t.Fatalf("completion=%q/%q", store.completedStatus, store.completedMessage)
			}
			if store.cleanupCanceled || !store.cleanupBounded {
				t.Fatal("cleanup must have its own live, bounded context")
			}
		})
	}
}

type executionFailureStore struct {
	pipelineExecutionStore
	run                               model.PipelineRun
	readCount, failRead               int
	readErr, snapshotErr, completeErr error
	beforeRead                        func()
	completedStatus, completedMessage string
	cleanupCanceled, cleanupBounded   bool
	completedStage                    model.PipelineStageRun
}

type executionWorkspace struct{ pipelinerunport.Workspace }

func (executionWorkspace) CreateRunDirectories(string, string) error { return nil }
func (w executionWorkspace) WorkspaceForTarget(context.Context, model.Environment) (pipelinerunport.Workspace, error) {
	return w, nil
}

func (s *executionFailureStore) CompletePipelineStageRun(ctx context.Context, _ string, stage model.PipelineStageRun) (bool, error) {
	s.completedStage = stage
	s.cleanupCanceled = ctx.Err() != nil
	_, s.cleanupBounded = ctx.Deadline()
	return true, nil
}

func TestStageFailurePersistsAfterExecutionCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	store := &executionFailureStore{}
	executor := Executor{store: store, logger: slog.New(slog.NewTextHandler(io.Discard, nil))}
	if executor.failStage(ctx, "project-1", model.PipelineStageRun{Id: "stage-1"}, "execution canceled") {
		t.Fatal("canceled stage succeeded")
	}
	if store.completedStage.Status != status.WorkStatusFaulted || store.cleanupCanceled || !store.cleanupBounded {
		t.Fatalf("stage cleanup failed: %+v", store.completedStage)
	}
}

func (*executionFailureStore) BeginPipelineRun(context.Context, string, string) (bool, error) {
	return true, nil
}
func (s *executionFailureStore) PipelineRun(context.Context, string, string) (model.PipelineRun, error) {
	s.readCount++
	if s.beforeRead != nil {
		s.beforeRead()
	}
	if s.readCount == s.failRead {
		return model.PipelineRun{}, s.readErr
	}
	return s.run, nil
}
func (*executionFailureStore) Repository(context.Context, string) (model.Repository, error) {
	return model.Repository{Id: "repo-1", Code: "repo"}, nil
}
func (s *executionFailureStore) PipelineSnapshot(context.Context, string, string) (model.PipelineSnapshot, error) {
	return model.PipelineSnapshot{StagesSnapshot: "[]"}, s.snapshotErr
}
func (*executionFailureStore) ListPipelineStageRuns(context.Context, string, string) ([]model.PipelineStageRun, error) {
	return nil, nil
}
func (*executionFailureStore) PipelineRunVersionBinding(context.Context, string, string) (model.PipelineRunVersionBinding, error) {
	return model.PipelineRunVersionBinding{}, repository.ErrNotFound
}
func (s *executionFailureStore) CompletePipelineRun(ctx context.Context, _, _, statusValue, message string) (bool, error) {
	s.completedStatus, s.completedMessage = statusValue, message
	s.cleanupCanceled = ctx.Err() != nil
	_, s.cleanupBounded = ctx.Deadline()
	return true, s.completeErr
}

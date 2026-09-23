package pipelinerunsvc

import (
	"context"
	"errors"
	"io"
	"io/fs"
	"log/slog"
	"strings"
	"testing"

	pipelinerunport "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/port"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/model"
	"github.com/leoninew/pomelo-orbit/internal/repository"
)

func TestDeletePipelineRunRemovesTerminalRecordWhenRunDirectoryIsMissing(t *testing.T) {
	run := pipelineRunForDeletion(status.WorkStatusRanToCompletion)
	pipelineRunStore := &pipelineRunDeletionStore{run: run}
	workspace := &pipelineRunDeletionWorkspace{removeErr: fs.ErrNotExist}
	service := newPipelineRunDeletionService(pipelineRunStore, workspace)

	if err := service.DeletePipelineRun(context.Background(), "user-1", "project-1", run.Id); err != nil {
		t.Fatalf("DeletePipelineRun returned error: %v", err)
	}
	if !pipelineRunStore.deleted {
		t.Fatal("pipeline run record was not deleted")
	}
	if workspace.removedRunId != run.Id {
		t.Fatalf("removed run Id = %q, want %q", workspace.removedRunId, run.Id)
	}
}

func TestDeletePipelineRunRejectsActiveRun(t *testing.T) {
	run := pipelineRunForDeletion(status.WorkStatusRunning)
	pipelineRunStore := &pipelineRunDeletionStore{run: run}
	workspace := &pipelineRunDeletionWorkspace{}
	service := newPipelineRunDeletionService(pipelineRunStore, workspace)

	err := service.DeletePipelineRun(context.Background(), "user-1", "project-1", run.Id)
	if err == nil || !strings.Contains(err.Error(), "Cannot delete pipeline run") {
		t.Fatalf("DeletePipelineRun error = %v, want active run validation error", err)
	}
	if pipelineRunStore.deleted {
		t.Fatal("active pipeline run must not be deleted")
	}
	if workspace.removedRunId != "" {
		t.Fatal("active pipeline run files must not be removed")
	}
}

func TestDeletePipelineRunDeletesRecordBeforeFileRemovalFails(t *testing.T) {
	run := pipelineRunForDeletion(status.WorkStatusFaulted)
	pipelineRunStore := &pipelineRunDeletionStore{run: run}
	workspace := &pipelineRunDeletionWorkspace{removeErr: errors.New("permission denied")}
	service := newPipelineRunDeletionService(pipelineRunStore, workspace)

	err := service.DeletePipelineRun(context.Background(), "user-1", "project-1", run.Id)
	if err == nil || !strings.Contains(err.Error(), "Failed to delete pipeline run files") {
		t.Fatalf("DeletePipelineRun error = %v, want file cleanup error", err)
	}
	if !pipelineRunStore.deleted {
		t.Fatal("pipeline run record must be deleted before file cleanup")
	}
	if workspace.removedRunId != run.Id {
		t.Fatalf("removed run Id = %q, want %q", workspace.removedRunId, run.Id)
	}
}

func TestDeletePipelineRunKeepsFilesWhenRecordDeletionFails(t *testing.T) {
	run := pipelineRunForDeletion(status.WorkStatusFaulted)
	pipelineRunStore := &pipelineRunDeletionStore{run: run, deleteErr: errors.New("db delete failed")}
	workspace := &pipelineRunDeletionWorkspace{}
	service := newPipelineRunDeletionService(pipelineRunStore, workspace)

	err := service.DeletePipelineRun(context.Background(), "user-1", "project-1", run.Id)
	if err == nil || !strings.Contains(err.Error(), "Failed to delete pipeline run") {
		t.Fatalf("DeletePipelineRun error = %v, want DB deletion error", err)
	}
	if pipelineRunStore.deleted {
		t.Fatal("pipeline run record must not be marked deleted when store fails")
	}
	if workspace.removedRunId != "" {
		t.Fatalf("workspace files must not be touched when DB delete fails, got %q", workspace.removedRunId)
	}
}

func pipelineRunForDeletion(runStatus string) model.PipelineRun {
	projectId := "project-1"
	environmentId, targetType, targetRevision := "environment-1", model.EnvironmentTargetTypeLocal, int64(1)
	return model.PipelineRun{Id: "run-1", ProjectId: &projectId, Status: runStatus, EnvironmentId: &environmentId, EnvironmentTargetType: &targetType, EnvironmentTargetRevision: &targetRevision}
}

func newPipelineRunDeletionService(pipelineRunStore *pipelineRunDeletionStore, workspace *pipelineRunDeletionWorkspace) Service {
	return Service{
		store: stores{
			project:     pipelineRunDeletionProjectStore{},
			pipelineRun: pipelineRunStore,
		},
		environments: pipelineRunDeletionEnvironmentStore{},
		workspace:    workspace,
		logger:       slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
}

type pipelineRunDeletionProjectStore struct {
	repository.ProjectReader
}

func (pipelineRunDeletionProjectStore) Project(context.Context, string) (model.Project, error) {
	return model.Project{Id: "project-1"}, nil
}

func (pipelineRunDeletionProjectStore) IsProjectMember(context.Context, string, string) (bool, error) {
	return true, nil
}

type pipelineRunDeletionStore struct {
	repository.PipelineRunStore
	run       model.PipelineRun
	deleted   bool
	deleteErr error
}

func (s *pipelineRunDeletionStore) PipelineRun(context.Context, string, string) (model.PipelineRun, error) {
	return s.run, nil
}

func (s *pipelineRunDeletionStore) DeletePipelineRun(context.Context, string, string) error {
	if s.deleteErr != nil {
		return s.deleteErr
	}
	s.deleted = true
	return nil
}

type pipelineRunDeletionWorkspace struct {
	pipelinerunport.Workspace
	removedRunId string
	removeErr    error
}

func (w *pipelineRunDeletionWorkspace) RemoveRunFiles(runId string) error {
	w.removedRunId = runId
	return w.removeErr
}

type pipelineRunDeletionEnvironmentStore struct {
	repository.EnvironmentStore
	environment *model.Environment
}

func (store pipelineRunDeletionEnvironmentStore) EnvironmentByProject(context.Context, string) (model.Environment, error) {
	if store.environment != nil {
		return *store.environment, nil
	}
	return model.Environment{Id: "environment-1", ProjectId: "project-1", TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: "/srv/orbit", TargetRevision: 1}, nil
}

func (w *pipelineRunDeletionWorkspace) WorkspaceForTarget(context.Context, model.Environment) (pipelinerunport.Workspace, error) {
	return w, nil
}

func TestDeletePipelineRunRefusesChangedTargetWithoutTouchingFiles(t *testing.T) {
	run := pipelineRunForDeletion(status.WorkStatusRanToCompletion)
	store := &pipelineRunDeletionStore{run: run}
	workspace := &pipelineRunDeletionWorkspace{}
	service := newPipelineRunDeletionService(store, workspace)
	environment := model.Environment{Id: "environment-1", ProjectId: "project-1",
		TargetType: model.EnvironmentTargetTypeLocal, WorkspaceRoot: "/srv/new-root", TargetRevision: 2}
	service.environments = pipelineRunDeletionEnvironmentStore{environment: &environment}
	err := service.DeletePipelineRun(context.Background(), "user-1", "project-1", run.Id)
	if err == nil || !strings.Contains(err.Error(), "changed") || store.deleted || workspace.removedRunId != "" {
		t.Fatalf("delete after target change: err=%v deleted=%v removed=%q", err, store.deleted, workspace.removedRunId)
	}
}

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

	if err := service.DeletePipelineRun(context.Background(), "user-1", run.Id); err != nil {
		t.Fatalf("DeletePipelineRun returned error: %v", err)
	}
	if !pipelineRunStore.deleted {
		t.Fatal("pipeline run record was not deleted")
	}
	if workspace.removedRunID != run.Id {
		t.Fatalf("removed run ID = %q, want %q", workspace.removedRunID, run.Id)
	}
}

func TestDeletePipelineRunRejectsActiveRun(t *testing.T) {
	run := pipelineRunForDeletion(status.WorkStatusRunning)
	pipelineRunStore := &pipelineRunDeletionStore{run: run}
	workspace := &pipelineRunDeletionWorkspace{}
	service := newPipelineRunDeletionService(pipelineRunStore, workspace)

	err := service.DeletePipelineRun(context.Background(), "user-1", run.Id)
	if err == nil || !strings.Contains(err.Error(), "Cannot delete pipeline run") {
		t.Fatalf("DeletePipelineRun error = %v, want active run validation error", err)
	}
	if pipelineRunStore.deleted {
		t.Fatal("active pipeline run must not be deleted")
	}
	if workspace.removedRunID != "" {
		t.Fatal("active pipeline run files must not be removed")
	}
}

func TestDeletePipelineRunKeepsRecordWhenFileRemovalFails(t *testing.T) {
	run := pipelineRunForDeletion(status.WorkStatusFaulted)
	pipelineRunStore := &pipelineRunDeletionStore{run: run}
	workspace := &pipelineRunDeletionWorkspace{removeErr: errors.New("permission denied")}
	service := newPipelineRunDeletionService(pipelineRunStore, workspace)

	err := service.DeletePipelineRun(context.Background(), "user-1", run.Id)
	if err == nil || !strings.Contains(err.Error(), "Failed to delete pipeline run files") {
		t.Fatalf("DeletePipelineRun error = %v, want file cleanup error", err)
	}
	if pipelineRunStore.deleted {
		t.Fatal("pipeline run record must remain when file cleanup fails")
	}
}

func pipelineRunForDeletion(runStatus string) model.PipelineRun {
	projectID := "project-1"
	return model.PipelineRun{Id: "run-1", ProjectId: &projectID, Status: runStatus}
}

func newPipelineRunDeletionService(pipelineRunStore *pipelineRunDeletionStore, workspace *pipelineRunDeletionWorkspace) Service {
	return Service{
		store: stores{
			project:     pipelineRunDeletionProjectStore{},
			pipelineRun: pipelineRunStore,
		},
		workspace: workspace,
		logger:    slog.New(slog.NewTextHandler(io.Discard, nil)),
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
	run     model.PipelineRun
	deleted bool
}

func (s *pipelineRunDeletionStore) PipelineRun(context.Context, string) (model.PipelineRun, error) {
	return s.run, nil
}

func (s *pipelineRunDeletionStore) DeletePipelineRun(context.Context, string) error {
	s.deleted = true
	return nil
}

type pipelineRunDeletionWorkspace struct {
	pipelinerunport.Workspace
	removedRunID string
	removeErr    error
}

func (w *pipelineRunDeletionWorkspace) RemoveRunFiles(runID string) error {
	w.removedRunID = runID
	return w.removeErr
}

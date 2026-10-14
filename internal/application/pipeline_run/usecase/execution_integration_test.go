//go:build !unit

package pipelinerunsvc

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	pipelinerundto "github.com/leoninew/pomelo-orbit/internal/application/pipeline_run/dto"
	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	"github.com/leoninew/pomelo-orbit/internal/config"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
	"github.com/leoninew/pomelo-orbit/internal/model"
	pipelinerunrepo "github.com/leoninew/pomelo-orbit/internal/repository/impl/sqlc/pipeline_run"
)

func TestRedeliveredRunRecoversFailedCleanupWithoutReplayingStages(t *testing.T) {
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = database.Close() })
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, config.DatabaseDriverSQLite); err != nil {
		t.Fatal(err)
	}
	if _, err := database.Exec(`INSERT INTO project (id, name, code) VALUES ('project-1', 'Project', 'project');
		INSERT INTO pipeline_run (id, project_id, repository_id, snapshot_id, pipeline_id, pipeline_name, pipeline_version, trigger, repository_ref, status)
		VALUES ('run-1', 'project-1', 'repo-1', 'snapshot-1', 'pipeline-1', 'Pipeline', 1, 'manual', 'main', 'waiting_to_run');`); err != nil {
		t.Fatal(err)
	}
	store := &interruptedExecutionStore{repo: pipelinerunrepo.NewRepository(database), unavailable: true}
	service := ExecutionService{executionStore: store, executionTimeout: time.Minute}
	ctx := context.Background()
	input := pipelinerundto.ExecutePipelineRunInput{ProjectId: "project-1", PipelineRunId: "run-1", Attempt: 1}
	if err := service.ExecutePipelineRun(ctx, input); err == nil {
		t.Fatal("expected storage failure")
	}
	run, err := store.repo.PipelineRun(ctx, "project-1", "run-1")
	if err != nil || run.Status != status.WorkStatusRunning {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	store.unavailable = false
	// A duplicate first delivery must not terminate an active execution.
	if err := service.ExecutePipelineRun(ctx, input); err == nil {
		t.Fatal("active execution was silently accepted")
	}
	input.Attempt = 2
	if err := service.ExecutePipelineRun(ctx, input); err != nil {
		t.Fatal(err)
	}
	run, err = store.repo.PipelineRun(ctx, "project-1", "run-1")
	if err != nil || run.Status != status.WorkStatusFaulted || run.FinishedAt == nil {
		t.Fatalf("run=%+v err=%v", run, err)
	}
	active, err := store.repo.RepositoryHasActivePipelineRun(ctx, "project-1", "repo-1")
	if err != nil || active {
		t.Fatalf("repository remains blocked: active=%v err=%v", active, err)
	}
	// Already canceled records keep their user-selected terminal state.
	if _, err := database.Exec(`UPDATE pipeline_run SET status = 'canceled' WHERE id = 'run-1'`); err != nil {
		t.Fatal(err)
	}
	if err := service.ExecutePipelineRun(ctx, input); err != nil {
		t.Fatal(err)
	}
	run, err = store.repo.PipelineRun(ctx, "project-1", "run-1")
	if err != nil || run.Status != status.WorkStatusCanceled {
		t.Fatalf("cancellation overwritten: run=%+v err=%v", run, err)
	}
}

type interruptedExecutionStore struct {
	pipelineExecutionStore
	repo        pipelinerunrepo.Repository
	unavailable bool
}

func (s *interruptedExecutionStore) BeginPipelineRun(ctx context.Context, projectId, id string) (bool, error) {
	return s.repo.BeginPipelineRun(ctx, projectId, id)
}
func (s *interruptedExecutionStore) PipelineRun(ctx context.Context, projectId, id string) (model.PipelineRun, error) {
	if s.unavailable {
		return model.PipelineRun{}, errors.New("read unavailable")
	}
	return s.repo.PipelineRun(ctx, projectId, id)
}
func (s *interruptedExecutionStore) CompletePipelineRun(ctx context.Context, projectId, id, state, message string) (bool, error) {
	if s.unavailable {
		return false, errors.New("completion unavailable")
	}
	return s.repo.CompletePipelineRun(ctx, projectId, id, state, message)
}

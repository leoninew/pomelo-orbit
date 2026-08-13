package taskrepo

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"

	status "github.com/leoninew/pomelo-orbit/internal/common/constant"
	db "github.com/leoninew/pomelo-orbit/internal/infrastructure/database"
)

func openTestDb(t *testing.T) *sql.DB {
	t.Helper()
	database, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	database.SetMaxOpenConns(1)
	if err := db.MigrateUp(database, "sqlite"); err != nil {
		t.Fatal(err)
	}
	return database
}

func TestRepositoryClaimComplete(t *testing.T) {
	database := openTestDb(t)
	defer func() { _ = database.Close() }()

	repo := NewRepository(database)
	ctx := context.Background()
	if err := repo.Enqueue(ctx, "task-1", status.TaskTypePipelineRunExecute, `{"pipeline_run_id":"run-1"}`, 1); err != nil {
		t.Fatal(err)
	}

	claimed, err := repo.ClaimNext(ctx, "worker-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if claimed == nil {
		t.Fatal("expected task to be claimed")
		return
	}
	if claimed.Id != "task-1" {
		t.Fatalf("unexpected task id: %s", claimed.Id)
	}

	if err := repo.Complete(ctx, claimed.Id); err != nil {
		t.Fatal(err)
	}

	completed, err := repo.FindById(ctx, claimed.Id)
	if err != nil {
		t.Fatal(err)
	}
	if completed.Status != status.TaskSucceeded {
		t.Fatalf("unexpected status: %s", completed.Status)
	}
}

func TestRepositoryFailTerminatesTask(t *testing.T) {
	database := openTestDb(t)
	defer func() { _ = database.Close() }()

	repo := NewRepository(database)
	ctx := context.Background()
	if err := repo.Enqueue(ctx, "task-1", status.TaskTypePipelineRunExecute, `{}`, 1); err != nil {
		t.Fatal(err)
	}

	claimed, err := repo.ClaimNext(ctx, "worker-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if err := repo.Fail(ctx, claimed.Id, "first failure"); err != nil {
		t.Fatal(err)
	}
	failed, err := repo.FindById(ctx, claimed.Id)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != status.TaskFailed {
		t.Fatalf("expected failed task, got %s", failed.Status)
	}
}

func TestRepositoryFailRequeuesUntilFrozenAttemptBudgetIsExhausted(t *testing.T) {
	database := openTestDb(t)
	defer func() { _ = database.Close() }()

	repo := NewRepository(database)
	ctx := context.Background()
	if err := repo.Enqueue(ctx, "task-1", status.TaskTypePipelineRunExecute, `{}`, 2); err != nil {
		t.Fatal(err)
	}

	firstAttempt, err := repo.ClaimNext(ctx, "worker-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if firstAttempt == nil || firstAttempt.Attempts != 1 || firstAttempt.MaxAttempts != 2 {
		t.Fatalf("unexpected first claim: %+v", firstAttempt)
	}
	if err := repo.Fail(ctx, firstAttempt.Id, "first failure"); err != nil {
		t.Fatal(err)
	}

	requeued, err := repo.FindById(ctx, firstAttempt.Id)
	if err != nil {
		t.Fatal(err)
	}
	if requeued.Status != status.TaskPending || requeued.Attempts != 1 || requeued.MaxAttempts != 2 || requeued.FinishedAt != nil {
		t.Fatalf("unexpected requeued task: %+v", requeued)
	}

	secondAttempt, err := repo.ClaimNext(ctx, "worker-1", time.Minute)
	if err != nil {
		t.Fatal(err)
	}
	if secondAttempt == nil || secondAttempt.Attempts != 2 {
		t.Fatalf("unexpected second claim: %+v", secondAttempt)
	}
	if err := repo.Fail(ctx, secondAttempt.Id, "second failure"); err != nil {
		t.Fatal(err)
	}

	failed, err := repo.FindById(ctx, secondAttempt.Id)
	if err != nil {
		t.Fatal(err)
	}
	if failed.Status != status.TaskFailed || failed.Attempts != 2 || failed.MaxAttempts != 2 || failed.FinishedAt == nil {
		t.Fatalf("unexpected terminal task: %+v", failed)
	}
}
